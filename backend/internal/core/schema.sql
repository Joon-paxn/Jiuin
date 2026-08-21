-- Single shared schema. Go embeds this file into the backup binary; PHP reads
-- this same source file from the deployment repository.
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS music (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    artist TEXT NOT NULL,
    album TEXT NOT NULL DEFAULT '',
    album_artist TEXT NOT NULL DEFAULT '',
    genre TEXT NOT NULL DEFAULT '',
    year TEXT NOT NULL DEFAULT '',
    source_name TEXT NOT NULL,
    source_path TEXT NOT NULL,
    original_path TEXT NOT NULL DEFAULT '',
    cover_path TEXT NOT NULL DEFAULT '',
    full_path TEXT NOT NULL DEFAULT '',
    lite_path TEXT NOT NULL DEFAULT '',
    duration_seconds REAL,
    full_size INTEGER,
    lite_size INTEGER,
    state TEXT NOT NULL CHECK(state IN ('queued','processing','ready','failed')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS upload_requests (
    idempotency_key TEXT PRIMARY KEY,
    upload_id TEXT NOT NULL UNIQUE,
    task_id TEXT NOT NULL UNIQUE,
    music_id TEXT NOT NULL UNIQUE REFERENCES music(id),
    content_sha256 TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS music_tasks (
    id TEXT PRIMARY KEY,
    music_id TEXT NOT NULL UNIQUE REFERENCES music(id),
    state TEXT NOT NULL CHECK(state IN ('queued','processing','done','failed')),
    locked_by TEXT NOT NULL DEFAULT '',
    lease_until TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS music_public_idx ON music(state, created_at DESC);
CREATE INDEX IF NOT EXISTS music_task_claim_idx ON music_tasks(state, lease_until, created_at);

CREATE TABLE IF NOT EXISTS page_statistics (
    path TEXT PRIMARY KEY,
    views INTEGER NOT NULL DEFAULT 0,
    last_visited_at TEXT NOT NULL
);
