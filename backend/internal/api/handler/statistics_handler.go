package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
	"github.com/Joon-paxn/Jiuin/backend/internal/service"
)

type StatisticsHandler struct {
	service service.StatisticsService
	logger  *slog.Logger
}

type recordVisitRequest struct {
	Path string `json:"path"`
}

func NewStatisticsHandler(service service.StatisticsService, logger *slog.Logger) StatisticsHandler {
	return StatisticsHandler{service: service, logger: logger}
}

func (handler StatisticsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	statistics, err := handler.service.Summary(r.Context())
	if err != nil {
		handler.logger.Error("failed to load statistics", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to load statistics")
		return
	}

	response.Success(w, statistics)
}

func (handler StatisticsHandler) Record(w http.ResponseWriter, r *http.Request) {
	request, err := decodeRecordVisitRequest(w, r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid statistics visit payload")
		return
	}

	page, err := handler.service.Record(r.Context(), request.Path)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid statistics visit payload")
		return
	}

	response.Success(w, page)
}

func decodeRecordVisitRequest(w http.ResponseWriter, r *http.Request) (recordVisitRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request recordVisitRequest
	if err := decoder.Decode(&request); err != nil {
		return recordVisitRequest{}, err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return recordVisitRequest{}, errors.New("request body must contain one JSON object")
	}

	return request, nil
}
