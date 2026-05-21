# 模块清单 (MODULES)

> 最后更新: 2026-05-21 | tag: v4.15.8
> 共 42 个 model, 57 个 handler, 78 个 service, 42 个 repository, 280+ API 端点

---

## 模块依赖关系

```
webhook ──────────→ alert-engine ←──── alert-rule (读取规则，含 channel_id)
                       │    ↑
                       │    └── datasource (查询数据)
                       │    └── label-registry (标签匹配)
                       │
                       ├──→ alert-v2-pipeline ←── noise-reducer (降噪)
                       │        ├──→ alert (v2 Alert + AlertEventV2)
                       │        ├──→ incident (故障生命周期)
                       │        └──→ dispatch (标签增强)
                       │
                       ├──→ notification ←── notify-rule, notify-media, message-template, subscribe-rule
                       │        └──→ lark, alert-channel (分发渠道)
                       │
                       └──→ escalation ──→ schedule (查找值班人)
                                └──→ user, team (查找通知目标)

integration (webhook接入) ──→ alert-v2-pipeline
  └── routing-rule (共享集成路由)

channel (协作空间) ──┬── incident (故障)
                     ├── exclusion-rule (排除规则)
                     ├── dispatch-policy (分派策略)
                     └── noise-reducer (降噪配置)

incident ──→ post-mortem (复盘) ──→ ai (AI 生成初稿)
schedule ──→ user (成员)
auth ──→ user (用户信息)
ai ──→ alert-engine (读取告警上下文)
ai-agent ──→ ai-service + knowledge-base + tool-registry
diagnostic-workflow ──→ incident-context + change-event + ai-agent
dashboard ──→ alert-event + incident + channel + team (统计数据)
user-notification ──→ user (按用户推送)
permissions ──→ team (团队角色查询)
```

改模块前查上方依赖：改 notification 会影响 alert-engine 和 escalation；改 schedule 会影响 escalation。

## 测试覆盖状态

| 模块 | 功能状态 | 单元测试 | 集成测试 | 覆盖率 |
|------|----------|----------|----------|--------|
| 告警引擎 | ✅ | ✅ evaluator_test.go (19) + evaluator_concurrent_test.go (4) + rule_eval_test.go + suppression_test.go (26) | ❌ | service 层 ~40% |
| 告警规则 | ✅ | ❌ | ❌ | 0% |
| 告警事件 | ✅ | ❌ | ❌ | 0% |
| 告警通道 | ✅ | ✅ alert_channel_test.go (handler + service) | ❌ | ~30% |
| 通知管道 | ✅ | ✅ notification_test.go (7 tests) | ❌ | service 层 ~25% |
| 静默规则 | ✅ | ❌ | ❌ | 0% |
| 抑制规则 | ✅ | ✅ inhibition_rule_test.go | ❌ | ~15% |
| 标签注册表 | ✅ | ❌ | ❌ | 0% |
| 数据源 | ✅ | ❌ | ❌ | 0% |
| 值班排班 | ✅ | ✅ schedule_test.go (32 tests) | ❌ | service 层 ~35% |
| 升级策略 | ✅ | ❌ | ❌ | 0% |
| 认证 | ✅ | ✅ auth_test.go (middleware) | ❌ | ~10% |
| 用户管理 | ✅ | ❌ | ❌ | 0% |
| 团队 | ✅ | ❌ | ❌ | 0% |
| 业务分组 | ✅ | ❌ | ❌ | 0% |
| 仪表盘 | ✅ | ❌ | ❌ | 0% |
| AI 助手 | ✅ | ❌ | ❌ | 0% |
| 飞书集成 | ✅ | ❌ | ❌ | 0% |
| 系统设置 | ✅ | ✅ encryption_test.go | ❌ | ~10% |
| 审计日志 | ✅ | ❌ | ❌ | 0% |
| Webhook 入站 | ✅ | ❌ | ❌ | 0% |
| 协作空间 | ✅ | ❌ | ❌ | 0% |
| 故障管理 | ✅ | ❌ | ❌ | 0% |
| 告警 v2 | ✅ | ❌ | ❌ | 0% |
| 集成中心 | ✅ | ❌ | ❌ | 0% |
| 路由规则 | ✅ | ❌ | ❌ | 0% |
| 分派策略 | ✅ | ❌ | ❌ | 0% |
| 排除规则 | ✅ | ❌ | ❌ | 0% |
| 故障复盘 | ✅ | ❌ | ❌ | 0% |
| 预设规则 | ✅ | ❌ | ❌ | 0% |
| 告警规则模板 | ✅ | ❌ | ❌ | 0% |
| AI 规则生成 | ✅ | ❌ | ❌ | 0% |
| 状态页面 | ✅ | ❌ | ❌ | 0% |
| Alertmanager 导入 | ✅ | ❌ | ❌ | 0% |
| 知识库 | ✅ | ❌ | ❌ | 0% |
| 诊断工作流 | ✅ | ❌ | ❌ | 0% |
| 变更事件 | ✅ | ❌ | ❌ | 0% |
| Incident 上下文 | ✅ | ❌ | ❌ | 0% |

