#!/usr/bin/env bash
set -euo pipefail

public_base="${JIUIN_PUBLIC_BASE_URL:?Set JIUIN_PUBLIC_BASE_URL, for example https://jiuin.cn}"
php_stop="${JIUIN_PHP_STOP_COMMAND:?Set the BaoTa/PHP-FPM stop command}"
php_start="${JIUIN_PHP_START_COMMAND:?Set the BaoTa/PHP-FPM start command}"

request() {
  curl --fail --silent --show-error --max-time 10 -D - "$public_base/api/v1/music" -o /dev/null
}

printf '%s\n' '[1/3] Checking PHP primary.'
request | grep -qi '^X-Jiuin-Backend: php'

printf '%s\n' '[2/3] Stopping PHP-FPM and checking Go fallback.'
eval "$php_stop"
trap 'eval "$php_start"' EXIT
request | grep -qi '^X-Jiuin-Backend: go'

printf '%s\n' '[3/3] Restoring PHP-FPM and checking primary recovery.'
eval "$php_start"
sleep 2
request | grep -qi '^X-Jiuin-Backend: php'
trap - EXIT
printf '%s\n' 'Failover and recovery passed.'
