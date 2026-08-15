#!/usr/bin/env bash
set -euo pipefail

# Build a portable Linux backend directory containing the Go binary, FFmpeg,
# FFprobe, configuration template and persistent-storage placeholder.
#
# Usage:
#   ./scripts/build-backend-linux.sh
#   ./scripts/build-backend-linux.sh --install-ffmpeg
#   ./scripts/build-backend-linux.sh --output /opt/jiuin/backend

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/release/jiuin-backend-linux-amd64"
INSTALL_FFMPEG=0

usage() {
  sed -n '1,14p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-ffmpeg) INSTALL_FFMPEG=1; shift ;;
    --output)
      [[ $# -ge 2 ]] || { echo '--output requires a directory' >&2; exit 2; }
      OUTPUT_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

command -v go >/dev/null 2>&1 || {
  echo 'Go is required. Install Go 1.24+ and retry.' >&2
  exit 1
}

install_ffmpeg() {
  if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y ffmpeg
  elif command -v dnf >/dev/null 2>&1; then
    sudo dnf install -y ffmpeg
  elif command -v yum >/dev/null 2>&1; then
    sudo yum install -y ffmpeg
  elif command -v apk >/dev/null 2>&1; then
    sudo apk add --no-cache ffmpeg
  else
    echo 'No supported package manager found; install FFmpeg and FFprobe manually.' >&2
    exit 1
  fi
}

if ! command -v ffmpeg >/dev/null 2>&1 || ! command -v ffprobe >/dev/null 2>&1; then
  if [[ "$INSTALL_FFMPEG" -eq 1 ]]; then
    install_ffmpeg
  else
    echo 'FFmpeg/FFprobe are required but were not found.' >&2
    echo 'Retry with --install-ffmpeg or install them using your distro package manager.' >&2
    exit 1
  fi
fi

FFMPEG_PATH="$(command -v ffmpeg)"
FFPROBE_PATH="$(command -v ffprobe)"
"$ROOT_DIR/scripts/package-backend.sh" linux-amd64 "$FFMPEG_PATH" "$FFPROBE_PATH" "$OUTPUT_DIR"
