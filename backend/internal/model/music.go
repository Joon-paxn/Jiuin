package model

import "time"

// AudioQuality is retained for the legacy /music/list endpoint consumed by
// older clients. New clients use PublicMusicTrack.Audio.
type AudioQuality struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	SourceURL   string `json:"sourceUrl"`
	BitrateKbps int    `json:"bitrateKbps,omitempty"`
}

// MusicTrack is the legacy public shape. It intentionally carries only public
// URLs, never storage paths.
type MusicTrack struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Artist          string         `json:"artist"`
	ArtworkURL      string         `json:"artworkUrl,omitempty"`
	SourceURL       string         `json:"sourceUrl,omitempty"`
	DurationSeconds int            `json:"durationSeconds,omitempty"`
	Qualities       []AudioQuality `json:"qualities,omitempty"`
}

type PublicAudioSources struct {
	Full string `json:"full,omitempty"`
	Lite string `json:"lite,omitempty"`
}

// PublicMusicTrack is the current player/API contract. The internal disk
// paths, source hash, and original filename are deliberately absent.
type PublicMusicTrack struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Artist          string             `json:"artist"`
	Album           string             `json:"album"`
	AlbumArtist     string             `json:"albumArtist,omitempty"`
	Genre           string             `json:"genre,omitempty"`
	Year            string             `json:"year,omitempty"`
	DurationSeconds int                `json:"durationSeconds,omitempty"`
	Cover           string             `json:"cover,omitempty"`
	Audio           PublicAudioSources `json:"audio"`
	FullSize        int64              `json:"fullSize,omitempty"`
	LiteSize        int64              `json:"liteSize,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
}

// MusicTaskResponse is the deliberately small public task contract. Processing
// diagnostics and private source/storage details stay in the database and logs.
type MusicTaskResponse struct {
	TaskID   string          `json:"task_id"`
	Status   MusicTaskStatus `json:"status"`
	Progress int             `json:"progress"`
	MusicID  string          `json:"music_id,omitempty"`
}

// MusicRecord is the persisted representation. Paths are private relative
// paths beneath the configured music storage root, not API values.
type MusicRecord struct {
	ID              string
	SourceHash      string
	Title           string
	Artist          string
	Album           string
	AlbumArtist     string
	Genre           string
	Year            string
	DurationSeconds int
	CoverPath       string
	OriginalPath    string
	FullPath        string
	LitePath        string
	FullSize        int64
	LiteSize        int64
	CreatedAt       time.Time
}

type MusicTaskStatus string

const (
	MusicTaskPending    MusicTaskStatus = "pending"
	MusicTaskProcessing MusicTaskStatus = "processing"
	MusicTaskCompleted  MusicTaskStatus = "completed"
	MusicTaskFailed     MusicTaskStatus = "failed"
)

type MusicTask struct {
	ID           string          `json:"task_id"`
	Status       MusicTaskStatus `json:"status"`
	Progress     int             `json:"progress"`
	MusicID      string          `json:"music_id,omitempty"`
	SourceHash   string          `json:"-"`
	OriginalPath string          `json:"-"`
	ErrorType    string          `json:"-"`
	ErrorDetail  string          `json:"-"`
	ExitCode     int             `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// MusicAsset is an internal resolved asset. It is intentionally not returned
// by public endpoints; media handlers pass it to http.ServeContent.
type MusicAsset struct {
	Path string
	Name string
}
