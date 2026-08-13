package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type managedMusicServiceStub struct {
	uploaded service.UploadInput
}

func (stub *managedMusicServiceStub) MaxUploadSize() int64  { return 1024 * 1024 }
func (stub *managedMusicServiceStub) Start(context.Context) {}
func (stub *managedMusicServiceStub) Stop()                 {}
func (stub *managedMusicServiceStub) List(context.Context) ([]model.MusicTrack, error) {
	return []model.MusicTrack{}, nil
}
func (stub *managedMusicServiceStub) Open(context.Context, string) (model.MusicAsset, error) {
	return model.MusicAsset{}, service.ErrMusicNotFound
}
func (stub *managedMusicServiceStub) Upload(_ context.Context, input service.UploadInput) (service.UploadResult, error) {
	contents, _ := io.ReadAll(input.Reader)
	stub.uploaded = service.UploadInput{Filename: input.Filename, ContentType: input.ContentType, Reader: bytes.NewReader(contents)}
	return service.UploadResult{TaskID: "12345678-1234-4234-9234-123456789abc", Status: model.MusicTaskPending}, nil
}
func (stub *managedMusicServiceStub) GetTask(context.Context, string) (model.MusicTask, error) {
	return model.MusicTask{ID: "12345678-1234-4234-9234-123456789abc", Status: model.MusicTaskProcessing, Progress: 50}, nil
}
func (stub *managedMusicServiceStub) ClaimTask(context.Context, string) (model.MusicTask, bool, error) {
	return model.MusicTask{}, false, nil
}
func (stub *managedMusicServiceStub) CompleteTask(context.Context, model.MusicTask, model.MusicRecord) error {
	return nil
}
func (stub *managedMusicServiceStub) GetMusic(context.Context, string) (model.PublicMusicTrack, error) {
	return model.PublicMusicTrack{}, service.ErrMusicNotFound
}
func (stub *managedMusicServiceStub) ListPublic(context.Context) ([]model.PublicMusicTrack, error) {
	return []model.PublicMusicTrack{{ID: "12345678-1234-4234-9234-123456789abc", Title: "Song", Artist: "Artist", Album: "Unknown", Audio: model.PublicAudioSources{Full: "/media/music/full/12345678-1234-4234-9234-123456789abc.mp3", Lite: "/media/music/lite/12345678-1234-4234-9234-123456789abc.mp3"}}}, nil
}
func (stub *managedMusicServiceStub) OpenVariant(context.Context, string, string) (model.MusicAsset, error) {
	return model.MusicAsset{}, service.ErrMusicNotFound
}

func TestRouterProtectsAndAcceptsMusicUploads(t *testing.T) {
	stub := &managedMusicServiceStub{}
	router := newManagedMusicTestRouter(stub)

	body, contentType := musicMultipartBody(t, "song.flac", []byte("fLaCvalid-content"))
	unauthorized := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	unauthorized.Header.Set("Content-Type", contentType)
	unauthorizedRecorder := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedRecorder, unauthorized)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized upload = %d, want %d", unauthorizedRecorder.Code, http.StatusUnauthorized)
	}

	body, contentType = musicMultipartBody(t, "song.flac", []byte("fLaCvalid-content"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer test-music-admin-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, want %d; body=%s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
	if stub.uploaded.Filename != "song.flac" || stub.uploaded.ContentType != "application/octet-stream" {
		t.Fatalf("uploaded input = %#v", stub.uploaded)
	}
	var envelope struct {
		Code int                  `json:"code"`
		Data service.UploadResult `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if envelope.Code != http.StatusAccepted || envelope.Data.Status != model.MusicTaskPending {
		t.Fatalf("upload envelope = %#v", envelope)
	}
}

func TestRouterAcceptsProtectedCompatibilityUploadAlias(t *testing.T) {
	stub := &managedMusicServiceStub{}
	router := newManagedMusicTestRouter(stub)
	body, contentType := musicMultipartBody(t, "song.flac", []byte("fLaCvalid-content"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/music/upload", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer test-music-admin-token")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("compatibility upload status = %d, want %d; body=%s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
}

func TestRouterExposesManagedMusicAndTaskEndpoints(t *testing.T) {
	router := newManagedMusicTestRouter(&managedMusicServiceStub{})
	for _, path := range []string{"/api/v1/music", "/api/v1/music/12345678-1234-4234-9234-123456789abc", "/api/v1/music/tasks/12345678-1234-4234-9234-123456789abc"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK && !(strings.Contains(path, "/music/") && recorder.Code == http.StatusNotFound) {
			t.Fatalf("GET %s = %d, body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestRouterTaskResponseDoesNotExposeProcessingDetails(t *testing.T) {
	router := newManagedMusicTestRouter(&managedMusicServiceStub{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/music/tasks/12345678-1234-4234-9234-123456789abc", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("task status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var envelope struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	if envelope.Code != http.StatusOK || envelope.Data["task_id"] == nil || envelope.Data["status"] == nil || envelope.Data["progress"] == nil {
		t.Fatalf("task envelope = %#v", envelope)
	}
	for _, privateField := range []string{"source_hash", "original_path", "error_type", "error_detail", "exit_code", "created_at", "updated_at"} {
		if _, exists := envelope.Data[privateField]; exists {
			t.Fatalf("task response exposed private field %q: %#v", privateField, envelope.Data)
		}
	}
}

func musicMultipartBody(t *testing.T, name string, contents []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatalf("write upload fixture: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func newManagedMusicTestRouter(music service.MusicProcessingService) http.Handler {
	return NewRouter(
		stubSiteService{}, music, stubStatisticsService{}, stubStatusService{}, stubLinkService{}, stubResourceService{},
		"test-shared-service-token", "test-music-admin-token", []string{"http://app.test"}, slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}
