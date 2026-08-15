#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
OUTPUT_PATH="${1:-$BACKEND_DIR/jiuin-server}"

cd "$BACKEND_DIR"
go test ./...
go vet ./...
go build -trimpath -ldflags='-s -w' -o "$OUTPUT_PATH" ./cmd/server
printf 'Built %s\n' "$OUTPUT_PATH"
