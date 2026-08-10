package repository

import (
	"context"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

// SiteRepository is the persistence boundary for shared site information.
// Future implementations may load the same model from MySQL, PostgreSQL, or SQLite.
type SiteRepository interface {
	Get(context.Context) (model.SiteInfo, error)
}

type configSiteRepository struct {
	site model.SiteInfo
}

func NewConfigSiteRepository(site model.SiteInfo) SiteRepository {
	return configSiteRepository{site: site}
}

func (repository configSiteRepository) Get(context.Context) (model.SiteInfo, error) {
	return repository.site, nil
}
