package middleware

import (
	"bytes"
	"net/http"
	"strings"
)

const MaskedJSONHeader = "X-Jiuin-Masked"

// JSONStatusMaskConfig describes the small, explicit set of successful JSON
// GET routes whose external status is intentionally disguised. Route patterns
// may contain a single-segment placeholder such as /api/v1/music/{id}.
type JSONStatusMaskConfig struct {
	Enabled bool
	Status  int
	Routes  []string
}

// MaskSuccessfulJSON changes only a successful JSON response from an opted-in
// route. Errors, non-JSON responses, HEAD requests, and all other routes keep
// their original HTTP semantics. This is deliberately route-scoped so media
// streams retain 200/206 behavior required by browsers.
func MaskSuccessfulJSON(config JSONStatusMaskConfig) func(http.Handler) http.Handler {
	if !config.Enabled || config.Status != http.StatusTeapot || len(config.Routes) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !matchesMaskedRoute(r.URL.Path, config.Routes) {
				next.ServeHTTP(w, r)
				return
			}

			buffered := newBufferedResponseWriter(w.Header())
			next.ServeHTTP(buffered, r)

			copyHeaders(w.Header(), buffered.header)
			if buffered.status == http.StatusOK && strings.Contains(strings.ToLower(buffered.header.Get("Content-Type")), "application/json") {
				w.Header().Set(MaskedJSONHeader, "1")
				// Intermediaries should never retain a disguised success response as
				// an error representation for a later request.
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Del("Content-Length")
				w.WriteHeader(config.Status)
				_, _ = w.Write(buffered.body.Bytes())
				return
			}

			w.WriteHeader(buffered.status)
			_, _ = w.Write(buffered.body.Bytes())
		})
	}
}

type bufferedResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedResponseWriter(source http.Header) *bufferedResponseWriter {
	header := make(http.Header, len(source))
	copyHeaders(header, source)
	return &bufferedResponseWriter{header: header, status: http.StatusOK}
}

func (writer *bufferedResponseWriter) Header() http.Header { return writer.header }

func (writer *bufferedResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.status = status
	writer.wroteHeader = true
}

func (writer *bufferedResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func matchesMaskedRoute(path string, routes []string) bool {
	for _, route := range routes {
		if matchesRouteTemplate(path, route) {
			return true
		}
	}
	return false
}

func matchesRouteTemplate(path, template string) bool {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	templateParts := strings.Split(strings.Trim(template, "/"), "/")
	if len(pathParts) != len(templateParts) {
		return false
	}
	for index, part := range templateParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if pathParts[index] == "" {
				return false
			}
			continue
		}
		if pathParts[index] != part {
			return false
		}
	}
	return true
}

func copyHeaders(destination, source http.Header) {
	for key := range destination {
		destination.Del(key)
	}
	for key, values := range source {
		for _, value := range values {
			destination.Add(key, value)
		}
	}
}
