package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type streamMusicService struct {
	asset model.MusicAsset
	err   error
}

type managedStreamMusicService struct {
	streamMusicService
}

func (stub managedStreamMusicService) MaxUploadSize() int64  { return 1024 }
func (stub managedStreamMusicService) Start(context.Context) {}
func (stub managedStreamMusicService) Stop()                 {}
func (stub managedStreamMusicService) Upload(context.Context, service.UploadInput) (service.UploadResult, error) {
	return service.UploadResult{}, nil
}
func (stub managedStreamMusicService) GetTask(context.Context, string) (model.MusicTask, error) {
	return model.MusicTask{}, repository.ErrMusicTaskNotFound
}
func (stub managedStreamMusicService) GetMusic(context.Context, string) (model.PublicMusicTrack, error) {
	return model.PublicMusicTrack{}, service.ErrMusicNotFound
}
func (stub managedStreamMusicService) ListPublic(context.Context) ([]model.PublicMusicTrack, error) {
	return nil, nil
}
func (stub managedStreamMusicService) OpenVariant(_ context.Context, _ string, variant string) (model.MusicAsset, error) {
	if variant != "full" {
		return model.MusicAsset{}, service.ErrMusicNotFound
	}
	return stub.asset, stub.err
}

type uploadMusicService struct {
	managedStreamMusicService
	upload func(context.Context, service.UploadInput) (service.UploadResult, error)
}

func (stub uploadMusicService) Upload(ctx context.Context, input service.UploadInput) (service.UploadResult, error) {
	return stub.upload(ctx, input)
}

type deadlineResponseRecorder struct {
	*httptest.ResponseRecorder
	deadline time.Time
}

func (recorder *deadlineResponseRecorder) SetWriteDeadline(deadline time.Time) error {
	recorder.deadline = deadline
	return nil
}

type uploadDeadlineResponseRecorder struct {
	*httptest.ResponseRecorder
	readDeadline time.Time
}

func (recorder *uploadDeadlineResponseRecorder) SetReadDeadline(deadline time.Time) error {
	recorder.readDeadline = deadline
	return nil
}

func (stub streamMusicService) List(context.Context) ([]model.MusicTrack, error) {
	return nil, nil
}

func (stub streamMusicService) Open(context.Context, string) (model.MusicAsset, error) {
	return stub.asset, stub.err
}

func TestNewMusicHandlerAssignsService(t *testing.T) {
	t.Parallel()

	musicService := streamMusicService{}
	handler := NewMusicHandler(musicService, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if handler.service == nil {
		t.Fatal("service must be assigned")
	}
	if _, ok := handler.service.(streamMusicService); !ok {
		t.Fatalf("service type = %T, want streamMusicService", handler.service)
	}
}

func TestMusicHandlerStreamsByteRanges(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "Test Artist - Test Song.mp3")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatalf("write music fixture: %v", err)
	}
	handler := NewMusicHandler(streamMusicService{asset: model.MusicAsset{Path: path, Name: filepath.Base(path)}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/test-track", nil)
	request.Header.Set("Range", "bytes=2-5")
	request.SetPathValue("id", "test-track")
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusPartialContent)
	}
	if recorder.Body.String() != "2345" {
		t.Fatalf("body = %q, want 2345", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("content type = %q, want audio/mpeg", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Cache-Control") == "" {
		t.Fatal("expected cache header")
	}
}

func TestManagedMusicHandlerStreamsOneMultipartUploadFile(t *testing.T) {
	contents := bytes.Repeat([]byte("a"), 64*1024)
	var receivedName, receivedType string
	var receivedContents []byte
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(_ context.Context, input service.UploadInput) (service.UploadResult, error) {
			var err error
			receivedName, receivedType = input.Filename, input.ContentType
			receivedContents, err = io.ReadAll(input.Reader)
			if err != nil {
				return service.UploadResult{}, err
			}
			return service.UploadResult{TaskID: "12345678-1234-4234-9234-123456789abc", Status: model.MusicTaskPending}, nil
		},
	}, 128*1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	body, contentType := handlerMusicMultipartBody(t, "file", "song.flac", contents)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want %d; body=%s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
	if receivedName != "song.flac" || receivedType != "application/octet-stream" {
		t.Fatalf("upload metadata = (%q, %q)", receivedName, receivedType)
	}
	if !bytes.Equal(receivedContents, contents) {
		t.Fatalf("upload contents differ: received %d bytes, want %d", len(receivedContents), len(contents))
	}
}

func TestManagedMusicHandlerClearsOnlyUploadReadDeadline(t *testing.T) {
	body, contentType := handlerMusicMultipartBody(t, "file", "song.flac", []byte("fLaCsource"))
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(_ context.Context, input service.UploadInput) (service.UploadResult, error) {
			_, err := io.ReadAll(input.Reader)
			if err != nil {
				return service.UploadResult{}, err
			}
			return service.UploadResult{TaskID: "12345678-1234-4234-9234-123456789abc", Status: model.MusicTaskPending}, nil
		},
	}, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	recorder := &uploadDeadlineResponseRecorder{ResponseRecorder: httptest.NewRecorder(), readDeadline: time.Now().Add(time.Minute)}

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want %d; body=%s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
	if !recorder.readDeadline.IsZero() {
		t.Fatalf("read deadline = %v, want zero deadline for upload", recorder.readDeadline)
	}
}

