package repository

import (
	"context"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

// MusicRepository is the future storage boundary for music metadata and stream URLs.
type MusicRepository interface {
	List(context.Context) ([]model.MusicTrack, error)
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
