# SREAgent

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?style=flat-square&logo=vue.js)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-hero1221%2Fsreagent-2496ED?style=flat-square&logo=docker)](https://hub.docker.com/r/hero1221/sreagent)
[![License](https://img.shields.io/badge/License-Proprietary-red?style=flat-square)](LICENSE)

自托管的告警管理与 OnCall 值班平台。从 Prometheus、VictoriaMetrics、Zabbix 接入告警，通过去重、降噪、路由管道处理后，分派到飞书、邮件或 Webhook。

## 核心能力

- **四层事件模型**: Integration → Event → Alert → Incident，告警全生命周期管理
- **智能降噪**: 规则聚合、风暴预警、抖动检测、快速静默
- **OnCall 值班**: 排班日历、班次覆盖、超时自动升级
- **集成中心**: 专属/共享集成，Pipeline 处理，路由规则
- **飞书深度集成**: 交互卡片、Bot DM 指令、卡片实时更新
- **AI 辅助**: 多 Provider 支持 (OpenAI/Azure/Ollama/Anthropic/Custom)，AI Agent 自主执行 + 会话持久化，知识库全文检索 (RAG)，诊断工作流编排，告警分析报告、故障复盘初稿、SOP 推荐
- **AIOps 诊断**: 诊断工作流编排、变更事件关联、AI Agent 自主执行
- **故障复盘**: Markdown 编辑器 + AI 辅助生成 + 一键发布

## 快速开始

### Docker 部署

前置条件: MySQL 8.0+ 和 Redis 7+。

```bash
git clone https://github.com/pputzyz/sreagent && cd sreagent
cp configs/config.example.yaml configs/config.yaml
# 编辑 config.yaml 填写数据库和 Redis 密码

docker build -f deploy/docker/Dockerfile -t sreagent:latest .
docker run -d --name sreagent \
  -p 8080:8080 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  sreagent:latest
```

打开 `http://your-server:8080`，用 `admin` / `admin123` 登录（首次登录请修改密码）。

### 本地开发

依赖: Go 1.25+, Node.js 20+, MySQL 8+, Redis 7+。

```bash
cp configs/config.example.yaml configs/config.yaml
make docker-up     # 启动 MySQL + Redis
make run           # 启动后端 (localhost:8080)
make web-dev       # 启动前端 (localhost:3000)
```

### Kubernetes

部署清单位于 `deploy/kubernetes/`，按顺序 apply: namespace → mysql → redis → secret → app。

## 配置

启动必需的四个环境变量:

| 环境变量 | 说明 |
|---------|------|
| `SREAGENT_DATABASE_PASSWORD` | MySQL 密码 |
| `SREAGENT_REDIS_PASSWORD` | Redis 密码 |
| `SREAGENT_JWT_SECRET` | JWT 签名密钥 (建议 `openssl rand -hex 32`) |
| `SREAGENT_SECRET_KEY` | AES-256-GCM 主密钥 (64 位 hex，`openssl rand -hex 32`) |

AI、飞书 Bot、SMTP 等敏感配置通过 Web UI → 系统设置管理，AES-256-GCM 加密存储在数据库中。

完整配置项见 `configs/config.example.yaml`。

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25, Gin, GORM, Zap, golang-migrate |
| 前端 | Vue 3, TypeScript, Naive UI, Pinia, Vite |
| 存储 | MySQL 8.0, Redis 7 |
| AI | OpenAI / Anthropic Claude / Ollama, Tool Registry, RAG Knowledge Base |
| 容器 | Docker, Kubernetes |

## CI/CD

推送 `v*` 格式 tag 自动触发 GitHub Actions: Go 测试 → 前端类型检查 → 构建镜像推送到 Docker Hub (`hero1221/sreagent`)。

```bash
# 发版流程
# 1. 更新版本号 (CLAUDE.md, MODULES.md, web/package.json)
# 2. 提交并打 tag
git tag v4.15.5
git push origin v4.15.5
# 3. Actions 自动构建并推送 :v4.15.5 + :latest
```

## 项目结构

```
cmd/server/              入口 + DI wiring
internal/
  model/ handler/ service/ repository/   分层架构
  engine/                告警评估引擎
  middleware/            JWT / CORS / Logger
  router/                270+ API 端点
  pkg/                   dbmigrate / datasource / lark / redis / errors
web/src/                 Vue 3 前端
deploy/                  Dockerfile + K8s 清单
```

## 文档

- API 路由: `docs/api.md`
- 模块说明: `docs/` 目录下各模块文档
- CLAUDE.md: 代码约定、变更追踪规则、自动路由规则

## License

内部项目，保留所有权利。
