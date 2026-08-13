package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/config"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

var (
	ErrInvalidMusicUpload = errors.New("invalid music upload")
	ErrUploadTooLarge     = errors.New("music upload is too large")
)

// UploadInput is created by the HTTP layer after it has imposed multipart
// limits. Filename is metadata only; it is never used as a storage path.
type UploadInput struct {
	Filename    string
	ContentType string
	Reader      io.Reader
}

type UploadResult struct {
	TaskID  string                `json:"task_id"`
	Status  model.MusicTaskStatus `json:"status"`
	MusicID string                `json:"music_id,omitempty"`
	Reused  bool                  `json:"reused,omitempty"`
}

type musicMetadata struct {
	Title           string
	Artist          string
	Album           string
	AlbumArtist     string
	Genre           string
	Year            string
	DurationSeconds int
}

type commandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type osCommandRunner struct{}

const (
	commandOutputLimit   = 64 << 10
	maxMetadataFieldSize = 1024
	uploadProbeSize      = 512
)

// cappedCommandOutput retains only enough process diagnostics for server-side
// troubleshooting. FFmpeg and FFprobe operate on untrusted files, so their
// stderr must never be allowed to grow without bound in process memory.
type cappedCommandOutput struct {
	mutex sync.Mutex
	data  []byte
}

func (output *cappedCommandOutput) Write(value []byte) (int, error) {
	output.mutex.Lock()
	defer output.mutex.Unlock()

	remaining := commandOutputLimit - len(output.data)
	if remaining > 0 {
		if remaining > len(value) {
			remaining = len(value)
		}
		output.data = append(output.data, value[:remaining]...)
	}
	// Accept and discard excess output. Returning a short write would make
	// os/exec surface an opaque I/O failure instead of the FFmpeg exit status.
	return len(value), nil
}

func (output *cappedCommandOutput) Bytes() []byte {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	return append([]byte(nil), output.data...)
}

func (osCommandRunner) Run(ctx context.Context, executable string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, executable, arguments...)
	output := &cappedCommandOutput{}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	return output.Bytes(), err
}

type MusicProcessingService interface {
	MusicService
	MaxUploadSize() int64
	Upload(context.Context, UploadInput) (UploadResult, error)
	GetTask(context.Context, string) (model.MusicTask, error)
	GetMusic(context.Context, string) (model.PublicMusicTrack, error)
	ListPublic(context.Context) ([]model.PublicMusicTrack, error)
	OpenVariant(context.Context, string, string) (model.MusicAsset, error)
	Start(context.Context)
	Stop()
}

type musicProcessingService struct {
	repository repository.ManagedMusicRepository
	config     config.MusicConfig
	logger     *slog.Logger
	runner     commandRunner

	queue   chan string
	stop    chan struct{}
	done    sync.WaitGroup
	started sync.Once
	closed  sync.Once

	lifecycleMutex sync.Mutex
	workerCancel   context.CancelFunc
}

func NewMusicProcessingService(repository repository.ManagedMusicRepository, settings config.MusicConfig, logger *slog.Logger) MusicProcessingService {
	if logger == nil {
		logger = slog.Default()
	}
	queueCapacity := settings.WorkerCount * 4
	if queueCapacity < 4 {
		queueCapacity = 4
	}
	return &musicProcessingService{
		repository: repository,
		config:     settings,
		logger:     logger,
		runner:     osCommandRunner{},
		queue:      make(chan string, queueCapacity),
		stop:       make(chan struct{}),
	}
}

func (service *musicProcessingService) MaxUploadSize() int64 {
	return service.config.MaxUploadSize
}

func (service *musicProcessingService) Start(ctx context.Context) {
	service.started.Do(func() {
		workerContext, cancel := context.WithCancel(ctx)
		service.lifecycleMutex.Lock()
		service.workerCancel = cancel
		service.lifecycleMutex.Unlock()
		// Normalize interrupted work before a local worker can claim anything.
		// This avoids a recovery reset racing an active conversion during startup.
		recoveredTaskIDs := service.recoverTasks(workerContext)
		for worker := 0; worker < service.config.WorkerCount; worker++ {
			service.done.Add(1)
			go service.work(workerContext, worker+1)
		}
		if len(recoveredTaskIDs) == 0 {
			return
		}
		// A large recovered queue must not delay HTTP startup. Workers are
		// already available to drain this producer while its durable IDs are
		// re-enqueued in order.
		service.done.Add(1)
		go func() {
			defer service.done.Done()
			for _, taskID := range recoveredTaskIDs {
				service.enqueue(taskID)
			}
		}()
	})
}