> 目标：service 层 > 60%，handler 层 > 40%（v1.11.0 起逐步补全）

---

## 告警引擎 (alert-engine)

- **功能**: 规则评估、状态机、指纹去重、心跳检测、升级策略、分组通知
- **后端文件**: `internal/engine/` (6 files), `internal/service/alert_group.go`
- **API**: `GET /engine/status`
- **状态**: ✅ 核心完成（含 heartbeat、inhibition、group_wait/interval）
- **文档**: [docs/architecture.md](docs/architecture.md)（引擎状态机 + 通知管道）

## 告警规则 (alert-rule)

- **功能**: 规则 CRUD、分类、导入导出 (Prometheus format)、版本历史
- **后端**: `model/alert_rule.go`, `handler/alert_rule.go`, `service/alert_rule.go`, `repository/alert_rule.go`
- **前端**: `web/src/pages/alerts/rules/Index.vue`
- **API**: `/api/v1/alert-rules` (9 endpoints)
- **状态**: ✅ 完成

## 告警事件 (alert-event)

- **功能**: 事件生命周期 (firing→ack→assign→resolve→close)、时间线、批量操作、CSV 导出
- **后端**: `model/alert_event.go`, `handler/alert_event.go`, `service/alert_event.go`, `repository/alert_event.go`
- **前端**: `web/src/pages/alerts/events/Index.vue`, `Detail.vue`
- **API**: `/api/v1/alert-events` (13 endpoints)
- **状态**: ✅ 完成

## 告警通道 (alert-channel)

- **功能**: 虚拟接收器，按 match_labels 分发到不同通知管道
- **后端**: `model/alert_channel.go`, `handler/alert_channel.go`, `service/alert_channel.go`, `repository/alert_channel.go`
- **前端**: `web/src/pages/notification/AlertChannels.vue`
- **API**: `/api/v1/alert-channels` (5 endpoints)
- **状态**: ✅ 完成

## 通知管道 (notification)

- **功能**: v2 规则管道（标签匹配 + 严重级别 + 节流 + 去重 + 模板渲染 + 多渠道发送）、订阅机制
- **后端**: `service/notification.go`, `service/notification_dedup.go`, `service/notify_rule.go`, `service/notify_media.go`, `service/message_template.go`, `service/subscribe_rule.go`
- **前端**: `web/src/pages/notification/` (Rules, Media, Templates, Subscribe)
- **API**: `/api/v1/notify-rules`, `/api/v1/notify-media`, `/api/v1/message-templates`, `/api/v1/subscribe-rules` (~25 endpoints)
- **状态**: ✅ 完成（v4.11.0 移除 v1 NotifyPolicy 管道，统一为 v2）
- **文档**: [docs/architecture.md](docs/architecture.md)（引擎状态机 + 通知管道）

## 静默规则 (mute-rule)

- **功能**: 时间窗口 + 周期性静默、命中预览
- **后端**: `model/mute_rule.go`, `handler/mute_rule.go`, `service/mute_rule.go`, `repository/mute_rule.go`
- **前端**: `web/src/pages/alerts/mute/Index.vue`
- **API**: `/api/v1/mute-rules` (6 endpoints, 含 preview)
- **状态**: ✅ 完成

## 抑制规则 (inhibition-rule)

- **功能**: Alertmanager 风格，source→target 条件抑制
- **后端**: `model/inhibition_rule.go`, `handler/inhibition_rule.go`, `service/inhibition_rule.go`, `repository/inhibition_rule.go`
- **前端**: `web/src/pages/alerts/inhibition/Index.vue`
- **API**: `/api/v1/inhibition-rules` (5 endpoints)
- **状态**: ✅ 完成

## 标签注册表 (label-registry)

