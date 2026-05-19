# 模块清单 (MODULES)

> 最后更新: 2026-05-19 | tag: v4.10.27
> 共 34 个 model, 46 个 handler, 46 个 service, 34 个 repository, 268+ API 端点

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
dashboard ──→ alert-event + incident + channel + team (统计数据)
```

改模块前查上方依赖：改 notification 会影响 alert-engine 和 escalation；改 schedule 会影响 escalation。

## 测试覆盖状态

| 模块 | 功能状态 | 单元测试 | 集成测试 | 覆盖率 |
|------|----------|----------|----------|--------|
| 告警引擎 | ✅ | ✅ evaluator_test.go (19) + rule_eval_test.go + suppression_test.go (26) | ❌ | service 层 ~40% |
| 告警规则 | ✅ | ❌ | ❌ | 0% |
| 告警事件 | ✅ | ❌ | ❌ | 0% |
| 告警通道 | ✅ | ✅ alert_channel_test.go (handler + service) | ❌ | ~30% |
| 通知管道 | ✅ | ✅ notification_test.go (20 tests) | ❌ | service 层 ~25% |
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
| 宠物系统 | ✅ | ❌ | ❌ | 0% |
| 状态页面 | ✅ | ❌ | ❌ | 0% |
| Alertmanager 导入 | ✅ | ❌ | ❌ | 0% |

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

- **功能**: v1 策略管道 + v2 规则管道、多渠道发送、订阅机制
- **后端**: `service/notification.go`, `service/notify_rule.go`, `service/notify_media.go`, `service/message_template.go`, `service/subscribe_rule.go`
- **前端**: `web/src/pages/notification/` (Rules, Media, Templates, Subscribe)
- **API**: `/api/v1/notify-rules`, `/api/v1/notify-media`, `/api/v1/message-templates`, `/api/v1/subscribe-rules` (~25 endpoints)
- **状态**: ✅ 完成
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

## 仪表盘 V2 (dashboard-v2)

- **功能**: 面板仪表盘、变量模板系统、PromQL 查询 + ECharts 时序图
- **后端**: `model/dashboard.go`, `handler/dashboard_v2.go`, `service/dashboard.go`, `repository/dashboard.go`
- **前端**: `web/src/pages/dashboard-v2/Index.vue`, `web/src/pages/dashboard-v2/View.vue`, `web/src/components/query/`, `web/src/components/time/`
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
- **API**: `/api/v1/users` (8 endpoints) + `/api/v1/me/*` (5 endpoints)
- **状态**: ✅ 完成

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
- **后端**: `handler/dashboard.go`
- **前端**: `web/src/pages/dashboard/Index.vue`
- **API**: `/api/v1/dashboard/*` (7 endpoints)
- **状态**: ✅ 完成

## AI 助手 (ai)

- **功能**: LLM 告警分析报告、SOP 建议、连接测试、多供应商配置、规则生成
- **后端**: `service/ai.go`, `handler/ai.go`, `service/alert_context.go`, `service/alert_pipeline.go`, `service/rule_generator.go`, `handler/ai_rule.go`
- **前端**: `web/src/pages/settings/AISettings.vue`, `web/src/composables/useAIModule.ts`
- **API**: `/api/v1/ai/*` (10 endpoints: config, test, chat, report, sop, modules, providers, test-provider, rules/generate, rules/validate)
- **状态**: ✅ 完成（含多供应商配置 + 模块级供应商选择）

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

## 宠物系统 (pet)

- **功能**: 用户虚拟宠物养成（喂食/玩耍/互动/升级）、互动记录、等级经验系统
- **后端**: `model/pet.go`, `handler/pet.go`, `service/pet.go`, `repository/pet.go`
- **前端**: `web/src/pages/settings/PetSettings.vue`（个人设置内嵌）
- **API**: `/api/v1/pet` (5 endpoints: GET 获取宠物, PUT 更新名称, POST /feed 喂食, POST /play 玩耍, GET /interactions 互动记录)
- **状态**: ✅ 完成

## 状态页面 (status-service)

- **功能**: 公开状态页面服务管理（运维/降级/中断/维护四种状态）、排序、图标配置
- **后端**: `model/status_service.go`, `handler/status_service.go`, `service/status_service.go`, `repository/status_service.go`
- **API**: `/api/v1/status-services` (5 endpoints: LIST 列表, GET 详情, POST 创建, PUT 更新, DELETE 删除)
- **权限**: 列表/详情已认证即可，创建/更新/删除仅管理员
- **状态**: ✅ 完成

## 预设规则 (preset-rule)

- **功能**: 预定义告警规则模板库（社区最佳实践/供应商推荐）、分类浏览、一键应用创建 AlertRule、YAML 导入
- **后端**: `model/preset_rule.go`, `handler/preset_rule.go`, `service/preset_rule.go`, `repository/preset_rule.go`
- **API**: `/api/v1/preset-rules` (6 endpoints: LIST 列表, GET 详情, GET /categories 分类列表, POST /:id/apply 应用, POST /import YAML 导入, DELETE 删除)
- **权限**: 列表/详情/分类已认证即可，应用/导入/删除需管理权限
- **状态**: ✅ 完成

## Alertmanager 导入 (alertmanager-import)

- **功能**: 解析 Alertmanager YAML 配置文件，自动导入 receivers 为 Channels、inhibit_rules 为 InhibitionRules
- **后端**: `handler/alertmanager_import.go`, `service/alertmanager_import.go`
- **API**: `POST /api/v1/integrations/import-alertmanager` (1 endpoint, 管理权限)
- **输入**: JSON body `{"yaml": "..."}` 或 multipart file upload
- **输出**: `{channels_created, inhibitions_created, warnings[], errors[]}`
- **状态**: ✅ 完成

---

## 文档索引

| 文档 | 内容 |
|------|------|
| [CLAUDE.md](CLAUDE.md) | AI 协作规范（代码约定、目录、错误码） |
| [MODULES.md](MODULES.md) | 本文件：40 个模块清单 + 状态 |
| [CHANGELOG.md](CHANGELOG.md) | 变更日志 |
| [docs/architecture.md](docs/architecture.md) | 架构设计 + ADR + 引擎状态机 + 通知管道 |
| [docs/api.md](docs/api.md) | REST API 参考（175+ 端点） |
| [docs/ci-deploy.md](docs/ci-deploy.md) | CI/CD 部署文档 |
| [docs/phases.md](docs/phases.md) | Phase 追踪 + QA 修复汇总 |
| [docs/PLAN-status.md](docs/PLAN-status.md) | v2.0 重构执行状态（Phase 1-5 全部完成） |

---

## v2.0 新增模块（Phase 1-5）

| 模块 | 文件 | 状态 | 说明 |
|------|------|------|------|
| **协作空间** Channel | model/channel.go + repo/service/handler/channel.go | ✅ 生产就绪 | CRUD + Star + 降噪配置 + 分派策略 |
| **故障** Incident | model/incident.go + repo/service/handler/incident.go | ✅ 生产就绪 | 完整生命周期：ack/close/reopen/snooze/merge/reassign/escalate + 自动关闭 |
| **告警 v2** Alert + AlertEventV2 | model/alert.go + repo/service/alert.go + handler/alert.go | ✅ 生产就绪 | 按 alert_key 去重，关联 Channel + Incident |
| **告警 v2 管道** AlertV2Pipeline | service/alert_v2_pipeline.go | ✅ 生产就绪 | 非侵入式引擎桥接，WrapOnAlert hook |
| **降噪引擎** NoiseReducer | service/noise_reducer.go | ✅ 生产就绪 | 排除规则 + 聚合 + 风暴预警 + 抖动检测 |
| **排除规则** ExclusionRule | repo/service/handler/exclusion_rule.go | ✅ 生产就绪 | Per-channel 排除规则 CRUD |
| **分派策略** DispatchPolicy | model/dispatch.go + repo/service/dispatch.go + handler/dispatch.go | ✅ 生产就绪 | 触发条件 + 延迟 + 重复 + 标签增强 + 升级绑定 |
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