func (service *musicProcessingService) Stop() {
	service.closed.Do(func() {
		service.lifecycleMutex.Lock()
		if service.workerCancel != nil {
			service.workerCancel()
		}
		service.lifecycleMutex.Unlock()
		close(service.stop)
		service.done.Wait()
	})
}

func (service *musicProcessingService) recoverTasks(ctx context.Context) []string {
	tasks, err := service.repository.ListRecoverableTasks(ctx)
	if err != nil {
		service.logger.Error("music task recovery query failed", "error", err)
		return nil
	}
	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task.Status == model.MusicTaskProcessing {
			task.Status = model.MusicTaskPending
			task.Progress = 0
			task.ErrorType = ""
			task.ErrorDetail = ""
			task.ExitCode = 0
			if err := service.repository.UpdateTask(ctx, task); err != nil {
				service.logger.Error("music task recovery update failed", "task_id", task.ID, "error", err)
				continue
			}
		}
		taskIDs = append(taskIDs, task.ID)
	}
	return taskIDs
}

func (service *musicProcessingService) work(parent context.Context, worker int) {
	defer service.done.Done()
	for {
		select {
		case <-service.stop:
			return
		case <-parent.Done():
			return
		case taskID := <-service.queue:
			if taskID == "" {
				continue
			}
			service.processTask(parent, worker, taskID)
		}
	}
}

func (service *musicProcessingService) enqueue(taskID string) {
	select {
	case <-service.stop:
		return
	case service.queue <- taskID:
	}
}

