package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	_ "modernc.org/sqlite"
)

var ErrMusicNotFound = errors.New("music track not found")

// MusicRepository is the storage boundary for public music metadata and local streams.
type MusicRepository interface {
	List(context.Context) ([]model.MusicTrack, error)
	Open(context.Context, string) (model.MusicAsset, error)
}

// ManagedMusicRepository is the persistent boundary used by the upload API.
// It extends the legacy public streaming contract so older route tests and
// clients can coexist during migration.
type ManagedMusicRepository interface {
	MusicRepository
	CreateTask(context.Context, model.MusicTask) error
	GetTask(context.Context, string) (model.MusicTask, error)
	FindTaskBySourceHash(context.Context, string) (model.MusicTask, error)
	ClaimTask(context.Context, string) (model.MusicTask, bool, error)
	UpdateTask(context.Context, model.MusicTask) error
	ListRecoverableTasks(context.Context) ([]model.MusicTask, error)
	FindMusicByHash(context.Context, string) (model.MusicRecord, error)
	CreateMusic(context.Context, model.MusicRecord) error
	CompleteTask(context.Context, model.MusicTask, model.MusicRecord) error
	GetMusic(context.Context, string) (model.MusicRecord, error)
	ListMusic(context.Context) ([]model.MusicRecord, error)
	OpenManagedAsset(context.Context, string, string) (model.MusicAsset, error)
}

var ErrMusicTaskNotFound = errors.New("music task not found")

// DuplicateMusicTaskError exposes the already-owned task without requiring a
// racy second database lookup during concurrent duplicate uploads.
type DuplicateMusicTaskError interface {
	error
	ExistingTask() model.MusicTask
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

type sqliteMusicRepository struct {
	database *sql.DB
	root     string
	mutex    sync.Mutex
}

const (
	musicDatabaseFilename = "music.db"
)

// NewSQLiteMusicRepository creates a small durable library database under the
// music storage root. SQLite is intentional here: the project has no central
// database dependency yet, and the worker/task state must survive restarts.
func NewSQLiteMusicRepository(directory string) (ManagedMusicRepository, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("music directory must not be empty")
	}

	root, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve music directory: %w", err)
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create music directory: %w", err)
	}
	resolvedRoot, err := resolveMusicDirectory(root)
	if err != nil {
		return nil, err
	}
	for _, child := range []string{"original", "full", "lite", "covers", "tmp"} {
		if err := ensureMusicStorageDirectory(resolvedRoot, child); err != nil {
			return nil, err
		}
	}

	databasePath, err := safeMusicDatabasePath(resolvedRoot)
	if err != nil {
		return nil, err
	}
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open music database: %w", err)
	}
	// SQLite PRAGMAs are connection-local for some drivers. One process owns
	// this library database, so a single connection gives every task operation
	// the same busy timeout and avoids spurious "database is locked" races.
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if _, err := database.Exec(`PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;`); err != nil {
		database.Close()
		return nil, fmt.Errorf("configure music database: %w", err)
	}
	repository := &sqliteMusicRepository{database: database, root: resolvedRoot}
	if err := repository.migrate(context.Background()); err != nil {
		database.Close()
		return nil, err
	}

	return repository, nil
}

func ensureMusicStorageDirectory(root, child string) error {
	if !validMusicStorageDirectoryName(child) {
		return errors.New("invalid music storage directory")
	}
	path := filepath.Join(root, child)
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(path, 0o750); err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("create music storage directory %q: %w", child, err)
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return fmt.Errorf("inspect music storage directory %q: %w", child, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("music storage directory %q must be a non-symlink directory", child)
	}
	return nil
}

func validMusicStorageDirectoryName(value string) bool {
	switch value {
	case "original", "full", "lite", "covers", "tmp":
		return true
	default:
		return false
	}
}

func safeMusicDatabasePath(root string) (string, error) {
	path := filepath.Join(root, musicDatabaseFilename)
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&fs.ModeSymlink != 0 {
		return "", errors.New("music database must not be a symlink")
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("inspect music database path: %w", err)
	}
	return path, nil
}

func ensureMusicDatabaseIsRegular(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect music database: %w", err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("music database must be a regular file")
	}
	return nil
}

