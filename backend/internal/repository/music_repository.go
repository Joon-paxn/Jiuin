package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

var ErrMusicNotFound = errors.New("music track not found")

// MusicRepository is the storage boundary for public music metadata and local streams.
type MusicRepository interface {
	List(context.Context) ([]model.MusicTrack, error)
	Open(context.Context, string) (model.MusicAsset, error)
}

type staticMusicRepository struct {
	tracks []model.MusicTrack
}

func NewStaticMusicRepository(tracks []model.MusicTrack) MusicRepository {
	if tracks == nil {
		tracks = []model.MusicTrack{}
	}

	return staticMusicRepository{tracks: tracks}
}

func (repository staticMusicRepository) List(context.Context) ([]model.MusicTrack, error) {
	return repository.tracks, nil
}

func (staticMusicRepository) Open(context.Context, string) (model.MusicAsset, error) {
	return model.MusicAsset{}, ErrMusicNotFound
}

type filesystemMusicRepository struct {
	directory string
}

type musicFile struct {
	id    string
	name  string
	path  string
	track model.MusicTrack
}

var supportedAudioExtensions = map[string]struct{}{
	".aac":  {},
	".flac": {},
	".m4a":  {},
	".mp3":  {},
	".ogg":  {},
	".wav":  {},
}

// NewFilesystemMusicRepository reads user-managed music files from a dedicated
// directory. Files are never written through the public HTTP service.
func NewFilesystemMusicRepository(directory string) (MusicRepository, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("music directory must not be empty")
	}

	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve music directory: %w", err)
	}
	if err := os.MkdirAll(absDirectory, 0o755); err != nil {
		return nil, fmt.Errorf("create music directory: %w", err)
	}
	resolvedDirectory, err := filepath.EvalSymlinks(absDirectory)
	if err != nil {
		return nil, fmt.Errorf("resolve music directory symlinks: %w", err)
	}
	info, err := os.Stat(resolvedDirectory)
	if err != nil {
		return nil, fmt.Errorf("stat music directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("music directory %q is not a directory", resolvedDirectory)
	}

	return filesystemMusicRepository{directory: resolvedDirectory}, nil
}

func (repository filesystemMusicRepository) List(ctx context.Context) ([]model.MusicTrack, error) {
	files, err := repository.scan(ctx)
	if err != nil {
		return nil, err
	}

	tracks := make([]model.MusicTrack, 0, len(files))
	for _, file := range files {
		tracks = append(tracks, file.track)
	}

	return tracks, nil
}

func (repository filesystemMusicRepository) Open(ctx context.Context, id string) (model.MusicAsset, error) {
	if !isMusicID(id) {
		return model.MusicAsset{}, ErrMusicNotFound
	}

	files, err := repository.scan(ctx)
	if err != nil {
		return model.MusicAsset{}, err
	}

	for _, file := range files {
		if file.id == id {
			info, err := os.Lstat(file.path)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return model.MusicAsset{}, ErrMusicNotFound
				}
				return model.MusicAsset{}, fmt.Errorf("stat music file before open: %w", err)
			}
			if !info.Mode().IsRegular() {
				return model.MusicAsset{}, ErrMusicNotFound
			}

			return model.MusicAsset{Path: file.path, Name: file.name}, nil
		}
	}

	return model.MusicAsset{}, ErrMusicNotFound
}

func (repository filesystemMusicRepository) scan(ctx context.Context) ([]musicFile, error) {
	entries, err := os.ReadDir(repository.directory)
	if err != nil {
		return nil, fmt.Errorf("read music directory: %w", err)
	}

	files := make([]musicFile, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}

		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if _, supported := supportedAudioExtensions[extension]; !supported {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("read music file information for %q: %w", entry.Name(), err)
		}
		if !info.Mode().IsRegular() {
			continue
		}

		id := musicID(entry.Name(), info)
		artist, title := parseMusicFileName(entry.Name())
		files = append(files, musicFile{
			id:   id,
			name: entry.Name(),
			path: filepath.Join(repository.directory, entry.Name()),
			track: model.MusicTrack{
				ID:        id,
				Title:     title,
				Artist:    artist,
				SourceURL: "/media/music/" + id,
			},
		})
	}

	sort.Slice(files, func(left, right int) bool {
		if files[left].track.Title == files[right].track.Title {
			return files[left].name < files[right].name
		}
		return files[left].track.Title < files[right].track.Title
	})

	return files, nil
}

func musicID(name string, info fs.FileInfo) string {
	seed := fmt.Sprintf("%s:%d:%d", name, info.Size(), info.ModTime().UnixNano())
	digest := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(digest[:12])
}

func isMusicID(id string) bool {
	if len(id) != 24 {
		return false
	}

	_, err := hex.DecodeString(id)
	return err == nil
}

func parseMusicFileName(name string) (artist string, title string) {
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	parts := strings.SplitN(base, " - ", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	return "Jiuin Music", base
}
