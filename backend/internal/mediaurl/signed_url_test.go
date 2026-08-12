package mediaurl

import (
	"errors"
	"testing"
	"time"
)

func TestSignerSignsAndVerifiesBoundPath(t *testing.T) {
	t.Parallel()

	signer, err := NewSigner("a-dedicated-test-media-signing-key-with-32-bytes")
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	now := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return now }

	values, err := signer.Sign("/media/private/track-123", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if err := signer.Verify("/media/private/track-123", values); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if err := signer.Verify("/media/private/other-track", values); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify() error = %v, want ErrInvalidSignature for another path", err)
	}
}

func TestSignerRejectsExpiredAndUnsafePaths(t *testing.T) {
	t.Parallel()

	signer, err := NewSigner("a-dedicated-test-media-signing-key-with-32-bytes")
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	now := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return now }

	if _, err := signer.Sign("/media/../admin", now.Add(time.Minute)); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Sign() error = %v, want ErrInvalidPath", err)
	}
	if _, err := signer.Sign("/media/private/track", now); !errors.Is(err, ErrExpired) {
		t.Fatalf("Sign() error = %v, want ErrExpired", err)
	}
}
