package repository

import (
	"context"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

type StatusRepository interface {
	Get(context.Context) (model.EcosystemStatus, error)
}

type staticStatusRepository struct {
	status model.EcosystemStatus
}

func NewStaticStatusRepository(status model.EcosystemStatus) StatusRepository {
	return staticStatusRepository{status: status}
}

func (repository staticStatusRepository) Get(context.Context) (model.EcosystemStatus, error) {
	return repository.status, nil
}
