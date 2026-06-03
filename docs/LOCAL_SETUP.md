# SREAgent 本地开发环境搭建指南

## 方案选择

**推荐：Windows 本地 Docker（MySQL + Redis）+ 本地运行 Go + Vue**

原因：
- 你已有 VSCode + Go + Node 环境
- Docker Desktop for Windows 提供 MySQL + Redis
- 热重载开发体验好
- 自动化 QA 可直接在本地运行

## 第一步：安装 Docker Desktop

1. 下载 https://www.docker.com/products/docker-desktop/
2. 安装后确保 Docker Desktop 正在运行
3. 验证：
```powershell
docker --version
docker compose version
```

## 第二步：启动 MySQL + Redis

```powershell
cd c:\project\sreagent
docker compose -f deploy/docker/docker-compose.dev.yml up -d
```

如果没有 `docker-compose.dev.yml`，手动启动：

```powershell
# MySQL
docker run -d --name sreagent-mysql `
  -p 3306:3306 `
  -e MYSQL_ROOT_PASSWORD=root `
  -e MYSQL_DATABASE=sreagent `
  -e MYSQL_USER=sreagent `
  -e MYSQL_PASSWORD=sreagent `
  mysql:8.0

# Redis
docker run -d --name sreagent-redis `
  -p 6379:6379 `
  redis:7-alpine
```

## 第三步：配置后端

```powershell
cp configs/config.example.yaml configs/config.yaml
```

编辑 `configs/config.yaml`：
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"

database:
  host: "127.0.0.1"
  port: 3306
  username: "sreagent"
  password: "sreagent"
  database: "sreagent"
  charset: "utf8mb4"

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0

jwt:
  secret: "dev-secret-key-at-least-32-characters-long-for-local"
  expire_seconds: 86400
```

## 第四步：设置环境变量

```powershell
$env:SREAGENT_ADMIN_PASSWORD = "admin123"
$env:SREAGENT_DATABASE_PASSWORD = "sreagent"
$env:SREAGENT_REDIS_PASSWORD = ""
```

## 第五步：启动后端

```powershell
cd c:\project\sreagent
go run cmd/server/main.go -config configs/config.yaml
```

首次启动会自动运行迁移。看到 `server started` 即成功。

## 第六步：启动前端

```powershell
cd c:\project\sreagent\web
npm install
npm run dev
```

前端运行在 http://localhost:5173，API 代理到 http://localhost:8080。

## 第七步：验证

```powershell
# 后端健康检查
curl http://localhost:8080/healthz

# 前端打开
start http://localhost:5173
```

## 常用命令

```powershell
# 查看日志
docker logs sreagent-mysql
docker logs sreagent-redis

# 停止服务
docker stop sreagent-mysql sreagent-redis

# 重启
docker start sreagent-mysql sreagent-redis

# 清理重来
docker rm -f sreagent-mysql sreagent-redis
```
