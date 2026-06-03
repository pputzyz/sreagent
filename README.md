# SREAgent

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?style=flat-square&logo=vue.js)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-hero1221%2Fsreagent-2496ED?style=flat-square&logo=docker)](https://hub.docker.com/r/hero1221/sreagent)
[![License](https://img.shields.io/badge/License-Proprietary-red?style=flat-square)](LICENSE)

AI 驱动的智能运维平台。从 Prometheus、VictoriaMetrics、Zabbix、Elasticsearch 接入数据，提供告警管理、值班排班、事件协作、数据探索和 AI 辅助诊断能力。

## 核心能力

- **告警引擎** — 多数据源规则评估、分组、去重、心跳检测、升级策略、可编程 Pipeline
- **通知管道** — 17 种通知渠道（飞书/钉钉/Slack/Webhook/邮件/短信等）、分派策略、模板系统、飞书卡片回调交互
- **值班排班** — 轮转排班、替班、升级策略、日历视图、订阅规则
- **事件协作** — 协作空间、事件时间线、指派、认领、解决、故障复盘、批量操作
- **数据查询** — Prometheus/VictoriaMetrics/VLogs/Zabbix 多数据源统一查询 + 记录规则
- **日志探索** — Elasticsearch 日志查询 + 索引模式管理
- **AI 助手** — 多 LLM Provider (OpenAI/Claude/DeepSeek/Ollama)、MCP Server、Skill 系统、SSE 流式输出、根因分析、SOP 推荐
- **内置规则** — 299 条预置告警规则 + 规则模板 + 一键导入
- **仪表盘** — 团队仪表盘 + 内置仪表盘 + 业务分组 + 指标视图 + 全屏模式
- **AIOps 诊断** — 诊断工作流编排、变更事件关联、知识库全文检索 (RAG)
- **飞书深度集成** — 卡片回调按钮（认领/静默/分配）、静默表单、分配表单、AI 对话、错误重试

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

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25, Gin, GORM, Wire (DI), Zap, golang-migrate |
| 前端 | Vue 3, TypeScript, Naive UI, Pinia, Vite, ECharts |
| 存储 | MySQL 8.0, Redis 7 |
| AI | OpenAI / Claude / DeepSeek / Ollama, MCP Protocol, Tool Registry, RAG |
| 容器 | Docker, Kubernetes |

## 项目结构

```
cmd/server/                    入口 + DI wiring
internal/
  model/       (62 个模型)     数据模型
  handler/     (77 个处理器)   HTTP 处理器
  service/     (98 个服务)     业务逻辑
  repository/  (59 个仓库)     数据访问层
  engine/      (22 个文件)     告警引擎：评估器 + 规则 + 抑制 + 心跳 + 升级 + Pipeline
  middleware/  (8 个中间件)     JWT / CORS / Logger / RBAC / RateLimit / WebhookAuth
  router/      (21 个文件)     350+ API 端点
  pkg/                         通用包：迁移 / 数据源代理 / 飞书 / Redis / 错误码 / MCP
web/src/                       Vue 3 前端
deploy/                        Dockerfile + K8s 清单
docs/                          文档
```

## 权限模型

三级角色体系:

| 角色 | 说明 |
|------|------|
| admin | 系统管理员，全部权限（用户管理/SSO/审计/系统设置） |
| team_lead | 团队负责人，可管理规则/排班/通知/数据源 |
| member | 普通成员，可查看/认领/解决事件，使用 AI 助手 |

## 环境变量

| 环境变量 | 说明 |
|---------|------|
| `SREAGENT_DATABASE_PASSWORD` | MySQL 密码 |
| `SREAGENT_REDIS_PASSWORD` | Redis 密码 |
| `SREAGENT_JWT_SECRET` | JWT 签名密钥 (建议 `openssl rand -hex 32`) |
| `SREAGENT_SECRET_KEY` | AES-256-GCM 主密钥 (64 位 hex，`openssl rand -hex 32`) |
| `SREAGENT_ADMIN_PASSWORD` | 初始管理员密码 |
| `SREAGENT_WEBHOOK_SECRET` | Webhook HMAC 签名密钥 (可选) |
| `SREAGENT_OIDC_CLIENT_SECRET` | OIDC 客户端密钥 (可选) |
| `SREAGENT_AGENT_MODEL` | AI Agent 执行模型: `tool-calling` (默认) 或 `plan-execute` |

AI、飞书 Bot、SMTP 等敏感配置通过 Web UI → 系统设置管理，AES-256-GCM 加密存储在数据库中。

## CI/CD

推送 `v*` 格式 tag 自动触发 GitHub Actions: Go 测试 → 前端类型检查 → 构建镜像推送到 Docker Hub (`hero1221/sreagent`)。

```bash
# 发版流程
# 1. 更新版本号 (CLAUDE.md, MODULES.md, web/package.json)
# 2. 提交并打 tag
git tag v4.61.0
git push origin v4.61.0
# 3. Actions 自动构建并推送 :v4.61.0 + :latest
```

## 文档

- [API 路由](docs/api.md)
- [架构设计](docs/architecture.md)
- [RBAC 权限体系](docs/rbac.md)
- [通知管道](docs/notification-pipeline.md)
- [AI Agent 路线图](docs/ai-agent-roadmap.md)
- [设计系统](docs/design-system.md)
- [CI/CD 部署](docs/ci-deploy.md)
- [平台操作手册](docs/PLATFORM_GUIDE.md)
- [测试指南](docs/TEST_GUIDE.md)
- [模块说明](MODULES.md)
- [代码约定](CLAUDE.md)
- [变更日志](CHANGELOG.md)

## License

内部项目，保留所有权利。
