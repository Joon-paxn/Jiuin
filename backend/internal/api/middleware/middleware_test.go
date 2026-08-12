package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitReturnsNumericRetryAfter(t *testing.T) {
	t.Parallel()

	handler := RateLimit(1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first response status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second response status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if retryAfter := second.Header().Get("Retry-After"); retryAfter == "" || retryAfter == "0s" {
		t.Fatalf("Retry-After = %q, want positive numeric seconds", retryAfter)
	}
	if contentType := second.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON response", contentType)
	}
}

func TestTrustedClientAddressUsesForwardedIPOnlyFromLoopbackProxy(t *testing.T) {
	t.Parallel()

	if address := trustedClientAddress("127.0.0.1:51234", "203.0.113.4"); address != "203.0.113.4" {
		t.Fatalf("loopback proxy address = %q, want forwarded client address", address)
	}
	if address := trustedClientAddress("198.51.100.12:51234", "203.0.113.4"); address != "198.51.100.12" {
		t.Fatalf("public peer address = %q, want direct peer address", address)
	}
	if address := trustedClientAddress("127.0.0.1:51234", "not-an-ip"); address != "127.0.0.1" {
		t.Fatalf("invalid forwarded address = %q, want loopback address", address)
	}
}

func TestAllowMethodsRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	handler := AllowMethods(http.MethodGet)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}

func TestCORSRejectsUnconfiguredOriginWithJSON(t *testing.T) {
	t.Parallel()

	handler := CORS([]string{"https://jiuin.cn"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON response", contentType)
	}
}

func TestRequestIDAddsServerGeneratedIdentifier(t *testing.T) {
	t.Parallel()

	var contextID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(requestIDHeader, "client-controlled")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if contextID == "" || contextID == "client-controlled" {
		t.Fatalf("context request ID = %q, want a server-generated ID", contextID)
	}
	if headerID := recorder.Header().Get(requestIDHeader); headerID != contextID {
		t.Fatalf("header request ID = %q, want %q", headerID, contextID)
	}
}
