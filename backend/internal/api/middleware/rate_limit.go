package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
)

type rateLimitEntry struct {
	windowStartedAt time.Time
	requests        int
	lastSeenAt      time.Time
}

// RateLimit is a bounded, in-process fixed-window limiter. It uses the direct
// peer address by default. A loopback reverse proxy may supply X-Real-IP; the
// header is deliberately ignored for non-loopback peers so internet clients
// cannot choose their own rate-limit bucket.
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	if maxRequests < 1 {
		maxRequests = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	var mutex sync.Mutex
	entries := make(map[string]rateLimitEntry)
	var lastPrunedAt time.Time

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			key := trustedClientAddress(r.RemoteAddr, r.Header.Get("X-Real-IP"))

			mutex.Lock()
			if now.Sub(lastPrunedAt) >= window {
				for entryKey, entry := range entries {
					if now.Sub(entry.lastSeenAt) > window*2 {
						delete(entries, entryKey)
					}
				}
				lastPrunedAt = now
			}

			entry := entries[key]
			if entry.windowStartedAt.IsZero() || now.Sub(entry.windowStartedAt) >= window {
				entry = rateLimitEntry{windowStartedAt: now}
			}
			entry.requests++
			entry.lastSeenAt = now
			entries[key] = entry
			allowed := entry.requests <= maxRequests
			retryAfter := window - now.Sub(entry.windowStartedAt)
			mutex.Unlock()

			if !allowed {
				w.Header().Set("Retry-After", retryAfterSeconds(retryAfter))
				response.Error(w, http.StatusTooManyRequests, "request rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func retryAfterSeconds(value time.Duration) string {
	seconds := int(math.Ceil(value.Seconds()))
	if seconds < 1 {
		seconds = 1
	}

	return strconv.Itoa(seconds)
}

func clientAddress(remoteAddress string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddress))
	if err != nil || host == "" {
		host = strings.TrimSpace(remoteAddress)
	}
	if host == "" {
		return "unknown"
	}

	return host
}

func trustedClientAddress(remoteAddress, forwardedAddress string) string {
	directAddress := clientAddress(remoteAddress)
	directIP := net.ParseIP(directAddress)
	if directIP == nil || !directIP.IsLoopback() {
		return directAddress
	}

	forwardedIP := net.ParseIP(strings.TrimSpace(forwardedAddress))
	if forwardedIP == nil {
		return directAddress
	}

	return forwardedIP.String()
}
