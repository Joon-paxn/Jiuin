package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Envelope{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

// Accepted is used for intentionally asynchronous requests such as media
// uploads. It keeps the standard response envelope while accurately exposing
// HTTP 202 to clients and intermediaries.
func Accepted(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusAccepted, Envelope{
		Code:    http.StatusAccepted,
		Message: "accepted",
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Envelope{
		Code:    status,
		Message: message,
	})
}

func writeJSON(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
