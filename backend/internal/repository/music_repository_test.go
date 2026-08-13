package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

func TestSQLiteMusicRepositoryEnforcesOneTaskPerSourceAndCompletesAtomically(t *testing.T) {
	directory := t.TempDir()
	repo, err := NewSQLiteMusicRepository(directory)
	if err != nil {
		t.Fatalf("NewSQLiteMusicRepository: %v", err)
	}
	defer func() {
		if closer, ok := repo.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()

	ctx := context.Background()
	hashBytes := sha256.Sum256([]byte("same source"))
	hash := hex.EncodeToString(hashBytes[:])
	task := model.MusicTask{
		ID: "12345678-1234-4234-9234-123456789abc", Status: model.MusicTaskPending, SourceHash: hash,
		OriginalPath: "original/12345678-1234-4234-9234-123456789abc.flac",
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	duplicate := task
	duplicate.ID = "12345678-1234-4234-9234-123456789abd"
	if err := repo.CreateTask(ctx, duplicate); err == nil {
		t.Fatal("second source hash task unexpectedly succeeded")
	} else {
		var duplicateError DuplicateMusicTaskError
		if !errors.As(err, &duplicateError) || duplicateError.ExistingTask().ID != task.ID {
			t.Fatalf("duplicate task error = %v, want existing task %s", err, task.ID)
		}
	}

	claimed, won, err := repo.ClaimTask(ctx, task.ID)
	if err != nil || !won || claimed.Status != model.MusicTaskProcessing {
		t.Fatalf("ClaimTask = (%#v, %t, %v), want processing win", claimed, won, err)
	}
	if _, won, err := repo.ClaimTask(ctx, task.ID); err != nil || won {
		t.Fatalf("second ClaimTask won=%t err=%v, want no winner", won, err)
	}

	record := model.MusicRecord{
		ID: "12345678-1234-4234-9234-123456789abe", SourceHash: hash,
		Title: "Song", Artist: "Artist", Album: "Album", AlbumArtist: "未知", Genre: "未知", Year: "未知", DurationSeconds: 1,
		OriginalPath: task.OriginalPath, FullPath: "full/12345678-1234-4234-9234-123456789abe.mp3", LitePath: "lite/12345678-1234-4234-9234-123456789abe.mp3",
	}
	completed := claimed
	completed.Status, completed.Progress, completed.MusicID = model.MusicTaskCompleted, 100, record.ID
	if err := repo.CompleteTask(ctx, completed, record); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	storedTask, err := repo.GetTask(ctx, task.ID)
	if err != nil || storedTask.Status != model.MusicTaskCompleted || storedTask.MusicID != record.ID {
		t.Fatalf("stored task = %#v err=%v", storedTask, err)
	}
	if _, err := repo.GetMusic(ctx, record.ID); err != nil {
		t.Fatalf("completed music record not found: %v", err)
	}
}

func TestSQLiteMusicRepositoryRejectsUnsafeManagedStoragePaths(t *testing.T) {
	directory := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(directory, "full")); err != nil {
		t.Skipf("symbolic links are unavailable in this test environment: %v", err)
	}
	if _, err := NewSQLiteMusicRepository(directory); err == nil {
		t.Fatal("NewSQLiteMusicRepository accepted a symlinked storage directory")
	}

	hashBytes := sha256.Sum256([]byte("unsafe paths"))
	hash := hex.EncodeToString(hashBytes[:])
	record := model.MusicRecord{
		ID: "12345678-1234-4234-9234-123456789abc", SourceHash: hash,
		Title: "Song", Artist: "Artist", Album: "Album", AlbumArtist: "Unknown", Genre: "Unknown", Year: "Unknown", DurationSeconds: 1,
		OriginalPath: "original/12345678-1234-4234-9234-123456789abc.flac",
		FullPath:     "music.db",
		LitePath:     "lite/12345678-1234-4234-9234-123456789abc.mp3",
	}
	if validMusicRecord(record) {
		t.Fatal("validMusicRecord accepted an unsafe full-path")
	}
	if validManagedMusicPath("full/../music.db") {
		t.Fatal("validManagedMusicPath accepted a traversal path with a managed prefix")
	}
}

func TestSQLiteMusicRepositoryRejectsMalformedTaskSourceAndOriginalPath(t *testing.T) {
	repo, err := NewSQLiteMusicRepository(t.TempDir())
	if err != nil {
		t.Fatalf("NewSQLiteMusicRepository: %v", err)
	}
	defer func() { _ = repo.(interface{ Close() error }).Close() }()

	task := model.MusicTask{
		ID: "12345678-1234-4234-9234-123456789abc", Status: model.MusicTaskPending,
		Progress: 0, SourceHash: strings.Repeat("z", 64), OriginalPath: "tmp/source.flac",
	}
	if err := repo.CreateTask(context.Background(), task); err == nil {
		t.Fatal("CreateTask accepted malformed source hash and non-original path")
	}
}

func TestFilesystemMusicRepositoryScansSupportedRegularFiles(t *testing.T) {
	directory := t.TempDir()
	writeMusicFixture(t, directory, "Zed - Zebra.mp3", []byte("mp3"))
	writeMusicFixture(t, directory, "Alpha - Apple.OGG", []byte("ogg"))
	writeMusicFixture(t, directory, "ambient.flac", []byte("flac"))
	writeMusicFixture(t, directory, "ignore.txt", []byte("not audio"))

	if err := os.Mkdir(filepath.Join(directory, "nested"), 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	writeMusicFixture(t, filepath.Join(directory, "nested"), "Nested - Track.mp3", []byte("nested"))

	repository, err := NewFilesystemMusicRepository(directory)
	if err != nil {
		t.Fatalf("NewFilesystemMusicRepository() error = %v", err)
	}

	tracks, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tracks) != 3 {
		t.Fatalf("List() returned %d tracks, want 3", len(tracks))
	}

	tracksByTitle := make(map[string]model.MusicTrack, len(tracks))
	for _, track := range tracks {
		tracksByTitle[track.Title] = track
	}
	if track := tracksByTitle["Apple"]; track.Artist != "Alpha" {
		t.Fatalf("Apple track = %#v, want artist Alpha", track)
	}
	if track := tracksByTitle["ambient"]; track.Artist != "Jiuin Music" {
		t.Fatalf("ambient track = %#v, want fallback metadata", track)
	}
	if track := tracksByTitle["Zebra"]; track.Artist != "Zed" {
		t.Fatalf("Zebra track = %#v, want artist Zed", track)
	}

	for _, track := range tracks {
		if track.SourceURL != "/media/music/"+track.ID {
			t.Fatalf("SourceURL = %q, want media route for id %q", track.SourceURL, track.ID)
		}
		if len(track.ID) != 24 {
			t.Fatalf("ID = %q, want opaque 24-character hex ID", track.ID)
		}
	}

	asset, err := repository.Open(context.Background(), tracksByTitle["Apple"].ID)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if filepath.Base(asset.Path) != "Alpha - Apple.OGG" {
		t.Fatalf("Open() resolved %q, want Alpha - Apple.OGG", asset.Path)
	}

	_, err = repository.Open(context.Background(), "../../outside.mp3")
	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf("Open() traversal error = %v, want ErrMusicNotFound", err)
	}
}

func TestFilesystemMusicRepositorySkipsSymbolicLinks(t *testing.T) {
	directory := t.TempDir()
	outsideDirectory := t.TempDir()
	outsidePath := filepath.Join(outsideDirectory, "outside.mp3")
	writeMusicFixture(t, outsideDirectory, "outside.mp3", []byte("outside"))

	if err := os.Symlink(outsidePath, filepath.Join(directory, "linked.mp3")); err != nil {
		t.Skipf("symbolic links are unavailable in this test environment: %v", err)
	}
	writeMusicFixture(t, directory, "Local - Track.mp3", []byte("local"))

	repository, err := NewFilesystemMusicRepository(directory)
	if err != nil {
		t.Fatalf("NewFilesystemMusicRepository() error = %v", err)
	}

	tracks, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tracks) != 1 || tracks[0].Title != "Track" {
		t.Fatalf("List() = %#v, want only the local regular file", tracks)
	}
}

func writeMusicFixture(t *testing.T, directory string, name string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), contents, 0o644); err != nil {
		t.Fatalf("write fixture %q: %v", name, err)
	}
}
