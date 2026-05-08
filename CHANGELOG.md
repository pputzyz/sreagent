# 变更日志 (CHANGELOG)

> 基于 git tag 和 commit 记录整理。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)

---

## [v2.4.0] - 2026-05-08

### Changed — UI 重构 Phase 3（配置类页面 FlashCat 对齐）

11 个配置类页面统一视觉语言：Geist 字体 / tabular nums / sre-row-card / sre-dot / hairline。

- **通知中心**：
  - `notification/Index.vue`：NTabs → 200px 左导航 + 内容区，URL hash 同步
  - `AlertChannels.vue`：NDataTable → sre-row-card 列表，标签匹配 chips、Webhook URL 复制、Throttle 显示
  - `Media.vue`：紧凑列表，type chip 着色（lark/email/webhook/script），测试发送按钮
  - `Rules.vue`：Match conditions chips → "→" 关联 media 列表
  - `Subscribe.vue`：订阅人头像组 + Match chips + 通知方式
  - `Templates.vue`：type chip + 内容预览 mono + 内置/自定义标记

- **告警引擎补齐**：
  - `mute/Index.vue`：状态分段（生效中/未来/已过期/禁用）+ Match chips + Schedule 摘要 + 命中预览抽屉
  - `inhibition/Index.vue`：Source/Target/Equal 分行展示 + 命中数 tnum
  - `history/Index.vue`：时间分段（7天/30天/90天/自定义）+ 紧凑历史列表（dim 0.85）+ Export CSV

- **集成与数据**：
  - `datasources/Index.vue`：表格 → 卡片网格（顶部 type 色条）+ 健康状态点 + Latency/Version stats + Test 按钮
  - `schedule/Index.vue`：Header + Sidebar 视觉对齐，班次色块（var(--sre-primary-soft) + 主色 marker），当前值班用主色实色

vue-tsc ✅ — 修复 4 处类型问题（DataSourceStatus 类型断言、SelectOption value 不能为 null、NSpin size="tiny" → "small"、duration/relTime null 兼容）

---

## [v2.3.0] - 2026-05-08

### Changed — UI 重构 Phase 2（详情页 FlashCat 对齐）

5 个详情/二级页面同步对齐 Phase 1 设计语言。所有页面：Geist 字体、tabular nums、severity 4px 左色条、sre-dot 圆点、hairline 边框、hover lift。

- **故障详情**（incidents/Detail.vue）：
  - Header 紧凑横条 + sre-row-card 副标题（圆点+severity+状态+空间+持续时间）
  - 操作栏分层：主操作（认领/关闭/重新打开）+ 横排次操作 + 右上 NDropdown 收纳
  - 三栏 Tab：Overview（紧凑 dl 网格 + tabular nums）/ Alerts (sre-row-card) / Timeline (竖向圆点+hairline 连接)/ Post-Mortem (md-editor-v3 dark)
  - 右栏 280px：KEY INFO + TIMELINE BRIEF
  - Snooze/Merge/Reassign 弹窗用 sre-row-card picker 行

- **协作空间详情**（channels/Detail.vue）：
  - Header 24px 700 标题 + 描述·团队副标
  - 4 张 KPI 卡片（Active/Today/MTTA/MTTR）+ 底部 tone 色条
  - 5 Tab：Incidents（sre-row-card）/ Overview / Noise / Dispatch / Settings
  - Settings Tab 含"危险区"删除卡片（红边二次确认）

- **告警 v2 列表 + 详情**（alerts-v2/）：
  - Index：sre-row-card + status segmented + severity/channel 筛选
  - Detail：sre-row-card subtitle + Tabs(Overview/Events) + 右栏 KEY INFO + LABELS

- **告警事件详情**（alerts/events/Detail.vue）：
  - 状态感知操作栏（firing→ack/resolve/close, acked→resolve/close/assign, etc）
  - 三 Tab：Overview（labels mono chips + annotations dl + rule card）/ Timeline（竖向 sre-dot 时间线 + 评论框）/ AI（报告+SOP 推荐）
  - 右栏 280px：Key Info + Responders + Labels + Related

- **集成中心**（integrations/Index.vue）：
  - n-data-table → 卡片网格（auto-fill minmax 320px）
  - Type+Mode 双层 segmented 筛选
  - 卡片：顶部 type 色条 + 状态点 + 标题 + type/mode badges + 描述 + webhook URL chip + 复制按钮 + 底部 alerts count + → 关联空间 + 操作行
  - 共享集成卡片显式"路由规则"按钮 → RoutingRules 抽屉

vue-tsc ✅

---

## [v2.2.0] - 2026-05-08

### Changed — UI 重构 Phase 1（FlashCat 全站对齐）

应用三个 skill：frontend-design (anthropics) / vue 3.5 (antfu) / web-design-guidelines (Vercel)

**字体与设计 tokens：**
- Geist + JetBrains Mono 通过 Google Fonts 全局引入（替代 system fonts，避免 AI slop）
- 新增设计 tokens：`--sre-stripe-w` `--sre-row-pad-y/x` `--sre-card-pad` `--sre-section-gap` `--sre-hairline`
- 新增 utility classes：`.sre-stagger`（错峰浮现）、`.sre-row-card[data-severity]`（4px 左色条卡片）、`.sre-dot[data-severity]`（圆点）、`.sre-meta-divider`、`.sre-stat-value`、`.sre-lift`、`.tnum`、`.sre-label-eyebrow`

**Phase 1 页面（4 个核心页）：**

