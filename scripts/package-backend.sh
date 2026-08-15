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
if [[ "$TARGET" == "linux-amd64" ]]; then
  cp "$ROOT_DIR/scripts/start-backend-linux.sh" "$OUTPUT_DIR/start-backend-linux.sh"
  chmod +x "$OUTPUT_DIR/start-backend-linux.sh"
fi
sed \
  -e 's|^JIUIN_MUSIC_DIRECTORY=.*|JIUIN_MUSIC_DIRECTORY=storage/music|' \
  -e "s|^JIUIN_FFMPEG_PATH=.*|JIUIN_FFMPEG_PATH=./$(basename "$FFMPEG_PATH")|" \
  -e "s|^JIUIN_FFPROBE_PATH=.*|JIUIN_FFPROBE_PATH=./$(basename "$FFPROBE_PATH")|" \
  "$ROOT_DIR/backend/configs/production.env.example" > "$OUTPUT_DIR/backend.env.example"
echo "Backend package created: $OUTPUT_DIR"