- **功能**: 从 Prom/VM 数据源自动同步 label key/value，支持自动补全
- **后端**: `model/label_registry.go`, `handler/label_registry.go`, `service/label_registry.go`, `repository/label_registry.go`
- **API**: `/api/v1/label-registry` (3 endpoints)
- **状态**: ✅ 完成

## 数据源 (datasource)

- **功能**: Prom/VM/VLogs/Zabbix 多源管理、健康检查、Instant/Range Query、日志查询、标签代理
- **后端**: `model/datasource.go`, `handler/datasource.go`, `service/datasource.go`, `repository/datasource.go`, `pkg/datasource/` (8 files)
- **前端**: `web/src/pages/datasources/Index.vue`, `web/src/pages/explore/Index.vue` (数据查询页, 路由 `/query`)
- **API**: `/api/v1/datasources` (12 endpoints: CRUD + health-check + query + query-range + log-query + labels/keys + labels/values + metrics)
- **状态**: ✅ 完成

## 仪表盘 V2 (dashboards)

- **功能**: 面板仪表盘、变量模板系统、PromQL 查询 + ECharts 时序图
- **后端**: `model/dashboard.go`, `handler/dashboard_v2.go`, `service/dashboard.go`, `repository/dashboard.go`
- **前端**: `web/src/pages/dashboards/Index.vue`, `web/src/pages/dashboards/View.vue`, `web/src/components/query/`, `web/src/components/time/`
- **API**: `/api/v1/dashboards` (5 endpoints: CRUD)
- **依赖**: datasource (查询数据)
- **状态**: ✅ 完成
- **迁移**: 000016_dashboards

## 值班排班 (schedule)

- **功能**: 排班管理、轮转 (daily/weekly/custom)、替班、iCal 导出
- **后端**: `model/schedule.go`, `handler/schedule.go`, `service/schedule.go`, `repository/schedule.go`, `repository/oncall_shift.go`
- **前端**: `web/src/pages/schedule/` (5 components)
- **API**: `/api/v1/schedules` (15 endpoints, 含 iCal)
- **状态**: ✅ 完成

## 升级策略 (escalation)

- **功能**: 多步骤升级，支持 user/team/schedule 目标，lark_personal/email/webhook 渠道
- **后端**: `model/schedule.go` (EscalationPolicy/Step), `handler/schedule.go`, `service/schedule.go`
- **前端**: `web/src/pages/oncall/EscalationPolicies.vue`（CRUD + 步骤管理）
- **API**: `/api/v1/escalation-policies` (8 endpoints)
- **状态**: ✅ 完成

## 认证 (auth)

- **功能**: JWT 本地登录 + Keycloak OIDC SSO + JWT 7天宽限续签
- **后端**: `handler/auth.go`, `handler/oidc.go`, `service/auth.go`, `service/oidc.go`, `middleware/auth.go`
- **前端**: `web/src/pages/Login.vue`, `web/src/stores/auth.ts`, `web/src/router/index.ts`
- **API**: `/api/v1/auth/*` (10 endpoints)
- **状态**: ✅ 完成

## 用户管理 (user)

- **功能**: 用户 CRUD、虚拟用户、密码管理、个人设置、飞书绑定
- **后端**: `model/user.go`, `handler/user.go`, `service/user.go`, `repository/user.go`
- **前端**: `web/src/pages/settings/UserManagement.vue`, `VirtualUsers.vue`
- **API**: `/api/v1/users` (8 endpoints) + `/api/v1/me/*` (7 endpoints, 含 preferences)
- **状态**: ✅ 完成（v4.11.0 新增用户偏好系统）

## 团队 (team)

- **功能**: 团队 CRUD、成员管理
- **后端**: `model/team.go`, `handler/team.go`, `service/team.go`, `repository/team.go`
- **前端**: `web/src/pages/settings/TeamManagement.vue`
- **API**: `/api/v1/teams` (7 endpoints)
- **状态**: ✅ 完成

## 业务分组 (biz-group)

- **功能**: 树形分组、match_labels 作用域
- **后端**: `model/biz_group.go`, `handler/biz_group.go`, `service/biz_group.go`, `repository/biz_group.go`
- **前端**: `web/src/pages/settings/BizGroupManagement.vue`
- **API**: `/api/v1/biz-groups` (9 endpoints)
- **状态**: ✅ 完成

## 仪表盘 (dashboard)