- **主仪表盘**（dashboard/Index.vue 1158 → 536 行）：
  - 删除 GlowCard / AnimatedNumber / AuroraBackground（AI slop 视觉）
  - 4 张 KPI 卡片（Active / MTTA / MTTR / Resolved Today）+ 底部 3px tone 色条
  - Geist 字体 + 所有数字带 tabular-nums
  - 告警趋势 ECharts 折线图（280px 渐变填充，节制配色）
  - Top 噪音规则自定义列表 + 严重程度环形图
  - sre-stagger 首屏 KPI 错峰浮现

- **设置页**（settings/Index.vue 81 → 332 行）：
  - 顶部 Tabs → 240px 左导航 + 内容区
  - 三组 eyebrow label：PLATFORM / ORGANIZATION / AUDIT
  - 选中态 primary-soft 背景 + 2px 主色左 marker + 主色文字
  - URL hash 同步（#ai / #lark-bot 等）+ hashchange 监听
  - UserManagement v-show 保留以便跨 tab user list 共享

- **告警规则**（alerts/rules/Index.vue 941 → 1288 行）：
  - 抛弃 n-data-table，自定义 sre-row-card div 列表
  - 220px 左侧分类导航（active 态 primary-soft + 2px marker + tnum count）
  - 严重程度作 4px 左色条 + sre-dot 圆点替代 tag
  - 紧凑工具栏（search 240px + 3 selects 160px）
  - 浮现选择栏 + 批量启用/禁用/删除
  - 行 actions：启用 switch + 省略号下拉

- **活跃告警**（alerts/events/Index.vue 817 → 750 行）：
  - 抛弃 n-data-table，自定义 sre-row-card 列表
  - 严重程度 4px 左色条 + sre-dot
  - resolved/closed 行 data-dim 淡化 0.6
  - 4 行结构：headline / context / labels chips / footer 元数据
  - 状态分段：[全部 | Firing | Acked | Resolved]
  - 自动刷新（Off/30s/60s/5min 持久化）+ Export CSV

vue-tsc ✅

---

## [v2.1.0] - 2026-05-08

### Changed — UI 重构（FlashCat 风格）

- **数据查询页（/query）**：
  - 时间范围：横排预设按钮（5m/15m/30m/1h/3h/6h/12h/24h/2d/7d/30d）+ 自定义 datetimerange
  - 自动刷新：5s/10s/30s/1min/5min，倒计时显示
  - Step 选择器（metrics）：自动/15s/30s/1m/5m/15m/1h
  - Limit 选择器：metrics 50-1000，logs 50-5000
  - 查询历史 Popover（localStorage，最近 20 条）+ 清空按钮
  - CSV 导出（前端 Blob）：metrics table 模式 + logs 模式
  - 视觉：三段式卡片布局（工具栏 / 编辑器 / 结果）

- **主侧栏（MainLayout）**：
  - 扁平化：去除"告警管理"父级嵌套，子项铺平
  - 6 个 group 分组：概览 / 故障管理 / 告警引擎 / 集成与数据 / 通知与值班 / 系统
  - Group label 小字号 uppercase，FlashCat 风格
  - 选中态：左侧色条 + 浅色背景 + 主色文字
  - 折叠态隐藏 group label

- **故障列表（/incidents）**：
  - 弃用 n-data-table 改自定义卡片列表
  - 左侧 4px 严重程度色条
  - 三行结构：圆点+严重程度+#ID+标题 / 元数据 / 状态+处理人+时间
  - Hover 背景变化 + 箭头浮现
  - 已关闭行 0.72 opacity 淡化
  - 顶部分段控件：全部/我的，状态/严重程度筛选

- **协作空间列表（/channels）**：
  - 顶部 4px 主色色条
  - 三栏指标（活跃故障 / MTTA / MTTR）
  - 卡片右上角 Star（hover 显隐）
  - 右下角省略号下拉（编辑/删除）
  - Hover translateY(-2px) + 主色边框
  - 视图切换占位（卡片/列表）+ 排序下拉

- **i18n**：menu.* 补齐 6 个 v2 路由的中英键，incident.empty/duration/unassigned 等新增

---

## [v2.0.3] - 2026-05-08

### Fixed — 启动 panic（路由冲突）

- **Gin 路由参数冲突**：`/api/v1/integrations/:token/alerts`（webhook 接收）与 `/api/v1/integrations/:id/routing-rules`（路由规则 CRUD）共享前缀 `/integrations/:X`，但参数名不同，导致 Gin 启动 panic
- **修复**：将路由规则 API 改为扁平路径
  - `GET /api/v1/routing-rules?integration_id=X`（query string）
  - `POST /api/v1/routing-rules`（integration_id 在 body）
  - `PUT/DELETE /api/v1/routing-rules/:id`（不变）
- 前端 `routingRuleApi.listByIntegration` 和 `create` 同步更新

---

## [v2.0.2] - 2026-05-08

### Added — UI 缺口补齐

- **故障详情页 — 暂缓（Snooze）**：操作栏新增"暂缓"按钮，提供 5 个时长预设（15m/30m/1h/2h/4h）+ 自定义截止时间选择
- **故障详情页 — 合并（Merge）**：操作栏新增"合并故障"按钮，支持搜索目标故障并二次确认合并；合并后跳转目标故障
- **故障详情页 — 重新分派（Reassign）**：操作栏新增"重新分派"按钮，展示用户列表并支持实时过滤
- **故障复盘 — Markdown 编辑器**：PostMortem Tab 从纯 textarea 升级为 `md-editor-v3`，支持实时预览、语法高亮、工具栏
- **路由规则 CRUD**：
  - 后端：`RoutingRuleHandler`（List/Create/Update/Delete）+ 路由注册（`GET/POST /integrations/:id/routing-rules`，`PUT/DELETE /routing-rules/:id`）+ main.go wiring
  - 前端：`RoutingRules.vue` — 规则列表 + 优先级上下调整 + 启用开关 + 条件 JSON 编辑 + 目标空间选择
  - 集成中心：共享集成行新增"路由规则"按钮，点击弹出右侧抽屉展示 `RoutingRules.vue`

