# 霁雪居 / Jiuin

霁雪居是一个由 React + Vite 前端和 Go 后端组成的个人站点项目，包含 Live2D 看板娘、站点信息、音乐、统计、服务状态、链接和资源清单等能力。

## 技术栈

- 前端：React、TypeScript、Vite、PixiJS、pixi-live2d-display
- 后端：Go 标准库
- 部署：Nginx 静态托管前端，反向代理 Go API

## 目录结构

```text
.
├── backend/                 Go 后端服务
│   ├── cmd/server/          服务入口
│   ├── configs/             开发与生产环境变量示例
│   └── internal/            API、配置、模型、仓储与服务层
├── public/                  会原样复制至前端构建产物的静态资源
│   └── live2d/              Live2D 模型与 Cubism Core Runtime
├── scripts/                 Live2D 构建与 HTTP 部署校验脚本
├── src/                     React 前端源码
└── vite.config.ts           Vite 开发服务器配置
```

## 环境要求

- Node.js 18 或更高版本
- npm
- Go 1.24 或更高版本

## 本地开发

### 1. 安装前端依赖

```bash
npm install
```

### 2. 配置前端环境变量

复制 `.env.example` 为 `.env`，并按需要修改：

```bash
cp .env.example .env
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
```

本地默认后端地址是 `http://127.0.0.1:8080`。

### 3. 启动后端

在一个终端中执行：

```bash
cd backend
```

Linux/macOS：

```bash
set -a
source configs/development.env.example
set +a
go run ./cmd/server
```

Windows PowerShell：

```powershell
Get-Content configs/development.env.example | ForEach-Object {
  if ($_ -match '^(?<key>[^=]+)=(?<value>.*)$') {
    Set-Item -Path "Env:$($matches.key)" -Value $matches.value
  }
}
go run ./cmd/server
```

后端默认监听 `127.0.0.1:8080`，可访问以下地址确认：

```text
http://127.0.0.1:8080/api/v1/health
```

### 4. 启动前端

回到项目根目录，在另一个终端执行：

```bash
npm run dev
```

开发服务器默认地址为（两者任选其一）：

```text
http://localhost:5173
http://127.0.0.1:5173
```

## 前端构建

```bash
npm run build
```

构建产物输出至 `dist/`。构建命令会自动校验 Live2D Runtime、模型配置、MOC 与纹理等部署资源；任一资源缺失时，构建会失败。

本地单独校验已生成的构建产物：

```bash
npm run verify:live2d:dist
```

## Live2D

当前使用 Cubism 4 模型 Noir：

- 模型入口：`public/live2d/my_model/noir.model3.json`
- Cubism Core Runtime：`public/live2d/runtime/live2dcubismcore.min.js`

Vite 会将 `public/` 下的资源原样复制到 `dist/`。部署后，下列地址必须可以正常访问：

```text
/live2d/my_model/noir.model3.json
/live2d/runtime/live2dcubismcore.min.js
```

在线检查部署资源：

```bash
npm run verify:live2d:http -- https://jiuin.cn
```

若 Nginx 将 JavaScript 标记为 `text/javascript`，可执行：

```bash
npm run verify:live2d:http -- https://jiuin.cn --allow-text-javascript
```

## 后端 API

- `GET /api/v1/health`
- `GET /api/v1/site/info`
- `GET /api/v1/site/copyright`
- `GET /api/v1/site`
- `GET /api/v1/music`：公开歌曲列表，供播放器使用
- `GET /api/v1/music/{id}`：公开单曲 Metadata
- `GET /api/v1/music/tasks/{task_id}`：查询异步处理任务状态
- `POST /api/v1/admin/music/upload`：管理员上传音频，`multipart/form-data`，字段名为 `file`
- `POST /api/v1/music/upload`：受同一管理员令牌保护的兼容别名
- `GET /media/music/...`：公开的 full、lite 与封面资源
- `GET /api/v1/statistics`
- `POST /api/v1/statistics/visit`
- `GET /api/v1/status`
- `GET /api/v1/links`
- `GET /api/v1/resources`

