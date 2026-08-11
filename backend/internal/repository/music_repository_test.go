package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

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
