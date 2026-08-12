package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDContextKey struct{}

const requestIDHeader = "X-Request-ID"

// RequestID assigns an opaque correlation ID. Client-supplied IDs are ignored
// so logs cannot be spoofed or correlated with sensitive client values.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		if requestID != "" {
			w.Header().Set(requestIDHeader, requestID)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID)))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}

	// crypto/rand failures are exceptionally rare. Do not fabricate a
	// predictable correlation value; leave it absent while keeping the request
	// functional.
	return ""
}
