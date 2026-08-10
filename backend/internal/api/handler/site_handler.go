package handler

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type SiteHandler struct {
	service service.SiteService
	logger  *slog.Logger
}

func NewSiteHandler(service service.SiteService, logger *slog.Logger) SiteHandler {
	return SiteHandler{service: service, logger: logger}
}

func (handler SiteHandler) Info(w http.ResponseWriter, r *http.Request) {
	info, err := handler.service.Info(r.Context())
	if err != nil {
		handler.logger.Error("failed to load site information", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load site information")
		return
	}

	response.Success(w, info)
}

func (handler SiteHandler) Copyright(w http.ResponseWriter, r *http.Request) {
	response.Success(w, handler.service.Copyright(r.Context()))
}

func (handler SiteHandler) Shared(w http.ResponseWriter, r *http.Request) {
	configuration, err := handler.service.Shared(r.Context())
	if err != nil {
		handler.logger.Error("failed to load shared site configuration", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load shared site configuration")
		return
	}

	response.Success(w, configuration)
}

func Health(w http.ResponseWriter, _ *http.Request) {
	response.Success(w, model.HealthStatus{Status: "ok"})
}
