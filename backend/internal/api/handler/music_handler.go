package handler

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type MusicHandler struct {
	service service.MusicService
	logger  *slog.Logger
}

func NewMusicHandler(service service.MusicService, logger *slog.Logger) MusicHandler {
	return MusicHandler{service: service, logger: logger}
}

func (handler MusicHandler) List(w http.ResponseWriter, r *http.Request) {
	tracks, err := handler.service.List(r.Context())
	if err != nil {
		handler.logger.Error("failed to load music list", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load music list")
		return
	}

	response.Success(w, tracks)
}
