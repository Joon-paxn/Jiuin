package repository

import (
	"context"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

type LinkRepository interface {
	List(context.Context) ([]model.ExternalLink, error)
}

type staticLinkRepository struct {
	links []model.ExternalLink
}

func NewStaticLinkRepository(links []model.ExternalLink) LinkRepository {
	return staticLinkRepository{links: append([]model.ExternalLink{}, links...)}
}

func (repository staticLinkRepository) List(context.Context) ([]model.ExternalLink, error) {
	return append([]model.ExternalLink{}, repository.links...), nil
}
