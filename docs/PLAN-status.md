# SREAgent 重构执行状态

> 本文件是实时执行状态追踪，所有参与者（人或 AI agent）在完成任务后必须更新此文件。
> 最后更新：2026-05-07

---

## 当前状态

| 字段 | 值 |
|------|-----|
| **当前 Phase** | Phase 1 — 进行中 |
| **当前版本** | v1.16.23 |
| **目标版本** | v2.0.0 (Phase 1 完成) |
| **阻塞项** | 无 |

---

## 执行规范

### 更新规则
1. **开始任务前**：将对应条目状态改为 `🔄 进行中`，填写开始时间和执行者
2. **完成任务后**：将状态改为 `✅ 完成`，填写完成时间，附注 commit hash 或版本号
3. **遇到阻塞**：将状态改为 `⛔ 阻塞`，在备注中说明原因
4. **任务取消**：将状态改为 `🚫 取消`，说明原因

### 状态标记
- `⬜ 待开始` — 未触碰
- `🔄 进行中` — 正在执行
- `✅ 完成` — 已完成并验证
- `⛔ 阻塞` — 被阻塞无法推进
- `🚫 取消` — 已决定不做

### 参与者标识
每个任务标注执行者，格式：`@人名` 或 `@agent-名称`

---

## Phase 1：核心模型重构（目标版本 v2.0.0）

### 1.1 后端 — 数据模型 & 迁移

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 1.1.1 | Channel 模型定义 (model/channel.go) | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | Channel + ChannelExclusionRule + ChannelStar + 降噪配置结构体 + FilterCondition |
| 1.1.2 | Channel Repository + Service + Handler | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | CRUD + Star/Unstar + 列表带收藏标记 |
| 1.1.3 | Incident 模型定义 (model/incident.go) | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | Incident + IncidentAssignee + IncidentTimeline + PostMortem |
| 1.1.4 | Incident Repository + Service + Handler | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | CRUD + ack/close/reopen/snooze/merge/reassign/escalate/comment + timeline |
| 1.1.5 | IncidentTimeline 模型 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | 已在 model/incident.go 定义，API 在 IncidentHandler 中实现 |
| 1.1.6 | 重构 AlertEvent → Alert + Event 双表 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | model/alert.go (Alert + AlertEventV2) + repo + service + handler，旧 AlertEvent 保留兼容 |
| 1.1.7 | RoutingRule 模型 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | 在 integration.go 中定义 |
| 1.1.8 | Integration 模型 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | Integration + AlertPipelineStep + LabelEnhancementRule |
| 1.1.9 | DB 迁移文件 (up + down) | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | 迁移 000019-000029：channels, channel_stars, channel_exclusion_rules, incidents, incident_assignees, incident_timelines, post_mortems, alerts, alert_events_v2, integrations, routing_rules |
| 1.1.10 | 创建"默认协作空间"迁移逻辑 | ⬜ 待开始 | | | | 现有告警全部归入默认空间 |

### 1.2 后端 — 告警引擎适配

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 1.2.1 | rule_eval 产出 Event 而非 AlertEvent | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | 非侵入式：AlertV2Pipeline.WrapOnAlert 拦截 onAlert 回调，并行驱动 v2 路径，原引擎保持不变 |
| 1.2.2 | Event → Alert 合入逻辑 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | AlertV2Pipeline.upsertAlert：按 alert_key 去重，同一序列累加 fire_count |
| 1.2.3 | Alert → Incident 触发逻辑 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | AlertV2Pipeline.ensureIncident：复用已有 open Incident 或新建，时间线自动记录 |
| 1.2.4 | Incident 自动关闭逻辑 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | AlertV2Pipeline.handleResolution：所有关联 Alert resolved → Incident resolved（尊重 follow_alert_close） |
| 1.2.5 | Incident 超时自动关闭 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | IncidentService.StartAutoCloseWorker：每 5 分钟检查 auto_close_minutes，appCtx 控制生命周期 |

### 1.3 后端 — API 路由注册

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 1.3.1 | /api/v1/channels CRUD | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | GET/POST/PUT/DELETE + Star/Unstar |
| 1.3.2 | /api/v1/incidents CRUD + 操作接口 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | acknowledge, close, reopen, snooze, merge, reassign, escalate |
| 1.3.3 | /api/v1/incidents/:id/timeline | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | GET 时间线 + POST 评论 |
| 1.3.4 | /api/v1/alerts 列表 + 详情 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | GET 列表 + GET 详情 + GET /:id/events |
| 1.3.5 | /api/v1/integrations CRUD + webhook 入口 | ⬜ 待开始 | | | | POST /api/v1/integrations/:token/alerts (Phase 4) |
| 1.3.6 | /api/v1/routing-rules CRUD | ⬜ 待开始 | | | | (Phase 4) |