func (repository *sqliteMusicRepository) migrate(ctx context.Context) error {
	if err := ensureMusicDatabaseIsRegular(filepath.Join(repository.root, musicDatabaseFilename)); err != nil {
		return err
	}
	_, err := repository.database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS music_records (
			id TEXT PRIMARY KEY,
			source_hash TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			album TEXT NOT NULL,
			album_artist TEXT NOT NULL,
			genre TEXT NOT NULL,
			year TEXT NOT NULL,
			duration_seconds INTEGER NOT NULL,
			cover_path TEXT NOT NULL,
			original_path TEXT NOT NULL,
			full_path TEXT NOT NULL,
			lite_path TEXT NOT NULL,
			full_size INTEGER NOT NULL,
			lite_size INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS music_tasks (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			progress INTEGER NOT NULL,
			music_id TEXT NOT NULL,
			source_hash TEXT NOT NULL,
			original_path TEXT NOT NULL,
			error_type TEXT NOT NULL,
			error_detail TEXT NOT NULL,
			exit_code INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_music_tasks_status ON music_tasks(status);
	`)
	if err != nil {
		return fmt.Errorf("migrate music database: %w", err)
	}
	if _, err := repository.database.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_music_tasks_source_hash_unique ON music_tasks(source_hash)`); err != nil {
		return fmt.Errorf("migrate music task source-hash uniqueness: %w", err)
	}
	if _, err := repository.database.ExecContext(ctx, `DROP INDEX IF EXISTS idx_music_tasks_source_hash`); err != nil {
		return fmt.Errorf("remove obsolete music task source-hash index: %w", err)
	}
	return nil
}

func (repository *sqliteMusicRepository) Close() error {
	return repository.database.Close()
}

func (repository *sqliteMusicRepository) List(ctx context.Context) ([]model.MusicTrack, error) {
	records, err := repository.ListMusic(ctx)
	if err != nil {
		return nil, err
	}
	tracks := make([]model.MusicTrack, 0, len(records))
	for _, record := range records {
		track := legacyTrackFromRecord(record)
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func (repository *sqliteMusicRepository) Open(ctx context.Context, id string) (model.MusicAsset, error) {
	return repository.OpenManagedAsset(ctx, id, "full")
}

func (repository *sqliteMusicRepository) CreateTask(ctx context.Context, task model.MusicTask) error {
	if !validMusicTask(task) {
		return errors.New("invalid music task")
	}
	if task.Progress < 0 || task.Progress > 100 {
		return errors.New("invalid music task progress")
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = task.CreatedAt
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if existing, err := repository.findTaskBySourceHash(ctx, task.SourceHash); err == nil {
		return duplicateMusicTaskError{task: existing}
	} else if !errors.Is(err, ErrMusicTaskNotFound) {
		return err
	}
	_, err := repository.database.ExecContext(ctx, `
		INSERT INTO music_tasks (id, status, progress, music_id, source_hash, original_path, error_type, error_detail, exit_code, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, string(task.Status), task.Progress, task.MusicID, task.SourceHash, task.OriginalPath, task.ErrorType, task.ErrorDetail, task.ExitCode, task.CreatedAt.UTC().Format(time.RFC3339Nano), task.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		if existing, lookupErr := repository.findTaskBySourceHash(ctx, task.SourceHash); lookupErr == nil {
			return duplicateMusicTaskError{task: existing}
		}
		return fmt.Errorf("create music task: %w", err)
	}
	return nil
}

func (repository *sqliteMusicRepository) GetTask(ctx context.Context, id string) (model.MusicTask, error) {
	if !validUUIDLikeID(id) {
		return model.MusicTask{}, ErrMusicTaskNotFound
	}
	row := repository.database.QueryRowContext(ctx, `
		SELECT id, status, progress, music_id, source_hash, original_path, error_type, error_detail, exit_code, created_at, updated_at
		FROM music_tasks WHERE id = ?
	`, id)
	task, err := scanMusicTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MusicTask{}, ErrMusicTaskNotFound
	}
	if err != nil {
		return model.MusicTask{}, fmt.Errorf("get music task: %w", err)
	}
	return task, nil
}

func (repository *sqliteMusicRepository) FindTaskBySourceHash(ctx context.Context, hash string) (model.MusicTask, error) {
	return repository.findTaskBySourceHash(ctx, hash)
}

// ClaimTask atomically moves a pending task to processing. Multiple enqueues
// are expected during startup recovery and duplicate upload races; only the
// caller that wins this compare-and-set is allowed to invoke FFmpeg.
func (repository *sqliteMusicRepository) ClaimTask(ctx context.Context, id string) (model.MusicTask, bool, error) {
	if !validUUIDLikeID(id) {
		return model.MusicTask{}, false, ErrMusicTaskNotFound
	}
	now := time.Now().UTC()
	result, err := repository.database.ExecContext(ctx, `
		UPDATE music_tasks
		SET status = ?, progress = ?, error_type = '', error_detail = '', exit_code = 0, updated_at = ?
		WHERE id = ? AND status = ?
	`, string(model.MusicTaskProcessing), 5, now.Format(time.RFC3339Nano), id, string(model.MusicTaskPending))
	if err != nil {
		return model.MusicTask{}, false, fmt.Errorf("claim music task: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return model.MusicTask{}, false, fmt.Errorf("inspect music task claim: %w", err)
	}
	if count == 0 {
		if _, err := repository.GetTask(ctx, id); err != nil {
			return model.MusicTask{}, false, err
		}
		return model.MusicTask{}, false, nil
	}
	task, err := repository.GetTask(ctx, id)
	if err != nil {
		return model.MusicTask{}, false, err
	}
	return task, true, nil
}

func (repository *sqliteMusicRepository) findTaskBySourceHash(ctx context.Context, hash string) (model.MusicTask, error) {
	if !validSHA256Hex(hash) {
		return model.MusicTask{}, ErrMusicTaskNotFound
	}
	row := repository.database.QueryRowContext(ctx, `
		SELECT id, status, progress, music_id, source_hash, original_path, error_type, error_detail, exit_code, created_at, updated_at
		FROM music_tasks WHERE source_hash = ?
	`, hash)
	task, err := scanMusicTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MusicTask{}, ErrMusicTaskNotFound
	}
	if err != nil {
		return model.MusicTask{}, fmt.Errorf("find music task by source hash: %w", err)
	}
	return task, nil
}

type duplicateMusicTaskError struct {
	task model.MusicTask
}

func (error duplicateMusicTaskError) Error() string { return "music task for source already exists" }
func (error duplicateMusicTaskError) ExistingTask() model.MusicTask {
	return error.task
}

func (repository *sqliteMusicRepository) UpdateTask(ctx context.Context, task model.MusicTask) error {
	if !validMusicTask(task) {
		return errors.New("invalid music task")
	}
	task.UpdatedAt = time.Now().UTC()
	result, err := repository.database.ExecContext(ctx, `
		UPDATE music_tasks SET status = ?, progress = ?, music_id = ?, source_hash = ?, original_path = ?, error_type = ?, error_detail = ?, exit_code = ?, updated_at = ? WHERE id = ?
	`, string(task.Status), task.Progress, task.MusicID, task.SourceHash, task.OriginalPath, task.ErrorType, task.ErrorDetail, task.ExitCode, task.UpdatedAt.Format(time.RFC3339Nano), task.ID)
	if err != nil {
		return fmt.Errorf("update music task: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect music task update: %w", err)
	}
	if count == 0 {
		return ErrMusicTaskNotFound
	}
	return nil
}

func (repository *sqliteMusicRepository) ListRecoverableTasks(ctx context.Context) ([]model.MusicTask, error) {
	rows, err := repository.database.QueryContext(ctx, `
		SELECT id, status, progress, music_id, source_hash, original_path, error_type, error_detail, exit_code, created_at, updated_at
		FROM music_tasks WHERE status IN (?, ?) ORDER BY created_at ASC
	`, string(model.MusicTaskPending), string(model.MusicTaskProcessing))
	if err != nil {
		return nil, fmt.Errorf("list recoverable music tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]model.MusicTask, 0)
	for rows.Next() {
		task, err := scanMusicTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recoverable music task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recoverable music tasks: %w", err)
	}
	return tasks, nil
}

func (repository *sqliteMusicRepository) FindMusicByHash(ctx context.Context, hash string) (model.MusicRecord, error) {
	if !validSHA256Hex(hash) {
		return model.MusicRecord{}, ErrMusicNotFound
	}
	row := repository.database.QueryRowContext(ctx, musicRecordSelect+` WHERE source_hash = ?`, hash)
	record, err := scanMusicRecord(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MusicRecord{}, ErrMusicNotFound
	}
	if err != nil {
		return model.MusicRecord{}, fmt.Errorf("find music by source hash: %w", err)
	}
	return record, nil
}

func (repository *sqliteMusicRepository) CreateMusic(ctx context.Context, record model.MusicRecord) error {
	if !validMusicRecord(record) {
		return errors.New("invalid music record")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	_, err := repository.database.ExecContext(ctx, `
		INSERT INTO music_records (id, source_hash, title, artist, album, album_artist, genre, year, duration_seconds, cover_path, original_path, full_path, lite_path, full_size, lite_size, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.SourceHash, record.Title, record.Artist, record.Album, record.AlbumArtist, record.Genre, record.Year, record.DurationSeconds, record.CoverPath, record.OriginalPath, record.FullPath, record.LitePath, record.FullSize, record.LiteSize, record.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("create music record: %w", err)
	}
	return nil
}

// CompleteTask writes the public song record and its task completion in the
// same SQLite transaction. Recovery therefore sees either a pending task or a
// complete song/task pair, never an orphaned published record.
func (repository *sqliteMusicRepository) CompleteTask(ctx context.Context, task model.MusicTask, record model.MusicRecord) error {
	if task.Status != model.MusicTaskCompleted || task.Progress != 100 || task.MusicID != record.ID || task.SourceHash != record.SourceHash || !validMusicRecord(record) {
		return errors.New("invalid completed music task")
	}
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin music completion transaction: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO music_records (id, source_hash, title, artist, album, album_artist, genre, year, duration_seconds, cover_path, original_path, full_path, lite_path, full_size, lite_size, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.SourceHash, record.Title, record.Artist, record.Album, record.AlbumArtist, record.Genre, record.Year, record.DurationSeconds, record.CoverPath, record.OriginalPath, record.FullPath, record.LitePath, record.FullSize, record.LiteSize, record.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("create completed music record: %w", err)
	}
	task.UpdatedAt = time.Now().UTC()
	result, err := transaction.ExecContext(ctx, `
		UPDATE music_tasks SET status = ?, progress = ?, music_id = ?, source_hash = ?, original_path = ?, error_type = ?, error_detail = ?, exit_code = ?, updated_at = ?
		WHERE id = ? AND status = ?
	`, string(task.Status), task.Progress, task.MusicID, task.SourceHash, task.OriginalPath, task.ErrorType, task.ErrorDetail, task.ExitCode, task.UpdatedAt.Format(time.RFC3339Nano), task.ID, string(model.MusicTaskProcessing))
	if err != nil {
		return fmt.Errorf("complete music task: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect music task completion: %w", err)
	}
	if count != 1 {
		return ErrMusicTaskNotFound
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit music completion transaction: %w", err)
	}
	return nil
}

func validMusicRecord(record model.MusicRecord) bool {
	return validUUIDLikeID(record.ID) &&
		validSHA256Hex(record.SourceHash) &&
		validMetadataField(record.Title) &&
		validMetadataField(record.Artist) &&
		validMetadataField(record.Album) &&
		validMetadataField(record.AlbumArtist) &&
		validMetadataField(record.Genre) &&
		validMetadataField(record.Year) &&
		record.DurationSeconds > 0 &&
		record.FullSize >= 0 &&
		record.LiteSize >= 0 &&
		validOriginalMusicPath(record.OriginalPath) &&
		validRenditionMusicPath(record.FullPath, "full", record.ID) &&
		validRenditionMusicPath(record.LitePath, "lite", record.ID) &&
		(record.CoverPath == "" || validCoverMusicPath(record.CoverPath, record.ID))
}

func (repository *sqliteMusicRepository) GetMusic(ctx context.Context, id string) (model.MusicRecord, error) {
	if !validUUIDLikeID(id) {
		return model.MusicRecord{}, ErrMusicNotFound
	}
	row := repository.database.QueryRowContext(ctx, musicRecordSelect+` WHERE id = ?`, id)
	record, err := scanMusicRecord(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MusicRecord{}, ErrMusicNotFound
	}
	if err != nil {
		return model.MusicRecord{}, fmt.Errorf("get music record: %w", err)
	}
	return record, nil
}

func (repository *sqliteMusicRepository) ListMusic(ctx context.Context) ([]model.MusicRecord, error) {
	rows, err := repository.database.QueryContext(ctx, musicRecordSelect+` ORDER BY created_at DESC, title COLLATE NOCASE ASC`)
	if err != nil {
		return nil, fmt.Errorf("list music records: %w", err)
	}
	defer rows.Close()

	records := make([]model.MusicRecord, 0)
	for rows.Next() {
		record, err := scanMusicRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan music record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate music records: %w", err)
	}
	return records, nil
}

func (repository *sqliteMusicRepository) OpenManagedAsset(ctx context.Context, id, variant string) (model.MusicAsset, error) {
	record, err := repository.GetMusic(ctx, id)
	if err != nil {
		return model.MusicAsset{}, err
	}
	var relativePath string
	switch variant {
	case "full":
		relativePath = record.FullPath
	case "lite":
		relativePath = record.LitePath
	case "cover":
		relativePath = record.CoverPath
	default:
		return model.MusicAsset{}, ErrMusicNotFound
	}
	if relativePath == "" {
		return model.MusicAsset{}, ErrMusicNotFound
	}
	path, err := repository.resolvePath(relativePath)
	if err != nil {
		return model.MusicAsset{}, ErrMusicNotFound
	}
	return model.MusicAsset{Path: path, Name: filepath.Base(path)}, nil
}

func (repository *sqliteMusicRepository) resolvePath(relativePath string) (string, error) {
	if !validManagedMusicPath(relativePath) {
		return "", errors.New("invalid music storage path")
	}
	path := filepath.Join(repository.root, filepath.FromSlash(relativePath))
	relative, err := filepath.Rel(repository.root, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		return "", errors.New("music storage path escaped root")
	}
	if err := rejectMusicSymlinkComponents(repository.root, path); err != nil {
		return "", err
	}
	return path, nil
}

func rejectMusicSymlinkComponents(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		return errors.New("music storage path escaped root")
	}
	current := root
	for _, component := range strings.FieldsFunc(relative, func(r rune) bool { return r == '/' || r == '\\' }) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return fmt.Errorf("inspect music storage path: %w", statErr)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return errors.New("music storage path must not traverse a symlink")
		}
	}
	return nil
}

const musicRecordSelect = `
	SELECT id, source_hash, title, artist, album, album_artist, genre, year, duration_seconds, cover_path, original_path, full_path, lite_path, full_size, lite_size, created_at
	FROM music_records`

type rowScanner interface {
	Scan(...any) error
}

func scanMusicRecord(row rowScanner) (model.MusicRecord, error) {
	var record model.MusicRecord
	var createdAt string
	err := row.Scan(&record.ID, &record.SourceHash, &record.Title, &record.Artist, &record.Album, &record.AlbumArtist, &record.Genre, &record.Year, &record.DurationSeconds, &record.CoverPath, &record.OriginalPath, &record.FullPath, &record.LitePath, &record.FullSize, &record.LiteSize, &createdAt)
	if err != nil {
		return model.MusicRecord{}, err
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.MusicRecord{}, fmt.Errorf("parse music creation time: %w", err)
	}
	record.CreatedAt = parsedCreatedAt
	if !validMusicRecord(record) {
		return model.MusicRecord{}, errors.New("stored music record is invalid")
	}
	return record, nil
}

func scanMusicTask(row rowScanner) (model.MusicTask, error) {
	var task model.MusicTask
	var status string
	var createdAt, updatedAt string
	err := row.Scan(&task.ID, &status, &task.Progress, &task.MusicID, &task.SourceHash, &task.OriginalPath, &task.ErrorType, &task.ErrorDetail, &task.ExitCode, &createdAt, &updatedAt)
	if err != nil {
		return model.MusicTask{}, err
	}
	if !validMusicTaskStatus(model.MusicTaskStatus(status)) {
		return model.MusicTask{}, errors.New("stored music task has invalid status")
	}
	task.Status = model.MusicTaskStatus(status)
	var errCreatedAt, errUpdatedAt error
	task.CreatedAt, errCreatedAt = time.Parse(time.RFC3339Nano, createdAt)
	task.UpdatedAt, errUpdatedAt = time.Parse(time.RFC3339Nano, updatedAt)
	if errCreatedAt != nil || errUpdatedAt != nil {
		return model.MusicTask{}, errors.New("stored music task has invalid timestamp")
	}
	if !validMusicTask(task) {
		return model.MusicTask{}, errors.New("stored music task is invalid")
	}
	return task, nil
}

func legacyTrackFromRecord(record model.MusicRecord) model.MusicTrack {
	qualities := []model.AudioQuality{
		{ID: "full", Label: "完整版", SourceURL: "/media/music/full/" + record.ID + ".mp3"},
		{ID: "lite", Label: "省流版", SourceURL: "/media/music/lite/" + record.ID + ".mp3"},
	}
	track := model.MusicTrack{
		ID: record.ID, Title: record.Title, Artist: record.Artist,
		DurationSeconds: record.DurationSeconds, SourceURL: qualities[0].SourceURL, Qualities: qualities,
	}
	if record.CoverPath != "" {
		track.ArtworkURL = "/media/music/covers/" + record.ID + ".jpg"
	}
	return track
}

func validUUIDLikeID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}

func validMusicTaskStatus(status model.MusicTaskStatus) bool {
	return status == model.MusicTaskPending || status == model.MusicTaskProcessing || status == model.MusicTaskCompleted || status == model.MusicTaskFailed
}

func validRelativeMusicPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "\\") || strings.ContainsRune(value, '\x00') {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	return clean == value && !strings.HasPrefix(clean, "/") && !strings.HasPrefix(clean, "../") && clean != ".."
}

func validSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func validMusicTask(task model.MusicTask) bool {
	return validUUIDLikeID(task.ID) &&
		validMusicTaskStatus(task.Status) &&
		task.Progress >= 0 && task.Progress <= 100 &&
		validSHA256Hex(task.SourceHash) &&
		validOriginalMusicPath(task.OriginalPath) &&
		(task.MusicID == "" || validUUIDLikeID(task.MusicID)) &&
		len(task.ErrorType) <= 64 &&
		len(task.ErrorDetail) <= 512
}

func validManagedMusicPath(value string) bool {
	// Prefix checks alone are not enough here: a value such as
	// "full/../music.db" has the expected prefix but is normalized outside the
	// managed rendition directory when it is later joined with the storage root.
	if !validRelativeMusicPath(value) {
		return false
	}
	return validOriginalMusicPath(value) ||
		strings.HasPrefix(value, "full/") ||
		strings.HasPrefix(value, "lite/") ||
		strings.HasPrefix(value, "covers/")
}

func validOriginalMusicPath(value string) bool {
	if !validRelativeMusicPath(value) || !strings.HasPrefix(value, "original/") {
		return false
	}
	name := strings.TrimPrefix(value, "original/")
	if strings.Contains(name, "/") || !validMusicFileName(name) {
		return false
	}
	extension := strings.ToLower(filepath.Ext(name))
	return supportedManagedInputExtension(extension) && validUUIDLikeID(strings.TrimSuffix(name, extension))
}

