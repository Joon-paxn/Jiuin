package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/config"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type fakeMusicRunner struct {
	mutex sync.Mutex
	calls [][]string
	fail  bool
	cover bool
}

func (runner *fakeMusicRunner) Run(_ context.Context, executable string, arguments ...string) ([]byte, error) {
	runner.mutex.Lock()
	runner.calls = append(runner.calls, append([]string{executable}, arguments...))
	runner.mutex.Unlock()
	if runner.fail {
		return []byte("transcoder rejected input"), errors.New("exit status 7")
	}
	if executable == "ffprobe-test" {
		return []byte(`{"streams":[{"codec_type":"audio"}],"format":{"duration":"180.4","tags":{"title":"Test Song","artist":"Test Artist","album":"Test Album","album_artist":"Album Artist","genre":"Test Genre","date":"2026"}}}`), nil
	}
	outputPath := arguments[len(arguments)-1]
	if strings.Contains(outputPath, "covers") {
		if runner.cover {
			return nil, os.WriteFile(outputPath, []byte("generated cover"), 0o600)
		}
		return nil, errors.New("no cover stream")
	}
	if err := os.WriteFile(outputPath, []byte("generated audio"), 0o600); err != nil {
		return nil, err
	}
	return nil, nil
}

func TestMusicUploadExtractsOptionalEmbeddedCover(t *testing.T) {
	service, repo, runner, directory := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)
	runner.cover = true
	service.Start(context.Background())
	defer service.Stop()

	result, err := service.Upload(context.Background(), UploadInput{
		Filename: "with-cover.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCcover-source"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	task := waitForCompletedTask(t, service, result.TaskID)
	track, err := service.GetMusic(context.Background(), task.MusicID)
	if err != nil {
		t.Fatalf("GetMusic: %v", err)
	}
	if track.Cover == "" || !strings.Contains(track.Cover, task.MusicID) {
		t.Fatalf("cover URL = %q, want public extracted cover", track.Cover)
	}
	if _, err := os.Stat(filepath.Join(directory, "covers", task.MusicID+".jpg")); err != nil {
		t.Fatalf("extracted cover is absent: %v", err)
	}
}

func newMusicProcessingTestService(t *testing.T) (*musicProcessingService, repository.ManagedMusicRepository, *fakeMusicRunner, string) {
	t.Helper()
	directory := t.TempDir()
	repo, err := repository.NewSQLiteMusicRepository(directory)
	if err != nil {
		t.Fatalf("NewSQLiteMusicRepository: %v", err)
	}
	service := NewMusicProcessingService(repo, config.MusicConfig{
		Directory: directory, MaxUploadSize: 1024 * 1024, FFmpegPath: "ffmpeg-test", FFprobePath: "ffprobe-test",
		FullBitrate: "320k", LiteBitrate: "128k", OutputCodec: "libmp3lame", WorkerCount: 1,
	}, slog.New(slog.NewTextHandler(io.Discard, nil))).(*musicProcessingService)
	runner := &fakeMusicRunner{}
	service.runner = runner
	return service, repo, runner, directory
}

func TestMusicUploadCreatesAsynchronousTaskAndPublishesBothRenditions(t *testing.T) {
	service, repo, runner, directory := newMusicProcessingTestService(t)
	service.Start(context.Background())
	defer service.Stop()
	defer closeMusicRepository(t, repo)

	result, err := service.Upload(context.Background(), UploadInput{
		Filename: "source.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCvalid-audio-source"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.TaskID == "" || result.Status != model.MusicTaskPending {
		t.Fatalf("upload result = %#v, want pending task", result)
	}

	task := waitForCompletedTask(t, service, result.TaskID)
	if task.Status != model.MusicTaskCompleted || task.Progress != 100 || task.MusicID == "" {
		t.Fatalf("completed task = %#v", task)
	}
	track, err := service.GetMusic(context.Background(), task.MusicID)
	if err != nil {
		t.Fatalf("GetMusic: %v", err)
	}
	if track.Title != "Test Song" || track.Artist != "Test Artist" || track.DurationSeconds != 180 {
		t.Fatalf("public track metadata = %#v", track)
	}
	if track.Cover != "" {
		t.Fatalf("cover = %q, want absent when extraction fails", track.Cover)
	}
	if !strings.Contains(track.Audio.Full, task.MusicID) || !strings.Contains(track.Audio.Lite, task.MusicID) {
		t.Fatalf("audio URLs = %#v", track.Audio)
	}
	for _, expected := range []string{
		filepath.Join(directory, "original"), filepath.Join(directory, "full"), filepath.Join(directory, "lite"),
	} {
		entries, err := os.ReadDir(expected)
		if err != nil || len(entries) == 0 {
			t.Fatalf("expected published asset under %s, entries=%v err=%v", expected, entries, err)
		}
	}

	runner.mutex.Lock()
	defer runner.mutex.Unlock()
	transcodeOutputs := make([]string, 0, 2)
	for _, call := range runner.calls {
		if call[0] == "ffmpeg-test" {
			transcodeOutputs = append(transcodeOutputs, call[len(call)-1])
		}
	}
	if len(transcodeOutputs) < 2 {
		t.Fatalf("ffmpeg transcode calls = %#v, want full and lite outputs", transcodeOutputs)
	}
	for _, output := range transcodeOutputs[:2] {
		if !strings.HasSuffix(output, ".part.mp3") {
			t.Fatalf("temporary FFmpeg output = %q, want a .part.mp3 suffix", output)
		}
	}
}

func TestMusicUploadRejectsInvalidAndOversizedFiles(t *testing.T) {
	service, repo, _, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)

	_, err := service.Upload(context.Background(), UploadInput{Filename: "../../evil.mp3", ContentType: "audio/mpeg", Reader: strings.NewReader("ID3payload")})
	if !errors.Is(err, ErrInvalidMusicUpload) {
		t.Fatalf("traversal upload error = %v, want ErrInvalidMusicUpload", err)
	}
	_, err = service.Upload(context.Background(), UploadInput{Filename: "malware.mp3", ContentType: "audio/mpeg", Reader: strings.NewReader("MZnot-audio")})
	if !errors.Is(err, ErrInvalidMusicUpload) {
		t.Fatalf("executable upload error = %v, want ErrInvalidMusicUpload", err)
	}
	_, err = service.Upload(context.Background(), UploadInput{Filename: "wrong-type.flac", ContentType: "audio/mpeg", Reader: strings.NewReader("fLaCvalid-audio")})
	if !errors.Is(err, ErrInvalidMusicUpload) {
		t.Fatalf("MIME mismatch upload error = %v, want ErrInvalidMusicUpload", err)
	}
	service.config.MaxUploadSize = 4
	_, err = service.Upload(context.Background(), UploadInput{Filename: "too-large.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCmore-than-four")})
	if !errors.Is(err, ErrUploadTooLarge) {
		t.Fatalf("oversized upload error = %v, want ErrUploadTooLarge", err)
	}
}

func TestMusicUploadProbeRespectsUploadLimitBoundary(t *testing.T) {
	service, repo, _, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)
	service.config.MaxUploadSize = 4

	result, err := service.Upload(context.Background(), UploadInput{
		Filename: "exact.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaC"),
	})
	if err != nil {
		t.Fatalf("exact-limit Upload: %v", err)
	}
	if result.Status != model.MusicTaskPending {
		t.Fatalf("exact-limit status = %q, want %q", result.Status, model.MusicTaskPending)
	}

	_, err = service.Upload(context.Background(), UploadInput{
		Filename: "over.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCx"),
	})
	if !errors.Is(err, ErrUploadTooLarge) {
		t.Fatalf("over-limit Upload error = %v, want ErrUploadTooLarge", err)
	}
}