- **功能**: 统计概览、MTTA/MTTR 分析、趋势图、Top 规则、CSV 导出
- **后端**: `handler/dashboard.go`（274 行）, `service/dashboard_stats.go`（821 行，12 个方法）
- **前端**: `web/src/pages/dashboard/Index.vue`
- **API**: `/api/v1/dashboard/*` (7 endpoints)
- **状态**: ✅ 完成（v4.11.0 handler/service 拆分）

## AI 助手 (ai)

- **功能**: LLM 告警分析报告、SOP 建议、连接测试、多供应商配置、规则生成、标签推荐、抑制规则生成、静默规则生成、规则优化（ImproveRule）、Few-shot 提示模板、生成结果缓存、**会话持久化（ai_conversations + ai_tool_calls）**、工具调用追踪
- **后端**: `service/ai.go`, `handler/ai.go`, `service/alert_context.go`, `service/alert_pipeline.go`, `service/rule_generator.go`, `service/rule_gen_prompts.go`, `service/rule_gen_cache.go`, `handler/ai_rule.go`, `model/ai_conversation.go`, `repository/ai_conversation.go`
- **前端**: `web/src/pages/settings/AISettings.vue`, `web/src/composables/useAIModule.ts`, `web/src/pages/alerts/rules/Index.vue`（AI 生成按钮 + 模态框）, `web/src/pages/alerts/mute/Index.vue`（AI 生成屏蔽按钮）
- **API**: `/api/v1/ai/*` (14 endpoints: config, test, chat, report, sop, modules, providers, test-provider, rules/generate, rules/validate, rules/suggest-labels, rules/generate-inhibition, rules/generate-mute, rules/improve)
- **迁移**: 000043_ai_conversations, 000044_ai_tool_calls
- **状态**: ✅ 完成（含多供应商配置 + 模块级供应商选择 + 规则页 AI 生成入口 + P1.4 会话持久化）

## 飞书集成 (lark)

- **功能**: Webhook 通知、Bot API (DM + 群消息)、卡片模板、Bot 指令回调
- **后端**: `pkg/lark/` (2 files), `service/lark.go`, `service/larkbot.go`, `handler/larkbot.go`
- **API**: `POST /lark/event`, `/api/v1/lark/bot/config` (2 endpoints)
- **状态**: ✅ 完成

## 系统设置 (system-setting)

- **功能**: AES-256-GCM 加密 KV 存储（AI/Lark/SMTP/OIDC 配置）
- **后端**: `model/system_setting.go`, `service/system_setting.go`, `repository/system_setting.go`, `handler/oidc_settings.go`, `handler/smtp_settings.go`
- **前端**: `web/src/pages/settings/` (AIConfig, LarkBotConfig, OIDCConfig, SMTPConfig)
- **API**: `/api/v1/settings/*`, `/api/v1/ai/config`, `/api/v1/lark/bot/config`
- **状态**: ✅ 完成

## 审计日志 (audit-log)

- **功能**: 操作审计（11 种 action, 9 种 resource）
- **后端**: `model/audit_log.go`, `handler/audit_log.go`, `service/audit_log.go`, `repository/audit_log.go`
- **前端**: `web/src/pages/settings/AuditLog.vue`
- **API**: `GET /api/v1/audit-logs`
- **状态**: ✅ 完成

## Webhook 入站 (webhook)

- **功能**: Alertmanager/VMAlert 格式接收、AlertChannel 路由
- **后端**: `model/webhook.go`, `handler/heartbeat.go`
- **API**: `POST /webhooks/alertmanager`, `POST /heartbeat/:token`
- **状态**: ✅ 完成（仅支持 Alertmanager 格式）

## 状态页面 (status-service)

- **功能**: 公开状态页面服务管理（运维/降级/中断/维护四种状态）、排序、图标配置
- **后端**: `model/status_service.go`, `handler/status_service.go`, `service/status_service.go`, `repository/status_service.go`
- **API**: `/api/v1/status-services` (5 endpoints: LIST 列表, GET 详情, POST 创建, PUT 更新, DELETE 删除)
- **权限**: 列表/详情已认证即可，创建/更新/删除仅管理员
- **状态**: ✅ 完成

## 预设规则 (preset-rule)

