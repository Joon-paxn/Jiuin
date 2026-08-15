#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-linux-amd64}"
FFMPEG_PATH="${2:?usage: package-backend.sh <linux-amd64|windows-amd64> <ffmpeg> <ffprobe> [output-dir]}"
FFPROBE_PATH="${3:?usage: package-backend.sh <linux-amd64|windows-amd64> <ffmpeg> <ffprobe> [output-dir]}"
OUTPUT_DIR="${4:-$ROOT_DIR/release/jiuin-backend-$TARGET}"

test -f "$FFMPEG_PATH" || { echo "FFmpeg not found: $FFMPEG_PATH" >&2; exit 1; }
test -f "$FFPROBE_PATH" || { echo "FFprobe not found: $FFPROBE_PATH" >&2; exit 1; }
mkdir -p "$OUTPUT_DIR/storage/music"

cd "$ROOT_DIR/backend"
go test ./...
go vet ./...
case "$TARGET" in
  linux-amd64) GOOS=linux GOARCH=amd64 BINARY=jiuin-server ;;
  windows-amd64) GOOS=windows GOARCH=amd64 BINARY=jiuin-server.exe ;;
  *) echo "unsupported target: $TARGET" >&2; exit 1 ;;
esac
GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags='-s -w' -o "$OUTPUT_DIR/$BINARY" ./cmd/server

cp "$FFMPEG_PATH" "$OUTPUT_DIR/$(basename "$FFMPEG_PATH")"
cp "$FFPROBE_PATH" "$OUTPUT_DIR/$(basename "$FFPROBE_PATH")"
if [[ "$TARGET" == "windows-amd64" ]]; then
  runtime_dir="$(dirname "$FFMPEG_PATH")"
  find "$runtime_dir" -maxdepth 1 -type f ! -name "$(basename "$FFMPEG_PATH")" ! -name "$(basename "$FFPROBE_PATH")" -exec cp {} "$OUTPUT_DIR/" \;
fi
cp README.md "$OUTPUT_DIR/README.backend.md"
cat > "$OUTPUT_DIR/backend.env.example" <<EOF
JIUIN_ENV=production
JIUIN_SERVER_HOST=127.0.0.1
JIUIN_SERVER_PORT=8080
JIUIN_MUSIC_DIRECTORY=storage/music
JIUIN_FFMPEG_PATH=./$(basename "$FFMPEG_PATH")
JIUIN_FFPROBE_PATH=./$(basename "$FFPROBE_PATH")
JIUIN_MUSIC_ADMIN_TOKEN=replace-with-a-long-random-production-token
JIUIN_SHARED_SERVICE_TOKEN=replace-with-a-long-random-production-token
EOF
echo "Backend package created: $OUTPUT_DIR"
