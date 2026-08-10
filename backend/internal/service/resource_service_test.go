package service

import (
	"context"
	"testing"

	"github.com/Joon-paxn/Jiuin/backend/internal/model"
	"github.com/Joon-paxn/Jiuin/backend/internal/repository"
)

func TestResourceServiceRestrictsResourceManifest(t *testing.T) {
	t.Parallel()

	valid := NewResourceService(repository.NewStaticResourceRepository([]model.ResourceDescriptor{{
		Name: "Site configuration", URL: "/api/v1/site", Priority: 1, CachePolicy: "config",
	}}))
	if _, err := valid.List(context.Background()); err != nil {
		t.Fatalf("valid resource was rejected: %v", err)
	}

	unsafe := NewResourceService(repository.NewStaticResourceRepository([]model.ResourceDescriptor{{
		Name: "External script", URL: "https://outside.example/script.js", Priority: 1, CachePolicy: "static",
	}}))
	if _, err := unsafe.List(context.Background()); err == nil {
		t.Fatal("expected cross-origin resource URL to be rejected")
	}
}
