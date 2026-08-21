package core

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type PublicMusic struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Artist          string      `json:"artist"`
	Album           string      `json:"album,omitempty"`
	AlbumArtist     string      `json:"albumArtist,omitempty"`
	Genre           string      `json:"genre,omitempty"`
	Year            string      `json:"year,omitempty"`
	Cover           string      `json:"cover,omitempty"`
	DurationSeconds *float64    `json:"durationSeconds,omitempty"`
	Audio           PublicAudio `json:"audio"`
	FullSize        *int64      `json:"fullSize,omitempty"`
	LiteSize        *int64      `json:"liteSize,omitempty"`
	CreatedAt       string      `json:"createdAt,omitempty"`
}

type PublicAudio struct {
	Full string `json:"full,omitempty"`
	Lite string `json:"lite,omitempty"`
}

type UploadInput struct {
	IdempotencyKey string
	Title          string
	Artist         string
	Album          string
	AlbumArtist    string
	Genre          string
	Year           string
	SourceName     string
	Source         io.Reader
}

type UploadResult struct {
	UploadID         string `json:"uploadId"`
	TaskID           string `json:"taskId"`
	MusicID          string `json:"musicId"`
	State            string `json:"state"`
	IdempotentReplay bool   `json:"idempotentReplay"`
}

type MusicStore struct {
	DB     *sql.DB
	Config Config
}

func (s MusicStore) ListPublic(ctx context.Context) ([]PublicMusic, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,title,artist,album,album_artist,genre,year,cover_path,duration_seconds,full_path,lite_path,full_size,lite_size,created_at FROM music WHERE state='ready' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var music []PublicMusic
	for rows.Next() {
		item, err := scanPublicMusic(rows)
		if err != nil {
			return nil, err
		}
		music = append(music, item)
	}
	return music, rows.Err()
}