---

## [v2.0.1] - 2026-05-07

### Added — 告警规则批量操作

- **后端**:
  - `AlertRuleRepository.BatchUpdateStatus(ctx, ids, status)` — 批量更新状态，version 字段自增
  - `AlertRuleRepository.BatchDelete(ctx, ids)` — 批量软删除
  - `AlertRuleService.BatchEnable/BatchDisable/BatchDelete` — 参数校验 + 错误封装
  - `AlertRuleHandler.BatchEnable/BatchDisable/BatchDelete` — Gin handler + 审计日志
  - 路由: `POST /api/v1/alert-rules/batch/enable|disable|delete`（manage 权限）
- **前端**:
  - `alertRuleApi.batchEnable/batchDisable/batchDelete` — API 层
  - `pages/alerts/rules/Index.vue`: columns 添加 `{ type: 'selection' }` 复选列；`v-model:checked-row-keys` 多选；批量工具栏（选中数量显示 + 启用/禁用/删除按钮 + Popconfirm 二次确认）
  - i18n: zh-CN + en `common.selected`、`alert.batchEnabled/Disabled/Deleted/DeleteConfirm` 新增键
- go build ✅ | vue-tsc 无新增错误 ✅

---

## [v2.0.0] - 2026-05-07

### Release — v2.0 正式版

本版本为 SREAgent v2.0 正式发布版，包含 Phase 1-5 全部功能，以及发版收尾工作。

**版本升级路径**：从任意 v1.x 直接升级即可。部署新镜像后 `golang-migrate` 自动执行 000019-000033 共 15 个迁移。

#### 新增功能汇总
- **协作空间**（Channel）：故障聚合、降噪、分派、统计的核心单元
- **故障管理**（Incident）：完整生命周期 + 自动关闭 + 复盘
- **告警 v2**（Alert/AlertEventV2）：去重、关联、事件流水线
- **智能降噪**：聚合规则、风暴预警、抖动检测、排除规则、**快速静默**
- **分派策略**：触发条件、延迟窗口、重复通知、标签增强、升级绑定
- **Webhook 集成**：Standard/AlertManager/Grafana 三格式 + Pipeline + 限流
- **故障复盘**：Markdown 编辑器 + AI 生成初稿 + 发布
- **增强仪表盘**：按协作空间/团队维度的故障统计 + 趋势图

#### 收尾工作
- 版本号更新：CLAUDE.md v2.0.0 / web/package.json 2.0.0
- MODULES.md 更新：34 个模块 + v2 模块清单 + 迁移文件索引
- PLAN-status.md 修正所有遗漏项
- QuickSilenceModal：Incident Detail + Alert Detail 集成快速静默

---

## [v2.4.0-alpha.1] - 2026-05-07

### Added — Phase 5 故障复盘 + 分析增强

- **PostMortem CRUD** (`internal/service/post_mortem.go`):
  - `GetOrCreate`: 按 incident_id 查找或自动创建草稿（含 Markdown 模板预填充）
  - `Update` / `Publish`: 保存内容并可一键发布
  - `List`: 支持按 channel_id（JOIN incidents）和 status 过滤
  - `defaultPostMortemTemplate`: 预填充故障标题/时间/等级
- **AI 故障分析** (`internal/handler/post_mortem.go`):
  - `AIGenerate`: 调用 `AIService.AnalyzeAlertWithContext` → 生成 Markdown 复盘初稿并保存
  - `AISummary`: 返回 `AlertAnalysis` JSON 供前端预览（不保存）
  - `buildPostMortemFromAnalysis`: 将 AI 输出拼装为标准 Markdown 复盘格式
- **API 端点**:
  - `GET/PUT /api/v1/incidents/:id/post-mortem`
  - `POST /api/v1/incidents/:id/post-mortem/publish`
  - `POST /api/v1/incidents/:id/post-mortem/ai-generate`
  - `POST /api/v1/incidents/:id/post-mortem/ai-summary`
  - `GET /api/v1/post-mortems` (全局列表)
- **分析看板增强** (`internal/handler/dashboard.go`):
  - `IncidentStats`: 活跃故障数/今日关闭/紧急/Avg MTTR/复盘统计
  - `ChannelStats`: 按协作空间的故障分布（total/triggered/closed/critical）
  - `TeamStats`: 按团队的故障分布 + Avg MTTR（JOIN channels→teams）
  - `IncidentTrend`: 按日汇总 triggered+closed 趋势
- **前端**:
  - Incident Detail 新增"复盘"Tab：Markdown textarea 编辑器 + 保存/发布/AI 生成按钮 + SparklesOutline 图标
  - `incidentApi` 扩展：getPostMortem/updatePostMortem/publishPostMortem/aiGeneratePostMortem/aiSummaryPostMortem
  - `dashboardV2StatsApi`: incidentStats/channelStats/teamStats/incidentTrend
  - `IncidentDashboard.vue`: 5 张统计卡片 + 趋势柱状图（纯 CSS） + 空间/团队排行表
  - 侧边栏新增"故障看板"菜单（BarChartOutline 图标）
  - i18n: zh-CN + en `postMortem.*` + `dashboardV2.*` 新增键（合并至已有 dashboardV2 节）

---

## [v2.3.0-alpha.1] - 2026-05-07

### Added — Phase 4 告警引擎增强 + Webhook 接入

