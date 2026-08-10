package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type LinkService interface {
	List(context.Context) ([]model.ExternalLink, error)
}

type linkService struct {
	repository repository.LinkRepository
}

func NewLinkService(repository repository.LinkRepository) LinkService {
	return linkService{repository: repository}
}

func (service linkService) List(ctx context.Context) ([]model.ExternalLink, error) {
	links, err := service.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if err := validateExternalLink(link); err != nil {
			return nil, err
		}
	}

	return links, nil
}

func validateExternalLink(link model.ExternalLink) error {
	if strings.TrimSpace(link.Name) == "" || len(link.Name) > 120 || len(link.Description) > 500 {
		return fmt.Errorf("external link metadata is invalid")
	}

	parsed, err := url.ParseRequestURI(link.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("external link URL must use HTTPS and include a host")
	}

	return nil
}
