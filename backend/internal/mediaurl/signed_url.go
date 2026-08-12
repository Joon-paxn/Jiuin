// Package mediaurl contains server-only primitives for issuing and validating
// short-lived media URLs. It deliberately has no HTTP route yet: public music
// remains public, while any future private-media handler can opt in without
// exposing its storage URL or a signing secret to browser code.
package mediaurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidPath      = errors.New("signed media path is invalid")
	ErrInvalidSignature = errors.New("signed media signature is invalid")
	ErrExpired          = errors.New("signed media URL has expired")
)

// Signer has a dedicated media signing secret. Do not reuse the service-token
// credential used for server-to-server API authentication.
type Signer struct {
	secret []byte
	now    func() time.Time
}

// NewSigner accepts a server-side, independent secret. A 32-byte minimum
// prevents accidentally using a short placeholder as a production key.
func NewSigner(secret string) (*Signer, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("media signing secret must contain at least 32 bytes")
	}

	return &Signer{secret: []byte(secret), now: time.Now}, nil
}

// Sign returns URL query values suitable for a future protected media route:
// ?expires=<unix-seconds>&signature=<base64url-hmac>. The path itself remains
// outside the token so it can be routed normally by net/http.
func (signer *Signer) Sign(mediaPath string, expiresAt time.Time) (url.Values, error) {
	normalizedPath, err := normalizePath(mediaPath)
	if err != nil {
		return nil, err
	}

	expiresAt = expiresAt.UTC().Truncate(time.Second)
	if !expiresAt.After(signer.now().UTC()) {
		return nil, ErrExpired
	}

	expires := expiresAt.Unix()
	values := make(url.Values, 2)
	values.Set("expires", strconv.FormatInt(expires, 10))
	values.Set("signature", signer.signature(normalizedPath, expires))
	return values, nil
}

// Verify validates a query token against the request path and the server's
// current time. Signature comparison is constant-time through hmac.Equal.
func (signer *Signer) Verify(mediaPath string, values url.Values) error {
	normalizedPath, err := normalizePath(mediaPath)
	if err != nil {
		return err
	}

	expires, err := strconv.ParseInt(values.Get("expires"), 10, 64)
	if err != nil || expires <= 0 {
		return ErrInvalidSignature
	}
	if !time.Unix(expires, 0).After(signer.now().UTC()) {
		return ErrExpired
	}

	provided, err := base64.RawURLEncoding.DecodeString(values.Get("signature"))
	if err != nil {
		return ErrInvalidSignature
	}
	expected, err := base64.RawURLEncoding.DecodeString(signer.signature(normalizedPath, expires))
	if err != nil || !hmac.Equal(provided, expected) {
		return ErrInvalidSignature
	}

	return nil
}

func (signer *Signer) signature(mediaPath string, expires int64) string {
	mac := hmac.New(sha256.New, signer.secret)
	_, _ = mac.Write([]byte("jiuin-media-v1\n"))
	_, _ = mac.Write([]byte(mediaPath))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(strconv.FormatInt(expires, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func normalizePath(value string) (string, error) {
	if strings.TrimSpace(value) == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsRune(value, '\x00') {
		return "", ErrInvalidPath
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path == "" {
		return "", ErrInvalidPath
	}
	if cleaned := path.Clean(parsed.Path); cleaned != parsed.Path {
		return "", ErrInvalidPath
	}

	return parsed.Path, nil
}
