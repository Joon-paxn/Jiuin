# Jiuin 音乐上传与 FFmpeg API：AI 接力文档

## 交接目标

继续完成并上线 Jiuin（霁雪居）的管理员音乐上传与异步 FFmpeg 处理能力。当前工作区已经有较完整的未提交实现；本轮交接**不要求重写**，优先审查、补齐已知问题、真实环境验收并提交。

用户的原始需求包括：管理员上传音频，后端异步生成 full/lite 两个版本，抽取 Metadata/封面，持久化任务和歌曲信息，播放器从 API 读取曲库，媒体支持 HTTP Range，具备鉴权、限流、文件校验、去重与可配置 FFmpeg。

## 当前状态

实现已存在于工作树，尚未 Git 提交。以下文件有音乐功能相关改动或新增：

- `backend/internal/service/music_processing.go`：上传验证、FFprobe、FFmpeg、Worker Pool、进度、去重与日志。
- `backend/internal/repository/music_repository.go`：SQLite 曲库/任务存储、迁移、安全路径解析。
- `backend/internal/model/music.go`：任务、数据库记录、公开 API 模型。
- `backend/internal/api/handler/music_handler.go`：上传、任务、列表、详情、媒体 Range 服务。
- `backend/internal/api/router.go`：公开与管理员路由。
- `backend/internal/config/config.go`：音乐处理配置校验。
- `backend/cmd/server/main.go`：初始化 SQLite repository 与 Worker 生命周期。
- `backend/internal/*/*_test.go`：新增单元/路由测试。
- `src/services/api/media.ts`、`src/services/api/types.ts`、`src/components/music/MusicPlayer.tsx`：播放器改为请求 `/api/v1/music` 并切换音质。
- `deploy/nginx/jiuin.conf.example`、`backend/configs/*.env.example`、`README.md`、`backend/README.md`：部署说明与示例配置。

## 已实现的 API 合约

所有 API 仍使用 `{ "code", "message", "data" }` 信封。

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/admin/music/upload` | `Authorization: Bearer <JIUIN_MUSIC_ADMIN_TOKEN>` | 管理员上传，`multipart/form-data`，字段名 `file`；HTTP 202 返回任务。 |
| `POST` | `/api/v1/music/upload` | 同上 | 与需求兼容的受保护别名。 |
| `GET` | `/api/v1/music` | 公开 | 新播放器曲库列表。 |
| `GET` | `/api/v1/music/{id}` | 公开 | 单曲完整公开 Metadata。 |
| `GET` | `/api/v1/music/tasks/{task_id}` | 公开（当前设计） | 返回安全任务状态。 |
| `GET` / `HEAD` | `/media/music/full/{id}.mp3` | 公开 | 完整版。 |
| `GET` / `HEAD` | `/media/music/lite/{id}.mp3` | 公开 | 省流版。 |
| `GET` / `HEAD` | `/media/music/covers/{id}.jpg` | 公开 | 抽取出的封面。 |

上传成功示例：

```json
{
  "code": 202,
  "message": "accepted",
  "data": {
    "task_id": "uuid",
    "status": "pending"
  }
}
```

任务返回刻意不暴露内部路径、SHA-256、FFmpeg 命令/参数、exit code 和诊断：

```json
{
  "task_id": "uuid",
  "status": "processing",
  "progress": 60,
  "music_id": "uuid"
}
```

公开歌曲结构包含 `id`、`title`、`artist`、`album`、`albumArtist`、`genre`、`year`、`durationSeconds`、`cover`、`audio.full`、`audio.lite`、`fullSize`、`liteSize` 和 `createdAt`。所有音频/封面 URL 均应是根相对 `/media/music/...` 路径，不能返回磁盘路径。

## 当前架构与数据流

```text
管理员浏览器/管理工具
  -> POST /api/v1/admin/music/upload
  -> 流式 multipart handler（大小/字段/文件名保护）
  -> MusicProcessingService.Upload
       -> 暂存、魔数检查、SHA-256、FFprobe 验证
       -> original/{task-id}.{ext}
       -> SQLite music_tasks
  -> Worker Pool（由 JIUIN_MUSIC_WORKER_COUNT 控制）
       -> FFprobe Metadata
       -> FFmpeg full MP3
       -> FFmpeg lite MP3
       -> 可选 FFmpeg cover JPG
       -> SQLite music_records + task completed（事务）
  -> GET /api/v1/music
  -> MusicPlayer 按 full/lite 偏好播放 `/media/music/...`