所有接口使用 `{ "code", "message", "data" }` JSON 响应格式。上传接口立即返回任务 ID，转码在服务器 Worker 中异步完成；随后通过任务查询接口获取 `pending`、`processing`、`completed` 或 `failed` 状态。`POST /api/v1/admin/music/upload` 需要 `Authorization: Bearer <JIUIN_MUSIC_ADMIN_TOKEN>`，令牌只能由受控的管理端或服务器端持有，不能嵌入公共前端。统计写入仍使用独立的 `JIUIN_SHARED_SERVICE_TOKEN`。

## 音乐上传、存储与播放

管理员向上传接口提交 `mp3`、`flac`、`wav`、`ogg`、`m4a` 或 `aac` 音频。后端会校验文件名、扩展名、与扩展名匹配的声明 MIME 类型、文件头及 FFprobe 实际音频流，并使用服务器生成的 ID 保存文件；不得把用户文件名当作存储路径。

`JIUIN_MUSIC_DIRECTORY` 是私有存储根目录，生产环境建议位于独立持久化卷，例如 `/var/lib/jiuin/music`：

```text
/var/lib/jiuin/music/
├── music.db             SQLite 音乐记录与处理任务
├── original/            保留的原始上传音源（不公开）
├── full/                完整音质播放文件（不直接暴露磁盘路径）
├── lite/                省流播放文件（不直接暴露磁盘路径）
└── covers/              从内嵌标签提取的封面（可为空）
```

后端以 SHA-256 去重；相同原始音频会复用既有结果，不重复触发 FFmpeg。它会用 FFprobe 读取标题、艺术家、专辑、流派、年份、时长与封面信息，缺失的文本 Metadata 以“未知”回退；缺封面不会导致任务失败。FFmpeg 在可配置数量的 Worker 中生成 full/lite 两个版本，完成后写入 SQLite，播放器只通过 API 获得 HTTP 资源地址，绝不会得到服务器文件系统路径。

媒体资源经 Go/Nginx 的 `/media/music/...` 路由提供，支持 `GET`、`HEAD`、HTTP Range、暂停、续播与 seek。不要直接把 `original/`、SQLite 文件或整个存储目录发布为静态目录。

## 生产部署（宝塔 / Nginx）

### 前端

在服务器项目根目录执行：

```bash
npm install
npm run build
```

将 Nginx 网站根目录设置为：

```text
/www/wwwroot/Jiuin/dist
```

网站根目录不能指向项目根目录 `/www/wwwroot/Jiuin`，否则 `/live2d/runtime/live2dcubismcore.min.js` 等构建资源会返回 404。

### 后端

将 `backend/configs/production.env.example` 复制为服务器本地配置文件，并修改生产值，尤其是：

- `JIUIN_CORS_ALLOWED_ORIGINS`
- `JIUIN_SHARED_SERVICE_TOKEN`
- `JIUIN_MUSIC_ADMIN_TOKEN`
- `JIUIN_SITE_DOMAIN`
- `JIUIN_MUSIC_DIRECTORY`
- `JIUIN_FFMPEG_PATH` 与 `JIUIN_FFPROBE_PATH`

在 Linux 生产机安装 FFmpeg（其包通常同时提供 `ffmpeg` 与 `ffprobe`），确保运行 Go 服务的用户可执行这两个程序，并对 `JIUIN_MUSIC_DIRECTORY` 有读写权限。推荐将其使用的 SQLite 文件和音乐目录放在持久化卷，不要放进网站静态根目录。

Go 后端在 `backend/` 目录启动：

```bash
go run ./cmd/server
```

生产环境建议预先编译：

```bash
cd /www/wwwroot/Jiuin/backend
go build -o jiuin-server ./cmd/server
./jiuin-server
```

后端默认仅监听 `127.0.0.1:8080`。请在 Nginx 中将 `/api/` 与 `/media/` 都反向代理至 `http://127.0.0.1:8080`，不要直接对公网暴露 8080 端口。使用 [deploy/nginx/jiuin.conf.example](deploy/nginx/jiuin.conf.example) 中的精确上传 location：它对管理员正式路径及兼容别名（含 `/jiuin/` 部署路径）应用独立、严格的限流、`100m` body 限制和适合大文件上传的超时。

