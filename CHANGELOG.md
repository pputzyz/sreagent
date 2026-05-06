# 变更日志 (CHANGELOG)

> 基于 git tag 和 commit 记录整理。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)

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
