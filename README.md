# 霁雪居 / Jiuin

Jiuin 是一个 React + TypeScript + Vite 主站，采用 PHP 主 API、Go 完整灾备
API 和 Go WebSocket 的裸机部署架构。生产环境使用 BaoTa、Nginx、PHP-FPM、
systemd、SQLite 与 FFmpeg，不使用 Docker。

```text
Browser
  -> https://jiuin.cn/api, /media, /ws
  -> public Nginx
  -> local TLS bkgapi.jiuin.cn gateway
       -> PHP-FPM primary API
       -> Go backup API + WebSocket
       -> shared SQLite and music storage
```

`bkgapi.jiuin.cn` 是服务器内部的统一网关入口，模板中只绑定
`127.0.0.1`。浏览器与前端构建产物仅使用 `jiuin.cn` 的同源相对路径，不会
包含后端 IP、端口或 `bkgapi.jiuin.cn`。

## 目录

```text
src/                    React frontend
backend/php/            PHP-FPM primary API and PHP CLI music worker
backend/cmd/jiuin-go/   Go backup API, backup worker, and WebSocket
backend/internal/core/  shared Go implementation and single SQLite schema
deploy/nginx/           BaoTa-friendly Nginx gateway templates
deploy/systemd/         Go and PHP worker service templates
scripts/                Go/PHP update, build, and failover smoke scripts
```

Go 1.22 or newer, PHP 8.2+ with `pdo_sqlite`, and FFmpeg/FFprobe are required
for the corresponding backend roles.

## Development

```bash
npm install
npm run build

cd backend
go test ./...
go vet ./...
go build -o bin/jiuin-go ./cmd/jiuin-go
```

The browser calls `/api`, `/media`, and `/ws` relative to its own origin.
`vite.config.ts` has a development-only server-side proxy to the local Go
service; no backend address is bundled into JavaScript.

Copy `.env.example` to the server-only `/etc/jiuin/jiuin.env` and replace all
tokens. PHP-FPM, the PHP worker, and Go must all receive the same
`JIUIN_STORAGE_DIR` and `JIUIN_DATABASE_PATH` values.

## Deployment

Do not replace BaoTa's generated site configuration. Use the minimal gateway
templates in [deploy/nginx](deploy/nginx/README.md), substituting the actual
PHP-FPM socket, Go port, local gateway TLS port, certificate paths, and PHP
document root discovered on the server. The detailed architecture, systemd,
build/update commands, DNS/certificate preflight, and failure test procedure
are in [deploy/OPERATIONS.md](deploy/OPERATIONS.md).

The common contract for PHP and Go is in
[backend/API_CONTRACT.md](backend/API_CONTRACT.md). Uploads require an
administrator bearer token and `Idempotency-Key`; the two runtimes share a
single WAL SQLite database, storage tree, request ledger, and task lease.

## Verification

```bash
npm run build
scripts/build-go.sh
```

On the server, run `nginx -t`, the Go `/health` and `/ready` checks, PHP syntax
checks through `scripts/update-php.sh`, and `scripts/test-failover.sh` with
the actual BaoTa stop/start commands. Browser Network must contain only
`https://jiuin.cn/api/...`, `https://jiuin.cn/media/...`, and
`wss://jiuin.cn/ws/...`.