func TestManagedMusicHandlerRejectsMultipartWithAdditionalParts(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile("file", "song.flac")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := file.Write([]byte("fLaCsource")); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.WriteField("unexpected", "value"); err != nil {
		t.Fatalf("create extra part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	called := false
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(_ context.Context, input service.UploadInput) (service.UploadResult, error) {
			called = true
			_, err := io.ReadAll(input.Reader)
			return service.UploadResult{}, err
		},
	}, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("additional multipart part status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !called {
		t.Fatal("upload service was not invoked to consume and validate the file part")
	}
}

func TestManagedMusicHandlerRejectsTraversalFilenameBeforeService(t *testing.T) {
	body, contentType := handlerMusicMultipartBody(t, "file", "../../song.flac", []byte("fLaCsource"))
	called := false
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(context.Context, service.UploadInput) (service.UploadResult, error) {
			called = true
			return service.UploadResult{}, nil
		},
	}, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("traversal filename status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if called {
		t.Fatal("traversal filename reached the upload service")
	}
}

func TestManagedMusicHandlerRejectsNonFormDataFilePartBeforeService(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"name": "file", "filename": "song.flac"})
	part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Disposition": {contentDisposition}, "Content-Type": {"audio/flac"}})
	if err != nil {
		t.Fatalf("create attachment part: %v", err)
	}
	if _, err := part.Write([]byte("fLaCsource")); err != nil {
		t.Fatalf("write attachment part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	called := false
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(context.Context, service.UploadInput) (service.UploadResult, error) {
			called = true
			return service.UploadResult{}, nil
		},
	}, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("attachment multipart part status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if called {
		t.Fatal("non-form-data part reached the upload service")
	}
}

func TestManagedMusicHandlerBoundsFileReaderToConfiguredLimit(t *testing.T) {
	body, contentType := handlerMusicMultipartBody(t, "file", "song.flac", bytes.Repeat([]byte("a"), 4096))
	var bytesRead int
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(_ context.Context, input service.UploadInput) (service.UploadResult, error) {
			contents, err := io.ReadAll(input.Reader)
			bytesRead = len(contents)
			if err != nil {
				return service.UploadResult{}, err
			}
			return service.UploadResult{}, service.ErrUploadTooLarge
		},
	}, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized upload status = %d, want %d; body=%s", recorder.Code, http.StatusRequestEntityTooLarge, recorder.Body.String())
	}
	if bytesRead != 1025 {
		t.Fatalf("handler passed %d file bytes to the service, want 1025 limit sentinel", bytesRead)
	}
}

func TestManagedMusicHandlerRejectsBodyBeyondMultipartAllowance(t *testing.T) {
	body, contentType := handlerMusicMultipartBody(t, "file", "song.flac", bytes.Repeat([]byte("a"), 2300))
	called := false
	handler := NewManagedMusicHandler(uploadMusicService{
		upload: func(context.Context, service.UploadInput) (service.UploadResult, error) {
			called = true
			return service.UploadResult{}, nil
		},
	}, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	// Set a known length after building the multipart body so the request is
	// rejected before a streaming reader can be instantiated.
	request.ContentLength = musicUploadBodyLimit(1024) + 1
	recorder := httptest.NewRecorder()

	handler.Upload(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized multipart body status = %d, want %d; body=%s", recorder.Code, http.StatusRequestEntityTooLarge, recorder.Body.String())
	}
	if called {
		t.Fatal("body exceeding multipart allowance reached the upload service")
	}
}

func handlerMusicMultipartBody(t *testing.T, field, filename string, contents []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatalf("write multipart contents: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}
	return body, writer.FormDataContentType()
}

func TestManagedMusicHandlerStreamsFullVariantByteRanges(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "song.mp3")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatalf("write music fixture: %v", err)
	}
	managed := managedStreamMusicService{streamMusicService: streamMusicService{asset: model.MusicAsset{Path: path, Name: filepath.Base(path)}}}
	handler := NewManagedMusicHandler(managed, 1024, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/full/12345678-1234-4234-9234-123456789abc.mp3", nil)
	request.Header.Set("Range", "bytes=3-6")
	request.SetPathValue("variant", "full")
	request.SetPathValue("file", "12345678-1234-4234-9234-123456789abc.mp3")
	recorder := httptest.NewRecorder()

	handler.StreamManaged(recorder, request)

	if recorder.Code != http.StatusPartialContent || recorder.Body.String() != "3456" {
		t.Fatalf("managed range = status %d body %q, want 206 / 3456", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("managed content type = %q, want audio/mpeg", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("Accept-Ranges = %q, want bytes", recorder.Header().Get("Accept-Ranges"))
	}
}

func TestMusicHandlerStreamSupportsResponseDeadlines(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "Test Artist - Test Song.mp3")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatalf("write music fixture: %v", err)
	}
	handler := NewMusicHandler(streamMusicService{asset: model.MusicAsset{Path: path, Name: filepath.Base(path)}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/test-track", nil)
	request.SetPathValue("id", "test-track")
	recorder := &deadlineResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !recorder.deadline.IsZero() {
		t.Fatalf("write deadline = %v, want zero deadline for stream", recorder.deadline)
	}
}

func TestMusicHandlerRejectsUnknownTrack(t *testing.T) {
	t.Parallel()

	handler := NewMusicHandler(streamMusicService{err: service.ErrMusicNotFound}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/unknown", nil)
	request.SetPathValue("id", "unknown")
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestMusicHandlerPreservesUnexpectedOpenErrors(t *testing.T) {
	t.Parallel()

	handler := NewMusicHandler(streamMusicService{err: errors.New("disk offline")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/media/music/broken", nil)
	request.SetPathValue("id", "broken")
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
