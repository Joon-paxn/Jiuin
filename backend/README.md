# Jiuin Backend

Go 标准库实现的 Jiuin 共享服务基础层。当前不连接数据库，站点信息由环境变量提供；`internal/repository.SiteRepository` 已为 MySQL、PostgreSQL 和 SQLite 实现预留边界。

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
- `GET /api/v1/music/list`
- `GET /media/music/{id}`
- `GET /api/v1/site`
- `GET /api/v1/statistics`
- `POST /api/v1/statistics/visit`（服务令牌保护）
- `GET /api/v1/status`
- `GET /api/v1/links`
- `GET /api/v1/resources`

全部接口以 `{ "code", "message", "data" }` 的 JSON 信封格式返回。

## 本地音乐目录

`JIUIN_MUSIC_DIRECTORY` 指向用户管理的音乐目录；开发环境默认使用 `storage/music`。服务会在读取歌单时扫描该目录，并将受支持的音频以不透明 ID 暴露在 `/media/music/{id}`。

- 支持：`mp3`、`m4a`、`aac`、`ogg`、`wav`、`flac`
- 文件名推荐：`艺术家 - 歌名.mp3`
- 音频流支持 HTTP Range，供浏览器拖动播放进度。
- 服务不提供公共上传或文件写入接口；部署时请通过受控文件同步将歌曲放入该目录。

## 生态服务边界

统计、状态、链接、资源清单和站点共享配置分别拥有独立的 model、repository、service 与 handler 边界；当前使用内存或环境配置实现，未来可迁移到 Core Database。

- Core Database：站点配置、统计、状态、链接与资源清单。
- Main Database：仅保留主站未来专属数据。
- Blog Database：仅保留 Blog 文章及其专属数据。

`POST /api/v1/statistics/visit` 只接受服务端持有的 `Authorization: Bearer <JIUIN_SHARED_SERVICE_TOKEN>`，浏览器代码不得包含该令牌。外链配置仅允许 HTTPS 地址；资源清单仅允许同源相对路径。

生产部署仅将 Go 服务监听在回环地址，并通过 Nginx 代理 `/api/` 与 `/media/`。代理必须覆盖客户端可伪造的 `X-Real-IP` 请求头：

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