- **4.3 AlertRule → Channel 关联**: `AlertRule.channel_id` 字段 + 迁移 `000033`；rule_eval 注入 `_channel_id` 标签，AlertV2Pipeline 按规则优先路由到指定 channel
- **4.4-4.6 Webhook 接入** (`internal/service/integration.go`):
  - `IntegrationService.ReceiveAlerts`: 按 token 查找集成，限流检查，格式解析，pipeline 处理，路由到 AlertV2Pipeline
  - `normaliseStandard`: `{alerts:[...]}` 或单对象格式
  - `normaliseAlertManager`: `{alerts:[{status,labels,annotations,startsAt,...}]}`
  - `normaliseGrafana`: `{alerts:[{title,state,labels,...}]}`，state=alerting/ok/normal/no_data
- **4.7 处理管道**: `applyPipeline` — `rewrite_severity`/`rewrite_title`/`rewrite_description`/`drop`；条件匹配复用 `FilterCondition`；模板变量 `{{title}}/{{severity}}/{{labels.xxx}}`
- **4.8 频率限制**: per-integration 令牌桶（in-memory），100/s + 1000/min 双窗口
- **Integration CRUD API**: `GET/POST /api/v1/integrations` + `GET/PUT/DELETE /api/v1/integrations/:id`
- **Webhook 接收端点**: `POST /api/v1/integrations/:token/alerts`（无 JWT，token 鉴权）
- **4.1 NoData**: 引擎已有实现（`NoDataEnabled`/`NoDataDuration`）
- **4.2 规则文件夹**: `AlertRule.category` 已支持，`listCategories` API 已有
- **前端**:
  - `pages/integrations/Index.vue`: CRUD 表格 + Webhook URL/Token 展示与复制 + 创建/编辑弹窗（type/mode/channel/pipeline/label 增强）
  - 侧边栏新增"集成中心"菜单项（GitNetworkOutline 图标）
  - `integrationV2Api` API 层
  - i18n: zh-CN + en `integration.*` / `ruleFolder.*` 键
- **DB 迁移 000033** `alert_rules.channel_id`

---

## [v2.2.0-alpha.1] - 2026-05-07

### Added — Phase 3 分派增强

- **DispatchPolicy 模型** (`internal/model/dispatch.go`):
  - Channel 绑定、多策略优先级、启用开关
  - 触发条件 `match_conditions` (JSON `FilterCondition[]`) + 生效时间段 `active_time_config` (时区/星期/时间段)
  - 延迟窗口 `delay_seconds` (0-3600)
  - 重复通知 `repeat_interval_seconds` + `max_repeats`
  - 通知方式 `notify_mode` (personal_preference | unified) + `unified_media_id`
  - 升级策略绑定 `escalation_policy_id`
  - 标签增强规则 `label_enhancement_rules` (JSON `LabelEnhancementAction[]`)
- **DispatchLog 模型** — 记录每次分派尝试状态
- **DispatchService** (`internal/service/dispatch.go`):
  - `FindMatchingPolicy`: 按优先级匹配第一个满足条件+时间窗口的策略
  - `ApplyLabelEnhancements`: set/extract(regex)/combine(template)/map(lookup)/delete 五种操作
  - `matchConditions` + `isActiveNow`: 复用 `FilterCondition` 匹配逻辑
- **AlertV2Pipeline 集成**: `SetDispatchService` → `process()` 在 upsert 前执行标签增强
- **API**: `GET/POST /api/v1/channels/:id/dispatch-policies` + `GET/PUT/DELETE /api/v1/dispatch-policies/:id`
- **DB 迁移 000031** `dispatch_policies` + **000032** `dispatch_logs`
- **前端 DispatchConfig.vue**: 策略列表 + 优先级上下移动 + 创建/编辑弹窗（全字段覆盖）
- Channel Detail 新增"分派配置" Tab
- i18n: zh-CN + en `channel.dispatch*` 全量键

---

## [v2.1.0-alpha.1] - 2026-05-07

### Added — Phase 2 智能降噪

- **NoiseReducer** (`internal/service/noise_reducer.go`): 降噪核心引擎
  - 排除规则：`matchAllConditions` 支持 eq/ne/contains/not_contains/regex/in/not_in
  - 聚合键计算：统一维度 / 细粒度条件分支，strictMode 控制空值处理
  - 风暴预警：滚动 1 分钟窗口计数，每阈值只触发一次告警
  - 抖动检测：in-memory flapStates，支持 off / notify_only / notify_then_silence 三种模式
- **AlertV2Pipeline 集成**：`SetNoiseReducer` + `process()` 在 upsert 前执行降噪，excluded→drop，silenced→跳过故障创建
- **ExclusionRuleRepository + Service + Handler**：`/api/v1/channels/:id/exclusion-rules` CRUD
- **前端 NoiseConfig.vue**：协作空间详情页新增"降噪配置" Tab，覆盖聚合规则/窗口/风暴预警/抖动检测/排除规则
- **i18n**：zh-CN + en 新增 `channel.noise*` / `channel.flapping*` / `channel.exclusion*` 全量键

---

## [v2.0.0-alpha.1] - 2026-05-07

### Added — Phase 1.1 核心模型重构
- **协作空间 (Channel)**：`model/channel.go` + repository + service + handler + API (`/api/v1/channels`)
  - CRUD + Star/Unstar 收藏 + 列表带收藏标记
  - 降噪配置（聚合规则/抖动检测）、自动关闭配置
- **故障 (Incident)**：`model/incident.go` + repository + service + handler + API (`/api/v1/incidents`)
  - 完整操作：acknowledge / close / reopen / snooze / merge / reassign / escalate / comment
  - 时间线 (IncidentTimeline) 自动记录所有操作
  - 分派人跟踪 (IncidentAssignee)
  - 复盘报告 (PostMortem) 模型
- **告警 v2 (Alert + AlertEventV2)**：`model/alert.go` + repository + service + handler + API (`/api/v1/alerts`)
  - Alert: 按 alert_key 去重的告警序列，关联 Channel + Incident
  - AlertEventV2: 原始事件数据（firing/resolved），按时间戳记录
  - UpsertFromEvent: 核心摄入路径，支持自动去重+合入