func validRenditionMusicPath(value, directory, id string) bool {
	return validRelativeMusicPath(value) && value == directory+"/"+id+".mp3"
}

func validCoverMusicPath(value, id string) bool {
	return validRelativeMusicPath(value) && value == "covers/"+id+".jpg"
}

func validMusicFileName(value string) bool {
	return value != "" && filepath.Base(value) == value && !strings.ContainsAny(value, "/\\") && !strings.ContainsRune(value, '\x00')
}

func supportedManagedInputExtension(extension string) bool {
	switch extension {
	case ".mp3", ".flac", ".wav", ".ogg", ".m4a", ".aac":
		return true
	default:
		return false
	}
}

func validMetadataField(value string) bool {
	return value != "" && len(value) <= 1024 && !strings.ContainsRune(value, '\x00')
}

func resolveMusicDirectory(directory string) (string, error) {
	resolved, err := filepath.EvalSymlinks(directory)
	if err == nil {
		return resolved, nil
	}
	// Restricted Windows environments can deny reparse-point inspection for an
	// otherwise ordinary temporary directory. In that case a verified,
	// non-symlink directory is still safe as the root; assets themselves are
	// separately Lstat-checked before public serving.
	info, statErr := os.Lstat(directory)
	if statErr == nil && info.IsDir() && info.Mode()&fs.ModeSymlink == 0 {
		return directory, nil
	}
	return "", fmt.Errorf("resolve music directory symlinks: %w", err)
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
	resolvedDirectory, err := resolveMusicDirectory(absDirectory)
	if err != nil {
		return nil, err
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
				ID:         id,
				Title:      title,
				Artist:     artist,
				SourceURL:  "/media/music/" + id,
				SourceSize: info.Size(),
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
