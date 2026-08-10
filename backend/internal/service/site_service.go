package service

import (
	"context"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

type SiteService interface {
	Info(context.Context) (model.SiteInfo, error)
	Copyright(context.Context) model.Copyright
	Shared(context.Context) (model.SharedSiteConfiguration, error)
}

type siteService struct {
	repository    repository.SiteRepository
	copyrightText string
	now           func() time.Time
}

func NewSiteService(repository repository.SiteRepository, copyrightText string) SiteService {
	return siteService{
		repository:    repository,
		copyrightText: copyrightText,
		now:           time.Now,
	}
}

func (service siteService) Info(ctx context.Context) (model.SiteInfo, error) {
	return service.repository.Get(ctx)
}

func (service siteService) Copyright(context.Context) model.Copyright {
	return model.Copyright{
		Year: service.now().Year(),
		Text: service.copyrightText,
	}
}

func (service siteService) Shared(ctx context.Context) (model.SharedSiteConfiguration, error) {
	info, err := service.Info(ctx)
	if err != nil {
		return model.SharedSiteConfiguration{}, err
	}

	return model.SharedSiteConfiguration{Site: info, Copyright: service.Copyright(ctx)}, nil
}