- **集成 (Integration) + 路由规则 (RoutingRule)**：模型已定义（repo/service/handler 待 Phase 4 实现）
- **DB 迁移 000019-000030**：
  - 000019: channels
  - 000020: channel_stars
  - 000021: channel_exclusion_rules
  - 000022: incidents
  - 000023: incident_assignees
  - 000024: incident_timelines
  - 000025: post_mortems
  - 000026: alerts
  - 000027: alert_events_v2
  - 000028: integrations
  - 000029: routing_rules
  - 000030: seed default channel

### Fixed
- **Settings 菜单点击无反应**：Naive UI n-menu 当 `:value` 等于点击项 key 时不触发 `@update:value`，改用 ref + 点击前清空解决

---

## [v1.16.23] - 2026-05-06

### Fixed
- **彻底消除 vue-i18n runtime SyntaxError**：移除所有 i18n 消息中的花括号示例文本（PromQL、JSON 示例），改用不含花括号的纯文字描述
  - `{'{'}` 转义在 vue-i18n v11 production JIT 编译模式下仍会触发 `EXPECTED_TOKEN` 错误
  - 涉及 14 条消息（zh-CN 7 条 + en 7 条）：datasource/explore/query placeholder、notifyRule hints、OIDC mapping、Lark bot hint

## [v1.16.22] - 2026-05-06

### Fixed
- `query.promqlPlaceholder` 中 `{instance=~"prod.*"}` 未转义花括号，vue-i18n v11 message-compiler 报 SyntaxError（与 v1.16.18 同类问题）

## [v1.16.21] - 2026-05-06

### Fixed
- 指标查询选择数据源后查询输入框不显示：移除 PromQLEditor 异步组件，改用稳定的 NInput textarea
- `v-if="selectedDsId"` 改为 `v-if="selectedDsId != null"` 显式空值检查

## [v1.16.20] - 2026-05-06

### Changed
- **「数据探索」重命名为「数据查询」(Data Query)**：路由 `/explore` → `/query`，保留旧路由重定向兼容
- **数据查询页面完全重写**，修复长期白屏问题：
  - 根因：`@codemirror/view`、`@codemirror/state`、`@codemirror/commands` 未声明为直接依赖，Rollup 打包解析失败
  - 新增 Tab 切换：「指标查询」(Prometheus/VM/Zabbix) + 「日志查询」(VictoriaLogs)
  - ECharts 改为懒加载 (dynamic import)：加载失败不阻塞页面，自动降级到表格模式
  - PromQLEditor (CodeMirror) 改为 defineAsyncComponent + 5s 超时：加载失败回退到 NInput textarea
  - 数据源按类型分组到对应 Tab，而非混合在一个下拉框
  - 新增 `query.*` i18n key set (中英双语)

### Fixed
- 安装缺失的 `@codemirror/view`、`@codemirror/state`、`@codemirror/commands` 包至 package.json
- 修复 vite build 因 CodeMirror 子包缺失导致的 Rollup resolve 错误

## [v1.16.19] - 2026-04-30

### Changed
- **Explore 页面 UI 重写**：使用 Naive UI 组件替代纯 HTML 元素
  - PromQLEditor（CodeMirror 6 + PromQL 语法高亮）用于指标数据源
  - 日志数据源使用简洁的 textarea + 等宽字体
  - ECharts 时序图表 + DataTable 表格切换
  - 数据源选择器显示类型标签（Prometheus/VM/VLogs/Zabbix）和版本号
  - 自动根据数据源类型切换查询模式（指标/日志）

## [v1.16.18] - 2026-04-30

### Fixed
- **真正的根因修复**：vue-i18n v11 的 message-compiler 对 `{` 字面量比 v9 更严格，i18n 消息中的 PromQL 示例 `{mode="idle"}` 和 JSON 示例 `[{"type":"aggregate"}]` 被错误解析为占位符，导致 `INVALID_TOKEN_IN_PLACEHOLDER` SyntaxError
- 修复 6 处 i18n 消息（zh-CN + en），使用 `{'{'}` / `{'}'}` 转义字面量花括号
- 恢复 esbuild 压缩器（terser 未生效，已移除）

### Changed
- `vite.config.ts`: 恢复 `minify: 'esbuild'`（误判，问题不在压缩器）

## [v1.16.17] - 2026-04-30

### Fixed (未生效)
- 尝试切换到 terser 压缩器，但错误依旧 — 证明问题不在压缩器，在于 i18n 消息内容

### Changed
- `vite.config.ts`: `minify: 'esbuild'` → `minify: 'terser'`（后被 v1.16.18 回滚）

## [v1.16.16] - 2026-04-30

### Fixed
- **DataView Symbol.toStringTag 报错**：lodash（Naive UI 内置）`getRawTag()` 在 ES module strict mode 下尝试覆写只读的 `DataView.prototype[Symbol.toStringTag]`，导致 "Cannot assign to read only property" TypeError
- 新增 `dataview-polyfill.ts`，在 main.ts 最开始执行，将 DataView 的 Symbol.toStringTag 属性设为 writable

## [v1.16.15] - 2026-04-30

### Fixed
- **真正的根因修复**：升级 vue-i18n 9.14.0 → 11.4.0，`@intlify/message-compiler` 新版修复了 esbuild 压缩产生的 `Unterminated closing brace` SyntaxError
- 恢复 esbuild 压缩器（`minify: 'esbuild'`），移除 terser 依赖

### Changed
- vue-i18n 升级到最新版 11.4.0（兼容，typecheck 通过）

## [v1.16.14] - 2026-04-30

