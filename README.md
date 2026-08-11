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

开发服务器默认地址为：

```text
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
- `GET /api/v1/music/list`
- `GET /api/v1/statistics`
- `POST /api/v1/statistics/visit`
- `GET /api/v1/status`
- `GET /api/v1/links`
- `GET /api/v1/resources`

所有接口使用 `{ "code", "message", "data" }` JSON 响应格式。`POST /api/v1/statistics/visit` 需要服务器端携带 `Authorization: Bearer <JIUIN_SHARED_SERVICE_TOKEN>`，不得在浏览器前端暴露该令牌。

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
- `JIUIN_SITE_DOMAIN`

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

后端默认仅监听 `127.0.0.1:8080`。请在 Nginx 中将 `/api/` 反向代理至 `http://127.0.0.1:8080`，不要直接对公网暴露 8080 端口。

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

- `VITE_API_BASE_URL`：后端 API 基础地址
- `VITE_LIVE2D_CORE_URL`：可选的 Cubism 3/4/5 Runtime 覆盖地址；留空时使用项目内置 Runtime
- `VITE_LIVE2D_CUBISM2_CORE_URL`：可选的 Cubism 2 Runtime 覆盖地址

后端环境变量示例见 `backend/configs/production.env.example`。

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
