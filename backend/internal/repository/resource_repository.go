package repository

import (
	"context"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

type ResourceRepository interface {
	List(context.Context) ([]model.ResourceDescriptor, error)
}

type staticResourceRepository struct {
	resources []model.ResourceDescriptor
}

func NewStaticResourceRepository(resources []model.ResourceDescriptor) ResourceRepository {
	return staticResourceRepository{resources: append([]model.ResourceDescriptor{}, resources...)}
}

func (repository staticResourceRepository) List(context.Context) ([]model.ResourceDescriptor, error) {
	return append([]model.ResourceDescriptor{}, repository.resources...), nil
}
