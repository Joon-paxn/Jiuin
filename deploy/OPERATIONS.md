# Jiuin Operations

## Architecture

```text
Browser
  -> https://jiuin.cn/{api,media}
  -> wss://jiuin.cn/ws/online
  -> public Nginx site
  -> local TLS gateway selected as bkgapi.jiuin.cn (127.0.0.1 only)
       -> PHP-FPM primary API + PHP CLI music worker
       -> Go backup API + Go music worker + Go WebSocket
       -> /var/lib/jiuin/music/{music.db,original,full,lite,covers,tmp}
```

PHP-FPM is the normal API path. The `jiuin-go-backup.service` process exposes
the complete same HTTP API, the media handler, a backup FFmpeg/FFprobe worker,
and `/ws/online`; it does not depend on PHP to run. WebSocket always uses Go.

## Shared state

Set `JIUIN_STORAGE_DIR=/var/lib/jiuin/music` and
`JIUIN_DATABASE_PATH=/var/lib/jiuin/music/music.db` in `/etc/jiuin/jiuin.env`
for both PHP-FPM, the PHP worker, and the Go service. Grant the `jiuin` user
and the PHP-FPM pool group access to the same directory; do not create per-run
runtime storage directories.

The shared schema enables WAL, `busy_timeout=5000`, and foreign keys. Writes
use short `BEGIN IMMEDIATE` transactions with bounded busy retries. FFmpeg and
FFprobe run outside the transaction. `music_tasks` has a worker ID and lease;
the lease owner must successfully claim the final state update before it can
publish output. Start with a two-hour lease and adjust only after measuring the
longest real transcode.

## Gateway and recovery

Install the templates described in [Nginx README](nginx/README.md). The public
site passes the original `Host`, real address, forwarding chain, and original
scheme to the local `bkgapi` TLS gateway. The gateway preserves those values
for both FastCGI and Go HTTP.

The read locations use `fastcgi_next_upstream` for `error`, `timeout`,
`invalid_header`, and HTTP `500`, `502`, `503`, `504`, then an internal named
location proxies the unchanged request to Go. It cannot fail over after a
response body has started, so no configuration promises that case is invisible.
The upload location is the only write route that can take the same path: it
requires an idempotency key, Nginx buffers the body, and the shared request
ledger/deterministic IDs prevent a duplicate row, file identity, task, or
FFmpeg job. Other writes never get Nginx replayed.

There is no permanent routing state to Go. The next request is always offered
to PHP first; after PHP-FPM recovers and its configured five-second
`fail_timeout` expires, it is selected again. Go may be down during normal
operation without affecting PHP. If both are down, the gateway returns the
normal 502/503 class failure.

## BaoTa and systemd

BaoTa manages the static site, TLS for `jiuin.cn`, PHP-FPM, its socket, and
the PHP FPM environment. It should not be replaced with a Docker deployment.
Install these services after substituting placeholders:

```text
jiuin-go-backup.service
jiuin-php-music-worker.service
```

Use `systemctl restart jiuin-go-backup.service` and
`systemctl restart jiuin-php-music-worker.service`; inspect Go with
`journalctl -u jiuin-go-backup.service -f`. PHP and Nginx logs remain the
BaoTa-managed logs. Do not restart the host for an application update.

## Build and updates

```bash
scripts/build-go.sh
JIUIN_GO_INSTALL_DIR=/opt/jiuin/go \
JIUIN_GO_HEALTH_URL=http://127.0.0.1:ACTUAL_GO_PORT/health \
JIUIN_GO_READY_URL=http://127.0.0.1:ACTUAL_GO_PORT/ready \
scripts/update-go.sh
```

The build script runs test, vet, and build into a temporary artifact before an
atomic replacement. The updater backs up the installed executable, restarts,
checks health/ready, and restores that binary if a check fails.

For PHP, set the actual BaoTa CLI binary and optional FPM reload command:

```bash
JIUIN_PHP_BIN=/actual/baota/php/bin/php \
JIUIN_PHP_FPM_RELOAD_COMMAND='actual-safe-reload-command' \
scripts/update-php.sh
```

## Preflight and acceptance

Discover, do not invent, the values that replace all Nginx placeholders:

```bash
ss -ltnp
find /www/server/php -type s -name '*.sock'
nginx -T
dig +short jiuin.cn
dig +short bkgapi.jiuin.cn
openssl s_client -connect bkgapi.jiuin.cn:ACTUAL_PORT -servername bkgapi.jiuin.cn </dev/null
nginx -t && nginx -s reload
```

The `bkgapi` listener must be loopback-only. The hostname still needs correct
DNS when its certificate issuance/validation requires it; a cert for
`jiuin.cn` is not automatically valid for `bkgapi.jiuin.cn`.

Run the automated primary/failover/recovery smoke test only with the actual
BaoTa PHP-FPM stop/start commands supplied explicitly:

```bash
JIUIN_PUBLIC_BASE_URL=https://jiuin.cn \
JIUIN_PHP_STOP_COMMAND='actual-stop-command' \
JIUIN_PHP_START_COMMAND='actual-start-command' \
scripts/test-failover.sh
```

Also test one upload with a generated idempotency key, repeat it unchanged,
then inspect `music`, `upload_requests`, and `music_tasks` for exactly one row
each. Verify cover, full, lite, Range playback, WebSocket connect/Ping/Pong,
and a PHP-write/Go-read plus PHP-read/Go-write SQLite scenario. Browser DevTools
must show only `https://jiuin.cn/api/...`, `https://jiuin.cn/media/...`, and
`wss://jiuin.cn/ws/...`; `bkgapi.jiuin.cn`, loopback, and real addresses must
not appear in Network or API payloads.
