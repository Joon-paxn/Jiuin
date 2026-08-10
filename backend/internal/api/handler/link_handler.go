package handler

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type LinkHandler struct {
	service service.LinkService
	logger  *slog.Logger
}

func NewLinkHandler(service service.LinkService, logger *slog.Logger) LinkHandler {
	return LinkHandler{service: service, logger: logger}
}

func (handler LinkHandler) List(w http.ResponseWriter, r *http.Request) {
	links, err := handler.service.List(r.Context())
	if err != nil {
		handler.logger.Error("failed to load external links", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load external links")
		return
	}

	response.Success(w, links)
}