func (service *musicProcessingService) Upload(ctx context.Context, input UploadInput) (UploadResult, error) {
	if input.Reader == nil || !validUploadedFilename(input.Filename) || !supportedMusicExtension(input.Filename) {
		return UploadResult{}, ErrInvalidMusicUpload
	}
	if !declaredMusicMIME(input.ContentType) || !musicMIMEMatchesExtension(input.Filename, input.ContentType) {
		return UploadResult{}, ErrInvalidMusicUpload
	}
	taskID, err := secureUUID()
	if err != nil {
		return UploadResult{}, err
	}
	service.logger.Info("music upload started", "task_id", taskID)

	storageRoot, err := safeMusicStorageRoot(service.config.Directory)
	if err != nil {
		return UploadResult{}, err
	}
	temporaryDirectory, err := safeJoin(storageRoot, "tmp")
	if err != nil {
		return UploadResult{}, fmt.Errorf("resolve upload temporary directory: %w", err)
	}
	temporaryFile, err := os.CreateTemp(temporaryDirectory, "upload-*.part")
	if err != nil {
		return UploadResult{}, fmt.Errorf("create upload temporary file: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	limited := &io.LimitedReader{R: input.Reader, N: uploadReadLimit(service.config.MaxUploadSize)}
	probeLimit := int64(uploadProbeSize)
	if limited.N < probeLimit {
		probeLimit = limited.N
	}
	probeBytes, err := io.ReadAll(io.LimitReader(limited, probeLimit))
	if err != nil {
		return UploadResult{}, fmt.Errorf("read upload: %w", err)
	}
	if int64(len(probeBytes)) > service.config.MaxUploadSize {
		_ = temporaryFile.Close()
		return UploadResult{}, ErrUploadTooLarge
	}
	if !looksLikeDeclaredAudio(probeBytes, filepath.Ext(input.Filename)) {
		_ = temporaryFile.Close()
		return UploadResult{}, ErrInvalidMusicUpload
	}
	hash := sha256.New()
	bytesWritten, err := io.Copy(io.MultiWriter(temporaryFile, hash), io.MultiReader(bytes.NewReader(probeBytes), limited))
	if err != nil {
		_ = temporaryFile.Close()
		return UploadResult{}, fmt.Errorf("write upload: %w", err)
	}
	if bytesWritten > service.config.MaxUploadSize {
		_ = temporaryFile.Close()
		return UploadResult{}, ErrUploadTooLarge
	}
	if err := temporaryFile.Sync(); err != nil {
		_ = temporaryFile.Close()
		return UploadResult{}, fmt.Errorf("sync upload: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return UploadResult{}, fmt.Errorf("close upload: %w", err)
	}

	sourceHash := hex.EncodeToString(hash.Sum(nil))
	if existing, err := service.repository.FindMusicByHash(ctx, sourceHash); err == nil {
		service.logger.Info("duplicate music upload reused", "upload_attempt_id", taskID, "source_hash", sourceHash, "music_id", existing.ID)
		return UploadResult{Status: model.MusicTaskCompleted, MusicID: existing.ID, Reused: true}, nil
	} else if !errors.Is(err, repository.ErrMusicNotFound) {
		return UploadResult{}, err
	}
	if existingTask, err := service.repository.FindTaskBySourceHash(ctx, sourceHash); err == nil {
		service.logger.Info("duplicate music upload joined existing task", "task_id", existingTask.ID, "upload_attempt_id", taskID, "source_hash", sourceHash)
		return UploadResult{TaskID: existingTask.ID, Status: existingTask.Status, MusicID: existingTask.MusicID, Reused: true}, nil
	} else if !errors.Is(err, repository.ErrMusicTaskNotFound) {
		return UploadResult{}, err
	}

	// ffprobe is the authoritative content validation step. MIME and magic
	// signatures are defence-in-depth filters, not the only determination.
	if _, err := service.probe(ctx, temporaryPath); err != nil {
		service.logger.Warn("music upload rejected by ffprobe", "error", err)
		return UploadResult{}, ErrInvalidMusicUpload
	}

	originalExtension := strings.ToLower(filepath.Ext(input.Filename))
	originalRelativePath := "original/" + taskID + originalExtension
	originalPath, err := safeJoin(storageRoot, originalRelativePath)
	if err != nil {
		return UploadResult{}, err
	}
	if err := os.Rename(temporaryPath, originalPath); err != nil {
		return UploadResult{}, fmt.Errorf("publish original upload: %w", err)
	}
	removeTemporary = false

	task := model.MusicTask{
		ID: taskID, Status: model.MusicTaskPending, Progress: 0,
		SourceHash: sourceHash, OriginalPath: originalRelativePath,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := service.repository.CreateTask(ctx, task); err != nil {
		_ = os.Remove(originalPath)
		var duplicateTask repository.DuplicateMusicTaskError
		if errors.As(err, &duplicateTask) {
			existingTask := duplicateTask.ExistingTask()
			service.logger.Info("duplicate music upload joined concurrent task", "task_id", existingTask.ID, "source_hash", sourceHash)
			return UploadResult{TaskID: existingTask.ID, Status: existingTask.Status, MusicID: existingTask.MusicID, Reused: true}, nil
		}
		if existingTask, lookupErr := service.repository.FindTaskBySourceHash(ctx, sourceHash); lookupErr == nil {
			service.logger.Info("duplicate music upload joined concurrent task", "task_id", existingTask.ID, "source_hash", sourceHash)
			return UploadResult{TaskID: existingTask.ID, Status: existingTask.Status, MusicID: existingTask.MusicID, Reused: true}, nil
		}
		return UploadResult{}, err
	}
	service.logger.Info("music upload completed", "task_id", task.ID, "source_hash", sourceHash)
	// Do not keep the HTTP response open if all workers are busy. The durable
	// task is already committed; restart recovery also re-enqueues it.
	go service.enqueue(task.ID)
	return UploadResult{TaskID: task.ID, Status: task.Status}, nil
}

func (service *musicProcessingService) processTask(parent context.Context, worker int, taskID string) {
	processingTimeout := service.config.ProcessingTimeout
	if processingTimeout <= 0 {
		processingTimeout = config.DefaultMusicProcessingTimeout
	}
	ctx, cancel := context.WithTimeout(parent, processingTimeout)
	defer cancel()
	task, claimed, err := service.repository.ClaimTask(ctx, taskID)
	if errors.Is(err, repository.ErrMusicTaskNotFound) {
		return
	}
	if err != nil {
		service.logger.Error("music task claim failed", "task_id", taskID, "error", err)
		return
	}
	if !claimed {
		return
	}
	service.logger.Info("music task started", "task_id", taskID, "worker", worker)

	record, err := service.process(task, ctx)
	if err != nil {
		failure := task
		failure.Status = model.MusicTaskFailed
		failure.Progress = 0
		failure.ErrorType, failure.ErrorDetail, failure.ExitCode = taskFailure(err)
		if updateErr := service.repository.UpdateTask(ctx, failure); updateErr != nil {
			service.logger.Error("music task failure update failed", "task_id", taskID, "error", updateErr)
		}
		service.logger.Error("music task failed", "task_id", taskID, "error_type", failure.ErrorType, "exit_code", failure.ExitCode, "ffmpeg_diagnostic", processingDiagnostic(err), "error", err)
		return
	}

	completed := task
	completed.Status = model.MusicTaskCompleted
	completed.Progress = 100
	completed.MusicID = record.ID
	completed.ErrorType, completed.ErrorDetail, completed.ExitCode = "", "", 0
	if err := service.repository.CompleteTask(ctx, completed, record); err != nil {
		failure := task
		failure.Status = model.MusicTaskFailed
		failure.Progress = 0
		failure.ErrorType, failure.ErrorDetail, failure.ExitCode = "database", "music could not be published", 0
		if existing, lookupErr := service.repository.FindMusicByHash(ctx, task.SourceHash); lookupErr == nil {
			if existing.ID != record.ID {
				service.removePublishedAssets(record)
			}
			completed.MusicID = existing.ID
			if updateErr := service.repository.UpdateTask(ctx, completed); updateErr == nil {
				service.logger.Info("music task completed from existing record", "task_id", taskID, "music_id", existing.ID)
				return
			}
		}
		// The task is terminal after a failed publication. Remove generated
		// assets so a database failure cannot leave unreferenced public files.
		service.removePublishedAssets(record)
		_ = service.repository.UpdateTask(ctx, failure)
		service.logger.Error("music task completion failed", "task_id", taskID, "error", err)
		return
	}
	service.logger.Info("music task completed", "task_id", taskID, "music_id", record.ID)
}

func (service *musicProcessingService) process(task model.MusicTask, ctx context.Context) (model.MusicRecord, error) {
	storageRoot, err := safeMusicStorageRoot(service.config.Directory)
	if err != nil {
		return model.MusicRecord{}, err
	}
	originalPath, err := safeJoin(storageRoot, task.OriginalPath)
	if err != nil {
		return model.MusicRecord{}, err
	}
	metadata, err := service.probe(ctx, originalPath)
	if err != nil {
		return model.MusicRecord{}, fmt.Errorf("probe original: %w", err)
	}
	musicID, err := secureUUID()
	if err != nil {
		return model.MusicRecord{}, err
	}
	fullRelativePath := "full/" + musicID + ".mp3"
	liteRelativePath := "lite/" + musicID + ".mp3"
	fullPath, err := safeJoin(storageRoot, fullRelativePath)
	if err != nil {
		return model.MusicRecord{}, err
	}
	litePath, err := safeJoin(storageRoot, liteRelativePath)
	if err != nil {
		return model.MusicRecord{}, err
	}
	// Keep the final media extension on temporary files so FFmpeg can select the
	// MP3 muxer from the output filename before the atomic publish rename.
	fullTemporaryPath := strings.TrimSuffix(fullPath, ".mp3") + ".part.mp3"
	liteTemporaryPath := strings.TrimSuffix(litePath, ".mp3") + ".part.mp3"
	published := false
	defer os.Remove(fullTemporaryPath)
	defer os.Remove(liteTemporaryPath)
	defer func() {
		if !published {
			_ = os.Remove(fullPath)
			_ = os.Remove(litePath)
		}
	}()

	service.setProgress(ctx, task, 20)
	service.logger.Info("ffmpeg full rendition started", "task_id", task.ID)
	if err := service.transcode(ctx, originalPath, fullTemporaryPath, service.config.FullBitrate); err != nil {
		return model.MusicRecord{}, fmt.Errorf("full rendition: %w", err)
	}
	if err := os.Rename(fullTemporaryPath, fullPath); err != nil {
		return model.MusicRecord{}, fmt.Errorf("publish full rendition: %w", err)
	}
	service.logger.Info("ffmpeg full rendition completed", "task_id", task.ID)
	service.setProgress(ctx, task, 60)
	service.logger.Info("ffmpeg lite rendition started", "task_id", task.ID)
	if err := service.transcode(ctx, originalPath, liteTemporaryPath, service.config.LiteBitrate); err != nil {
		return model.MusicRecord{}, fmt.Errorf("lite rendition: %w", err)
	}
	if err := os.Rename(liteTemporaryPath, litePath); err != nil {
		return model.MusicRecord{}, fmt.Errorf("publish lite rendition: %w", err)
	}
	service.logger.Info("ffmpeg lite rendition completed", "task_id", task.ID)
	service.setProgress(ctx, task, 85)

	coverRelativePath := ""
	coverRelativePathCandidate := "covers/" + musicID + ".jpg"
	coverPath, err := safeJoin(storageRoot, coverRelativePathCandidate)
	if err != nil {
		return model.MusicRecord{}, err
	}
	if err := service.extractCover(ctx, originalPath, coverPath); err != nil {
		// A cover is optional. The task remains valid without it, while detailed
		// diagnostics stay exclusively in the structured server log.
		service.logger.Info("embedded cover not extracted", "task_id", task.ID, "error", err)
		_ = os.Remove(coverPath)
	} else if info, err := os.Lstat(coverPath); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		coverRelativePath = coverRelativePathCandidate
	}

	fullInfo, err := os.Stat(fullPath)
	if err != nil {
		return model.MusicRecord{}, fmt.Errorf("stat full rendition: %w", err)
	}
	liteInfo, err := os.Stat(litePath)
	if err != nil {
		return model.MusicRecord{}, fmt.Errorf("stat lite rendition: %w", err)
	}
	record := model.MusicRecord{
		ID: musicID, SourceHash: task.SourceHash,
		Title: defaultMetadata(metadata.Title), Artist: defaultMetadata(metadata.Artist), Album: defaultMetadata(metadata.Album),
		AlbumArtist: defaultMetadata(metadata.AlbumArtist), Genre: defaultMetadata(metadata.Genre), Year: defaultMetadata(metadata.Year),
		DurationSeconds: metadata.DurationSeconds, CoverPath: coverRelativePath, OriginalPath: task.OriginalPath,
		FullPath: fullRelativePath, LitePath: liteRelativePath, FullSize: fullInfo.Size(), LiteSize: liteInfo.Size(), CreatedAt: time.Now().UTC(),
	}
	published = true
	return record, nil
}

func (service *musicProcessingService) removePublishedAssets(record model.MusicRecord) {
	root, err := safeMusicStorageRoot(service.config.Directory)
	if err != nil {
		return
	}
	for _, relativePath := range []string{record.FullPath, record.LitePath, record.CoverPath} {
		if relativePath == "" {
			continue
		}
		if path, err := safeJoin(root, relativePath); err == nil {
			_ = os.Remove(path)
		}
	}
}

func (service *musicProcessingService) setProgress(ctx context.Context, task model.MusicTask, progress int) {
	task.Progress = progress
	task.Status = model.MusicTaskProcessing
	if err := service.repository.UpdateTask(ctx, task); err != nil {
		service.logger.Warn("music task progress update failed", "task_id", task.ID, "error", err)
	}
}

func (service *musicProcessingService) transcode(ctx context.Context, inputPath, outputPath, bitrate string) error {
	output, err := service.runner.Run(ctx, service.config.FFmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-i", inputPath,
		"-vn", "-map", "0:a:0", "-c:a", service.config.OutputCodec, "-b:a", bitrate, outputPath,
	)
	if err != nil {
		return ffmpegCommandError{err: err, output: string(output)}
	}
	return nil
}

func (service *musicProcessingService) extractCover(ctx context.Context, inputPath, outputPath string) error {
	output, err := service.runner.Run(ctx, service.config.FFmpegPath,
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-i", inputPath,
		"-an", "-map", "0:v:0", "-frames:v", "1", "-c:v", "mjpeg", outputPath,
	)
	if err != nil {
		return ffmpegCommandError{err: err, output: string(output)}
	}
	return nil
}

func (service *musicProcessingService) probe(ctx context.Context, path string) (musicMetadata, error) {
	output, err := service.runner.Run(ctx, service.config.FFprobePath,
		"-v", "error", "-show_entries", "format=duration:format_tags=title,artist,album,album_artist,genre,date,year", "-show_streams", "-of", "json", path,
	)
	if err != nil {
		return musicMetadata{}, fmt.Errorf("ffprobe execution: %w", ffmpegCommandError{err: err, output: string(output)})
	}
	var result struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
		} `json:"streams"`
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return musicMetadata{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	hasAudio := false
	for _, stream := range result.Streams {
		if stream.CodecType == "audio" {
			hasAudio = true
			break
		}
	}
	if !hasAudio {
		return musicMetadata{}, errors.New("input has no audio stream")
	}
	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil || math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 || duration > maxMusicDuration.Seconds() {
		return musicMetadata{}, errors.New("input has no valid duration")
	}
	tags := lowerCaseKeys(result.Format.Tags)
	return musicMetadata{
		Title: tagValue(tags, "title"), Artist: tagValue(tags, "artist"), Album: tagValue(tags, "album"),
		AlbumArtist: tagValue(tags, "album_artist", "albumartist"), Genre: tagValue(tags, "genre"), Year: tagValue(tags, "year", "date"),
		DurationSeconds: int(duration + 0.5),
	}, nil
}

func (service *musicProcessingService) GetTask(ctx context.Context, id string) (model.MusicTask, error) {
	return service.repository.GetTask(ctx, id)
}

func (service *musicProcessingService) GetMusic(ctx context.Context, id string) (model.PublicMusicTrack, error) {
	record, err := service.repository.GetMusic(ctx, id)
	if errors.Is(err, repository.ErrMusicNotFound) {
		return model.PublicMusicTrack{}, ErrMusicNotFound
	}
	if err != nil {
		return model.PublicMusicTrack{}, err
	}
	return publicTrackFromRecord(record), nil
}

func (service *musicProcessingService) ListPublic(ctx context.Context) ([]model.PublicMusicTrack, error) {
	records, err := service.repository.ListMusic(ctx)
	if err != nil {
		return nil, err
	}
	tracks := make([]model.PublicMusicTrack, 0, len(records))
	for _, record := range records {
		tracks = append(tracks, publicTrackFromRecord(record))
	}
	return tracks, nil
}

func (service *musicProcessingService) OpenVariant(ctx context.Context, id, variant string) (model.MusicAsset, error) {
	asset, err := service.repository.OpenManagedAsset(ctx, id, variant)
	if errors.Is(err, repository.ErrMusicNotFound) {
		return model.MusicAsset{}, ErrMusicNotFound
	}
	return asset, err
}

func publicTrackFromRecord(record model.MusicRecord) model.PublicMusicTrack {
	track := model.PublicMusicTrack{
		ID: record.ID, Title: record.Title, Artist: record.Artist, Album: record.Album, AlbumArtist: record.AlbumArtist,
		Genre: record.Genre, Year: record.Year, DurationSeconds: record.DurationSeconds, FullSize: record.FullSize, LiteSize: record.LiteSize,
		CreatedAt: record.CreatedAt, Audio: model.PublicAudioSources{
			Full: "/media/music/full/" + record.ID + ".mp3", Lite: "/media/music/lite/" + record.ID + ".mp3",
		},
	}
	if record.CoverPath != "" {
		track.Cover = "/media/music/covers/" + record.ID + ".jpg"
	}
	return track
}

func (service *musicProcessingService) List(ctx context.Context) ([]model.MusicTrack, error) {
	return service.repository.List(ctx)
}

func (service *musicProcessingService) Open(ctx context.Context, id string) (model.MusicAsset, error) {
	return service.OpenVariant(ctx, id, "full")
}

type ffmpegCommandError struct {
	err    error
	output string
}

func (error ffmpegCommandError) Error() string { return error.err.Error() }
func (error ffmpegCommandError) Unwrap() error { return error.err }

func taskFailure(err error) (errorType, publicDetail string, exitCode int) {
	var commandError ffmpegCommandError
	if errors.As(err, &commandError) {
		var exitError *exec.ExitError
		if errors.As(commandError.err, &exitError) {
			return "ffmpeg", "audio processing failed", exitError.ExitCode()
		}
		return "ffmpeg", "audio processing failed", 0
	}
	return "processing", "audio processing failed", 0
}

func processingDiagnostic(err error) string {
	var commandError ffmpegCommandError
	if !errors.As(err, &commandError) {
		return ""
	}
	// Diagnostics remain in structured server logs only. Truncate output so a
	// malformed file cannot turn one task failure into an oversized log event.
	value := strings.TrimSpace(commandError.output)
	if len(value) > 2048 {
		return value[:2048]
	}
	return value
}

func safeMusicStorageRoot(directory string) (string, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err == nil {
		return resolvedRoot, nil
	}
	info, statErr := os.Lstat(root)
	if statErr == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return root, nil
	}
	return "", err
}

const maxMusicDuration = 7 * 24 * time.Hour

func uploadReadLimit(maximum int64) int64 {
	if maximum >= math.MaxInt64 {
		return math.MaxInt64
	}
	return maximum + 1
}

func safeJoin(root, relativePath string) (string, error) {
	if relativePath == "" || strings.Contains(relativePath, "\\") || strings.ContainsRune(relativePath, '\x00') {
		return "", errors.New("invalid storage path")
	}
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	if clean != relativePath || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", errors.New("storage path traversal")
	}
	path := filepath.Join(root, filepath.FromSlash(clean))
	relative, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		return "", errors.New("storage path escaped root")
	}
	if err := rejectSymlinkComponents(root, path); err != nil {
		return "", err
	}
	return path, nil
}

