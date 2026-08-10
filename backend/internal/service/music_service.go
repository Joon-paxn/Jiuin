package service

import (
	"context"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type MusicService interface {
	List(context.Context) ([]model.MusicTrack, error)
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