### Fixed (未生效)
- 尝试修复：切换到 terser 压缩器避开 `@intlify/message-compiler` esbuild 压缩 bug — 但引入了 "Cannot assign to read only property" 新问题
- Explore 页面简化为纯 HTML 元素，移除 Naive UI 组件依赖

## [v1.16.13] - 2026-04-30

### Debug
- Explore 页面移除 TimeRangePicker/RefreshPicker 依赖，用纯文本替代 — 隔离 DatePicker 是否为白屏根因

## [v1.16.12] - 2026-04-30

### Fixed
- Explore 页面 `row-key` 类型错误 — 单参数函数匹配 `CreateRowKey<any>` 签名
- Explore 页面日志数据添加 `_key` 索引

### Debug
- Explore 页面添加 `onErrorCaptured` 错误边界 + console 诊断日志，定位生产白屏根因

## [v1.16.11] - 2026-04-29

### Changed
- 重写 Explore 页面：移除 ECharts/vue-echarts 依赖，消除生产环境白屏。列 render 函数只返回纯字符串（不再用 `h()` 返回 VNode 数组），所有 Naive UI 组件显式导入 + PascalCase 模板用法

## [v1.16.10] - 2026-04-29

### Fixed
- 修复 Explore 页面生产环境白屏：移除未使用的 `shallowRef` 导入、模板内联 `.map()` 改为 computed `datasourceOptions`、全链路空值防御（`s.labels || {}`、`s.values || []`、`v.value ?? 0`、`Array.isArray` 守卫）

## [v1.16.9] - 2026-04-29

### Added
- P0-P4 严重级别支持：model 常量、前端类型、i18n 标签（P0-紧急/P1-严重/P2-一般/P3-轻微/P4-信息）、表单和过滤器选项
- `/metrics` 端点：Prometheus 暴露格式的 Go 运行时 + 应用指标
- PanelCard 新增 gauge/bar/pie 图表类型（ECharts GaugeChart + BarChart + PieChart）
- Dashboard V2 面板拖拽布局：拖拽标题栏移动面板位置 + 右下角拖拽调整面板尺寸（CSS Grid 24 列）
- Dashboard V2 面板类型扩展按钮：统计值/时序图/仪表盘/柱状图/饼图/表格
- 告警规则模板系统：CRUD + 分类 + "从模板加载"/"保存为模板"（前后端完整实现）
  - Model: `alert_rule_templates` 表（迁移: 000018_alert_rule_templates）
  - API: GET/POST/PUT/DELETE `/api/v1/alert-rule-templates` + `/categories` + `/:id/apply`
  - 前端：创建规则时可从模板加载，编辑时可保存为模板

### Changed
- Alert Detail 页面硬编码颜色全部替换为 CSS 自定义属性（banner、timeline、lifecycle、labels、annotations、responders）
- PanelCard Stat 面板支持阈值颜色：`panel.options.thresholds` 数组 `[{ value, color }]` 自动根据当前值匹配颜色

### Fixed
- PromQLEditor 防御性错误处理：onMounted 和 datasourceId watcher 中 EditorState.create 增加 try-catch

---

## [v1.16.8] - 2026-04-29

### Changed
- Alert Detail 页面硬编码颜色全部替换为 CSS 自定义属性（banner、timeline、lifecycle、labels、annotations、responders）
- PanelCard Stat 面板支持阈值颜色：`panel.options.thresholds` 数组 `[{ value, color }]` 自动根据当前值匹配颜色

## [v1.16.7] - 2026-04-29

### Removed
- 移除可编程告警处理链 (Event Pipeline) 功能：前端页面/路由/菜单/i18n、后端 handler/service/repository/model/engine 全部删除
- 从 onAlertFn 移除 Pipeline 拦截点，简化告警处理流程为: inhibition → mute → bizgroup → group → notify

### Fixed
- 恢复 6 个被误删的 i18n key（addQuery/runQueries/queryLabel/toggleOn/toggleOff/legendFormat），修复 Dashboard V2 查询组件显示原始 key 字符串
- Dashboard V2 列表页完整国际化 + 操作按钮（查看/编辑/删除）
- 补全英文 i18n 缺失的 dashboardV2 段
- Dashboard V2 面板网格渲染：CSS Grid 布局 + PanelCard 组件（支持 timeseries/stat/table 三种面板）
- Dashboard V2 硬编码颜色全部替换为 CSS 自定义属性，适配暗色模式

## [v1.16.4] - 2026-04-29

### Security
- P0-1: Webhook 端点增加共享密钥认证中间件 (`X-Webhook-Secret` header, constant-time compare)
- P0-2: 引入有界 goroutine 池 (`AlertWorkerPool`, 默认 64 并发)，防止告警风暴导致 goroutine 耗尽
- P0-2: `RuleEvaluator.createAlertEvent`/`resolveAlertEvent` 改用 worker pool 替代裸 `go func()`
- P0-2: `AlertEventService.processAlert`/`triggerLarkCardUpdate` 改用 worker pool
- P0-3: 修复优雅关闭顺序 (evaluator → heartbeat → groupMgr → escalation → pool.Wait() → HTTP → Redis)

### Changed
- **数据探索页面重写**: 移除复杂多目标 Grafana 风格 UI，改为简单交互：选数据源→自动匹配查询引擎→输入表达式→执行
- 自动根据数据源类型调整查询占位提示 (PromQL / LogsQL / Zabbix key)
- 查询结果图表自动适配 vector/matrix 类型
- **处理链页面完善**: 100% 国际化覆盖 (40+ i18n key)，列表页增加功能介绍说明，编辑器增加使用指南
- 处理链空状态增加引导文案和新建按钮
- 处理器节点增加 tooltip 功能描述
- 清理 `explore` i18n 中的无用 key (`addQuery`, `runQueries`, `legendFormat`, `toggleOn`, `toggleOff`, `queryLabel`)

