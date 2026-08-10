package model

import "time"

type SharedSiteConfiguration struct {
	Site      SiteInfo  `json:"site"`
	Copyright Copyright `json:"copyright"`
}

type PageStatistics struct {
	Path          string    `json:"path"`
	Views         int64     `json:"views"`
	LastVisitedAt time.Time `json:"lastVisitedAt"`
}

type SiteStatistics struct {
	TotalViews int64            `json:"totalViews"`
	Pages      []PageStatistics `json:"pages"`
}

type VisitRecord struct {
	Path      string
	VisitedAt time.Time
}

type EcosystemStatus struct {
	Site     string          `json:"site"`
	API      string          `json:"api"`
	Services []ServiceStatus `json:"services"`
	Checked  time.Time       `json:"checkedAt"`
}

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ExternalLink struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type ResourceDescriptor struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Priority    int    `json:"priority"`
	CachePolicy string `json:"cachePolicy"`
}