### 1.4 前端 — 导航 & 页面

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 1.4.1 | 侧边栏导航重构（新菜单结构） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | MainLayout.vue 新增 协作空间/故障管理/告警视图 三项 |
| 1.4.2 | 协作空间列表页 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/channels/Index.vue：卡片列表 + Star + 创建弹窗 |
| 1.4.3 | 协作空间详情页 — 故障列表 Tab | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/channels/Detail.vue Tab1 |
| 1.4.4 | 协作空间详情页 — 统计概览 Tab | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/channels/Detail.vue Tab2：4 张统计卡片 + channel 元信息 |
| 1.4.5 | 协作空间详情页 — 配置 Tab | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/channels/Detail.vue Tab3：inline 编辑表单 |
| 1.4.6 | 故障列表页（全局） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/incidents/Index.vue：全部/我的 + 筛选 + 认领/关闭操作 |
| 1.4.7 | 故障详情页 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/incidents/Detail.vue：操作栏 + Tab(概览/关联告警/时间线) + 右侧信息栏 |
| 1.4.8 | 告警列表页 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/alerts-v2/Index.vue：筛选 + 关联故障/空间 |
| 1.4.9 | 告警详情页 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | pages/alerts-v2/Detail.vue：概览+Labels+Annotations + Events Tab + 侧边栏链接 |
| 1.4.10 | 前端 API 层 + TypeScript 类型定义 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | types/index.ts: Channel/Incident/AlertV2/AlertEventV2; api/index.ts: channelV2Api/incidentApi/alertV2Api |

### 1.5 验证

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 1.5.1 | go build 通过 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | |
| 1.5.2 | vue-tsc --noEmit 通过 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | |
| 1.5.3 | 数据迁移测试 | ⬜ 待开始 | | | | 需在实际 MySQL 上运行验证 |
| 1.5.4 | CHANGELOG + MODULES.md 更新 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | CHANGELOG 已更新 |

---

## Phase 2：智能降噪（目标版本 v2.1.0）

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 2.1 | 规则聚合引擎（统一控制 + 细粒度控制） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | NoiseReducer.computeAggregationKey：统一/细粒度 + branch 条件匹配 |
| 2.2 | 聚合窗口（开关 + 计时起点 + 时长） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | ChannelNoiseAggregation.WindowEnabled/Origin/Minutes，存 JSON 到 aggregation_config |
| 2.3 | 风暴预警（阈值通知） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | NoiseReducer.checkStorm：滚动 1 分钟计数器，每阈值只通知一次 |
| 2.4 | 抖动检测（3 模式 + 参数） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | NoiseReducer.checkFlapping：off/notify_only/notify_then_silence，in-memory flapStates |
| 2.5 | 静默增强（周期-星期模式 + 快速静默） | ⬜ 待开始 | | | | 现有 MuteRule 已支持周期模式，快速静默待前端补充 |
| 2.6 | 排除规则 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | ExclusionRuleRepository + Service + Handler + API /channels/:id/exclusion-rules |
| 2.7 | 前端：空间降噪配置页 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | NoiseConfig.vue：聚合/窗口/风暴/抖动/排除规则全配置，嵌入 Channel Detail "降噪配置" Tab |
| 2.8 | 验证 + CHANGELOG | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | go build + vue-tsc 通过 |

---

## Phase 3：分派增强（目标版本 v2.2.0）

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 3.1 | 分派策略绑定 Channel + 多策略优先级 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | DispatchPolicy model: channel_id + priority + is_enabled；/api/v1/channels/:id/dispatch-policies CRUD |
| 3.2 | 触发条件（生效时间段 + 故障筛选） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | match_conditions (JSON FilterCondition[]) + active_time_config (DispatchActiveTimeConfig) |
| 3.3 | 延迟窗口（0-3600 秒） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | delay_seconds 字段；isActiveNow + matchConditions 匹配后延迟 N 秒分派 |
| 3.4 | 多环节升级（增删/移动/超时升级/重复通知） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | escalation_policy_id 绑定 + repeat_interval_seconds + max_repeats；前端支持优先级上下移动 |
| 3.5 | 通知方式（遵循个人偏好 vs 统一设置） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | notify_mode: personal_preference \| unified；unified_media_id |
| 3.6 | 标签增强（提取/组合/映射/删除） | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | LabelEnhancementAction: set/extract/combine/map/delete；DispatchService.ApplyLabelEnhancements；挂接到 AlertV2Pipeline.process() |
| 3.7 | 个人通知偏好配置 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | 已有 UserNotifyConfig + /me/notify-configs；notify_mode=personal_preference 时使用用户偏好 |
| 3.8 | 前端：空间分派配置页 | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | DispatchConfig.vue：策略列表 + 优先级排序 + 创建/编辑弹窗（延迟/重复/条件/时间/通知/升级/标签增强）；嵌入 Channel Detail "分派配置" Tab |
| 3.9 | 验证 + CHANGELOG | ✅ 完成 | @opencode | 2026-05-07 | 2026-05-07 | go build + vue-tsc 通过；DB 迁移 000031-000032 |

