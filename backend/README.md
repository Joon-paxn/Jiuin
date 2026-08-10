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

全部接口以 `{ "code", "message", "data" }` 的 JSON 信封格式返回。

## 媒体存储预留

`storage/music`、`storage/images` 与 `storage/models` 为未来部署卷中的媒体文件预留目录。当前 API 只返回空音乐列表，不提供上传或文件写入接口。