### Added
- `internal/middleware/webhook_auth.go` — Webhook 共享密钥认证中间件
- `internal/engine/workerpool.go` — 有界 goroutine 池 (semaphore + WaitGroup)
- `config.Server.WebhookSecret` 配置项
- pipeline i18n keys (zh-CN + en): title/subtitle/create/edit/noData/noDataHint/processors/filters/editorTitle/configureNode/proc*Desc 等 40+ 键
- explore i18n keys: promqlPlaceholder/zabbixPlaceholder/metricName/value/labelsHeader

### Added
- 侧栏新增「处理链」菜单项，Pipeline 页面入口
- i18n：menu.pipelines、explore.toggleOn/Off、common.loadFailed/updateSuccess/createSuccess/confirmDelete/filters/responders 等键值
- i18n：alert.datasourceType/datasourceRequired/selectDatasourceType 键值
- docs/n9e-gap-analysis.md — n9e 功能差距分析 + 三阶段实施路线图

### Fixed
- 修复 QueryRow/QueryPanel/Explore 页硬编码颜色 → CSS 自定义属性
- 修复 A/H 切换按钮未国际化
- 修复 resolveActiveKey/pageTitle 缺失 pipelines/schedule 路由匹配
- 修复 Inhibition 页面使用不存在的 i18n 键（显示原始 key 字符串）
- 修复 Alert Rules 页面缺少 i18n 的 datasourceType 相关键
- 修复路由守卫 role 检查优先使用 Pinia Store 而非 localStorage
- 修复迁移 000006 down.sql 错误删除未创建的索引
- 修复 MODULES.md 指向不存在的 docs/alert-engine.md 和 docs/notification.md

### Removed
- 移除未使用的 mutePreviewApi、heartbeatApi 前端 API 定义
- 移除未使用的 DocumentTextOutline/GridOutline 导入
- 移除未使用的 type Labels (model/base.go)
- 移除未使用的 useScrollReveal.ts、usePromQLCompletion.ts composables
- 移除未使用的 magnetic 指令 + 注册
- 移除未引用的 datasources/Query.vue 页面（路由已重定向到 /explore）

## [v1.16.2] - 2026-04-29

### Changed
- 简化 Explore 页面布局：数据源选择器移至顶栏，移除 QueryRow 内重复选择器
- 数据源切换自动同步到所有查询目标
- 完善 i18n 国际化

## [v1.16.1] - 2026-04-29

### Changed
- 统一数据探索页面（Explore）：合并 PromQL Explore 和 LogExplorer，根据数据源类型自动切换指标/日志模式
- 侧栏新增顶级「探索」菜单，旧路由 `/datasources/query` 和 `/explore/logs` 自动重定向
- 删除独立的 `LogExplorer.vue`

## [v1.16.0] - 2026-04-29

### Added
- 统一数据探索页面（Explore）：根据数据源类型自动切换指标/日志查询模式
- Prometheus/VM 数据源 → PromQL 编辑器 + 时序图表/表格
- VictoriaLogs 数据源 → LogsQL 查询 + 日志条目表格
- 侧栏新增顶级「探索」菜单入口
- 旧路由 `/datasources/query` 和 `/explore/logs` 自动重定向到 `/explore`
- VictoriaLogs 日志查询端点：`POST /api/v1/datasources/:id/log-query`
- 中英文 i18n 支持（所有错误提示和 UI 文本）

### Fixed
- 修复数据查询页白屏：`crypto.randomUUID` 在 HTTP 非安全上下文下不可用，改用 fallback UUID 生成
- 修复登录页 401 错误显示英文：拦截器现在优先使用后端返回的业务错误码进行本地化（如 10102 → "用户名或密码错误"）

### Removed
- 删除独立的 LogExplorer.vue（合并到统一 Explore 页面）

## [v1.15.0] - 2026-04-29

### Added
- 可编程告警处理链（Event Pipeline）：DAG 可视化编辑器 + 5 种处理器
- 处理器：If（条件分支）、Relabel（标签操作）、EventDrop（告警丢弃）、Callback（Webhook 回调）、AISummary（AI 摘要）
- Pipeline CRUD 端点：`/api/v1/event-pipelines`（7 个端点）
- Pipeline 试运行：`POST /api/v1/event-pipelines/tryrun`
- Pipeline 执行记录：`GET /api/v1/event-pipelines/:id/executions`
- 前端 Pipeline 列表页 + DAG 编辑器（原生 SVG + 拖拽连线）
- 前端节点配置面板（右侧抽屉，支持各处理器类型专属配置）
- Pipeline 引擎集成到 onAlertFn（inhibition → mute → bizgroup → **pipeline** → notify）
- 迁移: 000017_event_pipelines

## [v1.14.0] - 2026-04-29

### Added
- 数据源探索页面（Explore）：PromQL 编辑器（CodeMirror 6 + 语法高亮 + 自动补全）
- Range Query 支持：POST /api/v1/datasources/:id/query-range
- 数据源标签代理端点：GET /api/v1/datasources/:id/labels/keys、labels/values、metrics
- ECharts 时间序列图表（dataZoom、tooltip cross 指针、Legend 统计表格）
- 时间范围选择器（相对/绝对时间）+ 自动刷新
- 多查询支持、Legend 格式化、Chart/Table 视图切换
- 仪表盘 V2 系统：Dashboard CRUD 端点（/api/v1/dashboards）
- 变量模板系统：query/custom/textbox/constant 类型，$var 替换
- 仪表盘列表页和查看页（全局时间范围、变量选择器）
- 值格式化工具（bytes/seconds/percent/short/scientific）
- 迁移: 000016_dashboards