```

### SQLite

数据库位于 `JIUIN_MUSIC_DIRECTORY/music.db`，由 repository 自动迁移：

- `music_records`：ID、source hash、Metadata、时长、封面/原始/输出相对路径、输出大小、创建时间。
- `music_tasks`：ID、pending/processing/completed/failed、进度、关联音乐、source hash、原始路径、仅服务端保存的失败类型/详情/exit code、时间。
- `source_hash` 具有唯一性，防止相同源文件重复处理。
- SQLite 使用 WAL；同一服务进程限制为单连接以降低锁冲突。

### 存储目录

```text
<JIUIN_MUSIC_DIRECTORY>/
├── music.db
├── original/     # UUID 原始音源，不公开
├── full/         # UUID.mp3
├── lite/         # UUID.mp3
├── covers/       # UUID.jpg（可选）
└── tmp/          # 上传临时文件
```

临时转码输出使用 `*.part.mp3`，完成后原子 rename；失败时清理已生成但未发布的资产。

## 配置

已在 `backend/configs/development.env.example` 与 `production.env.example` 中加入：

```dotenv
JIUIN_MUSIC_DIRECTORY=/var/lib/jiuin/music
JIUIN_MUSIC_MAX_UPLOAD_SIZE=100MiB
JIUIN_FFMPEG_PATH=/usr/bin/ffmpeg
JIUIN_FFPROBE_PATH=/usr/bin/ffprobe
JIUIN_MUSIC_FULL_BITRATE=320k
JIUIN_MUSIC_LITE_BITRATE=128k
JIUIN_MUSIC_OUTPUT_CODEC=libmp3lame
JIUIN_MUSIC_WORKER_COUNT=2
JIUIN_MUSIC_PROCESSING_TIMEOUT=2h
JIUIN_MUSIC_ADMIN_TOKEN=<至少 32 字符的随机密钥>
```

配置校验要点：

- 上传限制可配，Go 上限为 2 GiB。
- Worker 数限为 1–32。
- full bitrate 不得低于 lite bitrate。
- FFmpeg/FFprobe 可执行路径可为 Linux、Windows 或 PATH 中的命令名。
- 生产 token 不可用示例/占位值。

目前转码参数由服务端集中构造：

```text
ffmpeg -nostdin -hide_banner -loglevel error -y -i <input>
  -vn -map 0:a:0 -c:a <JIUIN_MUSIC_OUTPUT_CODEC>
  -b:a <FULL_OR_LITE_BITRATE> <output>.part.mp3
```

封面提取是可选操作，失败不会导致音频任务失败。

## 已实现安全措施

- 管理上传的 Bearer Token、POST 方法限制、路由限流（3/min）与全局限流。
- 上传使用流式 multipart，避免 `ParseMultipartForm` 将大文件放入内存。
- 限制 multipart body 与文件流大小；超过限制返回明确 413。
- 要求仅一个 `file` 字段。
- 校验上传文件名：拒绝空名、路径分隔符、NUL、超长名和路径穿越。
- 扩展名白名单：`mp3`、`flac`、`wav`、`ogg`、`m4a`、`aac`。
- 校验 MIME 与扩展名对应关系（对浏览器的 `application/octet-stream` 有受限兼容）。
- 魔数/内容初检，显式拒绝 `MZ`、ELF 等可执行文件头；最终由 FFprobe 确认真实音频流与合法 duration。
- 所有服务器文件名使用 UUID，不使用用户文件名。
- `safeJoin`、相对路径校验、Lstat 与 symlink 防护，防止路径逃逸与软链接访问。
- 媒体使用 `http.ServeContent`，支持 GET/HEAD、ETag、Range / 206 / 416 和浏览器 seek。
- 任务对外隐藏内部诊断；详细 FFmpeg 输出仅进服务端日志，输出长度有上限。
- Worker 处理有上下文超时和 Stop 取消。
- 去重基于 SHA-256，并处理并发上传竞争。
- 限流 map 已有 10,000 项上限；跨实例仍必须由 Nginx/Redis 等边缘层兜底。

## 已验证结果

本地已实际运行并通过：

```powershell
cd backend
go test -count=1 -timeout 120s ./...
go build ./cmd/server

