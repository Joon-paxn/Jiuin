package model

type AudioQuality struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	SourceURL   string `json:"sourceUrl"`
	BitrateKbps int    `json:"bitrateKbps,omitempty"`
}

type MusicTrack struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Artist          string         `json:"artist"`
	ArtworkURL      string         `json:"artworkUrl,omitempty"`
	SourceURL       string         `json:"sourceUrl,omitempty"`
	DurationSeconds int            `json:"durationSeconds,omitempty"`
	Qualities       []AudioQuality `json:"qualities,omitempty"`
}

// MusicAsset is an internal, resolved audio file. It is intentionally not
// returned by the public API; callers stream it through a stable track ID.
type MusicAsset struct {
	Path string
	Name string
}
