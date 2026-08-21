#!/usr/bin/env bash
set -euo pipefail

project_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
backend_dir="$project_root/backend"
output_dir="$backend_dir/bin"
output="$output_dir/jiuin-go"
temporary="$output_dir/.jiuin-go.$$.tmp"

mkdir -p "$output_dir"
trap 'rm -f "$temporary"' EXIT

cd "$backend_dir"
go test ./...
go vet ./...
go build -o "$temporary" ./cmd/jiuin-go
chmod 0755 "$temporary"

# The running/deployed binary is untouched until all tests and compilation
# have succeeded. mv is atomic when output_dir is on one filesystem.
mv -f "$temporary" "$output"
trap - EXIT
printf 'Built %s\n' "$output"