---

## Phase 4：告警引擎增强 + Webhook 接入（目标版本 v2.3.0）

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 4.1 | 无数据告警 | ⬜ 待开始 | | | | |
| 4.2 | 规则文件夹/树形分类 | ⬜ 待开始 | | | | |
| 4.3 | 告警规则关联协作空间 | ⬜ 待开始 | | | | |
| 4.4 | 标准 Webhook 接入 API | ⬜ 待开始 | | | | |
| 4.5 | AlertManager 格式兼容 | ⬜ 待开始 | | | | |
| 4.6 | Grafana 格式兼容 | ⬜ 待开始 | | | | |
| 4.7 | 告警处理管道 Pipeline | ⬜ 待开始 | | | | |
| 4.8 | 频率限制（100/s, 1000/min） | ⬜ 待开始 | | | | |
| 4.9 | 前端：集成中心 + Pipeline 配置 + 规则文件夹 | ⬜ 待开始 | | | | |
| 4.10 | 验证 + CHANGELOG | ⬜ 待开始 | | | | |

---

## Phase 5：故障复盘 + 分析增强（目标版本 v2.4.0）

| # | 任务 | 状态 | 执行者 | 开始 | 完成 | 备注 |
|---|------|------|--------|------|------|------|
| 5.1 | PostMortem 模型 + CRUD | ⬜ 待开始 | | | | |
| 5.2 | AI 故障总结（概述/影响/建议） | ⬜ 待开始 | | | | |
| 5.3 | AI 辅助复盘（自动生成初稿） | ⬜ 待开始 | | | | |
| 5.4 | 分析看板增强（按空间/团队维度） | ⬜ 待开始 | | | | |
| 5.5 | 前端：故障详情复盘 Tab + AI 按钮 | ⬜ 待开始 | | | | |
| 5.6 | 前端：增强仪表盘 | ⬜ 待开始 | | | | |
| 5.7 | 验证 + CHANGELOG | ⬜ 待开始 | | | | |

---

## 变更记录

| 日期 | 变更内容 | 执行者 |
|------|---------|--------|
| 2026-05-07 | 创建 Plan 文件（PLAN-flashcat-alignment.md + PLAN-feature-checklist.md + PLAN-status.md） | @opencode |
| 2026-05-07 | 完成 1.1.1 + 1.1.3 + 1.1.7 + 1.1.8 模型定义（channel.go, incident.go, integration.go），go build 通过 | @opencode |
| 2026-05-07 | 完成 Phase 1.1 后端全部：1.1.2 Channel CRUD + 1.1.4 Incident CRUD(含 ack/close/reopen/snooze/merge/reassign/escalate/comment) + 1.1.5 Timeline + 1.1.6 Alert+Event v2 双表 + 1.1.9 迁移 000019-000030 + 1.1.10 seed default channel + 1.3.1-1.3.4 API 路由注册 | @opencode |
| 2026-05-07 | 完成 Phase 1.2 告警引擎适配：AlertV2Pipeline(非侵入式 WrapOnAlert hook) + IncidentService.StartAutoCloseWorker(超时自动关闭) + appCtx 生命周期管理 | @opencode |
| 2026-05-07 | 完成 Phase 1.4 前端：侧边栏新增协作空间/故障/告警视图菜单 + 协作空间列表页 + 故障列表+详情页 + 告警v2列表页 + TypeScript 类型 + API 层 + i18n(中英) | @opencode |
| 2026-05-07 | 完成 Phase 3 分派增强：DispatchPolicy+DispatchLog 模型 + DispatchService(FindMatchingPolicy/ApplyLabelEnhancements/时间窗口/触发条件) + AlertV2Pipeline 集成 + API + DB 迁移 000031-000032 + 前端 DispatchConfig.vue + Channel Detail Tab | @opencode |
| | | |
