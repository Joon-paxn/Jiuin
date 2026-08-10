package handler

import (
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type ResourceHandler struct {
	service service.ResourceService
	logger  *slog.Logger
}

func NewResourceHandler(service service.ResourceService, logger *slog.Logger) ResourceHandler {
	return ResourceHandler{service: service, logger: logger}
}

func (handler ResourceHandler) List(w http.ResponseWriter, r *http.Request) {
	resources, err := handler.service.List(r.Context())
	if err != nil {
		handler.logger.Error("failed to load resource manifest", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load resource manifest")
		return
	}

	response.Success(w, resources)
}
