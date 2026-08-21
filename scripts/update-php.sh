#!/usr/bin/env bash
set -euo pipefail

project_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
php_bin="${JIUIN_PHP_BIN:?Set JIUIN_PHP_BIN to the BaoTa PHP CLI binary}"
reload_command="${JIUIN_PHP_FPM_RELOAD_COMMAND:-}"

cd "$project_root"
git pull --ff-only
while IFS= read -r -d '' source; do
  "$php_bin" -l "$source" >/dev/null
done < <(find backend/php -type f -name '*.php' -print0)

if [ -n "$reload_command" ]; then
  eval "$reload_command"
fi
printf '%s\n' 'PHP syntax checks passed.'