### Changed
- /datasources/query 路由指向新的 Explore 页面（替代原生 HTML 查询页）

## [v1.11.0] - 2026-04-27

### Added
- 登录页密码/用户名错误 inline 提示（表单内 alert 替代右上角 message）
- 数据源卡片显示版本号（健康检查成功后持久化 version 字段）
- 数据源状态标签国际化（healthy/unhealthy/unknown 随语言切换）
- 密码复杂度校验（最少 8 位，含大小写字母和数字）
- JWT 超时可配置（系统设置 > 安全配置，预设 1h/4h/8h/24h/7d）
- 数据源查询页面（选择数据源 + 输入 PromQL/LogQL 执行查询）
- 迁移: 000015_datasource_version

### Removed
- 登录页默认账号 admin/admin123 提示

### Changed
- AuthService.Login / RefreshToken 读取 system_settings 中的 jwt_expire_seconds
- handler/auth.go, handler/user.go 密码最小长度约束从 6 提升至 8

## [v1.10.0] - 2026-04-26

### Added
- 测试框架：internal/testutil/ (TestDB, SeedUser, SeedAlertRule, CleanupDB)
- 测试骨架：service/alert_channel_test.go, handler/alert_channel_test.go
- docs/testing.md 测试策略和覆盖目标
- docs/prompts.md AI 提示词模板（新功能/Bug/审查/测试等）
- CLAUDE.md 对话规范（token 节省规则）
- config.example.yaml OIDC 配置段
- GET /schedules/:id/participants 后端 handler + 路由
- GET /schedules/:id/overrides 后端 handler + 路由
- POST /alert-channels/:id/test 后端 handler + 路由

### Fixed
- 修复 3 个前端 API 调用无后端路由的问题（schedule participants/overrides, alert-channel test）

### Removed
- 4 个孤立 Vue 组件（SpotlightCursor, SeverityTag, StatusTag, SkeletonCard）
- 废弃 TS 类型（NotifyChannel, NotifyPolicy v1）
- 无关文档（3th_monitor_readme.md）
- scripts/test-api.sh 中的 v1 通知端点

## [v1.9.10] - 2026-04-26

### Fixed
- label_registry.label_value 从 VARCHAR(512) 扩展到 VARCHAR(2048)，修复 Prometheus 长标签值导致 MySQL Error 1406
- SyncDatasource / RecordFromLabels 添加 >2048 截断安全网
- **迁移**: 000014_label_value_extend

## [v1.9.9] - 2026-04-26

### Added
- Alertmanager 风格 group_wait / group_interval 通知分组
- AlertGroupManager 在引擎回调和 RouteAlert 之间缓冲 firing 事件
- AlertRule 新增 group_wait_seconds / group_interval_seconds 字段
- 前端告警规则表单新增分组等待/间隔配置
- **迁移**: 000013_alert_rule_group_timing

## [v1.9.8] - 2026-04-25

### Added
- CLAUDE.md 与 .opencode/context.md 合并为单一 AI 导航文件
- .gitignore 添加 .claude/ 和 .opencode/ 排除
- Claude Code 全局 settings.json 权限配置

## [v1.6.0] - 2026-04-20

### Added
- 系统级 SMTP 配置（system_settings group=smtp）
- 升级策略 email 分支接入系统 SMTP 真实发送
- JWT 7天宽限续签（POST /auth/refresh）
- 前端 Axios 401 自动刷新 token
- 头像 Go 层大小校验（≤272KB data URL）
- GET /alert-events/export CSV 流式导出
- GET /mute-rules/preview 命中预览
- Lark OpenID → DB User 映射（user.lark_user_id）
- 个人设置新增「飞书账号绑定」tab
- 数据源健康检查返回 latency/version 富结果
- **迁移**: 000008_create_inhibition_rules, 000009_create_label_registry, 000010_alert_rule_datasource_optional, 000011_alert_rule_datasource_type

## [v1.5.0] - 2026-04-15

### Added
- 升级策略 lark_personal 分支接入 Lark Bot API（DM）
- 告警 AutoResolve 时同步 PATCH Lark 卡片
- LarkBotService.SendMessage 优先用 Bot API 回复 chatID
- NotifyChannel Bot API 类型在 TestChannel 支持真发送
- **迁移**: 000006_heartbeat_sla_alert_rules, 000007_sla_escalated_at_alert_events

## [v1.3.1] - 2026-04-10

### Added
- MTTA/MTTR P50/P95 百分位、按严重程度细分
- MTTA/MTTR 每日趋势折线图
- 品牌 logo.svg（sider/login/favicon 统一）
- 个人信息头像扩展为 32 个预设 emoji + 自定义上传

### Fixed
- 顶部栏保存头像后仍显示用户名首字母

## [v1.3.0] - 2026-04-08

### Changed
- 设计系统级视觉翻新：CSS token + Naive UI GlobalThemeOverrides
- 侧栏/顶栏/登录页玻璃态皮肤（dark + light）

## [v1.2.0] - 2026-04-05

### Added
- 告警规则分类 tab
- 仪表盘分析图表（趋势 + Top 规则）
- 操作审计日志
- 表达式实时测试
- **迁移**: 000004_audit_logs, 000005_add_rule_category

## [v1.1.x] - 2026-04-01

### Added
- 告警详情页改版（严重等级横幅 + 生命周期时间线）
- 通知模块合并为单页 Tabs
- **迁移**: 000003_alert_event_lark_message_id

## [v1.0.x] - 2026-03-25

### Added
- OIDC 配置 UI（存 DB）
- K8s 清单
- 多数据源集成
- RBAC 三级权限
- **迁移**: 000001_initial_schema, 000002_system_settings

> Phase 追踪和 QA 修复汇总已移至 [docs/phases.md](docs/phases.md)
