package handler

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type StatusHandler struct {
	service service.StatusService
	logger  *slog.Logger
}

func NewStatusHandler(service service.StatusService, logger *slog.Logger) StatusHandler {
	return StatusHandler{service: service, logger: logger}
}

func (handler StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	status, err := handler.service.Get(r.Context())
	if err != nil {
		handler.logger.Error("failed to load ecosystem status", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "ecosystem status is unavailable")
		return
	}

	response.Success(w, status)
}