cd ..
npm.cmd run build
git diff --check
```

最后一次全量 Go 测试输出全部通过，包含 `api`、`api/handler`、`middleware`、`config`、`repository` 与 `service`。前端 TypeScript/Vite 构建通过。

但本机**未安装** `ffmpeg`、`ffprobe` 和 `nginx`，所以尚未完成：

- 使用真实 MP3/FLAC 上传的端到端转码。
- 真实内嵌封面与 Metadata 抽取。
- Linux 上执行真实 `/usr/bin/ffmpeg`/`ffprobe`。
- 经 Nginx 的 Range/206、上传大小限制和 `nginx -t` 验证。

## 已知待办（下一位 AI 应按优先级执行）

### P0：先保存和清理工作区

1. 查看 `git status --short`。当前有大量音乐相关已修改/新增文件，均未提交。
2. 清理不应提交的临时 Go 缓存目录：`backend/.go-build-cache-*`。这些目录目前是测试遗留物，不属于产品。
3. 复查 `.gitignore` 是否已覆盖此类缓存，确认 `backend/go.sum`、新的 `music_processing.go` 和测试文件会被纳入提交。
4. 在不覆盖用户改动的前提下，提交前再次执行 `git diff --check` 与全量测试。

### P1：修复前端 URL/CSP 跨域问题

当前 `src/services/api/media.ts` 的 `resolveMediaUrl` 对 API 返回的绝对 `http(s)` URL 过于宽松，且未要求媒体 URL 必须落在 `/media/music/` 下。若 API 响应被篡改，浏览器可能加载任意外域资源。

建议：

1. 只接受后端实际返回的根相对 `/media/music/` 路径，或在配置了 `VITE_API_BASE_URL` 时仅接受该 API origin 下的 `/media/music/` 路径。
2. HTTPS 页面拒绝 `http:` 媒体。
3. 拒绝 protocol-relative URL、用户名/密码、非预期 origin、非 `/media/music/` path。
4. 为该解析规则增加 TypeScript 单元测试（若项目不引入测试框架，至少写纯函数测试计划或采用现有项目约定）。
5. `deploy/nginx/jiuin.conf.example` 的 CSP 目前仅默认允许同源媒体。若明确支持跨域 `VITE_API_BASE_URL`，需要让 `connect-src`、`media-src` 和 `img-src` 包含受信 API/media 域；不要使用无约束 `*`。如果生产只支持同源反代，应在 README 明确写出该限制并把跨域能力移除或完整设计配置白名单。

前端类型目前只消费部分公开 Metadata；如果播放器或管理界面需要显示更多资料，补上：`albumArtist`、`genre`、`year`、`fullSize`、`liteSize`、`createdAt`，并按 UI 需要映射到内部 `MusicTrack`。

### P1：Nginx 配置一致性与注释

1. `client_max_body_size 100m` 在 Nginx 示例里仍是具体值，而 Go 使用 `JIUIN_MUSIC_MAX_UPLOAD_SIZE=100MiB`；已在文档提醒同步，但仍容易漂移。至少保留醒目的同步说明；若有部署模板机制，应从同一变量生成两者。
2. Nginx 注释仍提到 Go 使用 `Host`/`X-Forwarded-Proto` 构造音乐 URL；当前新 API 已固定使用根相对 URL，应更新该注释以免误导。
3. 生产应有独立 default server 拒绝未知 Host，且 Go 仅监听 loopback。不要将 8080 暴露公网。
4. 执行 `nginx -t`，真实代理后验证 `Range: bytes=...` 返回 206、`Accept-Ranges`、GET/HEAD 行为、以及上传路径的 413/429。

### P1：真实 FFmpeg 端到端验收

Linux 上安装 FFmpeg（确保 MP3 encoder 可用）：

```bash
ffmpeg -hide_banner -encoders | grep libmp3lame
ffmpeg -version
ffprobe -version
```

然后：

```bash
curl -i \
  -H "Authorization: Bearer $JIUIN_MUSIC_ADMIN_TOKEN" \
  -F 'file=@fixture-with-tags-and-cover.flac;type=audio/flac' \
  https://<host>/api/v1/admin/music/upload

