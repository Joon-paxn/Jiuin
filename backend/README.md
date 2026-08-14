# Jiuin Backend

Jiuin 的 Go 后端提供公开站点 API、音乐播放、管理员音乐上传与异步 FFmpeg 处理。音乐 Metadata 与处理任务保存在 `JIUIN_MUSIC_DIRECTORY` 下的 SQLite 数据库中；SQLite 文件只供后端访问，不能作为静态资源发布。

## 本地启动

在 PowerShell 中加载 `configs/development.env.example` 的变量后运行：

```powershell
Get-Content configs/development.env.example | ForEach-Object {
  if ($_ -match '^(?<key>[^=]+)=(?<value>.*)$') {
    Set-Item -Path "Env:$($matches.key)" -Value $matches.value
  }
}
go run ./cmd/server
```

## API

- `GET /api/v1/health`
- `GET /api/v1/site/info`
- `GET /api/v1/site/copyright`
- `GET /api/v1/music`：公开歌曲列表
- `GET /api/v1/music/{id}`：公开单曲信息
- `GET /api/v1/music/tasks/{task_id}`：音乐处理任务状态
- `POST /api/v1/admin/music/upload`：管理员上传音频
- `POST /api/v1/music/upload`：管理员上传兼容别名（同一权限与限流）
- `GET|HEAD /media/music/...`：公开音频与封面资源
- `GET /api/v1/site`
- `GET /api/v1/statistics`
- `POST /api/v1/statistics/visit`（服务令牌保护）
- `GET /api/v1/status`
- `GET /api/v1/links`
- `GET /api/v1/resources`

全部接口以 `{ "code", "message", "data" }` 的 JSON 信封格式返回。上传请求采用 `multipart/form-data`，音频字段名为 `file`；成功接收后立即返回任务 ID，FFmpeg Worker 在后台处理。任务状态为 `pending`、`processing`、`completed` 或 `failed`。上传接口需要 `Authorization: Bearer <JIUIN_MUSIC_ADMIN_TOKEN>`，它是专门的管理员凭据，不能使用或暴露给公共浏览器代码。

## 音乐上传与存储

`JIUIN_MUSIC_DIRECTORY` 是私有音乐存储根目录；开发环境可使用 `storage/music`，生产环境应使用持久化卷，例如 `/var/lib/jiuin/music`。后端自动创建并管理以下内容：

```text
storage/music/
├── music.db             SQLite 音乐记录与任务状态
├── original/            原始上传文件，仅后端可读
├── full/                完整音质生成文件
├── lite/                省流生成文件
└── covers/              从音频内嵌封面提取出的图片
```

- 输入格式：`mp3`、`m4a`、`aac`、`ogg`、`wav`、`flac`。服务同时校验安全文件名、扩展名、与扩展名一致的请求 MIME、内容嗅探和 FFprobe 识别结果；不会仅根据扩展名接受文件。兼容保留在音乐根目录的 MP3 会在首次读取时自动提取内嵌封面，并缓存为供播放器显示的 JPEG。
- 原始文件采用服务器生成的 ID 命名，永不直接使用上传文件名或路径；服务以 SHA-256 去重，重复原始音频会复用已有资源而不重复转码。
- FFprobe 读取标题、艺术家、专辑、专辑艺术家、流派、年份、时长和内嵌封面；缺失文本信息使用“未知”，没有封面仍可成功处理。
- full/lite 输出均由服务器端 FFmpeg Worker 生成，Worker 数由配置控制，HTTP 上传请求不会等待转码完成。
- 播放器收到的是 `/media/music/...` HTTP 路径，而非内部 `original/`、`full/`、`lite/` 或 SQLite 的真实磁盘路径。资源处理器支持 `GET`、`HEAD` 与 HTTP Range，因此 HTML5 Audio 可以 seek、暂停、继续和部分下载。

## FFmpeg 与配置

Linux 上安装 FFmpeg 后，通常可使用 `/usr/bin/ffmpeg` 和 `/usr/bin/ffprobe`；Windows 则配置相应的绝对 `.exe` 路径。运行服务的账户必须能执行这两个程序，并对 `JIUIN_MUSIC_DIRECTORY` 有读写权限。FFmpeg 失败时，公开 API 只返回安全的任务失败信息；任务 ID、错误分类与 exit code 会记录到服务器日志，不会向普通访问者暴露命令行或内部路径。

音乐相关必填环境变量：

```text
JIUIN_MUSIC_DIRECTORY=storage/music
JIUIN_MUSIC_MAX_UPLOAD_SIZE=100MiB
JIUIN_FFMPEG_PATH=ffmpeg
JIUIN_FFPROBE_PATH=ffprobe
JIUIN_MUSIC_FULL_BITRATE=320k
JIUIN_MUSIC_LITE_BITRATE=128k
JIUIN_MUSIC_OUTPUT_CODEC=libmp3lame
JIUIN_MUSIC_WORKER_COUNT=2
JIUIN_MUSIC_ADMIN_TOKEN=development-only-change-me-to-a-different-32-character-minimum-token
```

`JIUIN_MUSIC_MAX_UPLOAD_SIZE` 必须与反向代理的 body limit 同步。仓库中 Nginx 示例使用 `client_max_body_size 100m`，与上述 `100MiB` 配套；若改变上传上限，请同时改变 Nginx 精确上传路由的值。后端也会强制该限制，因此 Nginx 不是唯一防线。

## 生态服务边界

统计、状态、链接、资源清单和站点共享配置分别拥有独立的 model、repository、service 与 handler 边界；当前使用内存或环境配置实现，未来可迁移到 Core Database。

- Core Database：站点配置、统计、状态、链接与资源清单。
- Main Database：仅保留主站未来专属数据。
- Blog Database：仅保留 Blog 文章及其专属数据。

`POST /api/v1/statistics/visit` 只接受服务端持有的 `Authorization: Bearer <JIUIN_SHARED_SERVICE_TOKEN>`，浏览器代码不得包含该令牌。音乐上传使用独立的 `JIUIN_MUSIC_ADMIN_TOKEN`，不可用共享服务令牌替代。外链配置仅允许 HTTPS 地址；资源清单仅允许同源相对路径。

生产部署仅将 Go 服务监听在回环地址，并通过 Nginx 代理 `/api/` 与 `/media/`。使用仓库的 Nginx 示例时，管理员正式上传路径、兼容别名及其 `/jiuin/` 版本均有精确 location，分别采用独立严格限流、`100m` 请求体限制和较长上传超时；不要只依赖通用 `/api/` location。代理必须覆盖客户端可伪造的 `X-Real-IP` 请求头：

```nginx
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header Host $host;
proxy_set_header X-Forwarded-Proto $scheme;
```

后端仅在直接对端为 loopback 时使用一个合法的 `X-Real-IP` 作为限流键；非回环连接忽略该头。音频流会显式取消其响应写入 deadline，以保留长时播放和 Range 支持；其他 API 仍使用正常的写入超时。

环境变量示例：

```text
JIUIN_EXTERNAL_LINKS_JSON=[{"name":"GitHub","url":"https://github.com/Joon-paxn/Jiuin","description":"Jiuin source repository"}]
JIUIN_RESOURCE_MANIFEST_JSON=[{"name":"shared-site-config","url":"/api/v1/site","priority":1,"cachePolicy":"config"}]
```
