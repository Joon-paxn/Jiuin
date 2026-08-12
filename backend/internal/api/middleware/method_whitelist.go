package middleware

import (
	"net/http"
	"strings"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
)

// AllowMethods rejects methods this read-mostly public API never supports
// before they reach a route.
func AllowMethods(methods ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(methods))
	orderedMethods := make([]string, 0, len(methods))

	for _, method := range methods {
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" {
			continue
		}
		if _, exists := allowed[method]; exists {
			continue
		}

		allowed[method] = struct{}{}
		orderedMethods = append(orderedMethods, method)
	}

	allowHeader := strings.Join(orderedMethods, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[r.Method]; !ok {
				if allowHeader != "" {
					w.Header().Set("Allow", allowHeader)
				}
				response.Error(w, http.StatusMethodNotAllowed, "request method is not allowed")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireMethods provides per-route method checks with the same JSON error
// envelope used by the rest of the API. This avoids net/http's default plain
// text 405 body when a known path is called with another allowed global method.
func RequireMethods(methods ...string) func(http.Handler) http.Handler {
	return AllowMethods(methods...)
}
