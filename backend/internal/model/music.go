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
