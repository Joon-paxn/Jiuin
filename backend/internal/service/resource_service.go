package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type ResourceService interface {
	List(context.Context) ([]model.ResourceDescriptor, error)
}

type resourceService struct {
	repository repository.ResourceRepository
}

func NewResourceService(repository repository.ResourceRepository) ResourceService {
	return resourceService{repository: repository}
}

func (service resourceService) List(ctx context.Context) ([]model.ResourceDescriptor, error) {
	resources, err := service.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		if err := validateResource(resource); err != nil {
			return nil, err
		}
	}

	return resources, nil
}

func validateResource(resource model.ResourceDescriptor) error {
	if strings.TrimSpace(resource.Name) == "" || resource.Priority < 1 || resource.Priority > 4 {
		return fmt.Errorf("resource descriptor is invalid")
	}
	if !strings.HasPrefix(resource.URL, "/") || strings.HasPrefix(resource.URL, "//") {
		return fmt.Errorf("resource URL must be a same-origin relative path")
	}
	if resource.CachePolicy != "static" && resource.CachePolicy != "config" && resource.CachePolicy != "media" {
		return fmt.Errorf("resource cache policy is invalid")
	}

	return nil
}