Nginx 的上传上限必须与 `JIUIN_MUSIC_MAX_UPLOAD_SIZE` 保持一致。当前环境示例使用 Go `100MiB`，配套 Nginx `100m`；若修改其中任一项，请同步修改另一项。Go 仍会强制执行限制，因此即使绕过反向代理也不能上传超限文件。

代理这两个路径时请保留站点主机名并传递协议头。音乐 API 只返回根相对 `/media/music/...` 地址，播放器不会信任任意外域媒体 URL：

```nginx
location /api/ {
  proxy_pass http://127.0.0.1:8080;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_set_header X-Real-IP $remote_addr;
}

location /media/ {
  proxy_pass http://127.0.0.1:8080;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_set_header X-Real-IP $remote_addr;
}
```

### 更新代码

```bash
cd /www/wwwroot/Jiuin
git fetch origin
git reset --hard origin/main
npm install
npm run build
```

如果服务器将项目交给 `www` 用户运行，注意保持目录读写权限与运行用户一致。

## 环境变量

前端环境变量示例见 `.env.example`：

- `VITE_API_BASE_URL`：仅用于开发环境覆盖后端 API 地址；留空时默认使用 `http://127.0.0.1:8080`。生产构建固定使用同源 `/api` 与 `/media` 反代，以匹配 CSP 和媒体 URL 白名单。
- `VITE_LIVE2D_CORE_URL`：可选的 Cubism 3/4/5 Runtime 覆盖地址；留空时使用项目内置 Runtime
- `VITE_LIVE2D_CUBISM2_CORE_URL`：可选的 Cubism 2 Runtime 覆盖地址

后端环境变量示例见 `backend/configs/production.env.example`。

音乐处理相关环境变量：

- `JIUIN_MUSIC_DIRECTORY`：私有音乐存储根目录；其中包含 SQLite 数据库、原始文件、转码文件和封面。
- `JIUIN_MUSIC_MAX_UPLOAD_SIZE`：单文件上传上限，例如 `100MiB`；须与 Nginx 上传 location 的 `client_max_body_size 100m` 同步。
- `JIUIN_FFMPEG_PATH`、`JIUIN_FFPROBE_PATH`：FFmpeg/FFprobe 可执行文件路径；Linux 可为 `/usr/bin/ffmpeg`、`/usr/bin/ffprobe`，Windows 可为绝对 `.exe` 路径。
- `JIUIN_MUSIC_FULL_BITRATE`、`JIUIN_MUSIC_LITE_BITRATE`：完整与省流 MP3 比特率，例如 `320k` 与 `128k`。
- `JIUIN_MUSIC_OUTPUT_CODEC`：输出编码器，例如 `libmp3lame`。
- `JIUIN_MUSIC_WORKER_COUNT`：并行 FFmpeg Worker 数量；生产环境通常从 `2` 开始，按 CPU/IO 容量调整。
- `JIUIN_MUSIC_ADMIN_TOKEN`：上传接口专用的至少 32 字符管理员 Bearer Token；生产环境必须由密钥管理系统提供，不能使用示例值。

## 常见问题

### Live2D Runtime 404

确认服务器执行过 `npm run build`，并检查：

```bash
ls /www/wwwroot/Jiuin/dist/live2d/runtime/live2dcubismcore.min.js
```

若文件存在但页面仍报 404，检查 Nginx 网站根目录是否为 `/www/wwwroot/Jiuin/dist`。

### Vite 提示域名不允许访问

开发服务器访问域名已在 `vite.config.ts` 的 `server.allowedHosts` 中配置 `jiuin.cn` 与 `www.jiuin.cn`。修改配置后需要重启 Vite。

### Git 拒绝拉取并提示本地改动

确认不需要保留服务器本地改动后，可强制同步远程主分支：

```bash
cd /www/wwwroot/Jiuin
git fetch origin
git reset --hard origin/main
```
