package service

import (
	"context"
	"testing"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
)

type stubSiteRepository struct {
	site model.SiteInfo
}

func (repository stubSiteRepository) Get(context.Context) (model.SiteInfo, error) {
	return repository.site, nil
}

func TestSiteServiceReturnsConfiguredInformationAndCurrentYear(t *testing.T) {
	t.Parallel()

	service := siteService{
		repository: stubSiteRepository{site: model.SiteInfo{
			Name:    "霁雪居",
			Project: "Jiuin",
			Domain:  "Jiuin.cn",
		}},
		copyrightText: "Jiuin.cn",
		now: func() time.Time {
			return time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
		},
	}

	info, err := service.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() returned an error: %v", err)
	}
	if info.Project != "Jiuin" {
		t.Fatalf("Info().Project = %q, want Jiuin", info.Project)
	}

	copyright := service.Copyright(context.Background())
	if copyright.Year != 2026 || copyright.Text != "Jiuin.cn" {
		t.Fatalf("Copyright() = %#v, want year 2026 and text Jiuin.cn", copyright)
	}
}
