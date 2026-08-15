#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ENV_FILE="${JIUIN_ENV_FILE:-$SCRIPT_DIR/backend.env}"
BINARY="${JIUIN_BINARY:-$SCRIPT_DIR/jiuin-server}"
PID_FILE="$SCRIPT_DIR/jiuin-server.pid"
LOG_FILE="$SCRIPT_DIR/jiuin-server.log"
SERVICE_NAME="${JIUIN_SERVICE_NAME:-jiuin-backend.service}"
MODE="foreground"

systemd_active() {
  command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet "$SERVICE_NAME"
}

usage() {
  cat <<'EOF'
Usage: ./start-backend-linux.sh [--daemon|--stop|--status]

  (default)  Start in the foreground
  --daemon   Start in the background and write jiuin-server.log
  --stop     Stop the background process started by --daemon
  --status   Show background process status
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
elif [[ "${1:-}" == "--daemon" ]]; then
  MODE="daemon"
elif [[ "${1:-}" == "--stop" ]]; then
  if systemd_active; then
    systemctl stop "$SERVICE_NAME"
    echo "Stopped Jiuin backend systemd service ($SERVICE_NAME)."
    exit 0
  fi
  if [[ -f "$PID_FILE" ]]; then
    pid="$(cat "$PID_FILE")"
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid"
      echo "Stopped Jiuin backend (PID $pid)."
    else
      echo "Stale PID file removed."
    fi
    rm -f "$PID_FILE"
  else
    echo 'Jiuin backend is not running.'
  fi
  exit 0
elif [[ "${1:-}" == "--status" ]]; then
  if systemd_active; then
    echo "Jiuin backend is running under systemd ($SERVICE_NAME)."
  elif [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "Jiuin backend is running (PID $(cat "$PID_FILE"))."
  else
    echo 'Jiuin backend is not running.'
  fi
  exit 0
elif [[ $# -gt 0 ]]; then
  usage >&2
  exit 2
fi

[[ -f "$ENV_FILE" ]] || {
  echo "Missing $ENV_FILE. Copy backend.env.example to backend.env first." >&2
  exit 1
}
[[ -x "$BINARY" ]] || {
  echo "Backend binary is missing or not executable: $BINARY" >&2
  echo "Run: chmod +x jiuin-server ffmpeg ffprobe" >&2
  exit 1
}

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

[[ "${JIUIN_MUSIC_ADMIN_TOKEN:-}" != replace-with-* ]] || {
  echo 'Set JIUIN_MUSIC_ADMIN_TOKEN in backend.env before starting.' >&2
  exit 1
}
[[ "${JIUIN_SHARED_SERVICE_TOKEN:-}" != replace-with-* ]] || {
  echo 'Set JIUIN_SHARED_SERVICE_TOKEN in backend.env before starting.' >&2
  exit 1
}

music_dir="${JIUIN_MUSIC_DIRECTORY:-storage/music}"
mkdir -p "$music_dir"

for tool_var in JIUIN_FFMPEG_PATH JIUIN_FFPROBE_PATH; do
  tool_path="${!tool_var:-}"
  [[ -n "$tool_path" ]] || { echo "$tool_var is missing in $ENV_FILE." >&2; exit 1; }
  if [[ "$tool_path" == */* ]]; then
    [[ -x "$tool_path" ]] || { echo "$tool_var is not executable: $tool_path" >&2; exit 1; }
  else
    command -v "$tool_path" >/dev/null 2>&1 || { echo "$tool_var executable not found: $tool_path" >&2; exit 1; }
  fi
done

if [[ "$MODE" == "daemon" ]]; then
  if systemd_active; then
    echo "Jiuin backend is already running under systemd ($SERVICE_NAME)."
    exit 0
  fi
  if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "Jiuin backend is already running (PID $(cat "$PID_FILE"))."
    exit 0
  fi
  nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
  echo $! > "$PID_FILE"
  echo "Jiuin backend started (PID $(cat "$PID_FILE")). Log: $LOG_FILE"
else
  exec "$BINARY"
fi
