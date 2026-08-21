#!/usr/bin/env bash
set -euo pipefail

project_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
install_dir="${JIUIN_GO_INSTALL_DIR:?Set JIUIN_GO_INSTALL_DIR to the systemd binary directory}"
service_name="${JIUIN_GO_SERVICE_NAME:-jiuin-go-backup.service}"
health_url="${JIUIN_GO_HEALTH_URL:?Set JIUIN_GO_HEALTH_URL to the loopback /health URL}"
ready_url="${JIUIN_GO_READY_URL:?Set JIUIN_GO_READY_URL to the loopback /ready URL}"
artifact="$project_root/backend/bin/jiuin-go"
installed="$install_dir/jiuin-go"
backup="$install_dir/jiuin-go.previous"
had_previous=0
deployed=0

rollback() {
  status=$?
  if [ "$deployed" -eq 1 ] && [ "$had_previous" -eq 1 ] && [ -f "$backup" ]; then
    printf '%s\n' 'Go update failed; restoring the previous binary.' >&2
    install -m 0755 "$backup" "$installed"
    systemctl restart "$service_name" || true
  fi
  exit "$status"
}
trap rollback ERR

cd "$project_root"
git pull --ff-only
"$project_root/scripts/build-go.sh"

if [ -f "$installed" ]; then
  cp -p "$installed" "$backup"
  had_previous=1
fi
install -m 0755 "$artifact" "$installed"
deployed=1
systemctl restart "$service_name"
systemctl is-active --quiet "$service_name"
curl --fail --silent --show-error --max-time 5 "$health_url" >/dev/null
curl --fail --silent --show-error --max-time 8 "$ready_url" >/dev/null
rm -f "$backup"
deployed=0
trap - ERR
printf '%s\n' 'Go update completed and health checks passed.'