- **功能**: 预定义告警规则模板库（社区最佳实践/供应商推荐）、分类浏览、一键应用创建 AlertRule、YAML 导入、monitoring-trading 全量导入
- **后端**: `model/preset_rule.go`, `handler/preset_rule.go`, `service/preset_rule.go`, `repository/preset_rule.go`
- **脚本**: `scripts/import-presets/main.go` — 从 monitoring-trading YAML 全量导入 299 条规则（支持 --dry-run）
- **种子数据**: 启动时自动 seed 45 条内置告警规则 + 16 条抑制规则模板（覆盖主机/容器/MySQL/Redis/MongoDB/ES/Kafka/RabbitMQ/Nginx/Blackbox/应用）
- **API**: `/api/v1/preset-rules` (6 endpoints: LIST 列表, GET 详情, GET /categories 分类列表, POST /:id/apply 应用, POST /import YAML 导入, DELETE 删除)
- **权限**: 列表/详情/分类已认证即可，应用/导入/删除需管理权限
- **兼容文档**: `docs/monitoring-trading-compat.md`
- **状态**: ✅ 完成

## 通知中心 (user-notification)

- **功能**: 用户级通知中心，推送系统/告警/事件通知，支持未读/已读状态管理
- **后端**: `model/user_notification.go`, `handler/user_notification.go`, `service/user_notification.go`, `repository/user_notification.go`
- **前端**: `web/src/pages/notification/Center.vue`, `web/src/components/common/NotificationBell.vue`
- **API**: `/api/v1/notifications` (5 endpoints: LIST 列表, GET /unread-count 未读数, PATCH /:id/read 标记已读, POST /read-all 全部已读, DELETE 删除)
- **迁移**: 000045_create_notifications
- **状态**: ✅ 完成

## RBAC 权限查询 (permissions)

- **功能**: 返回当前用户的完整权限列表（全局角色 + 团队角色 + 细粒度权限点）
- **后端**: `handler/permissions.go`，依赖 `TeamService.ListByUser`
- **前端**: `web/src/composables/usePermissions.ts`（hasPerm / hasAnyPerm / isTeamLead）, `web/src/permissions.ts`（50+ 权限常量）, `web/src/directives/vCan.ts`（v-can 指令）
- **API**: `GET /api/v1/me/permissions` (1 endpoint)
- **权限**: 已认证即可
- **已接入页面**: 告警规则页（创建/导入/AI 按钮）、告警事件页（认领/关闭按钮）
- **状态**: ✅ 完成

## Alertmanager 导入 (alertmanager-import)

- **功能**: 解析 Alertmanager YAML 配置文件，自动导入 receivers 为 Channels、inhibit_rules 为 InhibitionRules
- **后端**: `handler/alertmanager_import.go`, `service/alertmanager_import.go`
- **API**: `POST /api/v1/integrations/import-alertmanager` (1 endpoint, 管理权限)
- **输入**: JSON body `{"yaml": "..."}` 或 multipart file upload
- **输出**: `{channels_created, inhibitions_created, warnings[], errors[]}`
- **状态**: ✅ 完成

---

## 知识库 (knowledge-base) [v4.15.4]

- **功能**: 知识文档管理（SOP/故障案例/Runbook/模板/Markdown），FULLTEXT 全文检索，有用度投票
- **后端**: `model/knowledge.go`, `handler/knowledge.go`, `service/knowledge_base.go`, `repository/knowledge.go`
- **API**: `/api/v1/knowledge` (7 endpoints: LIST, GET, POST, PUT, DELETE, POST /search, POST /:id/helpful)
- **权限**: 列表/搜索/详情已认证即可，创建/更新/删除需管理权限
- **迁移**: 000054_knowledge_base
- **状态**: ✅ 完成（P1.3 知识库服务）

## 诊断工作流 (diagnostic-workflow) [v4.15.5]

- **功能**: 诊断工作流编排引擎，支持多步骤诊断流程定义、按告警匹配自动触发、执行记录追踪
- **后端**: `model/diagnostic_workflow.go`, `handler/diagnostic_workflow.go`, `service/diagnostic_workflow.go`, `repository/diagnostic_workflow.go`
- **API**: `/api/v1/diagnostic-workflows` (8 endpoints: LIST, GET, POST, PUT, DELETE, PUT /:id/steps, POST /:id/run, POST /match) + `/api/v1/diagnostic-runs` (2 endpoints: LIST, GET)
- **权限**: 列表/详情已认证即可，创建/更新/删除需管理权限，执行需操作权限
- **依赖**: incident-context + change-event + ai-agent
- **状态**: ✅ 完成（Phase 2-3 诊断工作流编排）

