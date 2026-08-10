package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
)

// RequireServiceToken protects server-to-server mutations. Browser code must never receive this token.
func RequireServiceToken(token string) func(http.Handler) http.Handler {
	expected := "Bearer " + token

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := strings.TrimSpace(r.Header.Get("Authorization"))
			if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				response.Error(w, http.StatusUnauthorized, "service authorization is required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
