package model

type SiteInfo struct {
	Name    string `json:"name"`
	Project string `json:"project"`
	Domain  string `json:"domain"`
}

type Copyright struct {
	Year int    `json:"year"`
	Text string `json:"text"`
}

type HealthStatus struct {
	Status string `json:"status"`
}