func TestMusicUploadReusesExistingTaskForDuplicateSource(t *testing.T) {
	service, repo, _, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)
	service.Start(context.Background())
	defer service.Stop()

	input := UploadInput{Filename: "same.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCsame-source")}
	first, err := service.Upload(context.Background(), input)
	if err != nil {
		t.Fatalf("first Upload: %v", err)
	}
	second, err := service.Upload(context.Background(), UploadInput{Filename: "copy.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCsame-source")})
	if err != nil {
		t.Fatalf("second Upload: %v", err)
	}
	if !second.Reused || second.TaskID != first.TaskID {
		t.Fatalf("duplicate result = %#v, want task reuse %q", second, first.TaskID)
	}
}

func TestMusicTaskIsClaimedOnlyOnceWhenEnqueuedMoreThanOnce(t *testing.T) {
	service, repo, runner, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)
	service.Start(context.Background())
	defer service.Stop()

	result, err := service.Upload(context.Background(), UploadInput{
		Filename: "claim-once.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCclaim-once-source"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	// This mirrors recovery plus an upload enqueue racing for the same durable
	// task. Exactly one worker is allowed to claim and transcode it.
	service.enqueue(result.TaskID)
	task := waitForCompletedTask(t, service, result.TaskID)
	if task.Status != model.MusicTaskCompleted {
		t.Fatalf("task status = %q, want completed", task.Status)
	}

	runner.mutex.Lock()
	defer runner.mutex.Unlock()
	transcodes := 0
	for _, call := range runner.calls {
		if call[0] == "ffmpeg-test" && !strings.Contains(call[len(call)-1], "covers") {
			transcodes++
		}
	}
	if transcodes != 2 {
		t.Fatalf("ffmpeg transcode calls = %d, want exactly full plus lite once", transcodes)
	}
}

func TestMusicUploadDefaultsMissingMetadataToUnknown(t *testing.T) {
	service, repo, _, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)
	metadata, err := service.probe(context.Background(), "unused")
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if defaultMetadata(metadata.Title) != "Test Song" || defaultMetadata("") != "未知" {
		t.Fatalf("metadata fallback behavior is incorrect")
	}
}