## 变更事件 (change-event) [v4.15.5]

- **功能**: 变更事件接入，记录部署/配置/基础设施变更，供诊断工作流关联分析
- **后端**: `model/change_event.go`, `handler/change_event.go`, `service/change_event.go`, `repository/change_event.go`
- **API**: `/api/v1/change-events` (4 endpoints: LIST, GET, POST /ingest, DELETE)
- **权限**: 列表/详情已认证即可，接入/删除需管理权限
- **状态**: ✅ 完成（Phase 2-3 变更事件接入）

## Incident 上下文聚合 (incident-context) [v4.15.5]

- **功能**: 聚合 Incident 相关上下文（告警、变更、知识库、历史故障），供 AI 分析和诊断工作流使用
- **后端**: `service/incident_context.go`（仅 service 层，无独立 handler/model/repository）
- **依赖**: alert-event, change-event, knowledge-base, incident
- **状态**: ✅ 完成（Phase 2-3 上下文聚合，service only）

## 定时巡检 Agent (inspection) [v4.15.8]

- **功能**: Cron 定时调度 AI 巡检任务，自主调用工具收集数据，生成结构化巡检报告，飞书卡片通知
- **后端**:
  - `model/inspection.go` — InspectionTask + InspectionRun
  - `repository/inspection.go` — CRUD + ListEnabledTasks
  - `service/inspection_prompt.go` — 巡检 prompt 模板
  - `service/inspection_executor.go` — 单次巡检执行器
  - `service/inspection_scheduler.go` — Cron 调度 + Leader 选举
  - `handler/inspection.go` — Task CRUD + Run CRUD + RunNow + ValidateCron
  - `router/admin_routes.go` — /inspection/tasks, /inspection/runs, /inspection/validate-cron
- **前端**:
  - `api/inspection.ts` — API 封装
  - `components/common/CronInput.vue` — Cron 表达式输入组件
  - `pages/platform/inspections/Index.vue` — 任务列表 + 创建/编辑 Modal
  - `pages/platform/inspections/RunDetail.vue` — 运行报告详情
- **迁移**: `000061_inspection_task.{up,down}.sql`
- **依赖**: ai-agent, ai-tools, leader-election, cron/v3
- **状态**: ✅ 完成

## AI 工具元数据增强 [v4.15.8]

- **功能**: AITool 新增 RiskLevel (0=read/1=write/2=destructive) + IO 标注 + /api/ai/tools/registry 端点
- **后端**: `service/ai_tools.go` (AITool 结构体), `handler/ai.go` (ListTools), `router/setting_routes.go`
- **状态**: ✅ 完成

## 文档索引

| 文档 | 内容 |
|------|------|
| [CLAUDE.md](CLAUDE.md) | AI 协作规范（代码约定、目录、错误码） |
| [MODULES.md](MODULES.md) | 本文件：44 个模块清单 + 状态 |
| [CHANGELOG.md](CHANGELOG.md) | 变更日志 |
| [docs/architecture.md](docs/architecture.md) | 架构设计 + ADR + 引擎状态机 + 通知管道 |
| [docs/api.md](docs/api.md) | REST API 参考（175+ 端点） |
| [docs/ci-deploy.md](docs/ci-deploy.md) | CI/CD 部署文档 |
| [docs/phases.md](docs/phases.md) | Phase 追踪 + QA 修复汇总 |
| [docs/PLAN-status.md](docs/PLAN-status.md) | v2.0 重构执行状态（Phase 1-5 全部完成） |
| [docs/rbac.md](docs/rbac.md) | RBAC 权限体系设计（角色 + 权限 + 中间件） |
| [docs/data-source-routing.md](docs/data-source-routing.md) | 多数据源路由 + 标签匹配引擎 |
| [docs/ai-rule-generation.md](docs/ai-rule-generation.md) | AI 规则生成引擎（dry-run + few-shot + 缓存） |
| [docs/preset-rule-library.md](docs/preset-rule-library.md) | 预置规则库（monitoring-trading 315 条导入） |
| [docs/notification-pipeline.md](docs/notification-pipeline.md) | v2 通知管道（规则匹配 + 模板渲染 + 多渠道分发） |

---

## v2.0 新增模块（Phase 1-5）

