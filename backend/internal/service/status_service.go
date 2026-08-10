package service

import (
	"context"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type StatusService interface {
	Get(context.Context) (model.EcosystemStatus, error)
}

type statusService struct {
	repository repository.StatusRepository
	now        func() time.Time
}

func NewStatusService(repository repository.StatusRepository) StatusService {
	return statusService{repository: repository, now: time.Now}
}

func (service statusService) Get(ctx context.Context) (model.EcosystemStatus, error) {
	status, err := service.repository.Get(ctx)
	if err != nil {
		return model.EcosystemStatus{}, err
	}
	status.Checked = service.now().UTC()
	return status, nil
}