func rejectSymlinkComponents(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		return errors.New("storage path escaped root")
	}
	current := root
	for _, component := range strings.FieldsFunc(relative, func(r rune) bool { return r == '/' || r == '\\' }) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return fmt.Errorf("inspect storage path: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("storage path must not traverse a symlink")
		}
	}
	return nil
}

func secureUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("create secure identifier: %w", err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

func validUploadedFilename(name string) bool {
	base := filepath.Base(name)
	return name != "" && base == name && !strings.ContainsAny(name, "/\\") && !strings.ContainsRune(name, '\x00') && len([]rune(name)) <= 255
}

func supportedMusicExtension(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp3", ".flac", ".wav", ".ogg", ".m4a", ".aac":
		return true
	default:
		return false
	}
}

func declaredMusicMIME(value string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	switch mediaType {
	case "audio/mpeg", "audio/mp3", "audio/flac", "audio/x-flac", "audio/wav", "audio/x-wav", "audio/ogg", "application/ogg", "audio/mp4", "audio/x-m4a", "audio/aac", "audio/x-aac", "application/octet-stream":
		return true
	default:
		return false
	}
}

func musicMIMEMatchesExtension(filename, value string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	// Some multipart clients (including browsers for FLAC/AAC) intentionally
	// send the generic binary type. It is accepted only as an unknown claim;
	// magic-byte and FFprobe checks below remain mandatory.
	if mediaType == "application/octet-stream" {
		return true
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp3":
		return mediaType == "audio/mpeg" || mediaType == "audio/mp3"
	case ".flac":
		return mediaType == "audio/flac" || mediaType == "audio/x-flac"
	case ".wav":
		return mediaType == "audio/wav" || mediaType == "audio/x-wav"
	case ".ogg":
		return mediaType == "audio/ogg" || mediaType == "application/ogg"
	case ".m4a":
		return mediaType == "audio/mp4" || mediaType == "audio/x-m4a"
	case ".aac":
		return mediaType == "audio/aac" || mediaType == "audio/x-aac"
	default:
		return false
	}
}