| 模块 | 文件 | 状态 | 说明 |
|------|------|------|------|
| **协作空间** Channel | model/channel.go + repo/service/handler/channel.go | ✅ 生产就绪 | CRUD + Star + 降噪配置 + 分派策略 |
| **故障** Incident | model/incident.go + repo/service/handler/incident.go + service/incident_aggregator.go | ✅ 生产就绪 | 完整生命周期：ack/close/reopen/snooze/merge/reassign/escalate + 自动关闭 + fingerprint 聚合 |
| **告警 v2** Alert + AlertEventV2 | model/alert.go + repo/service/alert.go + handler/alert.go | ✅ 生产就绪 | 按 alert_key 去重，关联 Channel + Incident |
| **告警 v2 管道** AlertV2Pipeline | service/alert_v2_pipeline.go | ✅ 生产就绪 | 非侵入式引擎桥接，WrapOnAlert hook + IncidentAggregator 钩子 |
| **降噪引擎** NoiseReducer | service/noise_reducer.go | ✅ 生产就绪 | 排除规则 + 聚合 + 风暴预警 + 抖动检测 |
| **排除规则** ExclusionRule | repo/service/handler/exclusion_rule.go | ✅ 生产就绪 | Per-channel 排除规则 CRUD |
| **分派策略** DispatchPolicy | model/dispatch.go + repo/service/dispatch.go + handler/dispatch.go | ✅ 生产就绪 | 触发条件 + 延迟 + 重复 + 标签增强 + 升级绑定 + 分派日志查看 |
| **Webhook 集成** Integration | model/integration.go + repo/service/integration.go + handler/integration.go | ✅ 生产就绪 | Standard/AlertManager/Grafana 三格式 + Pipeline + 限流 100/s |
| **路由规则** RoutingRule | model/integration.go + repo/integration.go | ✅ 生产就绪 | 共享集成的 label 路由 |
| **故障复盘** PostMortem | model/incident.go + repo/service/handler/post_mortem.go | ✅ 生产就绪 | CRUD + AI 生成初稿 + 发布 |

### v2.0 DB 迁移文件（000019-000033）

| 序号 | 文件 | 表 |
|------|------|----|
| 000019 | create_channels | channels |
| 000020 | create_channel_stars | channel_stars |
| 000021 | create_channel_exclusion_rules | channel_exclusion_rules |
| 000022 | create_incidents | incidents |
| 000023 | create_incident_assignees | incident_assignees |
| 000024 | create_incident_timelines | incident_timelines |
| 000025 | create_post_mortems | post_mortems |
| 000026 | create_alerts_v2 | alerts |
| 000027 | create_alert_events_v2 | alert_events_v2 |
| 000028 | create_integrations | integrations |
| 000029 | create_routing_rules | routing_rules |
| 000030 | seed_default_channel | INSERT default channel |
| 000031 | create_dispatch_policies | dispatch_policies |
| 000032 | create_dispatch_logs | dispatch_logs |
| 000033 | alert_rule_channel | ALTER alert_rules ADD channel_id |

---

## 路由规则 (routing-rule) [v2.0.2]

- **功能**: 共享集成的告警路由规则 CRUD（优先级排序、条件匹配、目标空间）
- **后端文件**: `internal/handler/routing_rule.go`, `internal/repository/integration.go`（RoutingRuleRepository 内联）
- **API**: `GET/POST /api/v1/integrations/:id/routing-rules`, `PUT/DELETE /api/v1/routing-rules/:id`
- **状态**: ✅ 完成

## 告警规则批量操作 (alert-rule-batch) [v2.0.1]

- **功能**: 批量启用/禁用/删除告警规则
- **后端文件**: `internal/handler/alert_rule.go`（BatchEnable/Disable/Delete）
- **API**: `POST /api/v1/alert-rules/batch/enable|disable|delete`（manage 权限）
- **状态**: ✅ 完成

## 故障复盘增强 (post-mortem-editor) [v2.0.2]

- **功能**: PostMortem Tab 使用 md-editor-v3 替换纯 textarea，支持 Markdown 实时预览
- **前端文件**: `web/src/pages/incidents/Detail.vue`
- **状态**: ✅ 完成

## 故障操作增强 (incident-ops) [v2.0.2]

- **功能**: 故障详情页新增暂缓（Snooze）/合并（Merge）/重新分派（Reassign）操作入口
- **前端文件**: `web/src/pages/incidents/Detail.vue`
- **后端 API**: 已有（POST /incidents/:id/snooze|merge|reassign）
- **状态**: ✅ 完成