curl -i https://<host>/api/v1/music/tasks/<task-id>
curl -i https://<host>/api/v1/music
curl -i -H 'Range: bytes=0-1023' https://<host>/media/music/full/<id>.mp3
```

验证：

- 原文件、full、lite、可选 cover 在正确目录中；无 `.part.mp3` 残留。
- Metadata 解析正确；缺少 tag 时为“未知”。
- full/lite 文件均可被 `ffprobe` 识别，码率接近设定值。
- 有封面的源可获得 JPG；无封面源仍成功。
- 重传完全相同源文件时返回复用结果，且不会新增转码调用或重复 DB 记录。
- 上传 EXE、伪造 MIME/扩展、超限 body、`../../x` 文件名均被拒绝。
- 让 FFmpeg 故意失败，任务应进入 `failed`，公开接口不泄露内部命令、路径或堆栈；日志应含 task_id、分类、exit code。

### P2：权限模型设计决策

当前上传鉴权是独立静态 Bearer capability：拥有 `JIUIN_MUSIC_ADMIN_TOKEN` 就可上传。它满足当前 API 层保护要求，但不等价于未来浏览器管理页的用户/角色体系。

- 绝不把 `JIUIN_MUSIC_ADMIN_TOKEN` 编译进 React/Vite 前端。
- 若开发 Music Admin 页面，先接入项目的 session/OIDC/JWT，再检查管理员角色/权限；可保留此 token 用于受信 server-to-server 操作。
- 当前任务状态是公开的，仅返回 UUID、状态、进度与完成 ID。如果任务隐私成为要求，应增加管理员鉴权或任务所有者模型。

### P2：可能需要的产品增强（当前非阻塞）

- 管理端列出任务、删除歌曲和补封面的 API/UI。
- 历史 `backend/storage/music` 根目录旧文件不会自动导入 SQLite 新曲库。若必须保留旧音乐，新增一次性迁移脚本或重新上传。
- 更精细的 FFmpeg 进度解析（当前使用阶段进度 20/60/85/100）。
- 跨进程/多实例任务队列与共享限流（例如 Redis）以及任务重试策略。
- 对超大/恶意 media 的更强资源配额（CPU、磁盘、文件时长/采样率）。

## 审查结论与注意事项

已做过两轮代码安全审查，主要修复已进入工作树：

- 旧音乐 URL 改为根相对，避免用请求 `Host` 生成播放地址。
- 全局 RateLimit 放在 CORS 之前，拒绝的 Origin 也会计数。
- 限流 map 容量受限为 10,000，避免无限 IP key 内存增长。
- Managed DB 路径须先通过规范相对路径验证，避免 `full/../music.db` 一类绕过。
- 临时上传目录也经 `safeJoin` 解析。
- multipart handler 近期修复了 `max+1` 文件读取哨兵边界；不要回退该逻辑。

仍需在生产部署层承担的边界：

- 单进程限流不是跨实例安全机制；保留 Nginx 限流，横向扩容时考虑 Redis。
- loopback 代理模型信任 `X-Real-IP`；确保只有受信 Nginx/本机进程可访问 Go 监听端口，最好以 socket/防火墙隔离。
- Nginx 的请求 Host 应有默认站点拒绝规则。

## 建议的接手顺序

1. 阅读本文件，运行 `git status --short`，确认没有其他人正在编辑。
2. 只清理 `backend/.go-build-cache-*`，不要 reset、checkout 或覆盖其它未提交改动。
3. 复跑：`cd backend && go test -count=1 ./... && go build ./cmd/server`；然后根目录 `npm.cmd run build`。
4. 处理 P1 前端 URL/CSP 与 Nginx 注释/配置一致性，新增对应测试。
5. 安装/使用真实 FFmpeg 环境完成 P1 端到端验证。
6. 更新 README 的真实验收结果；准备清晰、单一目的的 Git commit。
7. 向用户按九项交付：新增 API、数据库变化、FFmpeg 配置、Worker、存储、流程、音质参数、安全措施、测试结果，并明确真实 FFmpeg/Nginx 是否已验证。

## 不要做的事情

- 不要把管理员 Token 加到 `VITE_*` 环境变量、前端 bundle、README 示例中的真实值，或上传到 Git。
- 不要让 API 返回 `/home/...`、`/var/...`、`storage/...` 等内部路径。
- 不要用用户上传文件名作为磁盘路径或最终服务器文件名。
- 不要把 FFmpeg 执行搬到前端。
- 不要以 `git reset --hard`、`git checkout --` 清理当前未提交工作区。
- 不要把本地 `.go-build-cache-*`、`server.exe`、SQLite 测试产物提交。

