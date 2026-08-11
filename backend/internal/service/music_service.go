package service

import (
	"context"
	"errors"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

var ErrMusicNotFound = errors.New("music track not found")

type MusicService interface {
	List(context.Context) ([]model.MusicTrack, error)
	Open(context.Context, string) (model.MusicAsset, error)
}

type musicService struct {
	repository repository.MusicRepository
}

func NewMusicService(repository repository.MusicRepository) MusicService {
	return musicService{repository: repository}
}

func (service musicService) List(ctx context.Context) ([]model.MusicTrack, error) {
	return service.repository.List(ctx)
}

func (service musicService) Open(ctx context.Context, id string) (model.MusicAsset, error) {
	asset, err := service.repository.Open(ctx, id)
	if errors.Is(err, repository.ErrMusicNotFound) {
		return model.MusicAsset{}, ErrMusicNotFound
	}

	return asset, err
}