func (s MusicStore) GetPublic(ctx context.Context, id string) (PublicMusic, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id,title,artist,album,album_artist,genre,year,cover_path,duration_seconds,full_path,lite_path,full_size,lite_size,created_at FROM music WHERE id=? AND state='ready'`, id)
	item, err := scanPublicMusic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicMusic{}, ErrNotFound
	}
	return item, err
}

type scanner interface{ Scan(...any) error }

func scanPublicMusic(row scanner) (PublicMusic, error) {
	var item PublicMusic
	var coverPath, fullPath, litePath string
	var duration sql.NullFloat64
	var fullSize, liteSize sql.NullInt64
	err := row.Scan(&item.ID, &item.Title, &item.Artist, &item.Album, &item.AlbumArtist, &item.Genre, &item.Year, &coverPath, &duration, &fullPath, &litePath, &fullSize, &liteSize, &item.CreatedAt)
	if err != nil {
		return PublicMusic{}, err
	}
	if coverPath != "" {
		item.Cover = "/media/music/" + item.ID + "/cover"
	}
	if fullPath != "" {
		item.Audio.Full = "/media/music/" + item.ID + "/full"
	}
	if litePath != "" {
		item.Audio.Lite = "/media/music/" + item.ID + "/lite"
	}
	if duration.Valid {
		item.DurationSeconds = &duration.Float64
	}
	if fullSize.Valid {
		item.FullSize = &fullSize.Int64
	}
	if liteSize.Valid {
		item.LiteSize = &liteSize.Int64
	}
	return item, nil
}

func (s MusicStore) CreateUpload(ctx context.Context, input UploadInput) (UploadResult, error) {
	if strings.TrimSpace(input.IdempotencyKey) == "" || len(input.IdempotencyKey) > 255 {
		return UploadResult{}, ErrIdempotencyKeyRequired
	}
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Artist) == "" {
		return UploadResult{}, fmt.Errorf("title and artist are required")
	}
	if input.Source == nil || !safeFilename(input.SourceName) {
		return UploadResult{}, fmt.Errorf("a safe music file is required")
	}

	tmp, err := os.CreateTemp(filepath.Join(s.Config.StorageDir, "tmp"), "upload-*.part")
	if err != nil {
		return UploadResult{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	hash := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, hash), input.Source); err != nil {
		tmp.Close()
		return UploadResult{}, err
	}
	if err = tmp.Close(); err != nil {
		return UploadResult{}, err
	}

	// Both runtimes derive stable identifiers from the idempotency key. This
	// closes the crash window between moving the original and committing SQLite:
	// a controlled Go retry targets the same paths and rows, never new music.
	musicID := idForKey("music", input.IdempotencyKey)
	uploadID := idForKey("upload", input.IdempotencyKey)
	taskID := idForKey("task", input.IdempotencyKey)
	// The storage name deliberately does not depend on the client filename.
	// A retry with a different extension still resolves to the same object.
	originalPath := filepath.Join(s.Config.StorageDir, "original", musicID+".upload")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result := UploadResult{UploadID: uploadID, TaskID: taskID, MusicID: musicID, State: "queued"}

	contentHash := hex.EncodeToString(hash.Sum(nil))
	movedOriginal := false
	committed := false
	defer func() {
		if movedOriginal && !committed {
			_ = os.Remove(originalPath)
		}
	}()
	if err = withImmediateTx(ctx, s.DB, func(conn *sql.Conn) error {
		if existing, existingHash, err := queryUpload(ctx, conn, input.IdempotencyKey); err == nil {
			if existingHash != contentHash {
				return ErrIdempotencyConflict
			}
			result = existing
			result.IdempotentReplay = true
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := os.Rename(tmpPath, originalPath); err != nil {
			return err
		}
		movedOriginal = true
		if _, err := conn.ExecContext(ctx, `INSERT INTO music (id,title,artist,album,album_artist,genre,year,source_name,source_path,original_path,state,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?, 'queued',?,?)`, musicID, input.Title, input.Artist, input.Album, input.AlbumArtist, input.Genre, input.Year, input.SourceName, input.SourceName, originalPath, now, now); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, `INSERT INTO upload_requests (idempotency_key,upload_id,task_id,music_id,content_sha256,created_at) VALUES (?,?,?,?,?,?)`, input.IdempotencyKey, uploadID, taskID, musicID, contentHash, now); err != nil {
			return err
		}
		_, err := conn.ExecContext(ctx, `INSERT INTO music_tasks (id,music_id,state,created_at,updated_at) VALUES (?,?,'queued',?,?)`, taskID, musicID, now, now)
		return err
	}); err != nil {
		return UploadResult{}, err
	}
	committed = !result.IdempotentReplay
	return result, nil
}

func queryUpload(ctx context.Context, conn *sql.Conn, key string) (UploadResult, string, error) {
	row := conn.QueryRowContext(ctx, `SELECT u.upload_id,u.task_id,u.music_id,t.state,u.content_sha256 FROM upload_requests u JOIN music_tasks t ON t.id=u.task_id WHERE u.idempotency_key=?`, key)
	var result UploadResult
	var contentHash string
	err := row.Scan(&result.UploadID, &result.TaskID, &result.MusicID, &result.State, &contentHash)
	return result, contentHash, err
}

func (s MusicStore) OpenMedia(ctx context.Context, id, quality string) (*os.File, string, error) {
	if quality != "cover" && quality != "full" && quality != "lite" {
		return nil, "", ErrNotFound
	}
	column := map[string]string{"cover": "cover_path", "full": "full_path", "lite": "lite_path"}[quality]
	row := s.DB.QueryRowContext(ctx, `SELECT `+column+` FROM music WHERE id=? AND state='ready'`, id)
	var path string
	if err := row.Scan(&path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	if path == "" {
		return nil, "", ErrNotFound
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	contentType := "audio/mpeg"
	if quality == "cover" {
		contentType = "image/jpeg"
	}
	return file, contentType, nil
}

type task struct{ ID, MusicID, OriginalPath string }

func (s MusicStore) ClaimTask(ctx context.Context, workerID string, lease time.Duration) (*task, error) {
	var claimed *task
	now := time.Now().UTC()
	leaseUntil := now.Add(lease).Format(time.RFC3339Nano)
	err := withImmediateTx(ctx, s.DB, func(conn *sql.Conn) error {
		row := conn.QueryRowContext(ctx, `SELECT t.id,t.music_id,m.original_path FROM music_tasks t JOIN music m ON m.id=t.music_id WHERE t.state='queued' OR (t.state='processing' AND t.lease_until < ?) ORDER BY t.created_at LIMIT 1`, now.Format(time.RFC3339Nano))
		candidate := task{}
		if err := row.Scan(&candidate.ID, &candidate.MusicID, &candidate.OriginalPath); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		result, err := conn.ExecContext(ctx, `UPDATE music_tasks SET state='processing',locked_by=?,lease_until=?,attempts=attempts+1,updated_at=? WHERE id=? AND (state='queued' OR (state='processing' AND lease_until < ?))`, workerID, leaseUntil, now.Format(time.RFC3339Nano), candidate.ID, now.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 1 {
			claimed = &candidate
		}
		return nil
	})
	return claimed, err
}

func (s MusicStore) ProcessOne(ctx context.Context, workerID string) (bool, error) {
	job, err := s.ClaimTask(ctx, workerID, s.Config.ProcessingLease)
	if err != nil || job == nil {
		return false, err
	}
	err = s.processTask(ctx, *job)
	if finalizeErr := s.finishTask(ctx, *job, workerID, err); finalizeErr != nil {
		return true, finalizeErr
	}
	return true, err
}

func (s MusicStore) processTask(ctx context.Context, job task) error {
	full := filepath.Join(s.Config.StorageDir, "full", job.MusicID+".mp3")
	lite := filepath.Join(s.Config.StorageDir, "lite", job.MusicID+".mp3")
	cover := filepath.Join(s.Config.StorageDir, "covers", job.MusicID+".jpg")
	if err := run(ctx, s.Config.FFmpegPath, "-y", "-i", job.OriginalPath, "-vn", "-c:a", s.Config.OutputCodec, "-b:a", s.Config.FullBitrate, full); err != nil {
		return err
	}
	if err := run(ctx, s.Config.FFmpegPath, "-y", "-i", job.OriginalPath, "-vn", "-c:a", s.Config.OutputCodec, "-b:a", s.Config.LiteBitrate, lite); err != nil {
		return err
	}
	// A missing embedded cover is not a failed audio upload. ffmpeg's color
	// source creates the same deterministic JPEG shape for both workers.
	if err := run(ctx, s.Config.FFmpegPath, "-y", "-i", job.OriginalPath, "-an", "-map", "0:v:0", "-frames:v", "1", cover); err != nil {
		if fallbackErr := run(ctx, s.Config.FFmpegPath, "-y", "-f", "lavfi", "-i", "color=c=0x293241:s=1200x1200", "-frames:v", "1", cover); fallbackErr != nil {
			return fallbackErr
		}
	}
	return nil
}

func (s MusicStore) finishTask(ctx context.Context, job task, workerID string, processingErr error) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if processingErr != nil {
		return withImmediateTx(ctx, s.DB, func(conn *sql.Conn) error {
			result, err := conn.ExecContext(ctx, `UPDATE music_tasks SET state='failed',last_error=?,lease_until='',updated_at=? WHERE id=? AND locked_by=? AND state='processing'`, truncateError(processingErr), now, job.ID, workerID)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected != 1 {
				return ErrLeaseLost
			}
			_, err = conn.ExecContext(ctx, `UPDATE music SET state='failed',updated_at=? WHERE id=?`, now, job.MusicID)
			return err
		})
	}
	full := filepath.Join(s.Config.StorageDir, "full", job.MusicID+".mp3")
	lite := filepath.Join(s.Config.StorageDir, "lite", job.MusicID+".mp3")
	cover := filepath.Join(s.Config.StorageDir, "covers", job.MusicID+".jpg")
	fullInfo, err := os.Stat(full)
	if err != nil {
		return err
	}
	liteInfo, err := os.Stat(lite)
	if err != nil {
		return err
	}
	duration, err := probeDuration(ctx, s.Config.FFprobePath, job.OriginalPath)
	if err != nil {
		return err
	}
	return withImmediateTx(ctx, s.DB, func(conn *sql.Conn) error {
		result, err := conn.ExecContext(ctx, `UPDATE music_tasks SET state='done',lease_until='',updated_at=? WHERE id=? AND locked_by=? AND state='processing'`, now, job.ID, workerID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrLeaseLost
		}
		if _, err := conn.ExecContext(ctx, `UPDATE music SET state='ready',full_path=?,lite_path=?,cover_path=?,duration_seconds=?,full_size=?,lite_size=?,updated_at=? WHERE id=?`, full, lite, cover, duration, fullInfo.Size(), liteInfo.Size(), now, job.MusicID); err != nil {
			return err
		}
		return nil
	})
}

func (s MusicStore) RunWorker(ctx context.Context, workerID string) error {
	ticker := time.NewTicker(s.Config.WorkerInterval)
	defer ticker.Stop()
	for {
		for {
			worked, err := s.ProcessOne(ctx, workerID)
			if err != nil {
				return err
			}
			if !worked {
				break
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func run(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w: %s", command, err, truncate(string(output), 1200))
	}
	return nil
}

func probeDuration(ctx context.Context, command, input string) (float64, error) {
	cmd := exec.CommandContext(ctx, command, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", input)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("%s failed: %w", command, err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || duration < 0 {
		return 0, fmt.Errorf("%s returned invalid duration", command)
	}
	return duration, nil
}

func safeFilename(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.ContainsAny(name, "\\/") && len(name) <= 255
}

func idForKey(prefix, key string) string {
	sum := sha256.Sum256([]byte(key))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

func truncateError(err error) string { return truncate(err.Error(), 1000) }
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

var (
	ErrNotFound               = errors.New("not found")
	ErrIdempotencyKeyRequired = errors.New("Idempotency-Key header is required")
	ErrIdempotencyConflict    = errors.New("Idempotency-Key was already used with different content")
	ErrLeaseLost              = errors.New("music task lease was lost")
)
