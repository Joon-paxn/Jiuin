package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error(
						"request panic recovered",
						"request_id", RequestIDFromContext(r.Context()),
						"error_type", "panic",
						"error", fmt.Sprint(recovered),
						"stack", string(debug.Stack()),
					)
					if recorder, ok := w.(*responseRecorder); !ok || !recorder.hasWrittenHeader() {
						response.Error(w, http.StatusInternalServerError, "internal server error")
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