func looksLikeDeclaredAudio(contents []byte, extension string) bool {
	if len(contents) < 2 || bytes.HasPrefix(contents, []byte("MZ")) || bytes.HasPrefix(contents, []byte("\x7fELF")) {
		return false
	}
	switch strings.ToLower(extension) {
	case ".mp3":
		return bytes.HasPrefix(contents, []byte("ID3")) || (contents[0] == 0xff && contents[1]&0xe0 == 0xe0)
	case ".flac":
		return len(contents) >= 4 && bytes.HasPrefix(contents, []byte("fLaC"))
	case ".wav":
		return len(contents) >= 12 && bytes.HasPrefix(contents, []byte("RIFF")) && bytes.Equal(contents[8:12], []byte("WAVE"))
	case ".ogg":
		return bytes.HasPrefix(contents, []byte("OggS"))
	case ".m4a":
		return len(contents) >= 12 && bytes.Equal(contents[4:8], []byte("ftyp"))
	case ".aac":
		return len(contents) >= 2 && contents[0] == 0xff && contents[1]&0xf6 == 0xf0
	default:
		return false
	}
}

func lowerCaseKeys(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key == "" || len(key) > maxMetadataFieldSize || len(value) > maxMetadataFieldSize || strings.ContainsRune(value, '\x00') {
			continue
		}
		result[key] = value
	}
	return result
}

func tagValue(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(tags[key]); value != "" {
			return value
		}
	}
	return ""
}

func defaultMetadata(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxMetadataFieldSize || strings.ContainsRune(value, '\x00') {
		return "未知"
	}
	return value
}
