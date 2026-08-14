package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type MusicHandler struct {
	service           service.MusicService
	processingService service.MusicProcessingService
	maxUploadSize     int64
	logger            *slog.Logger
}

const musicUploadMultipartOverhead int64 = 1 << 20

var (
	errInvalidMusicMultipartRequest = errors.New("invalid music multipart request")
	errUnexpectedMusicMultipartPart = errors.New("unexpected music multipart part")
)

func NewMusicHandler(musicService service.MusicService, logger *slog.Logger) MusicHandler {
	return MusicHandler{service: musicService, logger: logger}
}

func NewManagedMusicHandler(processingService service.MusicProcessingService, maxUploadSize int64, logger *slog.Logger) MusicHandler {
	return MusicHandler{service: processingService, processingService: processingService, maxUploadSize: maxUploadSize, logger: logger}
}

func (handler MusicHandler) List(w http.ResponseWriter, r *http.Request) {
	tracks, err := handler.service.List(r.Context())
	if err != nil {
		handler.logger.Error("failed to load music list", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load music list")
		return
	}

	response.Success(w, tracks)
}

// ListPublic is the current player-facing endpoint. Its payload contains only
// stable HTTP paths, not host-specific filesystem locations.
func (handler MusicHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	if handler.processingService == nil {
		response.Success(w, []model.PublicMusicTrack{})
		return
	}
	tracks, err := handler.processingService.ListPublic(r.Context())
	if err != nil {
		handler.logger.Error("failed to load managed music list", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load music list")
		return
	}
	response.Success(w, tracks)
}

func (handler MusicHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	if handler.processingService == nil {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	track, err := handler.processingService.GetMusic(r.Context(), r.PathValue("id"))
	if errors.Is(err, service.ErrMusicNotFound) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to load music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load music track")
		return
	}
	response.Success(w, track)
}

func (handler MusicHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if handler.processingService == nil || handler.maxUploadSize <= 0 {
		response.Error(w, http.StatusNotFound, "music upload is not configured")
		return
	}
	// Server.ReadTimeout covers a whole request and is intentionally short for
	// ordinary API calls. A configured large upload can legitimately take longer
	// than that, so clear it only after routing/authentication/rate-limit checks.
	// The production Nginx upload locations retain their 10-minute body timeout;
	// direct development traffic is loopback-only.
	if err := http.NewResponseController(w).SetReadDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		handler.logger.Warn("failed to clear music upload read deadline", "error", err)
	}
	// ParseMultipartForm would retain up to the configured file size in RAM.
	// Keep multipart parsing streaming instead: the processing service copies
	// the one file part straight into its private temporary file.
	bodyLimit := musicUploadBodyLimit(handler.maxUploadSize)
	if r.ContentLength > bodyLimit {
		response.Error(w, http.StatusRequestEntityTooLarge, "uploaded file exceeds the configured size limit")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	multipartReader, err := r.MultipartReader()
	if err != nil {
		response.Error(w, http.StatusBadRequest, "request must contain exactly one file field")
		return
	}
	part, err := multipartReader.NextPart()
	if err != nil {
		if isMusicUploadBodyTooLarge(err) {
			response.Error(w, http.StatusRequestEntityTooLarge, "uploaded file exceeds the configured size limit")
			return
		}
		response.Error(w, http.StatusBadRequest, "request must contain exactly one file field")
		return
	}
	filename, err := originalMultipartFilename(part)
	if err != nil || part.FormName() != "file" || !validMultipartMusicFilename(filename, part.FileName()) {
		response.Error(w, http.StatusBadRequest, "uploaded file could not be read")
		return
	}

	result, err := handler.processingService.Upload(r.Context(), service.UploadInput{
		Filename: part.FileName(), ContentType: part.Header.Get("Content-Type"),
		// The processing service owns the file-size sentinel so it can return a
		// precise 413. Do not wrap this in a second io.LimitReader: at exactly
		// the configured limit it would hide the multipart EOF and skip the
		// trailing-part validation below.
		Reader: &musicMultipartFileReader{part: part, reader: multipartReader, limit: musicUploadReadLimit(handler.maxUploadSize)},
	})
	if errors.Is(err, service.ErrUploadTooLarge) || isMusicUploadBodyTooLarge(err) {
		response.Error(w, http.StatusRequestEntityTooLarge, "uploaded file exceeds the configured size limit")
		return
	}
	if errors.Is(err, errInvalidMusicMultipartRequest) || errors.Is(err, errUnexpectedMusicMultipartPart) {
		response.Error(w, http.StatusBadRequest, "request must contain exactly one file field")
		return
	}
	if errors.Is(err, service.ErrInvalidMusicUpload) {
		response.Error(w, http.StatusUnsupportedMediaType, "uploaded file is not a supported audio file")
		return
	}
	if err != nil {
		handler.logger.Error("music upload failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "music upload could not be accepted")
		return
	}
	// Upload processing is asynchronous. Duplicate completed sources reuse the
	// existing resource but still receive a clear completed response.
	response.Accepted(w, result)
}

// musicMultipartFileReader verifies that a successfully consumed file was the
// entire multipart body. The validation runs while the processing service is
// still reading its temporary source, before it can create a durable task.
// This keeps the public contract strict (exactly one `file` field) without
// buffering a potentially large upload in the HTTP handler.
type musicMultipartFileReader struct {
	part     *multipart.Part
	reader   *multipart.Reader
	limit    int64
	read     int64
	finished bool
}

func (reader *musicMultipartFileReader) Read(buffer []byte) (int, error) {
	if reader.limit > 0 {
		remaining := reader.limit - reader.read
		if remaining <= 0 {
			return 0, io.EOF
		}
		if int64(len(buffer)) > remaining {
			buffer = buffer[:remaining]
		}
	}
	count, err := reader.part.Read(buffer)
	reader.read += int64(count)
	if err != nil && !errors.Is(err, io.EOF) {
		if isMusicUploadBodyTooLarge(err) {
			return count, err
		}
		return count, fmt.Errorf("%w: %v", errInvalidMusicMultipartRequest, err)
	}
	if !errors.Is(err, io.EOF) || reader.finished {
		return count, err
	}
	reader.finished = true
	if validationErr := reader.validateTrailingParts(); validationErr != nil {
		return count, validationErr
	}
	return count, err
}

func (reader *musicMultipartFileReader) validateTrailingParts() error {
	part, err := reader.reader.NextPart()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		if isMusicUploadBodyTooLarge(err) {
			return err
		}
		return fmt.Errorf("%w: %v", errInvalidMusicMultipartRequest, err)
	}
	_ = part.Close()
	return errUnexpectedMusicMultipartPart
}

func originalMultipartFilename(part *multipart.Part) (string, error) {
	disposition, parameters, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(disposition, "form-data") {
		return "", errInvalidMusicMultipartRequest
	}
	return parameters["filename"], nil
}

func validMultipartMusicFilename(original, normalized string) bool {
	return original != "" && original == normalized && !strings.ContainsAny(original, "/\\") && !strings.ContainsRune(original, '\x00') && len([]rune(original)) <= 255
}

func musicUploadBodyLimit(maxUploadSize int64) int64 {
	if maxUploadSize > (1<<63-1)-musicUploadMultipartOverhead {
		return 1<<63 - 1
	}
	return maxUploadSize + musicUploadMultipartOverhead
}

func musicUploadReadLimit(maxUploadSize int64) int64 {
	if maxUploadSize >= (1<<63)-1 {
		return (1 << 63) - 1
	}
	return maxUploadSize + 1
}

func isMusicUploadBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func (handler MusicHandler) Task(w http.ResponseWriter, r *http.Request) {
	if handler.processingService == nil {
		response.Error(w, http.StatusNotFound, "music task was not found")
		return
	}
	task, err := handler.processingService.GetTask(r.Context(), r.PathValue("task_id"))
	if errors.Is(err, repository.ErrMusicTaskNotFound) {
		response.Error(w, http.StatusNotFound, "music task was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to load music task", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load music task")
		return
	}
	// Internal FFmpeg diagnostics, source hashes, storage paths, and timestamps
	// stay server-side. Ordinary callers only receive task state and its public
	// completed music ID.
	response.Success(w, model.MusicTaskResponse{
		TaskID: task.ID, Status: task.Status, Progress: task.Progress, MusicID: task.MusicID,
	})
}

// Stream serves a resolved local track by opaque ID. http.ServeContent provides
// seek/range support required by the browser audio element.
func (handler MusicHandler) Stream(w http.ResponseWriter, r *http.Request) {
	// Audio playback may legitimately outlast the API's normal write timeout.
	// Clear only this response deadline after routing/auth/rate-limit checks;
	// regular API endpoints retain the server-wide write deadline.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		handler.logger.Warn("failed to clear music stream write deadline", "error", err)
	}

	asset, err := handler.service.Open(r.Context(), r.PathValue("id"))
	if errors.Is(err, service.ErrMusicNotFound) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to resolve music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to resolve music track")
		return
	}

	beforeOpen, err := os.Lstat(asset.Path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && !beforeOpen.Mode().IsRegular()) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to inspect music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to inspect music track")
		return
	}

	file, err := os.Open(asset.Path)
	if errors.Is(err, os.ErrNotExist) {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to open music track", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to open music track")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(beforeOpen, info) {
		if err != nil {
			handler.logger.Error("failed to inspect music track", "error", err)
		}
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("ETag", fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixNano()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if contentType := audioContentType(asset.Name); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, asset.Name, info.ModTime(), file)
}

func (handler MusicHandler) StreamManaged(w http.ResponseWriter, r *http.Request) {
	if handler.processingService == nil {
		response.Error(w, http.StatusNotFound, "music track was not found")
		return
	}
	variant := r.PathValue("variant")
	fileName := r.PathValue("file")
	var id string
	switch variant {
	case "full", "lite":
		if !strings.HasSuffix(fileName, ".mp3") {
			response.Error(w, http.StatusNotFound, "music resource was not found")
			return
		}
		id = strings.TrimSuffix(fileName, ".mp3")
	case "covers":
		if !strings.HasSuffix(fileName, ".jpg") {
			response.Error(w, http.StatusNotFound, "music resource was not found")
			return
		}
		id = strings.TrimSuffix(fileName, ".jpg")
		variant = "cover"
	default:
		response.Error(w, http.StatusNotFound, "music resource was not found")
		return
	}
	asset, err := handler.processingService.OpenVariant(r.Context(), id, variant)
	if errors.Is(err, service.ErrMusicNotFound) {
		response.Error(w, http.StatusNotFound, "music resource was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to resolve music asset", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to resolve music asset")
		return
	}
	handler.serveAsset(w, r, asset)
}

func (handler MusicHandler) serveAsset(w http.ResponseWriter, r *http.Request, asset model.MusicAsset) {
	// Audio playback may legitimately outlast the API's normal write timeout.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		handler.logger.Warn("failed to clear music stream write deadline", "error", err)
	}

	beforeOpen, err := os.Lstat(asset.Path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && !beforeOpen.Mode().IsRegular()) {
		response.Error(w, http.StatusNotFound, "music resource was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to inspect music resource", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to resolve music resource")
		return
	}
	file, err := os.Open(asset.Path)
	if errors.Is(err, os.ErrNotExist) {
		response.Error(w, http.StatusNotFound, "music resource was not found")
		return
	}
	if err != nil {
		handler.logger.Error("failed to open music resource", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to resolve music resource")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(beforeOpen, info) {
		response.Error(w, http.StatusNotFound, "music resource was not found")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("ETag", fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixNano()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if contentType := mediaContentType(asset.Name); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, asset.Name, info.ModTime(), file)
}

func audioContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".m4a":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	default:
		return ""
	}
}

func mediaContentType(name string) string {
	if strings.EqualFold(filepath.Ext(name), ".jpg") || strings.EqualFold(filepath.Ext(name), ".jpeg") {
		return "image/jpeg"
	}
	return audioContentType(name)
}
