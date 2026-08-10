package service

import (
	"context"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

func TestLinkServiceRejectsUnsafeDestinations(t *testing.T) {
	t.Parallel()

	valid := NewLinkService(repository.NewStaticLinkRepository([]model.ExternalLink{{
		Name: "GitHub", URL: "https://github.com/Joon-paxn/Jiuin", Description: "Source code",
	}}))
	if _, err := valid.List(context.Background()); err != nil {
		t.Fatalf("valid HTTPS link was rejected: %v", err)
	}

	unsafe := NewLinkService(repository.NewStaticLinkRepository([]model.ExternalLink{{
		Name: "Unsafe", URL: "javascript:alert(1)", Description: "Unsafe URL",
	}}))
	if _, err := unsafe.List(context.Background()); err == nil {
		t.Fatal("expected unsafe link URL to be rejected")
	}
}