func TestMusicProbeRejectsNonFiniteOrExcessiveDuration(t *testing.T) {
	service, repo, _, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)

	for _, duration := range []string{"NaN", "Inf", "604801"} {
		duration := duration
		runnerOverride := commandRunnerFunc(func(_ context.Context, executable string, _ ...string) ([]byte, error) {
			if executable != "ffprobe-test" {
				return nil, errors.New("unexpected executable")
			}
			return []byte(`{"streams":[{"codec_type":"audio"}],"format":{"duration":"` + duration + `","tags":{}}}`), nil
		})
		service.runner = runnerOverride
		if _, err := service.probe(context.Background(), "unused"); err == nil {
			t.Fatalf("probe accepted invalid duration %q", duration)
		}
	}
}

func TestCappedCommandOutputDoesNotGrowWithoutBound(t *testing.T) {
	output := &cappedCommandOutput{}
	input := bytes.Repeat([]byte("x"), commandOutputLimit*2)
	if count, err := output.Write(input); err != nil || count != len(input) {
		t.Fatalf("Write = (%d, %v), want (%d, nil)", count, err, len(input))
	}
	if actual := len(output.Bytes()); actual != commandOutputLimit {
		t.Fatalf("stored command output = %d bytes, want %d", actual, commandOutputLimit)
	}
}

func TestMusicProcessingStopCancelsActiveWorkerContext(t *testing.T) {
	service, repo, _, _ := newMusicProcessingTestService(t)
	defer closeMusicRepository(t, repo)

	started := make(chan struct{})
	cancelled := make(chan struct{})
	service.runner = commandRunnerFunc(func(ctx context.Context, executable string, _ ...string) ([]byte, error) {
		if executable == "ffprobe-test" {
			return []byte(`{"streams":[{"codec_type":"audio"}],"format":{"duration":"1","tags":{}}}`), nil
		}
		select {
		case <-started:
		default:
			close(started)
		}
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	})
	service.Start(context.Background())
	result, err := service.Upload(context.Background(), UploadInput{
		Filename: "cancel.flac", ContentType: "audio/flac", Reader: strings.NewReader("fLaCcancel-source"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start processing")
	}
	service.Stop()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel the active FFmpeg context")
	}
	if _, err := service.GetTask(context.Background(), result.TaskID); err != nil {
		t.Fatalf("GetTask after stop: %v", err)
	}
}

type commandRunnerFunc func(context.Context, string, ...string) ([]byte, error)

func (function commandRunnerFunc) Run(ctx context.Context, executable string, arguments ...string) ([]byte, error) {
	return function(ctx, executable, arguments...)
}

func waitForCompletedTask(t *testing.T, service MusicProcessingService, taskID string) model.MusicTask {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := service.GetTask(context.Background(), taskID)
		if err == nil && (task.Status == model.MusicTaskCompleted || task.Status == model.MusicTaskFailed) {
			return task
		}
		time.Sleep(10 * time.Millisecond)
	}
	task, err := service.GetTask(context.Background(), taskID)
	t.Fatalf("task %s did not finish: task=%#v err=%v", taskID, task, err)
	return model.MusicTask{}
}

func closeMusicRepository(t *testing.T, repository repository.ManagedMusicRepository) {
	t.Helper()
	if closer, ok := repository.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			t.Errorf("close repository: %v", err)
		}
	}
}
