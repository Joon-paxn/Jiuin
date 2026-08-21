package core

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateUploadIsIdempotentAcrossRetries(t *testing.T) {
	storage := t.TempDir()
	config := Config{StorageDir: storage, DatabasePath: filepath.Join(storage, "music.db")}
	if err := config.EnsureStorage(); err != nil {
		t.Fatal(err)
	}
	db, err := OpenDatabase(config.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := MusicStore{DB: db, Config: config}
	input := UploadInput{IdempotencyKey: "same-request", Title: "title", Artist: "artist", SourceName: "track.mp3", Source: bytes.NewBufferString("not real audio")}
	first, err := store.CreateUpload(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.IdempotentReplay {
		t.Fatal("first upload was marked as replay")
	}

	input.Source = bytes.NewBufferString("not real audio")
	second, err := store.CreateUpload(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !second.IdempotentReplay || second.MusicID != first.MusicID || second.UploadID != first.UploadID || second.TaskID != first.TaskID {
		t.Fatalf("unexpected replay result: %#v %#v", first, second)
	}
	if _, err := os.Stat(filepath.Join(storage, "original", first.MusicID+".upload")); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM music").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("got %d music rows, want 1", count)
	}

	input.Source = bytes.NewBufferString("different body")
	if _, err := store.CreateUpload(context.Background(), input); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("got %v, want idempotency conflict", err)
	}
}
