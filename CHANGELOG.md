# 变更日志 (CHANGELOG)

> 基于 git tag 和 commit 记录整理。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)

---

## [v4.15.15] — 2026-05-22

### 审查遗漏修复（核对后补充）

**安全修复**
- `internal/pkg/crypto/crypto.go`：EncryptString 无 key 时返回错误（原先静默返回明文）

**错误处理**
- `internal/service/alert_event.go`：Acknowledge/Resolve/Close 三处 GetByID 错误不再吞没，改为 warn 日志

**前端修复**
- `web/src/utils/format.ts`：relTime() i18n key 从 `events.secsAgo` 修正为 `alert.secsAgo`

**Context 传播**
- `internal/service/dashboard_stats.go`：11 个方法新增 `ctx context.Context` 参数，使用请求级 context
- `internal/handler/dashboard.go`：11 处调用传递 `c.Request.Context()`

---

## [v4.15.14] — 2026-05-22

### Round 10 Medium 级别问题全量修复（6 agent 并行）

**安全加固**
- `deploy/kubernetes/app/secret.yaml`：真实密码替换为 `CHANGE_ME` 占位符
- `deploy/kubernetes/mysql/secret.yaml`：真实密码替换为 `CHANGE_ME` 占位符
- `deploy/docker/entrypoint.sh`：DB 密码改用 `MYSQL_PWD` 环境变量传递，避免暴露在 `/proc/cmdline`
- `deploy/kubernetes/app/deployment.yaml`：移除硬编码版本号，改用 Helm values

**审计日志扩展（9 个 handler）**
- `model/audit_log.go`：新增 Schedule/EscalationPolicy/MuteRule/InhibitionRule/NotifyMedia/BizGroup/Channel/RoutingRule 资源类型常量
- `handler/schedule.go`：新增 auditSvc + 6 个写操作审计（Schedule + EscalationPolicy）
- `handler/mute_rule.go`：新增 auditSvc + 3 个写操作审计
- `handler/inhibition_rule.go`：新增 auditSvc + 3 个写操作审计
- `handler/notify_rule.go`：新增 auditSvc + 3 个写操作审计
- `handler/notify_media.go`：新增 auditSvc + 3 个写操作审计
- `handler/biz_group.go`：新增 auditSvc + 3 个写操作审计
- `handler/channel.go`：新增 auditSvc + 3 个写操作审计
- `handler/routing_rule.go`：新增 auditSvc + 3 个写操作审计
- `cmd/server/wire.go`：注入所有新增 handler 的 SetAuditService

**迁移文件健壮性**
- 21 个 `.up.sql` 文件添加 `IF NOT EXISTS`（CREATE TABLE + CREATE INDEX）

**错误码对齐**
- `internal/pkg/errors/errors.go`：新增 `ErrBusiness`(10002) 和 `ErrUnauth`(40001)，与 CLAUDE.md 文档一致

---

## [v4.15.13] — 2026-05-22

### 全项目多视角深度审查修复

**Critical 修复**
- `cmd/server/wire.go`：Leader election 使用 `redisClient` 替代 `d.RedisClient`（赋值前检查导致永远不激活）
- `internal/handler/inspection.go`：异步执行使用 `context.Background()` 替代请求 context（避免请求结束后 context 被取消）
- `internal/engine/heartbeat_checker.go`：拆分 `sync.Once` 为 `startOnce`/`stopOnce`，防止 Start/Stop 互相干扰
- `internal/handler/datasource.go`：Create 时尊重 `req.IsEnabled` 字段（原先硬编码 `true`）
- `internal/pkg/crypto/crypto.go`：DecryptString 遇到 `enc:` 前缀但无 key 时返回错误（原先静默返回原始值）

**High 修复**
- `internal/handler/auth.go`：GetProfile/UpdateMe/BindLark/ChangeMyPassword 使用 `GetCurrentUserIDOK` 检查认证状态
- `internal/repository/schedule.go`：`HasExecuted` 返回 `(bool, error)` 不再吞没 DB 错误
- `internal/engine/escalation_executor.go`：适配 HasExecuted 新签名，错误时记录日志并跳过
- `internal/model/change_event.go`：嵌入 `BaseModel`，移除冗余 ID/CreatedAt/DeletedAt 字段

**前端修复**
- `web/src/utils/format.ts`：`relTime()` i18n key 从 `common.secsAgo` 修正为 `events.secsAgo`
- `web/src/components/common/CronInput.vue`：全部硬编码中文改为 i18n（`cronInput.*` 命名空间）
- `web/src/composables/useVariable.ts`：`replaceVariables` 对变量名做 regex 转义，防止注入
- `web/src/composables/usePermissions.ts`：`loadPermissions` 失败时也设置 `loaded = true`
- `web/src/stores/preferences.ts`：`update` 失败时向上抛出错误（不再静默吞没）

**CI 修复**
- `.github/workflows/docker-build.yml`：Node 版本从 24 改为 20（与 Dockerfile 一致，24 非 LTS）

---

## [v4.15.12] — 2026-05-22

### 前端测试框架 + Service 接口抽象 + 数据源审计日志

**前端测试**
- 安装 vitest + @vue/test-utils + jsdom
- `vite.config.ts` 添加 test 配置（jsdom 环境）
- `utils/format.test.ts`：formatDuration、kvArrayToRecord、recordToKVArray、getErrorMessage 测试
- `utils/severity.test.ts`：severityType 测试
- `composables/useFilterMemory.test.ts`：restore/save/clear/隔离 测试
- `package.json`：添加 test/test:watch 脚本

**Service 接口抽象**
- `service/alert_rule.go`：新增 `AlertRuleOperator` 接口，编译期断言 `*AlertRuleService` 实现
- `service/datasource.go`：新增 `DataSourceQuerier` 接口，编译期断言 `*DataSourceService` 实现
- `service/ai_tools.go`：RegisterBuiltinTools 参数改为接口类型
- `service/rule_generator.go`：dsSvc/ruleSvc 字段改为接口类型
- `service/diagnostic_workflow.go`：dsSvc 字段改为接口类型

**数据源审计日志**
- `handler/datasource.go`：新增 auditSvc 字段 + SetAuditService，Create/Update/Delete 后记录审计日志
- `cmd/server/wire.go`：注入 DataSource.SetAuditService

---

## [v4.15.11] — 2026-05-22

### Round 10 多视角审查 — 安全 + 健壮性 + CI + i18n + 服务端过滤

**安全加固**
- `handler/datasource.go`：数据源 endpoint URL scheme 白名单校验（仅 http/https），阻断 SSRF
- `stores/auth.ts`：401 检测从字符串匹配改为 HTTP status code 检查，避免误判

**引擎层修复**
- `engine/escalation_executor.go`：Start/Stop 拆分为独立 sync.Once，修复 Stop() 永远是 no-op 的 bug

**健壮性**
- `service/dashboard_stats.go`：WaitGroup+裸 goroutine → errgroup+panic 恢复，错误可传播
- `model/auto_action.go`：AutoAction 标注为未接入状态（模型保留，待后续实现）

**CI/CD**
- `.github/workflows/docker-build.yml`：添加 PR 触发 + main 分支 push 触发 + `-race` 标志 + golangci-lint

**服务端过滤**
- `repository/alert_rule.go`：List() 新增 keyword/datasourceID 参数，支持 LIKE 搜索和精确匹配
- `service/alert_rule.go`：List() 透传新参数
- `handler/alert_rule.go`：List() 读取 keyword/datasource_id query params
- `web/pages/alerts/rules/Index.vue`：移除客户端 filteredRules computed，改为 extraParams 服务端过滤

**i18n**
- `i18n/zh-CN.ts` + `i18n/en.ts`：新增 inspection 命名空间（55 个键）
- `pages/platform/inspections/Index.vue`：全部中文硬编码替换为 t() 调用
- `pages/platform/inspections/RunDetail.vue`：全部中文硬编码替换为 t() 调用 + 修复缺失的 h 导入

---

## [v4.15.10] — 2026-05-21

### Round 9 审查补漏

- `rule_eval.go`：break→continue+Unlock 修复 createAlertEvent 失败时 mutex 泄漏（死锁风险）
- `leader_election.go`：renew() 使用包级 checkAndExtendScript 替代内联 Lua
- `escalation_executor.go`：Start() 用 sync.Once 防止重复启动
- `handler/integration.go`：io.LimitReader→http.MaxBytesReader 超限返回 413
- `rule_generator_suggest.go`：suggestLabelsHeuristic 传播 ctx 替代 context.Background()
- `handler/inspection.go`：ListTasks 统一使用 SuccessPage() 响应格式

---

## [v4.15.9] — 2026-05-21

### 框架级审查 Round 9 — 引擎 Bug + 安全加固 + 分页统一 + BaseModel 嵌入

**引擎层修复**
- `evaluator_cache.go`：collectAllEvaluators() 修复 perDS 模式下 GetFiringEvents/GetStatus 遗漏规则
- `rule_eval.go`：createAlertEvent 失败后检测 EventID==0 防止状态不一致
- `rule_eval.go`：NoData 解析失败时保留内存状态避免孤立 firing 事件
- `escalation_executor.go`：SetInterval 校验 d>0
- `leader_election.go`：Lua 脚本提取为包级变量避免重复创建
- `suppression.go`：sync.Once 防止重复 Start

**安全加固**
- `crypto.go`：key 缺失/格式错误时 stderr 告警
- `safehttp/client.go`：移除 debug 模式 SSRF 绕过，loopback 始终阻断
- `datasource/auth.go`：Unmarshal 失败记录日志
- `handler/integration.go`：请求体 1MB 限流
- `repository/oncall_shift.go`：errors.Is 替代 == 比较

**数据层修复**
- `repository/label_registry.go`：4 个方法补全 context.Context 参数
- `repository/diagnostic_workflow.go`：json.Marshal 替代 fmt.Sprintf 防 JSON 注入
- 13+ 处调用链同步传递 ctx（service → handler）

**Handler 层统一**
- 7 个分页端点统一使用 SuccessPage() 替代 gin.H{"list":...,"total":...}

**Model 层规范**
- InspectionTask / DiagnosticWorkflow / AIConversation / KnowledgeDocument / AutoAction 嵌入 BaseModel 替代手动字段

**测试更新**
- safehttp 测试更新：loopback 在 debug 模式下同样阻断

---

## [v4.15.8] — 2026-05-21

### 定时巡检 Agent

- `internal/model/inspection.go`：新增 InspectionTask + InspectionRun 模型
- `internal/repository/inspection.go`：CRUD + ListEnabledTasks 仓储层
- `internal/service/inspection_prompt.go`：巡检系统提示词模板（结构化报告输出）
- `internal/service/inspection_executor.go`：单次巡检执行器，调用 RunUntilDone + 解析 JSON 报告
- `internal/service/inspection_scheduler.go`：robfig/cron 定时调度 + LeaderChecker 接口避免 import cycle
- `internal/handler/inspection.go`：Task CRUD + Run 列表/详情 + RunNow 手动触发 + ValidateCron 校验
- `internal/router/admin_routes.go`：/inspection/tasks, /inspection/runs, /inspection/validate-cron 路由注册
- `internal/service/lark_cards.go`：BuildInspectionReportCard 飞书巡检报告卡片
- `internal/pkg/dbmigrate/migrations/000061_inspection_task.{up,down}.sql`：迁移文件
- `cmd/server/wire.go`：inspectionRepo → InspectionExecutor → InspectionScheduler → InspectionHandler 全链路 DI

### AI Agent 增强

- `internal/service/ai_agent.go`：新增 RunUntilDone 方法（自主工具调用循环，直到 LLM 给出最终回答）
- `internal/service/ai.go`：新增 callLLMWithToolsCustom 支持自定义工具执行器 + ToolCallRecord
- `internal/service/ai_tools.go`：新增 ListFiltered + ToOpenAIToolsFiltered 工具白名单过滤

### AI 工具元数据

- `internal/service/ai_tools.go`：AITool 新增 RiskLevel (0=read/1=write/2=destructive) + IO 标注
- `internal/handler/ai.go`：新增 ListTools 端点暴露工具注册表元数据
- `internal/router/setting_routes.go`：GET /api/ai/tools/registry 路由

### 前端

- `web/src/api/inspection.ts`：巡检任务 + 运行记录 API 封装
- `web/src/components/common/CronInput.vue`：Cron 表达式输入组件（预设 + 自定义 + 校验）
- `web/src/pages/platform/inspections/Index.vue`：巡检任务列表 + 创建/编辑 Modal + 执行记录表格
- `web/src/pages/platform/inspections/RunDetail.vue`：巡检报告详情页（摘要 + 发现项 + 完整报告）
- `web/src/router/index.ts`：/platform/inspections 路由
- `web/src/i18n/en.ts` + `zh-CN.ts`：inspection/inspectionDetail 菜单文案

### 依赖

- `github.com/robfig/cron/v3`：Cron 定时调度

---

## [v4.15.7] — 2026-05-21

### Review Round 2 — 22 项全量整改

**安全 (PR-A)**
- `internal/service/larkbot.go`：飞书 Webhook 防重放攻击，±5 分钟时间戳窗口校验
- `internal/pkg/upload/validator.go`：新增文件上传 MIME + magic number 校验工具，YAML/JSON 内容验证
- `internal/handler/alert_rule.go`：Import 使用 `ValidateYAMLUpload` 校验上传文件
- `internal/handler/alertmanager_import.go`：readYAMLInput 使用 `ValidateYAMLUpload` 校验
- `internal/handler/oidc.go`：Callback/CallbackJSON 入口检查 `Enabled()`，OIDC 未启用返回 403

**引擎 (PR-B)**
- `internal/engine/rule_eval_state.go`：新增 `gcStates()` 方法，每小时清理 24h 前的 resolved 状态，防止 sync.Map 无限增长
- `internal/engine/rule_eval.go`：Run 循环新增 gcTicker 触发 gcStates
- `internal/engine/evaluator.go`：AlertState 新增 `Revision int64` 字段，每次状态变更自增
- `internal/engine/rule_eval.go`：所有 `state.Status =` 赋值后追加 `state.Revision++`
- `internal/engine/escalation_executor.go`：新增 `stepExecRepo` 字段，升级步骤使用 INSERT IGNORE 原子去重
- `internal/repository/schedule.go`：新增 `EscalationStepExecutionRepository`（InsertIgnore/HasExecuted）
- `internal/model/schedule.go`：新增 `EscalationStepExecution` 模型
- `internal/pkg/dbmigrate/migrations/000060_escalation_step_exec.{up|down}.sql`：迁移文件
- `cmd/server/wire.go`：注入 `stepExecRepo` 到 `NewEscalationExecutor`

**错误处理 (PR-C)**
- `internal/handler/alert_event.go`：CSV 导出 `w.Write()` 和 `w.Flush()` 错误处理
- `internal/service/ai_tools.go`：新增 `marshalJSONOrError` 辅助函数，替换 10 处 `data, _ := json.Marshal(...)`
- `internal/service/ai_agent.go`：`paramsBytes` JSON Marshal 错误处理
- `internal/service/ai_agent.go`：cleanupLoop / StartAgent 后台 goroutine 新增 panic recovery
- `internal/service/dashboard_stats.go`：GetStats 7 个 goroutine 新增 panic recovery

**诊断工作流 (PR-D)**
- `internal/repository/diagnostic_workflow.go`：FindMatchingWorkflows 使用 MySQL `JSON_CONTAINS` 替代 Go 层标签匹配

**前端 (PR-E)**
- `internal/handler/metrics.go`：未授权响应改用 `Error()` 统一封装
- `internal/handler/user_preference.go`：3 处 `c.JSON` 改用 `Error()` / `apperr` 统一响应
- `web/src/composables/useFilterMemory.ts`：`bindRefs` 新增 `onScopeDispose(stop)` 自动清理 watcher

**迁移文件**: 000060_escalation_step_exec

---

## [v4.15.6] — 2026-05-21

### Review Round 3 — 剩余项全量实现

**后端**
- `internal/model/diagnostic_workflow.go`：DiagnosticRun 新增 `Version` 字段，`UpdateRun` 改用乐观锁（`WHERE version = ?`）
- `internal/pkg/dbmigrate/migrations/000059_diagnostic_run_version.{up|down}.sql`：迁移文件
- `internal/service/ai_agent.go`：`StartAgent` 接收 `ctx`，DB 写入用请求 ctx，后台 goroutine 用 `context.WithTimeout(context.Background(), 30m)`
- `internal/handler/ai_agent.go`：传递 `c.Request.Context()` 给 `StartAgent`
- `internal/engine/evaluator.go`：`GetFiringEvents` 新增 5s TTL 缓存 + `firingCacheMu` RWMutex；`syncRules` 完成后调用 `invalidateFiringCache()`
- `internal/service/larkbot.go`：`LarkBotService` 新增 `lastMessageAt`/`lastError`/`consecutiveErrors` 生命周期指标；`GetBotStatus` 返回运行时指标；`SendMessage` 记录成功/失败

**文件拆分**
- `internal/engine/evaluator.go`（785→638 行）：拆出 `evaluator_cache.go`（GetFiringEvents/GetFiringAlertEvents/GetStatus/copyAlertState）
- `internal/engine/rule_eval.go`（688→370 行）：拆出 `rule_eval_state.go`（lockState/deleteState/rangeStates/persistState 等）+ `rule_eval_actions.go`（createAlertEvent/updateFiringEvent/resolveAlertEvent）
- `internal/service/alert_event.go`（541→280 行）：拆出 `alert_event_batch.go`（BatchAcknowledge/BatchClose）+ `alert_event_webhook.go`（ProcessWebhook/processAlert/triggerLarkCardUpdate/addTimeline）

**测试**
- `internal/service/lark_cards_test.go`：sanitizeLarkMarkdown / formatDuration / larkSeverityTemplate / larkSeverityEmoji / BuildResolvedCard
- `internal/engine/workerpool_test.go`：NewAlertWorkerPool / Submit / Wait / panic recovery / deadline / concurrent submit
- `internal/pkg/lark/bot_api_test.go`：LarkError / IsRetryable / doWithRetry / tokenCache
- `internal/engine/leader_election_test.go`：IsLeader 状态管理 + 并发安全

**前端**
- `web/src/utils/timeStep.ts`：提取 `computeTimeStep()` 公共函数（从 useQueryEngine/useVariable 重复代码提取）
- `web/src/composables/useQueryEngine.ts`：`autoStep` 委托给 `computeTimeStep`
- `web/src/composables/useVariable.ts`：`autoInterval` 委托给 `computeTimeStep`
- `web/src/composables/useVariable.ts`：补充缺失的 `computeTimeStep` 导入

**迁移文件**: 000059_diagnostic_run_version

---

## [v4.15.5] — 2026-05-21

### Full Review Fix — 6 P0 + 23 P1 + 18 P2 修复

**P0 修复**
- `internal/engine/leader_election.go`：`isLeader` 改为 `atomic.Bool` + `TryAcquire` 改用 Lua 脚本原子操作，消除数据竞争和脑裂风险
- `internal/service/diagnostic_workflow.go`：`ReplaceSteps` 改用 DB 事务，崩溃不再丢步骤
- `internal/pkg/lark/bot_api.go`：Token TTL 最小值 clamp 到 30s，防止 `expire<=60` 时缓存风暴
- `internal/service/lark.go`：`BotClient` 缓存到 `LarkService`，避免每次调用都重新获取 token
- `web/src/composables/useAppNav.ts`：修复 AI Module Config 菜单路由 404

**P1 修复**
- `internal/engine/evaluator.go`：`GetFiringEvents` 返回深拷贝，消除数据竞争；`startRuleEvaluator` 先 Stop 旧 evaluator 防 goroutine 泄漏
- `internal/engine/rule_eval.go`：`resolveAlertEvent` 返回 error，失败时不清除状态防丢告警；`createAlertEvent` 错误时查询已有事件防重复；所有 context 改为 `re.ctx`
- `internal/engine/escalation_executor.go`：批次超时 55s→5m；步骤排序改用 `continue` 替代 `break`
- `internal/engine/heartbeat_checker.go`：`onAlert` 回调改为异步，不再阻塞心跳循环
- `internal/service/alert_event.go`：`context.Background()` 改为 `serverCtx`；`Acknowledge/Resolve/Close` 改用原子 `TransitionStatus`；`BatchAcknowledge/Close` 只为实际更新的事件创建 timeline
- `internal/service/diagnostic_workflow.go`：`executeRun` goroutine 加 30 分钟超时
- `internal/handler/larkbot.go`：请求体加 1MB 限制
- `internal/service/ai.go`：LLM 响应加 10MB 限制（5 处）
- `internal/service/larkbot.go`：响应体加 1MB 限制；`handleMessageEvent` 消除重复 config 加载
- `internal/repository/alert_event.go`：新增 `TransitionStatus` / `GetByFingerprintAndStatus` 方法
- `web/src/components/alert-rule/AIGenerateModal.vue`：17 处硬编码中文改为 i18n
- `web/src/pages/oncall/EscalationPolicies.vue`：修复 i18n key 错误 + 添加 saving 状态
- `web/src/pages/alerts/events/Index.vue`：翻页时清除选中状态
- `web/src/layouts/AppRail.vue`：图标按钮添加 aria-label

**P2 修复**
- `internal/engine/suppression.go`：移除 `log.Printf`，`RemoveSeverity` 不再要求 severity 精确匹配
- `internal/engine/state_store.go`：`toStateEntry/fromStateEntry` 深拷贝 map 字段
- `internal/service/larkbot.go`：`resolveUserID` fallback 时记录警告日志
- `internal/service/lark.go`：`isWithinBusinessHours` 验证 Sscanf 返回值和范围；错误时返回实际 error
- `web/src/stores/preferences.ts`：catch 块添加 console.warn
- `web/src/pages/settings/SMTPConfig.vue`：i18n key 统一为 `common.savedSuccess`

### Review Round 2 — 安全加固 + 引擎稳定性 + 错误处理 + UX 改进

**安全 (PR-1)**
- `internal/service/larkbot.go`：新增 HMAC-SHA256 签名验证，`HandleEvent` 接收 `X-Lark-Signature`/`X-Lark-Request-Timestamp`/`X-Lark-Request-Nonce` 头
- `internal/handler/larkbot.go`：传递 Lark 签名头到 service 层
- `internal/service/lark_cards.go`：`sanitizeLarkMarkdown` 转义 `[]()` 等特殊字符，防止标签值注入 markdown 链接/代码块
- `internal/handler/alert_rule.go`：文件上传加 10MB 大小限制 + `LimitReader` 双重保护
- `internal/handler/alertmanager_import.go`：同上，文件上传加 10MB 限制

**引擎稳定性 (PR-2)**
- `internal/engine/evaluator.go`：`GetFiringEvents`/`GetStatus` 先快照 evaluator 列表再释放读锁，减少锁竞争
- `internal/engine/escalation_executor.go`：`executeStep` 加 30s per-step 超时，单个 webhook 慢调用不再饿死后续步骤
- `internal/engine/workerpool.go`：`NewAlertWorkerPool` 接收 `*zap.Logger`，panic recovery 记录堆栈而非静默吞没
- `internal/engine/rule_eval.go`：`state.Status = "firing"` 移到 `createAlertEvent` 成功后，消除 `GetFiringEvents` 幻影状态
- `internal/engine/heartbeat_checker.go`：新增 `computeMissed` 时钟偏移容忍，负间隔或 >5x interval 时跳过检查
- `cmd/server/wire.go`：`NewAlertWorkerPool(64, zapLogger)` 适配新签名

**错误与并发 (PR-3)**
- `internal/handler/alert_action.go`：`strconv.Atoi` 错误不再丢弃，无效 duration 使用默认值
- `internal/handler/diagnostic_workflow.go`：`ShouldBindJSON` 错误返回 400 而非静默忽略
- `internal/handler/dashboard.go`：CSV 导出 writeRow 包装函数，写入失败提前终止
- `internal/handler/alert_event.go`：CSV 导出同上
- `internal/service/alert_group.go`：`getGroupTiming`/`getGroupKey` 接收 `ctx` 参数；timer 回调使用 `serverCtx`
- `internal/repository/diagnostic_workflow.go`：`ReplaceSteps` 改用 `CreateInBatches` 减少 N+1 插入
- `internal/service/system_setting.go`：`UpdateAIModules` 写入后清除 `aiCache`/`providersCache`

**前端 UX (PR-4)**
- `web/src/api/request.ts`：`errorCodeMap` 从 6 项扩展到 31 项，覆盖所有后端错误码；修复 10400→conflict / 10401→nameTaken 映射错误
- `web/src/i18n/en.ts` + `zh-CN.ts`：新增 10 个 errorCode i18n 条目
- `web/src/pages/alerts/events/Detail.vue`：Ack/Resolve/Close/Comment 按钮加 `actionLoading`/`commentLoading` 状态
- `web/src/pages/alerts/events/Index.vue`：批量 Ack/Close 按钮加 `batchLoading` 状态

**技术债 (PR-5)**
- `internal/pkg/lark/bot_api.go`：新增 `LarkError` 类型 + `IsRetryable()` 判断（99991663/99991668/99991672/10012/10006）；`doWithRetry` 指数退避包装，所有 API 调用自动重试 3 次
- `web/src/stores/preferences.ts`：`load()` catch 块添加 `console.warn`

### Track A — AI 全局配置 + Track B — 飞书 Bot 重设计 + AIOps Phase 2 接入

**后端**
- `internal/service/system_setting.go`：新增 AI 全局配置（`ai_global` group）：retry_max / context_max_chars / default_temperature / default_max_tokens / monthly_token_budget / data_masking_enabled
- `internal/handler/ai.go`：新增 `GetAIGlobal` / `SaveAIGlobal` handler
- `internal/service/lark.go`：新增 `HandleCardLifecycle` — 根据 resolve_strategy（update/delete）处理告警恢复/关闭卡片，支持 business_hours 时间窗口判断
- `internal/service/lark.go`：新增 `isWithinBusinessHours` 辅助函数，支持跨午夜时间范围
- `internal/service/lark_cards.go`：新增 `BuildResolvedCard` — 构建恢复卡片（含持续时间、严重等级 emoji）
- `internal/pkg/lark/bot_api.go`：新增 `DeleteMessage` 方法（HTTP DELETE 删除消息）
- `internal/service/larkbot.go`：新增 `TestBotAPI` / `GetBotStatus` / `mapNaturalLanguage` 方法
- `internal/service/larkbot.go`：`handleMessageEvent` 支持 `commands_enabled` 开关 + `natural_language_enabled` 自然语言命令映射
- `internal/service/alert_event.go`：`triggerLarkCardUpdate` 改为调用 `HandleCardLifecycle`（替代直接 UpdateAlertCard）
- `cmd/server/wire.go`：接入 AIOps Phase 2 — DiagnosticWorkflowService / ChangeEventService + 对应 Handler
- `internal/router/router.go`：Handlers 新增 `DiagnosticWorkflow` / `ChangeEvent` 字段
- `internal/router/admin_routes.go`：注册诊断工作流 CRUD + Run + Match 路由，变更事件 CRUD 路由
- 新增 API：`GET/PUT /ai/global`、`POST /lark/bot/test`、`GET /lark/bot/status`、`/diagnostic-workflows/*`、`/diagnostic-runs/*`、`/change-events/*`

**前端**
- `web/src/pages/settings/AISettings.vue`：新增 Global Config Tab（retry_max / context_max_chars / temperature / max_tokens / monthly_token_budget / data_masking）
- `web/src/pages/settings/LarkBotConfig.vue`：全面重写 — 4 个区块（App Credentials / Behavior / Commands / Debug）
- `web/src/api/admin.ts`：新增 `aiApi.getGlobal/saveGlobal`、`larkBotApi.testBotAPI/getBotStatus`，扩展 config 响应字段
- `web/src/types/ai-module.ts`：新增 `AIGlobalConfig` 接口
- `web/src/i18n/en.ts` + `zh-CN.ts`：新增 ~37 个 i18n key（AI 全局配置 + 飞书 Bot 新功能）
- `web/src/router/index.ts`：合并重复 AI 路由，LarkBot 路由直指向 LarkBotConfig.vue

**清理**
- 删除 `web/src/pages/settings/AI.vue`、`AIConfig.vue`、`LarkBot.vue`（空壳 wrapper）

---

## [v4.15.4] — 2026-05-21

### P1.3 + P1.4 — 知识库服务 + AI Agent 会话持久化

**后端**
- 新增 `knowledge_documents` 表（迁移 000054）：MySQL FULLTEXT ngram 全文检索
- 新增 KnowledgeDocument 模型 / KnowledgeRepository / KnowledgeBaseService / KnowledgeHandler
- 注册 `search_knowledge` AI 工具（支持 query/source/top_k 参数）
- 新增 `ai_conversations` + `ai_tool_calls` 表（迁移 000055）
- 新增 AIConversation / AIToolCall 模型 / AIConversationRepository
- AgentService 集成 DB 持久化：启动时创建会话，执行时记录工具调用
- 新增 API: `GET/POST/PUT/DELETE /knowledge`, `POST /knowledge/search`, `POST /knowledge/:id/helpful`
- 新增 API: `GET/DELETE /ai/agent/conversations`, `GET /ai/agent/conversations/:id/tool-calls`

---

## [v4.15.3] — 2026-05-21

### 紧急修复 — Migration 000049 启动崩溃

- `000049_alert_rule_status_column.up.sql`：改为 no-op（`status` 列和索引已在初始 schema `000001` 中存在，该迁移引用不存在的 `enabled` 列导致 `Error 1054`）

---

## [v4.15.2] — 2026-05-21

### PR9 — PromQL 真解析 + 校验顺序修正 + 错误类型分离 + 关键单测

**后端**
- `internal/service/rule_generator_improve.go`：`validatePromQLSyntax` 改用 `prometheus/prometheus/promql/parser.ParseExpr` 真解析（替换括号匹配）
- `internal/service/rule_generator_improve.go`：`ImproveRule` 校验顺序修正 — LLM 前校验输入表达式、LLM 后校验输出表达式、语法错误直接阻断返回
- `internal/service/rule_generator_dryrun.go`：`ValidateExpression` 区分 `syntax:` 和 `query:` 两类错误（先离线 parse 再查数据源）
- `internal/service/alert_v2_pipeline_test.go`：新增 5 个 `buildAlertKey` 单测（跨数据源无碰撞、稳定性、标签排序、nil ID）
- `internal/service/preset_rule_test.go`：新增 4 个 `autoMatchDatasource` 测试（空集群、匹配、无匹配、禁用数据源忽略）

**前端**
- `web/src/pages/ai/AgentView.vue`：`isPolling` computed 返回 `!!` 修复 `boolean | null` 类型错误
- `web/src/pages/oncall/MyAlerts.vue`：`r.data?.items` 修正为 `r.data?.data?.list`（匹配 ApiResponse<PageData> 结构）
- `web/src/pages/oncall/MyAlerts.vue`：`formatTime(t)` 参数 `t` 遮蔽 i18n `t` 函数，重命名为 `timeStr`

**依赖**
- 新增 `github.com/prometheus/prometheus v0.304.2`（promql/parser）

---

## [v4.15.1] — 2026-05-20

### 代码审查修复 — Agent 并发安全 + DI 两阶段 + 异步执行 + i18n

**后端**
- `cmd/server/wire.go`：移除重复的空 toolRegistry 创建，改用 `SetToolRegistry` 延迟注入（DI 两阶段初始化）
- `internal/service/ai_agent.go`：添加 `cleanupLoop` 定时清理过期任务（每 10 分钟清理 1 小时前已完成任务），防止 OOM
- `internal/service/ai_agent.go`：提取 `StartAgent`（异步）+ `runTask`（核心逻辑），handler 改为异步返回任务 ID
- `internal/service/ai_agent.go`：`runTask` 使用 `task.Query` 替代闭包变量，修复 `undefined: query` 编译错误
- `internal/handler/ai_agent.go`：`RunAgent` handler 改为调用 `StartAgent` 异步执行

**前端**
- `web/src/pages/settings/AISettings.vue`：硬编码中文 `默认 Provider 未启用或不可用` 替换为 `t('aiSettings.defaultProviderUnhealthy')`
- `web/src/composables/useAppNav.ts`：`/ai` 路由正确映射到 `platform` app（之前默认映射到 `oncall`）
- `web/src/i18n/en.ts` + `zh-CN.ts`：新增 `aiSettings.defaultProviderUnhealthy` 翻译 key

---

## [v4.15.0] — 2026-05-20

### AI Phase 1 — 原生 Anthropic/Claude Provider 支持

**后端**
- `internal/service/ai.go`：新增 `callLLMAnthropic` 方法，原生调用 Anthropic Messages API（`/v1/messages`）
- `internal/service/ai.go`：新增 `chatAnthropic` 方法，支持多轮对话走 Anthropic 原生协议
- `internal/service/ai.go`：`callLLMWithSystem` 和 `Chat` 方法根据 `provider` 类型自动分发（`anthropic` → 原生 API，其余 → OpenAI 兼容）
- `internal/service/ai.go`：`callLLMJSON` 通过 `callLLMWithSystem` 自动继承 Anthropic 支持
- `internal/service/ai.go`：Anthropic provider 不强制要求 BaseURL（默认 `https://api.anthropic.com`）
- `internal/service/system_setting.go`：`AIConfig` 和 `AIProviderConfig` 注释新增 `anthropic` 类型

**前端**
- `web/src/pages/settings/AISettings.vue`：`providerOptions` 新增 `Anthropic Claude` 选项
- `web/src/pages/settings/AISettings.vue`：`providerTypeLabel` 新增 `anthropic` 映射
- `web/src/i18n/zh-CN.ts`：新增 `providerAnthropic: 'Anthropic Claude'`
- `web/src/i18n/en.ts`：新增 `providerAnthropic: 'Anthropic Claude'`
- `web/src/types/ai-module.ts`：`AIProvider.provider` 注释新增 `anthropic`

---

## [v4.14.0] — 2026-05-20

### PR8 — 收尾完成（v4.14.0 最终发版）

**删除废弃模块**
- 删除 Pet 系统（model/handler/service/repository/前端页面 + 迁移 000051_drop_pets）
- 删除 Todo 系统（model/handler/service/repository/前端页面 + 迁移 000052_drop_todo_items）
- Pet/Todo 从 wire.go、router、前端路由、侧边栏、i18n、types 全链路清除

**新增功能**
- 新建 `MyAlerts.vue` 值班视图 + 侧边栏入口 + i18n（中英文）
- 后端 `view_mode=mine` 已支持，前端直接对接

**可观测性**
- handler 层 60+ 端点补 zap.Info 操作日志（user_id + request_id + 实体标识）

**数据库**
- 迁移 000051：DROP pets / pet_interactions
- 迁移 000052：DROP todo_items
- 迁移 000053：alert_events 复合索引（fp+status, status+created_at, datasource_id+rule_id）

**AI 规则生成增强**
- SuggestLabels 改为 LLM 动态推荐（回退到启发式）
- ImproveRule 加冲突检测（PromQL 语法校验 + Jaccard 相似度检查）
- rule_generator.go 拆分为 4 文件：main / dryrun / suggest / improve

**可观测性增强**
- 新增 Prometheus gauge `sreagent_engine_last_heartbeat_timestamp`（deadman switch）
- heartbeat checker 每次成功 pass 更新时间戳

**文档**
- 新增 `docs/v1-v2-alerts.md`：v1/v2 双轨评估引擎说明

**测试**
- 新增 `rule_generator_improve_test.go`：validatePromQLSyntax / tokenizeExpression / jaccardSimilarity / extractMetricNames / extractKeywords / postProcessResult 共 20+ 测试用例

### Added — Sprint 3: High Availability + Cleanup

**Leader Election（S3.1）**
- `engine/leader_election.go`：Redis-based distributed leader election（SET NX EX + Lua atomic release）
- `RedisLeaderElection`：15s TTL，5s 续期，自动重新竞选
- `Evaluator.SetLeaderElection`：非 leader 实例暂停所有 rule evaluator，leader 恢复时自动重启
- `HeartbeatChecker.SetLeaderElection`：非 leader 跳过心跳检查
- `EngineStatus` 新增 `is_leader` 字段，`GET /engine/status` 返回 leader 状态
- Prometheus gauge `sreagent_engine_leader_status`（1=leader, 0=follower）
- 首页 Dashboard 显示待机（follower）徽章

**Heartbeat Checker 增强（S3.2）**
- `EngineConfig.HeartbeatInterval`：配置化心跳检查间隔（秒，默认 60）
- Prometheus counter `sreagent_heartbeat_checks_total`（labels: result=ok/missed/resolved/error）
- Prometheus gauge `sreagent_heartbeat_active_rules`：当前监控的活跃心跳规则数

**Tech Debt（S3.3）**
- `go vet` / `vue-tsc` / `vite build` 全部通过，无 dead code

### Added — Sprint 2: AI Intelligence + Observability

**AI 增强（S2.1-S2.2）**
- `chatCompletionRequest` 新增 `TopP` 字段，从 `AIConfig.TopP` 读取并传递给 LLM
- `chatCompletionResponse` 新增 `Usage` 字段，解析 `prompt_tokens` / `completion_tokens`
- `metrics.IncAITokensUsed` 新增 Prometheus counter `sreagent_ai_tokens_used_total`（labels: provider, direction）
- `callLLMWithSystem` + `Chat` 两个调用路径均记录 token usage
- `POST /ai/analyze-alert` 端点：接入已有的 `AnalyzeAlertWithContext` 方法，前端 `aiApi.analyzeAlert` 方法
- `AIConfig` 新增 `TopP` 字段（0.0-1.0）

**已确认完成的前序工作**
- S2.3: GenerateMute + Improve 端点已存在（`aiRules.POST("/generate-mute")` + `aiRules.POST("/improve")`）
- S2.4: request_id 中间件已在 `logger.go` 中实现（UUID + header propagation + log enrichment）

### Added — Sprint 1: 多数据源自动路由 + 前端批量应用

**多数据源自动路由（S1.1）**
- `PresetRuleService.autoMatchDatasource`：按 preset.Cluster 自动匹配 DataSource.Labels["cluster"]
- `PresetRuleService.Apply` 改造：DatasourceID 为 0 时自动匹配，无需手动选择
- `PresetRuleService.BatchApply`：批量应用预置规则，支持 autoMatch + fallbackDatasourceID
- `POST /preset-rules/batch-apply` 端点（handler + route）
- `AlertEvent` 模型新增 `DataSourceID *uint` 字段（迁移 `000050_alert_event_datasource_id`）
- `rule_eval.go` createAlertEvent 自动填充 DataSourceID
- `alert_v2_pipeline.go` buildAlertKey 加入 datasource_id 维度，防止跨数据源 key 碰撞
- `PresetRule` 前端类型新增 `cluster` 字段

**前端批量应用 UI（S1.1.3）**
- `preset-rules.ts` 新增 `batchApply` API 方法
- `preset-rule.ts` 新增 `BatchApplyResult` / `BatchApplyRequest` 类型
- `Presets.vue` 新增「批量应用」按钮 + Modal：
  - 按集群分组展示预置规则
  - 自动匹配数据源预览（cluster → datasource labels）
  - 兜底数据源选择
  - 批量应用结果展示（成功/失败列表）
- i18n 新增 16 个批量应用相关中英文 key

### 已确认完成的前序工作

- Phase 2.1: AlertV2Service 死代码已在前次清理中移除
- Phase 2.2: NotifyPolicy v1 已完全迁移到 NotifyRule v2
- Phase 2.3: Dashboard handler/service 已拆分（handler 274 行 / service 821 行）
- Phase 2.4: IncidentAggregator 已存在并集成到 AlertV2Pipeline
- Phase 2.5: dashboard-v2 已重定向到 alert/dashboards
- S1.3: On-call UI 已实现（Detail action bar + list inline actions + MyAlerts view mode）

---

## [v4.13.0] — 2026-05-19

### Added — 多数据源路由 + RBAC 后端强制 + AI Dry-Run

**多数据源路由（Theme A）**
- 新增 `internal/pkg/labelmatch/matcher.go`：统一标签匹配引擎，支持 `Match` / `MatchWithSourceID` / `CompileRegex`（带缓存）
- 新增 `internal/pkg/labelmatch/matcher_test.go`：16 个单元测试（精确/正则/否定/通配/数据源维度）
- 迁移 `000047_add_datasource_id_to_routing.{up,down}.sql`：alert_channels / notify_rules / dispatch_policies 新增 `datasource_id` 列
- `AlertChannel` / `NotifyRule` / `DispatchPolicy` 模型新增 `DataSourceID *uint` + `DataSource` 外键
- `NotifyRuleRepository.FindMatchingRules` 支持 `dataSourceID` 参数，使用 `labelmatch.MatchWithSourceID` 过滤
- `NotificationService.RouteAlert` 自动从 event.RuleID 解析 DataSourceID 并传递给规则匹配
- `NotificationService` 新增 `ruleRepo` 依赖，用于解析 DataSourceID
- 全面迁移：`alert_channel.go` / `biz_group.go` / `mute_rule.go` / `dispatch.go` / `noise_reducer.go` / `notification.go`（repository） / `subscribe_rule.go` / `team.go` 均改用 `labelmatch` 包

**RBAC 后端强制（Theme B）**
- 新增 `internal/pkg/rbac/rbac.go`：权限逻辑集中管理（`HasPerm` / `EffectivePerms` / `HighestTeamRole`）
- 新增 `internal/pkg/rbac/rbac_test.go`：10 个单元测试
- `handler/permissions.go`：`GetMyPermissions` 合并全局角色 + 团队角色，返回有效权限集
- 新增 `internal/middleware/permission.go`：`RequirePerm` 中间件，支持全局 + 团队权限检查

**AI Dry-Run（Theme C）**
- `RuleGeneratorService.DryRun`：生成规则 + 自动验证 PromQL 表达式，一步到位
- `POST /ai/rules/dry-run` 端点（handler + 路由）
- 前端 `DryRunResult` 类型 + `aiRuleApi.dryRun` 方法

**v1/v2 清理（Theme D）**
- 删除 `AuditResourceNotifyPolicy` 死常量（NotifyPolicy 已在 v4.11.0 删除）

**Explore 页面升级（Theme E）**
- 替换 NInput textarea 为 `PromQLEditor` 组件（CodeMirror 6 + PromQL 语法高亮 + 自动补全）
- Logs 模式保留 textarea 回退

**引擎可靠性（Theme F）**
- `RuleEvaluator` 新增 `consecutiveErrors` 计数器
- 连续 5 次查询失败升级为 Error 级别日志
- 查询恢复时记录恢复日志

**文档更新（Theme G）**
- `docs/api.md` 新增 AI 规则生成端点文档（generate / dry-run / validate / suggest-labels / generate-inhibition / generate-mute / improve）
- 新增 AI 模块配置端点文档

**PR4 — 枚举统一 + 并发重构 + AI 试算 + useCrudPage 迁移（Theme H）**

- 迁移 `000049_alert_rule_status_column.{up,down}.sql`：`alert_rules.status` 列 DEFAULT `'active'`（单语句，含索引）
- `AlertRuleStatus` 枚举统一为 `draft / active / disabled`（移除 `enabled` / `muted`）
- 后端全量替换 `RuleStatusEnabled → RuleStatusActive`（8 文件）
- 前端 `AlertRuleStatus` 类型 + `rules/Index.vue` 状态标签/开关对齐 `active`
- `rbac.go` 补全 `.write` 后缀权限（rules.write / mute.write / inhibition.write / notify.write / channels.write / dispatch.write / datasource.write / integration.write / team.write / user.write）
- `admin_routes.go`：DispatchPolicy / Integration 路由补挂 `RequirePerm`
- `audit_log.go` 新增 `AuditResultDenied` / `AuditResultSuccess` 常量
- 引擎 B1：`RuleEvaluator.states` 改为 `sync.Map` + per-fingerprint `stateLock`，移除全局 mutex
- 引擎 B2：`startRuleEvaluators` 改为 fan-out 所有匹配数据源（非首个）
- 新增 `evaluator_concurrent_test.go`：4 个并发安全测试
- AI Modal（B3）：dry-run 试算（series_count / sample_series / would_fire / eval_duration_ms）+ 标签预览 + 三按钮（重新生成 / 保存草稿 / 直接启用）
- `AISettings.vue` 响应式网格 `repeat(auto-fit, minmax(350px, 1fr))`
- `inhibition/Index.vue` 新增 `hit_count` 显示（>0 红色高亮）
- 7 页迁移至 `useCrudPage` composable，删除 `useCrudModal`

> **注**：原计划的 `000050_drop_notifications_legacy` 已**撤销**。
> notifications 表是 v4.12.0 新增的用户通知中心（user_notifications），属于通知中心功能，不应被删除。
> 该决策已在第二轮整改时确认，参见 plan-rework-v4.13.0.md。

**PR5 — 第五轮返工修复（PR4 残留补全）**
- 修 `evaluator.lockState()` 辅助方法补全 `deleteState()` + `rangeStates()`（第四轮残留）
- 所有 `re.states.Delete` / `re.states.Range` 改用 `deleteState()` / `rangeStates()` 包装
- 完整实现 `PerDatasourceEvaluator` 分桶 + `per_datasource_eval` feature flag（第四轮零落地补全）
- `Evaluator` 新增 `perDS sync.Map` + `getOrCreateDSBucket` / `removeDSBucket` / `listDSBuckets` 辅助方法
- `syncRules` 分流：perDS 模式走 bucket、legacy 模式走 `e.evaluators` map
- `RuleEvaluator` 新增 `Stop()` 方法（关闭 stopCh）
- 修 AI Modal `handleDryRun` 传 `expr`/`labels`/`severity` 而非 `description`
- 修 AI Modal `handleSaveAsDraft` 调 `alertRuleApi.create`（status=draft, enabled=false）而非 `aiRuleApi.generate`
- 修 AI Modal `handleSaveAsActive` 加 `enabled: true`
- 修 AI Modal `handleLabelPreview` 调 `labelValues` 返回 `registryValues`
- 补 `dispatch-policies` POST 端点 + 漏挂 `RequirePerm("dispatch.write")` 中间件
- 修 `wire.go` audit_log `Status` 硬编码 `"denied"` 改用 `model.AuditResultDenied` 常量
- 修 `rules/Index.vue` `statusFilterOptions` value `'enabled'` → `'active'`
- 补 `web/src/permissions.ts` 10 个 `.write` 后缀常量
- 修 `000049` down.sql 多语句违规（合并为单条 ALTER）
- `config.go` 新增 `EngineConfig.PerDatasourceEval` + `config.example.yaml` 配置项
- `wire.go` 接线 `evaluator.SetPerDatasourceEval(cfg.Engine.PerDatasourceEval)`

**PR6 — 第六轮返工修复（实质补全）**
- 修 handler `CreateAlertRuleRequest` 新增 `Status` 字段，不再硬编码 `RuleStatusActive`（前端 `status: 'draft'` 之前被忽略）
- AI Modal `handleSaveAsDraft` 补 `enabled: false` 字段
- AI Modal `handleSaveAsActive` 补 `enabled: true` 字段
- 前端 `AlertRule` 类型新增 `enabled?: boolean` 可选字段
- 新建 `evaluator_datasource_test.go`：5 个分桶测试（CreateAndCaches / Remove / Empty / StopCleanup / Concurrent）
- 确认 `RuleEvaluator.lockState` / `deleteState` / `rangeStates` 三件套在 `rule_eval.go` 已完整实现
- 确认 `Engine.Stop()` 清理 perDS 所有桶（`e.perDS.Range` + `PerDatasourceEvaluator.Stop()`）

**PR7 — feature flag 行为测试补全**
- 补 `Test_Evaluator_PerDatasourceEval_IsolatedExecution`：验证 perDSEval=true 时 `startRuleEvaluator` 路径创建独立分桶
- 补 `Test_Evaluator_PerDatasourceEval_FlagOff_FallbackLegacy`：验证 perDSEval=false 时退回 legacy 路径不创建分桶
- helper `newTestEvaluatorForDS` 补 `queryClient` 初始化避免 nil panic

---

## [v4.12.1] — 2026-05-19

### Added — RBAC 权限体系 + AI 规则引擎增强 + 前端体验优化

**RBAC 权限体系完善**
- 新增 `web/src/permissions.ts`：50+ 权限常量（rules.create / events.ack / incidents.manage 等）
- 新增 `web/src/directives/vCan.ts`：`v-can` 指令，支持单权限和多权限（OR）条件渲染
- `main.ts` 注册全局 `v-can` 指令
- 告警规则页：创建/导入/AI 生成按钮接入 `hasPerm` 权限检查
- 告警事件页：认领/关闭按钮接入 `hasPerm` 权限检查

**AI 规则引擎增强**
- 新增 `rule_gen_prompts.go`：Few-shot 示例模板（告警/抑制/静默三种规则类型）
- 新增 `rule_gen_cache.go`：内存 TTL 缓存（10 分钟），避免重复 LLM 调用
- `RuleGeneratorService.Generate` 集成缓存 + few-shot prompts
- `RuleGeneratorService.GenerateInhibition` / `GenerateMute` 集成 few-shot prompts
- 新增 `RuleGeneratorService.ImproveRule`：基于用户反馈优化已有规则
- 新增 `POST /ai/rules/improve` 端点（`ai_rule.go` handler + `setting_routes.go` 路由）
- 静默规则页新增"AI 生成屏蔽"按钮，支持自然语言生成静默规则并一键应用

**前端体验优化**
- 新增 `web/src/stores/preferences.ts`：用户偏好 Pinia store（主题/语言/时区/默认时间范围）
- 集成 `UserPreferences` API，支持持久化偏好设置

**国际化对齐**
- `en.ts` 补齐 mute AI 生成相关 key（aiGenerate / aiMatchLabels / aiSeverities / aiTimeMode 等）
- `zh-CN.ts` + `en.ts` 新增 aiImprove 相关 key（aiImprove / aiImproveTitle / aiImproveDesc / aiImprovePlaceholder / aiImproveFailed）

---

## [v4.12.0] — 2026-05-19

### Added — 通知中心 + 待办事项 + RBAC 权限增强

**通知中心 (Notification Center)**
- 新增 `user_notifications` 表（迁移: 000045），支持用户级通知推送
- 后端：`UserNotificationRepository` / `UserNotificationService` / `UserNotificationHandler`
- API：`GET /notifications`、`GET /notifications/unread-count`、`PATCH /notifications/:id/read`、`POST /notifications/read-all`、`DELETE /notifications/:id`
- 前端：`/notifications` 页面，支持未读/已读筛选、标记已读、全部已读
- 顶栏新增通知铃铛图标（`NotificationBell` 组件），30s 轮询未读数

**待办事项 (Todo / Task Center)**
- 新增 `todo_items` 表（迁移: 000046），支持个人任务管理
- 后端：`TodoItemRepository` / `TodoItemService` / `TodoItemHandler`
- API：`GET /todos`、`GET /todos/pending-count`、`POST /todos`、`PUT /todos/:id`、`PATCH /todos/:id/complete`、`DELETE /todos/:id`
- 前端：`/platform/todos` 页面，侧边栏"待办事项"入口，支持优先级排序、截止时间

**RBAC 权限增强**
- 新增 `GET /me/permissions` 端点，返回全局角色 + 团队角色 + 权限列表
- 新增 `PermissionsHandler`，基于角色生成细粒度权限（users.manage / rules.create / events.ack 等）
- `TeamRepository` / `TeamService` 新增 `ListByUser` 方法
- 前端新增 `usePermissions` composable，提供 `hasPerm` / `hasAnyPerm` / `isTeamLead` 等方法

**其他**
- `router.go` Handlers 新增 `UserNotification`、`TodoItem`、`Permissions` 字段

### 迁移文件
- `000045_create_notifications.up.sql` / `000045_create_notifications.down.sql`
- `000046_create_todo_items.up.sql` / `000046_create_todo_items.down.sql`

---

## [v4.11.3] — 2026-05-19

### Added — monitoring-trading 全量兼容

- `scripts/import-presets/main.go` — 全量导入脚本，扫描 monitoring-trading 299 条 YAML 告警规则
- `docs/monitoring-trading-compat.md` — 完整兼容方案文档（规则导入/抑制模板/多租户标签/通知路由/AI 辅助）
- 种子抑制规则从 4 条扩展到 16 条，与 Alertmanager inhibit_rules 完全对齐
- 新增抑制规则：容器 P0/P1 级联、Kafka/Redis/MongoDB/RabbitMQ/Nacos/RocketMQ 宕机级联、ES Red→Yellow、TCP 探测失败级联
- 所有抑制规则使用 `biz_project` equal_labels 防止跨租户误抑制

## [v4.11.2] — 2026-05-19

### Added — 内置预置规则库 seed

- 启动时自动 seed 45 条内置预置规则到 `preset_rules` 表
- 覆盖 8 大类：主机/系统(8)、Kubernetes(7)、MySQL(4)、Redis(3)、MongoDB(2)、Elasticsearch(3)、中间件(7)、网络探测(3)、应用(2)、抑制模板(4)

## [v4.11.1] — 2026-05-19

### Fixed — MySQL JSON DEFAULT 错误

- `cmd/server/main.go` — 从 AutoMigrate 列表移除 `UserPreference`（表由迁移 000044 创建）
- `internal/model/user_preference.go` — 移除 NotificationSeverities 的 default gorm tag
- `internal/pkg/dbmigrate/migrations/000044_create_user_preferences.up.sql` — `JSON NULL` 替代 `JSON DEFAULT`

## [v4.11.0] — 2026-05-19

### Changed — V1/V2 清理 & 架构统一

**Phase 2.1: 删除 AlertV2Service 死代码**
- `internal/service/alert.go` — 删除 `UpsertFromEvent`（从未调用）和 `LinkToIncident`（pipeline 直接调 repo）

**Phase 2.2: NotifyPolicy v1 → NotifyRule v2 完全迁移**
- `internal/service/notification.go` — 完全重写：移除 v1 策略管道（SendNotification、isThrottled、email/webhook helper），新构造函数仅依赖 subscribeRuleSvc + notifyRuleSvc
- `internal/model/notification.go` — 删除 `NotifyPolicy` 结构体
- `internal/repository/notification.go` — 删除 `NotifyPolicyRepository`
- `internal/engine/escalation_executor.go` — 改用 `NotifyMediaService` + `sendViaChannel` 适配器
- `internal/service/notify_rule.go` — 新增 `FindMatchingRules` 方法
- `internal/service/notification_dedup.go` — 移入 `routeDedup` 变量初始化
- `cmd/server/wire.go` — 移除 policyRepo，更新 NotificationService 构造参数
- `cmd/server/main.go` — AutoMigrate 移除 `NotifyPolicy`
- `internal/service/notification_test.go` — 重写：移除 v1 测试，保留 v2 测试
- 迁移: `000042_drop_notify_policies` — DROP TABLE notify_policies

**Phase 2.3: Dashboard handler/service 拆分**
- `internal/service/dashboard_stats.go` — 新建：DashboardStatsService（821 行，12 个方法）
- `internal/handler/dashboard.go` — 906→274 行：handler 仅做参数解析 + 调用 service + 返回 JSON

**Phase 2.4: IncidentAggregator 告警-故障桥接**
- `internal/service/incident_aggregator.go` — 新建：基于 fingerprint 的 Incident 聚合（OnEventFired/OnEventResolved）
- `internal/model/incident.go` — 新增 `Fingerprint` 字段
- `internal/repository/incident.go` — 新增 `FindOpenByFingerprint` 方法
- `internal/repository/alert_event.go` — 新增 `CountByFingerprintAndStatus` 方法
- `internal/service/alert_v2_pipeline.go` — 新增 `SetIncidentAggregator`，process() 中调用 aggregator 钩子
- `cmd/server/wire.go` — 创建 IncidentAggregator 并注入 pipeline
- 迁移: `000043_add_incident_fingerprint` — ALTER TABLE incidents ADD COLUMN fingerprint

**Phase 2.5: dashboard-v2 重命名**
- `web/src/pages/dashboard-v2/` → `web/src/pages/dashboards/` — 目录重命名
- `web/src/router/index.ts` — 更新 import 路径

### Added — 后端能力暴露（Section 6 P0）

**Gap 6: 用户偏好系统**
- `internal/model/user_preference.go` — 新建 UserPreference 模型（theme/language/timezone/default_time_range/notification_severities/ai_chat_mode）
- `internal/repository/user_preference.go` — 新建 repository（GetByUserID + Upsert）
- `internal/service/user_preference.go` — 新建 service
- `internal/handler/user_preference.go` — 新建 handler（GET/PUT /me/preferences）
- `internal/router/auth_routes.go` — 添加 /me/preferences 路由
- `cmd/server/wire.go` — 接入 UserPreference 依赖链
- `web/src/pages/platform/Profile.vue` — 新增"偏好设置"Tab（主题/语言/时区/默认时间范围/AI 对话模式）
- `web/src/types/index.ts` — 新增 UserPreferences 类型
- `web/src/api/admin.ts` — 新增 getPreferences/updatePreferences API
- `web/src/i18n/zh-CN.ts` + `en.ts` — 新增偏好设置 i18n keys
- 迁移: `000044_create_user_preferences` — CREATE TABLE user_preferences

**Gap 1: AlertRule 高级字段 UI**
- `web/src/components/alert/RuleFormModal.vue` — 新增"高级设置"折叠面板（rule_type/eval_interval/recovery_hold/nodata_enabled/nodata_duration/suppress_enabled/heartbeat_token/heartbeat_interval/ack_sla_minutes）
- `web/src/types/index.ts` — AlertRule 接口新增 eval_interval/recovery_hold/nodata_enabled/nodata_duration/suppress_enabled 字段
- `web/src/i18n/zh-CN.ts` + `en.ts` — 新增高级设置 i18n keys（advancedSettings/ruleType/evalInterval/recoveryHold/nodata/suppress/heartbeat/ackSla）

**Gap 4: 升级策略管理页面**
- `web/src/pages/oncall/EscalationPolicies.vue` — 新建：完整 CRUD 页面（多步骤升级，支持 user/team/schedule 目标）
- `web/src/router/index.ts` — 添加 `/oncall/config/escalation-policies` 路由
- `web/src/composables/useAppNav.ts` — 侧边栏"配置中心"组新增"升级策略"入口
- `web/src/i18n/zh-CN.ts` + `en.ts` — 新增 `escalation` 命名空间 + `menu.escalationPolicies`

**Gap 5: 通知分派记录**
- `internal/service/dispatch.go` — 新增 `ListLogsByIncident` 方法
- `internal/handler/dispatch.go` — 新增 `ListLogs` handler（GET /incidents/:id/dispatch-logs）
- `internal/router/admin_routes.go` — 注册 dispatch-logs 路由
- `web/src/api/incident.ts` — 新增 `getDispatchLogs` API 方法
- `web/src/types/index.ts` — 新增 `DispatchLog` 类型
- `web/src/pages/incidents/Detail.vue` — 新增"通知记录"Tab（NDataTable 展示分派日志）
- `web/src/i18n/zh-CN.ts` + `en.ts` — 新增 dispatchLog i18n keys

**Gap 7: AI temperature/max_tokens/system_prompt 透传**
- `internal/service/system_setting.go` — AIConfig 新增 Temperature/MaxTokens/SystemPrompt 字段 + GetAIConfig 加载 + SaveAIConfig 持久化 + parseFloatDef/parseIntDef helpers
- `internal/service/ai.go` — chatCompletionRequest 新增 Temperature/MaxTokens 字段，callLLMWithSystem 和 Chat 方法透传配置值
- `web/src/api/admin.ts` — aiApi.getConfig/updateConfig 类型定义新增 temperature/max_tokens/system_prompt

**Gap 8: AI 高级配置（重试 + 上下文预算）**
- `internal/service/system_setting.go` — AIConfig 新增 RetryMax/ContextMaxChars 字段 + 持久化
- `internal/service/ai.go` — callLLMJSON 使用 cfg.RetryMax 替代硬编码；AnalyzeAlertWithContext 增加上下文截断
- `web/src/pages/settings/AIConfig.vue` — 新增"最大重试次数"和"上下文字符上限"控件
- `web/src/i18n/zh-CN.ts` + `en.ts` — 新增 aiRetryMax/aiContextMaxChars

---

## [v4.10.37] — 2026-05-19

### Changed — 前端国际化完善

- `web/src/utils/severity.ts` — 严重级别标签改用 `i18n.global.t()` 动态翻译
- `web/src/api/request.ts` — 错误码消息改用 vue-i18n（去除手动 `[zh, en]` 双语数组）
- `web/src/pages/alerts/Presets.vue` — 30+ 硬编码中文字符串替换为 `t()` 调用
- `web/src/pages/settings/AISettings.vue` — 55+ 硬编码字符串替换为 `t()` 调用 + 添加 `useI18n`
- `web/src/pages/settings/AIConfig.vue` — 供应商选项标签国际化
- `web/src/pages/alerts/rules/Index.vue` — AI 生成结果标签（Duration/Labels/Summary）国际化
- `web/src/components/common/UserAvatar.vue` — 头像 alt 文本国际化
- `web/src/components/common/LabelMatcherEditor.vue` — placeholder 国际化
- `web/src/pages/settings/UserManagement.vue` — 4 个 placeholder 国际化
- `web/src/pages/settings/TeamManagement.vue` — placeholder 国际化
- `web/src/pages/notification/Templates.vue` — placeholder 国际化
- `web/src/pages/notification/Media.vue` — 8 个 placeholder 国际化
- `web/src/pages/notification/AlertChannels.vue` — fallback 模式修复
- `web/src/pages/datasources/Index.vue` — 2 个 placeholder 国际化
- `web/src/pages/schedule/ScheduleModal.vue` — 3 个 placeholder 国际化
- `web/src/pages/alerts/mute/Index.vue` — placeholder 国际化
- `web/src/pages/alerts/inhibition/Index.vue` — placeholder + fallback 国际化
- `web/src/pages/alerts/events/Index.vue` — fallback 模式修复
- `web/src/pages/alerts/events/Detail.vue` — fallback 模式修复
- `web/src/pages/alerts/history/Index.vue` — fallback 模式修复
- `web/src/components/alert/BatchOperations.vue` — fallback 模式修复
- `web/src/pages/dashboard-v2/Index.vue` — fallback 模式修复
- `web/src/i18n/zh-CN.ts` — 新增 `errorCode`、`alert.aiGen*`、`aiSettings.provider*` 等 i18n key
- `web/src/i18n/en.ts` — 对应英文翻译同步
- `web/src/components/time/TimeRangePicker.vue` — 改用响应式 `useRelativeTimeOptions()`

## [v4.10.36] — 2026-05-19

### Fixed — OIDC 配置热加载

- `cmd/server/wire.go` — `Dependencies` 新增 `ReloadOIDC()` 方法 + `oidcHdlr` 字段
- `internal/handler/oidc_settings.go` — `UpdateConfig` 保存后自动调用 `onReload()` 回调热加载 OIDC 配置
- `web/src/pages/settings/OIDCConfig.vue` — 移除"需要重启 Pod 才能生效"警告提示

## [v4.10.35] — 2026-05-19

### Security — P0 生产风险修复（7 项）

**P0-1: 登录暴力破解防护**
- `internal/middleware/rate_limit.go` — 新增 token bucket 限流中间件（通用 `RateLimit` + 专用 `LoginRateLimit`）
- `internal/router/router.go` — 登录端点：5 RPS / burst 5 / 5 次失败锁定 15 分钟

**P0-2: AI Chat 端点防护**
- `internal/handler/ai.go` — Chat message 字段增加 `max=4000` 长度校验

**P0-3: CORS 启动校验**
- `internal/middleware/cors.go` — 启动时校验：AllowCredentials + wildcard origin 直接 panic

**P0-4: DataSource AuthConfig 加密**
- `internal/pkg/crypto/crypto.go` — 新增共享 AES-256-GCM 加密包（`enc:` 前缀约定）
- `internal/service/datasource.go` — Create/Update 加密，5 个读取路径解密
- `internal/service/system_setting.go` — 重构为委托 crypto 包

**P0-5: HeartbeatToken 自动生成**
- `internal/service/alert_rule.go` — heartbeat 规则创建时自动生成 crypto/rand token
- `internal/model/alert_rule.go` — HeartbeatToken 增加 uniqueIndex + MaskHeartbeatToken()
- `internal/handler/alert_rule.go` — Get/List 返回掩码 token，新增 admin 专用 full-token 端点

**P0-6: 通知 v1/v2 跨管道去重**
- `internal/service/notification_dedup.go` — 内存去重缓存（TTL 5 分钟 + 定时清理）
- `internal/service/notification_dedup_test.go` — 6 个单元测试
- `internal/service/notification.go` + `notify_rule.go` — v1/v2 发送前去重守卫

**P0-7: PostMortem AI 限流**
- `internal/router/admin_routes.go` — AI 生成端点：0.1 RPS / burst 3

### Fixed — P1 健壮性修复（12 项）

- `internal/service/biz_group.go` — BizGroup 循环引用检测（`wouldCreateCycle` 祖先链遍历）
- `internal/service/incident.go` — 故障状态机白名单（`validTransitions` + `validateTransition`）
- `internal/service/schedule.go` — 值班 Override 优先级修正（override 现在优先于 shift）
- `internal/service/schedule.go` — EscalationStep 顺序校验（sequential order + target 非空 + delay >= 0）
- `internal/service/schedule_test.go` — 16 个升级步骤校验单元测试
- `internal/handler/user.go` — ChangePassword 审计日志（action=update, resource=user_password）
- `internal/service/team.go` — AddMember 幂等性（已存在同角色 → no-op，不同角色 → 更新）
- `internal/service/message_template.go` — 模板渲染 5 秒超时（context.WithTimeout）
- `internal/service/alert_channel.go` — 正则缓存（sync.Map + getOrCompileRegex）
- `internal/service/dispatch.go` — 分派策略正则缓存 + 预编译 template regex
- `internal/service/mute_rule.go` + `noise_reducer.go` — 共享正则缓存
- `internal/handler/oidc.go` — OIDC 错误标准化（9 个错误路径统一使用 Error(c, err)，不泄露内部细节）
- `internal/service/ai.go` — stripMarkdownCodeBlock 增强（嵌套 backtick / CRLF / indented blocks）

### Changed — 性能优化

- `internal/handler/dashboard.go` — GetStats 7 个查询并行化（sync.WaitGroup）
- `internal/handler/dashboard.go` — IncidentStats 7 个查询并行化

---

## [v4.10.34] — 2026-05-19

### Added — 侧边栏图标彩色化

- `web/src/composables/useAppNav.ts` — 新增 `iconColorMap`：30 个图标组件 → 语义色映射（amber/rose/emerald/violet/indigo/cyan/sky 等）
- `web/src/layouts/AppSidebar.vue` — 图标渲染使用 `NIcon` `color` prop 传入语义色；hover 时切换为 app accent 色 + 微缩放；selected 保持 accent 色；新增 `transition: color 180ms + transform 180ms`

---

## [v4.10.33] — 2026-05-19

### Fixed — 死代码清理（-402 行）

- `web/src/pages/settings/Index.vue` — 删除旧版统一设置页（已被独立路由替代）
- `web/src/api/incident.ts` — 移除从未调用的 `getNoiseConfig` 方法
- `web/src/api/oncall.ts` — 移除无后端对应的 escalation steps API（createStep/updateStep/deleteStep）
- `internal/handler/exclusion_rule.go` — 移除未注册路由的 `GetNoiseCfg` / `UpdateNoiseCfg` handler

---

## [v4.10.32] — 2026-05-19

### Fixed — AlertChannels KVEditor 类型修复

- `web/src/pages/notification/AlertChannels.vue` — 修复 TS2322：`form.match_labels`（`Record<string, string>`）通过 computed getter/setter 桥接为 `KVItem[]`，解决 `useCrudPage<T>` 泛型约束与 KVEditor 组件类型不兼容问题

---

## [v4.10.31] — 2026-05-19

### Fixed — 偷懒补丁 + 性能优化

**移动端 nav bug 修复**
- `web/src/styles/global.css` — 640px 断点从 `display: none` 改为 rail 模式（60px icon-only），手机用户可正常切页

**性能优化**
- `internal/repository/notify_rule.go` — `ListEnabled` 加 30s TTL 内存缓存，`FindMatchingRules` 告警评估时不再每次查 DB
- `web/src/pages/dashboard/UnifiedDashboard.vue` — on-call schedule 串行 fetch 改 `Promise.allSettled` 并行

**文档同步**
- `docs/PLAN-status.md` — 同步至 v4.10.31

---

## [v4.10.30] — 2026-05-19

### Fixed — 收尾优化（落地差距 + 响应式 + ARIA + 测试）

**落地差距修复**
- `web/src/pages/alerts/mute/Index.vue` — 重构：抽取 `utils.ts`（11 个纯函数），SFC 减 ~70 行
- `web/src/composables/useCrudModal.ts` — 标记 `@deprecated`（计划 v4.11 移除）
- `docs/PLAN-status.md` — 同步至 v4.10.29，补充 4 个新模块条目

**前端 — 响应式 + 滚动条**
- `web/src/styles/global.css` — 新增全局响应式断点（1024px 平板 + 640px 手机）
- `web/src/styles/global.css` — 滚动条改为始终可见（移除 hover-reveal）

**前端 — ARIA 关键场景**
- `EmptyState.vue` — 补 `role="status"` + `aria-live="polite"`
- 4 个页面 RowMenu 补 `aria-label`
- 4 个页面 inline switch 补 `:aria-label="item.name"`

**测试**
- 新建 `internal/handler/alert_rule_test.go`（7 个测试）
- 新建 `internal/handler/mute_rule_test.go`（5 个测试）
- `internal/service/schedule_test.go` — 2 个 DB 集成测试（GetCurrentOnCall + OverridePriority）
- `internal/service/notification_test.go` — 2 个 DB 集成测试（LabelSubset + NoMatch）

---

## [v4.10.29] — 2026-05-19

### Fixed — 自审修复（a11y + 后端一致性 + UI token + i18n）

**后端 — ErrorWithMessage 全量清除**
- 31 个 handler 文件替换 ~120 处 `ErrorWithMessage(c, CODE, msg)` → `Error(c, apperr.WithMessage(...))`
- 删除 `handler.go` 中 `ErrorWithMessage` 函数定义（0 调用点残留）
- 错误码映射：10001→ErrInvalidParam、10002→ErrMissingParam、10100→ErrUnauthorized、50000→ErrInternal、50001→ErrDatabase、50003→ErrExternalAPI

**前端 — a11y**
- 4 个 Modal 组件补焦点归还（MergeModal、SnoozeModal、ReassignModal、ChangePasswordModal）
- Dashboard/Index.vue + UnifiedDashboard.vue — 14 个可点击 div 补 `tabindex="0"` + `@keydown.enter` + `role="button"`
- alerts/events/Index.vue + BatchOperations.vue — 批量操作栏补 `role="toolbar"` + `aria-label`
- `--sre-text-tertiary` 亮色对比度 2.9:1 → 4.5:1（#5C5650），`--sre-text-muted` 2.3:1 → 3.7:1（#6B6560）

**前端 — CSS token 统一**
- explore/Index.vue、Login.vue、UnifiedDashboard.vue、dashboard/Index.vue、IncidentDashboard.vue — 10 处硬编码颜色改用 `var(--sre-*)` token
- BizGroupManagement.vue、TeamManagement.vue、VirtualUsers.vue — modal 加 `max-width: 90vw`

**前端 — UI 一致性**
- 6 个页面补齐 `<PageHeader>`（StatusPage、Rules、Templates、Media、Subscribe、BizGroupManagement）
- VirtualUsers.vue + Media.vue 补 loading 指示器

**前端 — i18n**
- 6 个页面删除 29 处 `|| 'English fallback'` 反模式

---

## [v4.10.28] — 2026-05-19

### Fixed — 终审优化（UI 一致性 + 交互增强 + a11y）

**后端**
- `internal/handler/alert_event.go` — 替换 8 处 `ErrorWithMessage(c, 10001, ...)` → `Error(c, apperr.WithMessage(...))`
- `internal/handler/alert_rule.go` — 替换 13 处
- `internal/handler/mute_rule.go` — 替换 5 处 + 新增 `PreviewOne` 单规则预览端点
- `internal/handler/notify_rule.go` — 替换 5 处
- `internal/router/admin_routes.go` — 注册 `GET /mute-rules/:id/preview`

**前端 — UI 一致性**
- 6 个页面补齐 `<PageHeader>`（events、history、integrations、notification/Index、schedule、settings/Index）
- `web/src/pages/notification/AlertChannels.vue` — 补 `<PageHeader>`
- `web/src/styles/global.css` — 补 `prefers-reduced-motion` 兜底（ripple、page-transition、hover 动效全部关闭）

**前端 — 交互增强**
- `web/src/api/notify.ts` — 新增 `previewOne(id)` 单规则预览 API
- `web/src/pages/alerts/mute/Index.vue` — 改用 `previewOne` 直接调用，不再拉全量筛选
- 5 个高频页面加 `<LoadingSkeleton>`（AuditLog、TeamManagement、UserManagement、StatusPage、incidents）

---

## [v4.10.27] — 2026-05-19

### Fixed — 三轮审查最终优化（Batch 7-8）

**Batch 7: 一致性清理**
- `internal/service/inhibition_rule.go` — `Delete` 方法补 `apperr.Wrap` 错误包装（与 Create/Update 一致）
- 删除 `internal/pkg/errors/codes.go`（9 个 int 常量，与 AppError 系统重复）
- `internal/handler/handler.go` — 错误码引用迁移到 AppError 系统（`ErrNotFound.Code` / `ErrInternal.Code`）
- `internal/handler/metrics.go` — `CodeTokenInvalid` → `ErrUnauthorized.Code`
- `internal/handler/mute_rule.go` — 构造函数改为接收 `eventSvc` 参数，移除 `SetAlertEventService` setter
- `cmd/server/wire.go` — 更新 MuteRuleHandler 构造调用
- `internal/handler/handler_test.go` — 断言码 10002 → 10300（匹配 AppError 系统）

**Batch 7: 前端清理（-518 行 0 用户代码）**
- 删除 `web/src/components/common/CrudListPage.vue`（193 行，0 引用）
- 删除 `web/src/components/common/ErrorRetry.vue`（71 行，0 引用）
- 删除 `web/src/utils/formRules.ts`（254 行，0 引用，硬编码英文）
- 删除 `web/src/pages/settings/AuditLogs.vue`（7 行包装器，合并到 AuditLog.vue）
- `web/src/router/index.ts` — 修复 AuditLog 导入路径，清理 alerts-v2 遗留 redirect
- `web/src/composables/useAppNav.ts` — 移除 alerts-v2 legacy route mapping
- `web/src/composables/useCrudPage.ts` — 新增 `normalizePageData` 适配器支持 `{list,total}` / `{items,count}` 双格式
- `web/src/composables/useCrudModal.ts` — 标记 `@deprecated`

**Batch 8: 用户侧改进**
- `internal/service/alert_rule.go` — 新增 `PreviewLabelValidation` 方法（dry-run 标签校验预览）
- `internal/handler/alert_rule.go` — 新增 `LabelValidationPreview` handler
- `internal/router/alert_routes.go` — 注册 `GET /alert-rules/label-validation-preview`
- `internal/service/notification_test.go` — 3 个 DB 集成测试（label matching / severity filter / batch update）
- `internal/service/schedule_test.go` — 2 个 DB 集成测试（shift CRUD / weekly rotation）
- `web/src/api/alert.ts` — 新增 `labelValidationPreview` API 方法
- `web/src/pages/settings/AISettings.vue` — 新增 "Preview Impact" 按钮 + 标签校验结果弹窗

---

## [v4.10.26] — 2026-05-19

### Fixed — 二轮审查修复（Batch 7-10）

**Batch 7: 紧急修复**
- `internal/repository/mute_rule.go` — `BatchUpdateEnabled`/`BatchDelete` 包裹 `db.Transaction` 事务边界
- `internal/repository/notify_rule.go` — 同上，防止批量操作部分提交
- `web/src/composables/index.ts` — 补 `useCrudPage` export（之前漏写导致导入报错）
- `web/src/pages/alerts/events/Index.vue` — `useFilterMemory` 补 `customRange` 持久化（timePreset='custom' 时日期丢失）

**Batch 8: 前端 CRUD 真采纳**
- `web/src/api/index.ts` — 1021 行 → barrel re-export（~70 行），引用 6 个域文件
- `web/src/pages/notification/AlertChannels.vue` — 迁移使用 `useCrudPage` composable
- `web/src/pages/alerts/mute/Index.vue` — 迁移使用 `useCrudPage` composable + preview 内联
- `web/src/pages/notification/Rules.vue` — 更新使用新 API 导入路径

**Batch 9: 遗留清理**
- 删除 `web/src/pages/alerts-v2/` 目录（Detail.vue + Index.vue，-898 行死代码）
- 路由中 `alerts-v2` redirect 保留（指向有效页面，兼容旧书签）

**Batch 10: 文档修正**
- MODULES.md：文件计数实测修正（34 model / 46 handler / 46 service / 34 repo / 268+ 端点）
- MODULES.md：4 个新模块完整条目（宠物/状态页面/预设规则/Alertmanager 导入）
- MODULES.md：测试覆盖表更新（7 个模块有测试标记）
- docs/api.md：补充 4 个新模块端点文档（§39-42）
- docs/PLAN-status.md：更新至 v4.10.25

### Added — 全栈审查 6 批次优化（首轮）

**Batch 1: 测试安全网（97 个测试函数）**
- `internal/engine/evaluator_test.go` — 19 个测试：状态序列化、生命周期、group_wait、recovery hold
- `internal/engine/suppression_test.go` — 26 个测试：severity 匹配、并发安全、端到端抑制场景
- `internal/service/notification_test.go` — 20 个测试：邮件构建、webhook 配置、路由匹配
- `internal/service/schedule_test.go` — 32 个测试：轮转计算、时区处理、值班查询

**Batch 2: 后端精简**
- 路由拆分：`admin_routes.go` → `datasource_routes.go` / `team_routes.go` / `setting_routes.go`
- 修复 `AlertEventService.DB()` 暴露：新增 `ListGrouped()` 服务方法替代直接 DB 访问
- 错误码统一：`internal/pkg/errors/codes.go` 集中管理（CodeInvalidParam/CodeForbidden/CodeUnauthorized 等）

**Batch 3: 前端 CRUD 通用化**
- `web/src/composables/useCrudPage.ts` — 通用 CRUD composable（封装分页+增删改查+Modal）
- `web/src/components/common/CrudListPage.vue` — 通用列表页面组件
- API 拆分：`api/index.ts`（1021 行）→ 6 个域文件（alert/notify/oncall/admin/data/incident）
- `usePaginatedList` 适配 `{list,total}` 和 `{items,count}` 两种响应格式
- 重构 Subscribe.vue 和 Rules.vue 使用新组件

**Batch 4: 前端交互体验**
- `web/src/utils/formRules.ts` — 13 个表单校验规则工厂（required/email/url/json/promql/severity 等）
- `web/src/components/common/ErrorRetry.vue` — 错误重试组件
- `web/src/composables/useFilterMemory.ts` — 筛选条件 localStorage 持久化
- 应用到告警事件和告警规则页面

**Batch 5: 文档对齐**
- MODULES.md：修正文件计数（34 model / 44 handler / 46 service / 34 repo / 173+ 端点）
- MODULES.md：补充 4 个缺失模块条目（宠物/状态页面/预设规则/Alertmanager 导入）
- api.md：移除已删除端点文档，补充新模块端点，章节重编号 1-38
- PLAN-status.md：更新至 v4.10.24

**Batch 6: AI 标签校验 + 批量端点**
- 标签语义校验：AlertRule 保存前校验 severity/job/instance 标签（可配置开关）
- NotifyRule 批量端点：`POST /notify-rules/batch/enable|disable|delete`
- MuteRule 批量端点：`POST /mute-rules/batch/enable|disable|delete`

**修改文件清单（40+ 文件）：**
- 后端：engine/*_test.go, service/*_test.go, router/*_routes.go, handler/alert_event.go, handler/handler.go, handler/metrics.go, handler/mute_rule.go, handler/notify_rule.go, repository/mute_rule.go, repository/notify_rule.go, service/alert_event.go, service/alert_rule.go, service/mute_rule.go, service/notify_rule.go, service/system_setting.go, pkg/errors/codes.go, cmd/server/wire.go
- 前端：api/alert.ts, api/notify.ts, api/oncall.ts, api/admin.ts, api/data.ts, api/incident.ts, api/index.ts, composables/useCrudPage.ts, composables/useFilterMemory.ts, components/common/CrudListPage.vue, components/common/ErrorRetry.vue, utils/formRules.ts, pages/alerts/events/Index.vue, pages/alerts/rules/Index.vue, pages/notification/Subscribe.vue, pages/notification/Rules.vue
- 文档：MODULES.md, CLAUDE.md, docs/api.md, docs/PLAN-status.md, CHANGELOG.md

---

## [v4.10.24] — 2026-05-19

### Removed — 删除 17 个孤立后端端点

清理前端零调用的后端端点，减少死代码和维护负担。

**Group 1 — v1 Notify Channels (6 endpoints):**
- 删除 `GET/POST /notify-channels`, `GET/PUT/DELETE /notify-channels/:id`, `POST /notify-channels/:id/test`
- 删除 `handler/notification.go` 整个文件（所有方法均为孤立端点）
- 删除 `NotificationHandler` struct + `NewNotificationHandler` 构造函数
- 删除 `service.NotificationService` 的 Channel CRUD 方法: `CreateChannel`, `GetChannel`, `ListChannels`, `UpdateChannel`, `DeleteChannel`, `TestChannel`
- 删除 `repository.NotifyChannelRepository` 的 CRUD 方法: `Create`, `List`, `Update`, `Delete`（保留 `GetByID` — 被 escalation_executor 和 noise_reducer 使用）

**Group 2 — v1 Notify Policies (5 endpoints):**
- 删除 `GET/POST /notify-policies`, `GET/PUT/DELETE /notify-policies/:id`
- 删除 `service.NotificationService` 的 Policy CRUD 方法: `CreatePolicy`, `GetPolicy`, `ListPolicies`, `UpdatePolicy`, `DeletePolicy`
- 删除 `repository.NotifyPolicyRepository` 的 CRUD 方法: `Create`, `GetByID`, `List`, `Update`, `Delete`（保留 `FindMatchingPolicies` — 被 `RouteAlert` 使用）

**Group 3 — Escalation Steps (3 endpoints):**
- 删除 `POST /escalation-policies/:id/steps`, `PUT/DELETE /escalation-policies/:id/steps/:stepId`
- 删除 `handler.ScheduleHandler` 的 `CreateEscalationStep`, `UpdateEscalationStep`, `DeleteEscalationStep` 方法
- 删除 `handler.CreateEscalationStepRequest`, `handler.UpdateEscalationStepRequest` 请求类型
- 删除 `service.ScheduleService` 的 `CreateEscalationStep`, `UpdateEscalationStep`, `DeleteEscalationStep` 方法（保留 `ListEscalationSteps` — 被 `GetEscalationPolicy` 使用）
- 删除 `repository.EscalationStepRepository` 的 `Create`, `Update` 方法（保留 `ListByPolicyID`, `Delete` — 被 engine 和 `DeleteEscalationPolicy` 使用）

**Group 4 — Label Registry Datasource Variants (2 endpoints):**
- 删除 `GET /label-registry/datasource-keys`, `GET /label-registry/datasource-values`
- 删除 `handler.LabelRegistryHandler` 的 `GetKeysByDatasource`, `GetValuesByDatasource` 方法
- 删除 `service.LabelRegistryService` 的 `GetKeysByDatasource`, `GetValuesByDatasource` 方法
- 删除 `repository.LabelRegistryRepository` 的 `GetKeysByDatasource`, `GetValuesByDatasource` 方法

**Group 5 — OIDC Settings Reload (1 endpoint):**
- 删除 `POST /settings/oidc/reload`
- 删除 `handler.OIDCSettingsHandler` 的 `Reload` 方法, `SetReloadFn` 方法, `reloadFn` 字段
- 删除 `cmd/server/wire.go` 的 `Dependencies.ReloadOIDC` 函数 + `SetReloadFn` 调用
- 更新 `UpdateConfig` 响应消息（不再提示调用 reload 端点）

**保留的代码（有其他调用者）:**
- `NotificationService` struct + `RouteAlert`/`SendNotification`/`processSubscriptions` — 被 alert routing pipeline 使用
- `NotifyChannelRepository.GetByID` — 被 escalation_executor 和 noise_reducer 使用
- `NotifyChannelRepository.ListByLabels` — 被 noise_reducer 使用
- `NotifyPolicyRepository.FindMatchingPolicies` — 被 `RouteAlert` 使用
- `EscalationStepRepository.ListByPolicyID` — 被 engine 和 service 使用
- `EscalationStepRepository.Delete` — 被 `DeleteEscalationPolicy` 使用
- `ListEscalationSteps` service method — 被 `GetEscalationPolicy` handler 使用
- `OIDCSettingsHandler.GetConfig`/`UpdateConfig` — 正常使用中
- `Handlers.Notification` 字段从 router.go 移除

---

## [v4.10.23] — 2026-05-18

### Fixed — 全栈 Review 批量修复（33 issues / 6 batches）

**Batch 1 — P0 引擎稳定性：**
- 修复 `syncRules` 并发更新 `rule.Version` 时的锁竞争（evaluator.go）
- 引擎 worker pool 4 处 `context.Background()` 改为可取消 ctx，shutdown 时正确退出
- `LabelRegistrySvc.StartSyncWorker` 改用 appCtx，shutdown 时 goroutine 不再泄漏
- 规则删除后 `suppressor.activeSeverities` 正确清理（新增 `RemoveRule` 方法）
- `eventRepo.Create/Update` 失败时状态回退为 pending/firing，下次评估重试（不再卡死）

**Batch 2 — P0 前端正确性：**
- Channel 星标切换改为乐观更新 + 失败回滚
- 通知规则删除从 `confirm()` 改为 Naive UI `dialog.warning()`
- Channel 类型补全 `mtta_label`/`mttr_label` 字段，移除 `as unknown` 不安全转型

**Batch 3 — P1 一致性与性能：**
- 引擎状态恢复时重建 suppressor 状态（含 pending 状态）
- 心跳检查 N+1 查询改为批量 `GetLatestByFingerprints`
- 未知 severity 不再绕过抑制，降级为 info 级别
- `SyncAll` 添加 atomic.Bool 并发保护
- AI chat 历史加载/清空错误从静默吞掉改为 console.error
- 通知规则 modal 关闭时自动 reset 表单

**Batch 4 — 配置集中 + 测试基线：**
- `METRICS_TOKEN` / `CORS_ALLOWED_ORIGINS` 从散落 `os.Getenv` 集中到 viper config
- `alert_channel_test.go` 4 个 TODO 占位替换为真实集成测试

**Batch 5 — P2 UX 清理：**
- 升级执行器 limit=10000 提取常量 + 超限时 warn 日志
- `GetFiringEvents` 添加 TODO(perf) 缓存优化注释
- 批量操作 `selectedKeys` 改为成功后才清空
- 批量删除添加确认弹窗
- `useCrudModal.closeModal` 完整重置 editingId/modalTitle
- MODULES.md 补全 6 个遗漏模块（预设规则/AI规则生成/宠物/状态页等）
- 审计发现 17 个孤儿后端端点（v1 通知 11 个 + 其他 6 个）

**Batch 6 — P3 文档与重构：**
- `appliedTemplateId` 添加使用说明注释
- 重定向 debounce 提取为 `REDIRECT_DEBOUNCE_MS` 常量
- MODULES.md 移除不存在的 `n9e-gap-analysis.md` 引用
- CHANGELOG.md v4.10.19 补充迁移文件编号

### Changed — 文件变更（25 files, +443 -102）

- `internal/engine/evaluator.go` — ctx 注入 + suppressor 清理 + GetFiringEvents TODO
- `internal/engine/rule_eval.go` — 可取消 ctx + 状态回退重试
- `internal/engine/suppression.go` — RemoveRule + severityRank 默认值
- `internal/engine/heartbeat_checker.go` — 批量查询重构
- `internal/engine/escalation_executor.go` — limit 常量 + warn
- `internal/config/config.go` — MetricsToken/CORSAllowedOrigins 字段
- `internal/handler/metrics.go` — 闭包工厂替代 os.Getenv
- `internal/middleware/cors.go` — 参数化 origins
- `internal/handler/label_registry.go` — atomic.Bool 并发保护
- `internal/handler/alert_channel_test.go` — 4 个真实集成测试
- `internal/repository/alert_event.go` — GetLatestByFingerprints 批量方法
- `cmd/server/main.go` + `wire.go` — appCtx 传递
- `web/src/pages/channels/Index.vue` — 乐观更新 + 类型补全
- `web/src/pages/notification/Rules.vue` — dialog + form reset
- `web/src/pages/alerts/rules/Index.vue` — 批量操作修复
- `web/src/composables/useCrudModal.ts` — closeModal 完整重置
- `web/src/composables/useAIChat.ts` — 错误日志

---

## [v4.10.22] — 2026-05-18

### Fixed — Bug 修复

- `preset_rules` 表缺少 `deleted_at` 列导致所有查询 500 内部错误（迁移: 000040）
- `preset_rule.go` List 响应缺少 `page`/`page_size` 字段，改用 `SuccessPage`
- `alert_rule.go` ListCategories 移除冗余 `deleted_at IS NULL`（GORM 自动添加）

### Added — AI 多供应商配置

- 后端: `AIProviderConfig` / `AIProvidersConfig` 结构体，支持多个命名供应商
- 后端: `GetProvidersConfig` / `SaveProvidersConfig` / `GetProviderConfig` 方法（AES-256-GCM 加密存储）
- 后端: `AIModule` 新增 `ProviderKey` 字段，每个模块可选择对接的供应商
- 后端: 3 个新 API 端点: `GET/PUT /ai/providers` + `POST /ai/test-provider`
- 前端: `AIProvider` / `AIProvidersConfig` 类型定义
- 前端: AISettings 页面重构 — 供应商管理器（增删改查 + 设默认）+ 模块供应商选择器
- 前端: `useAIModule` composable 新增 `getProviderForModule` / `isProviderEnabled`
- 向后兼容: 无供应商配置时回退到传统单供应商模式

### Improved — UI 一致性

- 所有 ~35 个侧边栏菜单项统一添加 `@vicons/ionicons5` 图标
- 首页快捷入口新增「预置规则库」和「AI 模块配置」
- 路由 meta + 侧边栏标签改为 i18n key（v4.10.21 遗留）

### Changed — 文件变更（12 files, +811 -155）

- `internal/service/system_setting.go` — 多供应商类型 + 存储逻辑
- `internal/service/ai.go` — 供应商解析层
- `internal/handler/ai.go` — 3 个新端点
- `internal/router/admin_routes.go` — 注册新路由
- `internal/handler/preset_rule.go` — SuccessPage 修复
- `internal/repository/alert_rule.go` — 冗余条件移除
- `web/src/pages/settings/AISettings.vue` — 供应商管理 UI 重构
- `web/src/composables/useAppNav.ts` — 全量图标
- `web/src/pages/dashboard/UnifiedDashboard.vue` — 新快捷入口
- `web/src/composables/useAIModule.ts` — 供应商感知
- `web/src/types/ai-module.ts` — 新类型
- `web/src/api/index.ts` — 新 API 调用
- 迁移: 000040_add_deleted_at_to_preset_rules

---

## [v4.10.21] — 2026-05-18

### Fixed — Code Review 全量修复（27 files, +251 -904）

**P0 安全/正确性：**
- `/ai/rules` 4 个端点添加 `operate` RBAC 中间件（防止 viewer 无限调用 LLM）
- 抑制规则 AI 生成改用正则 `=~".*"` 匹配器（原空字符串语义错误）
- AI 规则创建 `datasource_id: null` 改为 `?? undefined`

**P1 代码质量：**
- 统一 `getErrorMessage(err)` 替代不安全的 `(err as Error).message`（rules + inhibition，~10 处）
- `json.Marshal` 错误不再被 `_` 吞掉（preset_rule + alertmanager_import，3 处）
- `page/pageSize` 添加下界校验（handler/preset_rule）
- `label_registry` ctx 透传到 repository 层
- `Presets.vue` searchTimer 添加 `onUnmounted` 清理
- 删除未使用的导入和计算属性（Presets.vue、AISettings.vue）

**P2 改善：**
- 提取 `readYAMLInput` helper 消除 3 处 YAML 解析重复
- `rule_generator` stop-word map 提取为包级变量
- 魔法数字 `10401` 改为 `apperr.ErrDuplicateName.Code`
- 提取 `@/utils/severity.ts` 共享 severity 辅助函数
- `rules/Index.vue` 移除不必要的 `DynamicScroller`（50 条数据用普通 v-for）
- 拆分 `types/preset-rule.ts` → `preset-rule.ts` + `ai-module.ts`（关注点分离）
- 路由 meta + 侧边栏硬编码中文改为 i18n key
- 分类侧边栏 `<a>` 改为 `<button>` 提升可访问性
- AI 置信度显示添加颜色编码（绿 ≥80% / 黄 ≥50% / 红 <50%）
- 删除 `useAIModule.ts` 冗余 `isAIAvailable()`，统一用 `globalEnabled`
- `AISettings.vue` / `api/preset-rules.ts` 统一从 `@/api` 导入

**死代码清理：**
- 删除 `scripts/import-preset-rules.go`（546 行一次性脚本 + Makefile targets）
- 删除 `ImportPresets` 死端点（零前端调用）+ 相关 service/router 代码（~80 行）

---

## [v4.10.20] — 2026-05-18

### Fixed — TypeScript 类型错误修复（22 处）

vue-tsc --noEmit 全量类型检查修复，涉及 13 个文件：

- `types/index.ts` — AlertRuleTemplate 新增 group_name 字段；DropdownOption.icon 返回类型修正为 VNodeChild
- `RuleFormModal.vue` — datasource_type 空字符串转 undefined 兼容
- `PanelCard.vue` — statColor 返回类型、thresholds 类型断言、数值运算类型
- `alerts-v2/Index.vue` — SelectMixedOption null→undefined
- `mute/Index.vue` — AlertEvent 属性访问修正（rule?.name / annotations?.summary）
- `DispatchConfig.vue` — 升级策略选项 null→undefined
- `IncidentDashboard.vue` — IncidentStats 类型断言
- `UnifiedDashboard.vue` — icon: null→undefined
- `explore/Index.vue` — formatLabelsStr 参数类型 Record<string, unknown>
- `incidents/Detail.vue` — DropdownMixedOption 类型兼容
- `AlertChannels.vue` — 模板选项 null→undefined
- `Templates.vue` — h() 返回类型修正
- `BizGroupManagement.vue` — TreeNodeRow 递归类型打破循环引用

---

## [v4.10.19] — 2026-05-18

### Added — AI 智能规则引擎（Phase A-D）

**Phase A: 预置规则库 + AI 模块配置**
- `model/preset_rule.go` — PresetRule 模型（迁移 000038）
- `repository/preset_rule.go` + `service/preset_rule.go` + `handler/preset_rule.go` — CRUD + YAML 导入 + 一键应用
- `scripts/import-preset-rules.go` — 从 monitoring-trading 导入 299 条预置规则（6 类：infrastructure/kubernetes/middleware/database/probe/windows）
- `service/system_setting.go` — AIModuleConfig 5 模块独立开关（platform/chat/rule_gen/analysis/agent）
- `handler/ai.go` — GET/PUT /ai/modules 端点
- 前端 `/alert/presets` — 预置规则库页面（分类筛选 + 搜索 + 一键应用 + YAML 导入）
- 前端 `/platform/ai-settings` — AI 模块配置页面（总开关 + 5 模块开关 + 连接测试）
- `composables/useAIModule.ts` — AI 模块状态 composable（控制 UI 工况）

**Phase B: AI 规则生成引擎**
- `service/rule_generator.go` — RuleGeneratorService（Context Builder + LLM prompt + label_registry 上下文 + 预置规则参考）
- `handler/ai_rule.go` — 4 个端点：
  - `POST /ai/rules/generate` — 口述生成告警规则
  - `POST /ai/rules/validate` — PromQL 实时验证（dry-run 查询数据源）
  - `POST /ai/rules/suggest-labels` — AI 标签推荐
  - `POST /ai/rules/generate-inhibition` — 口述生成抑制规则
- 前端 alerts/rules/Index.vue — AI 生成按钮 + 对话式创建模态框 + 结果预览
- 前端 alerts/inhibition/Index.vue — 抑制规则 AI 生成入口

**Phase C: 传统平台兼容**
- `service/alertmanager_import.go` — Alertmanager YAML 解析 + Channel/InhibitionRule 创建
- `handler/alertmanager_import.go` — POST /integrations/import-alertmanager + import-alertmanager-presets
- `service/preset_rule.go` — ImportPresetInhibitions：13 条内置抑制规则预置模板
- `router/admin_routes.go` — 新增 /integrations 路由组

**Phase D: 多数据源标签增强**
- `model/label_registry.go` — 新增 Source 字段（sync/event/manual）
- 迁移 000039 — ALTER TABLE label_registry ADD COLUMN source
- `handler/label_registry.go` — 新增 /label-registry/datasource-keys + datasource-values 端点
- 支持按数据源查询标签，为 AI 规则生成提供精准上下文

### DB 迁移

- 迁移: 000038_create_preset_rules, 000039_add_source_to_label_registry

---

## [v4.10.18] — 2026-05-18

### Improved — 前端架构全面升级

**P1-2: usePaginatedList composable 全量迁移（12 页）**
- 剩余 7 个列表页迁移到 `usePaginatedList`：incidents/Index、incidents/PostMortems、channels/Index、dashboard-v2/Index、notification/AlertChannels、alerts-v2/Index、settings/AuditLog
- 统一分页模式：`extraParams` 回调 + `fetchList`/`refresh` + `:item-count`/`:page-size` 绑定

**P1-8: TypeScript any 全面清零（209 处 → 0）**
- 54 个前端文件中的所有 `any` 类型替换为具体类型定义
- `api/index.ts`（26 处）、alerts/events/Detail.vue（10 处）等重灾区全部修复
- 新增 `types/` 目录下的类型定义文件

**P1-10: vue-virtual-scroller 真实虚拟滚动**
- `env.d.ts` — 新增 `vue-virtual-scroller` TypeScript 声明
- `main.ts` — 引入 `vue-virtual-scroller/dist/vue-virtual-scroller.css`
- alerts/events/Index、alerts/rules/Index、alerts/history/Index — 使用 `DynamicScroller` + `DynamicScrollerItem` 替代 CSS containment hack
- 支持动态高度（`size-dependencies`），自动回收离屏 DOM 节点

### Refactored — 后端架构优化

**P2-2: setter 注入全部转为构造函数参数（12 处/6 文件）**
- `service/lark.go` — `NewLarkService` 新增 `settingSvc` 参数
- `service/notification.go` — `NewNotificationService` 新增 `eventRepo`/`subscribeSvc`/`notifyRuleSvc` 参数
- `service/larkbot.go` — `NewLarkBotService` 新增 `userRepo` 参数
- `service/alert_event.go` — `NewAlertEventService` 新增 `notifySvc`/`onCallSvc`/`larkSvc`/`workerPool` 参数
- `engine/escalation_executor.go` — `NewEscalationExecutor` 新增 `larkSvc`/`settingSvc`/`ruleRepo` 参数
- `cmd/server/wire.go` — 重排初始化顺序，消除所有 Set* 调用

**P2-8: MODULES.md 同步校验脚本**
- `scripts/check-modules.go` — 基于 Go AST 的精确计数校验
- `scripts/check-modules.sh` — CI 环境的轻量版
- `Makefile` — 新增 `check-modules` target
- MODULES.md 头部计数修正：46 model / 41 handler / 37 service / 42 repository

---

## [v4.10.17] — 2026-05-18

### Added — SSRF 防护 + 业务指标 + goroutine 背压

**P1-4: SSRF 防护**
- `pkg/safehttp/client.go` — `SafeTransport` 阻止 loopback/link-local/RFC1918/ULA/metadata 等内网地址外连
- 9 个文件 11 处 `http.Client` 替换为 `safehttp.NewSafeClient()`（datasource/lark/notification/ai 等）
- `pkg/safehttp/client_test.go` — 19 个测试覆盖所有阻断场景

**P1-7: /metrics 鉴权 + Prometheus 业务指标**
- `handler/metrics.go` — 新增 `METRICS_TOKEN` 环境变量鉴权，未设置时保持向后兼容
- `pkg/metrics/metrics.go` — 3 个 Prometheus 计数器：`sreagent_alerts_evaluated_total`、`sreagent_notifications_sent_total`、`sreagent_escalation_steps_total`
- `engine/rule_eval.go` — 规则评估结果写入指标
- `service/notification.go` — 通知成功/失败写入指标
- `engine/escalation_executor.go` — 升级步骤执行写入指标

**P1-5: goroutine 背压**
- `service/audit_log.go` — `dispatchSem` (cap 50)，异步审计日志写入有界
- `service/integration.go` — `dispatchSem` (cap 100)，集成回调有界
- `service/alert_v2_pipeline.go` — `dispatchSem` (cap 100)，v2 pipeline 异步处理有界

**P1-6: 批量 GetByIDs 接口**
- `repository/alert_event.go`、`channel.go`、`user.go`、`schedule.go` — 新增 `GetByIDs(ctx, ids)` 方法，`WHERE id IN ?`

**P1-2: usePaginatedList composable 迁移**
- `composables/usePaginatedList.ts` — 新建通用分页 composable（loading/items/total/page/fetchList/refresh）
- 5 个列表页迁移到 composable：alerts/events、alerts/rules、alerts/history、settings/UserManagement、settings/TeamManagement

**P1-8: TypeScript any 清理**
- explore/Index.vue、PanelCard.vue 等重灾区文件类型化
- 新增 `types/query.ts` 类型定义

**P1-10: 虚拟滚动**
- explore/Index.vue — `n-data-table` 启用 `virtual-scroll`

### Fixed — 分层修复 + 安全加固

**P2-1: TeamRepository 分层修正**
- 删除 `service/team.go` 中的重复 repository 实现（84 行）
- `repository/team.go` 补齐 `GetByName`、`GetMember`、`Preload("Members")` 等方法
- `cmd/server/main.go` 改用 `repository.NewTeamRepository(db)`

**P2-3: dispatchSem 全局变量改为实例级**
- `service/alert_event.go` — `var dispatchSem` 移除，改为 `AlertEventService.dispatchSem` 实例字段

**P2-4: testutil SQL 注入修复**
- `testutil/testutil.go` — `DELETE FROM` 表名改用反引号包裹

**P2-6: alert_rule_template 限制 pageSize**
- `handler/alert_rule_template.go` — `pageSize` 上限 100

### Refactored — DI 拆分 + OIDC 热重载 + 路由拆分

**P1-1: DI wiring 从 main.go 拆分到 wire.go**
- `cmd/server/wire.go` — 新增 `Dependencies` 结构体 + `initDependencies()` 函数，承载所有 repo/service/handler/engine 初始化
- `cmd/server/wire.go` — 新增 `Shutdown()` 方法，按正确顺序停止所有后台组件
- `cmd/server/main.go` — 从 778 行精简至 275 行，仅保留 config/logger/DB/migration/graceful shutdown

**P1-9: OIDC 热重载**
- `handler/oidc.go` — `OIDCHandler` 新增 `SetService()` 方法，支持运行时替换 OIDC 服务
- `handler/oidc_settings.go` — 新增 `Reload` 端点 + `SetReloadFn` 回调注入
- `cmd/server/wire.go` — `Dependencies.ReloadOIDC()` 从 DB 重新读取 OIDC 配置并重建服务
- `router/router.go` — 新增 `POST /api/v1/settings/oidc/reload`（admin only）

**P2-5: router.go 拆分为按模块文件**
- `router/auth_routes.go` — 用户 profile + OIDC 设置路由
- `router/alert_routes.go` — AlertRule + AlertEvent + AlertV2 + Heartbeat 路由
- `router/notify_routes.go` — 通知规则/媒体/模板/订阅/通道/策略路由
- `router/schedule_routes.go` — 值班排班 + 升级策略路由
- `router/admin_routes.go` — 数据源/用户/团队/静默/抑制/分组/审计/设置/Dashboard/AI 等管理路由
- `router/router.go` — 从 673 行精简至 193 行，仅保留 middleware/health/public routes/registrar 调用

---

## [v4.10.16] — 2026-05-18

### Fixed — 安全修复 + 性能优化 + 测试补全

**P0-1: JWT 算法混淆攻击防护**
- `middleware/auth.go` — `ParseToken` 的 keyFunc 新增 HMAC 签名方法校验，拒绝 non-HMAC 算法（如 `none`、RSA），与 `ParseTokenIgnoreExpiry` 行为一致

**P0-2: 告警链路内存视图替代 DB 全表扫描**
- `engine/evaluator.go` — 新增 `GetFiringEvents()` 和 `GetFiringAlertEvents()` 方法，从内存 states map 获取 firing 告警
- `cmd/server/main.go` — `onAlertFn` 中 evaluator 可用时走内存路径，避免每次告警扫描 2000 行 DB

**P0-5: handler.Error 吞错增加日志**
- `handler/handler.go` — 未知错误现在通过 request-scoped zap logger 记录，不再静默返回 500
- `handler/handler.go` — `gorm.ErrRecordNotFound` 特判为 404 而非 500
- `middleware/logger.go` — RequestLogger 将 zap logger 注入 gin context 供 handler 使用

**P1-3: CORS release 模式安全加固**
- `middleware/cors.go` — release 模式下若未设置 `CORS_ALLOWED_ORIGINS`，不再默认放行 localhost，返回空 origins 列表

**P2-7: release 模式 admin 密码强制**
- `cmd/server/main.go` — `GIN_MODE=release` 时若未设置 `SREAGENT_ADMIN_PASSWORD`，直接 Fatal 退出而非使用默认密码

**P2-9: gin.Recovery 接入 zap**
- `router/router.go` — 替换 `gin.Recovery()` 为自定义 recovery middleware，panic 信息写入 zap 而非 stderr

**P0-6: 首批关键单元测试（37 个用例，5 个测试文件）**
- `middleware/auth_test.go` — JWT 解析、过期、密钥错误、算法校验（4 个测试）
- `handler/handler_test.go` — Error 响应码、AppError、RecordNotFound（3 个测试）
- `service/encryption_test.go` — AES-256-GCM 加解密 roundtrip、随机 nonce、向后兼容（5 个测试）
- `service/inhibition_rule_test.go` — 抑制规则匹配、源/目标标签、EqualLabels 约束（5 个测试）
- `engine/rule_eval_test.go` — 指纹生成、状态机转换 pending→firing→resolved、duration 解析（8 个测试）

### Fixed — 升级执行器性能优化 + 去重机制加固

**P0-3: 修复全表扫描 + N+1 查询**
- `escalation_executor.go` — `runOnce` 改用 `ListFiringForEscalation` 只查 firing 事件，不再全表扫描 10000 条
- `escalation_executor.go` — 新增 `batchLoadRules` 方法，SLA 检查从逐条 `GetByID` 改为单次 `GetByIDs` 批量查询
- `repository/alert_event.go` — 新增 `ListFiringForEscalation(ctx, limit)` 方法，`WHERE status='firing' ORDER BY fired_at ASC LIMIT ?`
- `repository/alert_rule.go` — 新增 `GetByIDs(ctx, ids)` 方法，`WHERE id IN ?`

**P0-4: 升级去重从 Note 字符串改为 EscalationStepID**
- `model/alert_event.go` — AlertTimeline 新增 `EscalationStepID *uint` 字段（indexed）
- `escalation_executor.go` — `executedStepOrders` 主键改为 `step:<id>` 格式，Note 文本仅作旧数据兜底
- `escalation_executor.go` — `recordTimeline` 新增 stepID 参数，执行步骤时写入 EscalationStepID
- 迁移: 000037_add_escalation_step_id_to_alert_timelines

## [v4.10.15] — 2026-05-16

### Changed — 首页 widget 自定义增强 + 3 个新组件

**新增 widget：**
- 值班人员（oncallSchedule）：显示各排班当前值班人，调用 `scheduleApi.getCurrentOnCall`
- 置顶便签（pinnedItems）：用户自建书签，支持添加/编辑/删除/选色，存 `sre-home-pinned`
- 快捷入口升级为可自选：10 个内置入口，用户可逐个开关，存 `sre-home-quick-links`

**设置面板升级：**
- 3 个标签页：组件排序、快捷入口选择、置顶便签管理
- 便签支持内联编辑（名称 + URL + 颜色选择器）
- 每个标签页独立重置按钮

**i18n：**
- 新增 15 个键：`homepage.tabWidgets/tabQuickLinks/tabPinned/oncallSchedule/onDuty/noOnCall/pinnedItems/noPinned/addViaSettings/pinTitle/pinUrl/addPin/resetLinks/resetPinned`

## [v4.10.13] — 2026-05-16

### Changed — 首页全面重构：插件化 widget 系统 + 全宽 bento 布局

**首页架构重构：**
- 首页不再显示侧边栏菜单（`useAppNav` 返回空菜单），保留左侧 Rail 用于 App 切换
- `AppShell.vue`：首页隐藏侧边栏和页面标题栏，内容区全宽铺满
- 移除 `max-width: 1100px` 限制，改为 `max-width: 1400px` 自适应容器

**插件化 Widget 系统：**
- 用户可通过右上角「自定义」按钮管理首页小组件
- 支持添加/删除/排序 widget，配置持久化到 `localStorage`（key: `sre-home-widgets`）
- 5 个内置 widget：问候语、模块状态、我的待办、最近活动、快捷入口
- 设置面板：显示/隐藏切换、上下排序、重置为默认布局

**Bento 网格布局：**
- 2 列自适应网格，问候语和快捷入口跨满全宽
- 我的待办 + 最近活动并排显示（各占 1 列）
- 模块状态 4 卡片等宽排列

**视觉升级：**
- 问候卡片：橙色渐变背景 + 装饰性半透明圆形
- 模块图标：每个模块独立配色（Monitor 蓝、Oncall 粉、Deploy 绿、AI 紫）
- 快捷入口：彩色图标 + 悬停上浮动画
- 状态指示点：绿/琥珀/红三色 + 光晕效果

**i18n：**
- 新增 4 个键：`homepage.customize`、`homepage.widgetSettings`、`homepage.resetLayout`、`homepage.widgetGreeting`

## [v4.10.12] — 2026-05-16

### Fixed — 遗留路由清理（13 文件 23 处路径修复）

所有内部 `router.push` / `router-link` 目标从遗留路径迁移到规范路径：
- `/incidents/` → `/oncall/incidents/`
- `/channels/` → `/oncall/spaces/`
- `/alerts/rules/` → `/alert/rules/`
- `/alerts/events/` → `/alert/events/`
- `/notification` → `/alert/notify/policies`
- 命名路由改为路径字符串（dashboard-v2）

## [v4.10.10] — 2026-05-15

### Fixed — 首页不可达 + i18n 硬编码清理

**首页不可达修复：**
- Login.vue：登录后默认跳转从 `/oncall/overview` 改为 `/`（用户之前根本到不了首页）
- useAppNav.ts：新增 `'home'` 应用状态，`/` 路径不再默认为 oncall
- 问候语 i18n：逗号从硬编码中文改为 i18n 参数 `{name}`，英文 locale 下使用英文逗号

**i18n 硬编码修复：**
- mute/Index.vue："Once" → `t('mute.oneTime')`，"Periodic" → `t('mute.periodic')`
- rules/Index.vue："for" → `t('alert.forPrefix')`
- explore/Index.vue：CSV 导出表头改为 `t()` 调用
- 新增 `alert.forPrefix`、`query.csvTimestamp/csvMessage/csvLabels/csvName/csvValue` 键

## [v4.10.9] — 2026-05-15

### Fixed — 首页设计合规 + i18n 完善

**impeccable 设计审查修复（5 项违规）：**
- 移除模块卡片左侧 3px 彩色竖条（违反"禁止侧边彩条"规则）
- Monitor 卡片改为跨 2 列的主卡片布局（打破重复网格）
- 模块图标统一使用品牌橙色 `--sre-primary-soft`（消除 4 色竞争）
- 字体层级比例修正：`.mod-status` 和 `.task-meta` 降至 `--sre-fs-2xs`，`.task-title` 升至 `--sre-fs-md`（比例 ≥ 1.30）
- 活动文本破折号（—）改为冒号（:）

**快捷入口优化：**
- 6 个导航按钮各添加对应图标（DocumentText、Calendar、Search、StatsChart、Notifications、Shield）

**i18n 修复：**
- 问候语从硬编码中文改为 i18n 键（`greetingMorning` / `greetingAfternoon` / `greetingEvening`）
- 模块状态文字从硬编码 `"活跃"` 改为 `t('homepage.nActive', { count })`
- 错误提示从硬编码 `"Load failed"` 改为 `t('homepage.loadFailed')`
- 新增 5 个 i18n 键到 zh-CN.ts 和 en.ts

## [v4.10.8] — 2026-05-15

### Changed — 平台首页完全重写

**首页重构（UnifiedDashboard.vue）：**
- 从"KPI+图表仪表盘"重构为"平台运营中心"
- 问候条：用户名 + 引擎运行状态 + uptime
- 模块健康卡片：Monitor / Oncall / Deploy Agent / AI Agent 四模块状态
- Deploy Agent 和 AI Agent 为"即将推出"占位状态
- 我的待办：跨模块活跃故障列表（最多 5 条），带严重等级标签
- 最近活动：合并故障+告警事件的时间线（最多 10 条）
- 快捷入口：6 个导航按钮
- 模块命名统一英文：Monitor、Oncall、Deploy Agent、AI Agent
- 移除所有趋势图和 KPI 数字卡片（留给子模块仪表盘）

**i18n：**
- 新增 `homepage` 命名空间（30+ 键），覆盖问候、模块状态、待办、活动、导航

## [v4.10.7] — 2026-05-15

### Fixed — 橙色疲劳修复 + i18n 硬编码清理 + 主页路由修正

**颜色系统修复：**
- `--sre-bg-hover` / `--sre-bg-active` / `--sre-bg-subtle` 从橙色调改为中性暖灰
- 修复前：`rgba(249, 115, 22, 0.04/0.07/0.02)` 全局橙色 hover 反馈导致视觉疲劳
- 修复后：`rgba(28, 25, 23, 0.04/0.07/0.02)` 中性暖灰，橙色仅保留在主按钮和导航激活态

**i18n 修复：**
- OIDCConfig.vue: 页面标题 `SSO / OIDC` → `t('settings.oidcConfig')`
- OIDCConfig.vue: 角色选项 admin/team_lead/member/viewer 使用 `t()` 翻译
- AlertChannels.vue: `message.error('Copy failed')` → `t('common.copyFailed')`
- 新增 `common.copyFailed` 中英文键
- UnifiedDashboard.vue: `t('menu.rules')` → `t('menu.alertRules')`（修复缺失键）
- Index.vue (alert dashboard): 同步修复 `menu.rules` → `menu.alertRules`
- 新增 `menu.dashboards` 中英文键

## [v4.10.6] — 2026-05-15

### Added — 统一主页仪表盘 + i18n 完善

**统一主页（UnifiedDashboard）：**
- 新增 `/` 入口主页，替代原 `/oncall/overview` 重定向
- 6 个 KPI 卡片：活跃故障、活跃告警、今日关闭、严重活跃、MTTA P50、MTTR P50
- 告警趋势 SVG 面积图 + 严重等级分布堆叠条
- 故障趋势 CSS 柱状图 + Top 噪音规则列表
- 快捷操作网格：6 个导航卡片直达各模块
- 6 个并行 API 调用（Promise.allSettled），响应式 bento grid 布局
- 入场交错动画，支持 1200px / 768px 断点

**i18n 国际化修复：**
- 修复 23 个文件中的硬编码英文字符串，新增 40+ i18n 键
- commandPalette（18 键）、timeRangeOptions（11 键）、tooltip（11 键）
- 统一 `relTime()` 函数支持 vue-i18n `t()` 参数
- MTTA/MTTR 标签、Pet 等级前缀、查询错误提示等全部 i18n 化

**路由变更：**
- 根路径 `/` 现在渲染 UnifiedDashboard.vue（原为重定向到 /oncall/overview）
- 登录后跳转、OIDC 回调均指向 `/`
- `/oncall/overview` 和 `/alert/overview` 仍保留为各模块详细仪表盘

## [v4.10.5] — 2026-05-15

### Fixed — 配色系统重构 + 回退过度装饰

**配色重构（遵循 PRODUCT.md "Warm neutrality" 原则）：**
- 背景从 `#FFFBF7`（暖奶油色）改为 `#FAFAF9`（中性暖灰，stone-50）
- 主文本从 `#111827` 改为 `#1C1917`（stone-900，更深沉）
- 次要文本从 `#6b7280` 改为 `#57534E`（stone-600，对比度提升）
- 三级文本从 `#9ca3af` 改为 `#78716C`（stone-500，WCAG AA 达标）
- 静音文本从 `#d1d5db` 改为 `#A8A29E`（stone-400）
- Naive UI 主题覆盖同步更新

**回退违反 PRODUCT.md 的装饰性改动：**
- 移除 main-content 暖色径向渐变背景（anti-reference: decorative overload）
- 移除 dashboard 卡片 ::before 渐变覆盖层（anti-reference: gradient accent lines）
- 移除侧边栏图标浮动动画（anti-reference: animation for animation's sake）
- 保留：卡片 hover 浮起、交错入场动画、页面过渡（符合 "subtle spring easing"）

## [v4.10.4] — 2026-05-15

### Changed — UI 视觉提升 + AI 聊天框全屏

**AI 聊天框：**
- 新增全屏切换按钮（右上角最大化/最小化图标）
- 全屏模式下 drawer 占满整个视口宽度
- i18n 新增 `fullscreen` / `exitFullscreen` 键

**Logo 重新设计：**
- 背景从绿色渐变改为暖橙色系（`#FB923C` → `#F97316` → `#EA580C`）
- 添加 SVG glow 滤镜增强 "S" 字母质感
- 告警圆点改为金黄色 `#FBBF24` + 脉冲光环

**头像库美化：**
- PetAvatar 所有 8 种宠物添加柔和渐变背景圆
- 改进阴影效果：双层 drop-shadow 增强立体感
- UserAvatar SVG 添加 radialGradient 高光 + 增强阴影

**动画与过渡：**
- Dashboard bento 卡片添加交错入场动画（400ms，60ms 间隔）
- 卡片 hover 增加暖橙渐变光泽覆盖层
- 页面过渡改为 scale + opacity 组合（280ms spring easing）
- 主内容区添加微妙暖色径向渐变背景
- 侧边栏活跃图标增加浮动动画（3s 周期）

## [v4.10.3] — 2026-05-15

### Fixed — 字体离线化

- 移除 Google Fonts 外部依赖（Plus Jakarta Sans），改用本地 `@fontsource/inter`
- `index.html` 删除 `fonts.googleapis.com` 和 `fonts.gstatic.com` preconnect/link 标签
- `global.css` 字体栈更新：Inter（本地）+ Segoe UI + 系统字体回退
- `App.vue` Naive UI 主题 fontFamily 同步更新

## [v4.10.0] — 2026-05-14

### Changed — 全平台 UI 视觉重构：暖橙主题 + Bento 布局 + 春季动画

**配色系统重构：**
- 全局主色从 teal `#0d9488` 迁移至暖橙 `#F97316`
- App.vue Naive UI 主题覆盖全部使用暖橙色系
- 新增伴侣色 CSS 变量：`--sre-rose-light`、`--sre-emerald-light`、`--sre-violet-light`、`--sre-mint`
- 侧边栏 accent-soft RGBA 值对齐新品牌色（oncall `#F43F5E`、alert `#3B82F6`、platform `#8B5CF6`）
- mascot-fox.svg 内耳颜色从 `#0d9488` 更新为 `#F97316`
- UserAvatar.vue 调色板更新为新品牌色

**布局重构：**
- Index.vue 和 IncidentDashboard.vue 改为 12 列 CSS Grid bento 布局
- 卡片移除装饰性渐变 `::before` 线条
- 响应式断点：1200px（8+4）、768px（单列）

**动画系统：**
- 新增 `--sre-ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1)` 弹簧缓动
- AppRail 图标 hover 弹跳放大 + active 缩放
- AppSidebar 指示器 scaleY 入场动画 + `transform-origin: top`
- AppShell 页面过渡改为 opacity + transform

**可访问性：**
- Index.vue action-btn/rule-item 添加 `role="button"` + `aria-label`
- tooltip 从硬编码 `#1C1917` 改为 CSS 变量，适配深色模式
- `.sev-seg` transition 从 `flex` 改为 `flex-grow` 避免布局抖动

**其他：**
- AppShell topbar 添加暖色阴影
- AppShell/AppSidebar 使用 display font
- Login.vue mesh blob 更新为暖橙色调
- i18n 补齐 `dashboard.quickActions` 中英文 key
- App.vue border-radius 添加 CSS 变量关联注释

---

## [v4.9.7] — 2026-05-13

### Fixed — 协作空间国际化 + 状态页面 CRUD

**协作空间 (Channels) 国际化：**
- 移除 Detail.vue 中 `|| 'Overview'` / `|| 'Settings'` 硬编码回退
- 修复 `auto_close_minutes` 后硬编码 "min"，改为 i18n `autoCloseMinutesUnit`
- 补齐 `channel.deleteDesc` 中英文 key

**状态页面 (StatusPage) CRUD 管理：**
- 新增"管理服务"入口，支持直接在状态页面增删改查服务
- 管理模态框：服务列表 + 编辑/删除操作
- 创建/编辑表单：名称、状态、描述、图标、链接、排序
- 空状态直接提供"添加服务"按钮
- statusOptions 改为 i18n computed
- 新增 13 个 statusPageModule 中英文 key

## [v4.9.6] — 2026-05-13

### Fixed — 状态页面 + AI 聊天面板 + 国际化清理

**StatusPage：**
- 移除"即将上线"预览文案，改为展示真实服务状态
- 移除 feature cards 营销内容
- 改为"订阅通知"文案

**AIChatPanel：**
- 重构布局：chat-body flex column 结构，消息区自动撑满
- 输入框区域固定在底部，带 border-top 分隔
- textarea autosize 调整为 minRows:2 / maxRows:6

**国际化：**
- 清理 10+ 处不必要的 `|| 'fallback'` 模式
- 修复 2 处硬编码 aria-label（Search / Change language）
- 补齐 statusPageModule 新键（currentStatus / subscribe / noServicesHint）

## [v4.9.5] — 2026-05-13

### Security — 安全加固 + 代码审查 + 品牌重命名

**安全修复：**
- 修复 JWT ParseTokenIgnoreExpiry 未验证签名算法（CRITICAL）
- 修复 OIDC CallbackJSON state 验证可选导致 CSRF 绕过（HIGH）
- 新增全局请求体大小限制 10MB（HIGH）
- 新增 Datasource Endpoint SSRF 防护（MEDIUM）
- 移除 Webhook Secret URL 查询参数支持（MEDIUM）
- 修复 Integration Token 日志泄露（MEDIUM）
- 修复 Lark Bot Config Secret 部分泄露（MEDIUM）

**代码质量：**
- 修复 GetByFingerprint 缺少软删除过滤
- 修复 Goroutine 使用 request context 导致后台任务中断
- 删除死代码：itoa() 包装函数、废弃 fmt.Sprintf
- JWT 签名方法验证添加 fmt 导入

**UI/前端：**
- 品牌重命名：Nexus → SREAgent（7 处）
- 更新 tagline：一站式 DevOps 解决方案平台 · 具身 Agent
- 补齐 en.ts rolesModule 缺失翻译键（14 个）
- 补齐 dashboard.lastSync、profile.selectAvatar 等缺失翻译
- 4 处 window.confirm 替换为 Naive UI useDialog
- AppShell 添加 768px 响应式断点（移动端隐藏侧边栏）

## [v4.9.2] — 2026-05-13

### Fixed — 代码审查修复 + 公开状态页端点

- 移除 global.css 中已删除的 sre-jelly 动画残留引用
- 补全 en.ts statusPageModule + rolesModule 英文翻译
- 补全 zh-CN.ts statusPageModule.invalidEmail 翻译键
- 修复 bizGroup.created 重复键（toast 改为 createSuccess）
- 新增公开端点 `GET /api/v1/status-services`（无需认证）
- 修复 StatusService.GetByID 错误遮蔽（区分 not-found 和 DB 错误）
- 修复 StatusPage.vue 验证消息键 + 移除内联 fallback
- AIChatButton 补充呼吸灯动画效果

## [v4.8.0] — 2026-05-12

### Added — AI Chat + 宠物系统 + UI 动画系统

**AI Chat（AI 对话）：**
- 右下角浮动按钮，点击展开 400px 右侧抽屉
- 三种模式：告警分析 / 通用对话 / 宠物对话
- 多轮对话上下文（自动加载最近 20 条历史）
- 后端：`POST /ai/chat`、`GET /ai/history`、`DELETE /ai/history`
- 新增 `chat_histories` 表（迁移: 000034）
- 组件：AIChatPanel、AIChatMessage、AIChatButton
- Composable：useAIChat

**宠物系统（Pet System）：**
- 狐狸宠物，自动创建（名字"小狐"）
- 喂食（饥饿 -20，经验 +5）、玩耍（心情 +15，经验 +5）
- 升级公式：所需经验 = 等级 × 100
- 互动历史记录
- 角落常驻显示（PetCorner）+ 弹出面板（PetPanel）+ 独立详情页（/pet）
- 后端：`GET/PUT /pet`、`POST /pet/feed`、`POST /pet/play`、`GET /pet/interactions`
- 新增 `pets` + `pet_interactions` 表（迁移: 000035）
- Pinia store：usePetStore

**UI 动画系统：**
- 页面切换动画：fade + translateY(8px)
- 卡片入场 stagger 动画（`.stagger-card`）
- 列表行入场 stagger 动画（`.stagger-row`）
- Per-app 背景色彩染（5%）：`.bg-app-oncall`、`.bg-app-alert`、`.bg-app-platform`
- Rail 图标从 20px 放大到 24px
- 卡片圆角从 12px 微调到 10px

**i18n：**
- 新增 `ai.*` 和 `pet.*` 中英文翻译

### 技术细节

- 迁移: 000034 (chat_histories)、000035 (pets + pet_interactions)
- 新增 composable: useAIChat
- 新增 Pinia store: usePetStore
- 新增 8 个 Vue 组件（ai/ 3 个 + pet/ 3 个 + pages/pet/ 1 个 + composable 1 个）

---

## [v4.7.1] — 2026-05-12

### Changed — UI 视觉深度 + 交互反馈 + 图标优化

**卡片视觉层级：**
- 全局阴影从极淡 (`0 1px 2px rgba(0,0,0,0.05)`) 升级到可见深度
- 新增 `--sre-shadow-lift` 用于 hover 浮起效果
- 所有卡片 hover 时 `translateY(-1px)` 浮起 + 阴影增强
- 涉及：surface-card、surface-clay、surface-glass、content-card、sre-row-card、sre-notify-card、sre-lift

**间距节奏优化：**
- 新增 `--sre-card-pad-compact: 16px`（KPI 卡片、列表行）
- 新增 `--sre-card-pad-relaxed: 24px`（图表卡片、内容区域）
- dashboard KPI 卡片更紧凑，图表区域更舒展
- incidents/alerts 列表行间距用语义化 token

**交互细节：**
- 图标按钮 hover 时 `scale(1.05)` 微放大
- 侧边栏菜单项 hover 时 `translateX(2px)` 微右移 + 文字颜色变深
- 选中菜单项左边框指示器加宽 + 发光效果
- rail 按钮 active 状态加 inset border ring

**侧边栏 hover 展开恢复：**
- 恢复被误删的 hover 展开功能（collapsed 状态下鼠标悬停临时展开）
- 新增 `pinned` 状态管理

**图标更换：**
- On-Call: `FlashOutline`（闪电）→ `CallOutline`（电话）— 更直观
- Alert: `AlertCircleOutline`（感叹号）→ `NotificationsOutline`（铃铛）— 更显眼

**配色：**
- per-app 背景色从 3% 提升到 5%（更易区分 oncall/alert/platform）

---

## [v4.7.0] — 2026-05-12

### Changed — UI 全面重构：去 AI 味 + 无障碍 + 交互优化

**Phase 1 · quieter — 去除 AI slop：**
- Login.vue：删除 mesh blob 动画背景、gradient-text 渐变文字
- global.css：bounce easing (`cubic-bezier(0.34,1.56,0.64,1)`) → 指数减速 (`cubic-bezier(0.16,1,0.3,1)`)
- 删除 `sre-bounce-in`、`sre-glow-pulse`、`sre-rail-active-pulse` 等过度动画

**Phase 2 · typeset + colorize — 字体与主题统一：**
- 字体：Plus Jakarta Sans → Inter（`@fontsource/inter` 本地加载）
- 暗色主题：暖棕 stone → 冷蓝灰 navy（`#0a1018`、`rgba(15,23,42,0.65)`）
- 浅色主题：stone-50 `#fafaf9` → slate-50 `#f8fafc`
- 图表配色：explore、dashboard、QueryResultChart 统一为 slate 色系

**Phase 3 · layout — 列表行简化：**
- incidents/Index.vue：3 行布局 → 2 行（标题行 + 元数据行），操作移入 hover dropdown
- alerts-v2/Index.vue：同上，删除冗余 severity 文字，添加 status pill

**Phase 4 · distill — 侧边栏精简：**
- AppShell.vue：删除 hover-expand 行为（`handleNavEnter/Leave`、`hoverTimeout`、`pinned`）
- AppSidebar.vue：删除 `pinned` prop，简化为纯点击切换

**Phase 5 · harden — 键盘快捷键 + 无障碍：**
- AppShell.vue：添加 skip-to-content 链接、`aria-label`、命令面板动作（主题切换、App 切换）
- 为 icon-only 按钮添加 `aria-label`

**Phase 6 · onboard — 空状态引导：**
- i18n：新增 incidents/alerts/channels 空状态描述文案 + 筛选提示
- 列表空状态：有筛选时显示"无匹配结果"，无筛选时显示功能介绍

**Phase 7 · polish — 最终打磨：**
- 全局 bounce easing → exponential ease-out（9 处修复）
- schedule shift blocks：`border-left: 4px` → 背景色着色
- AuditLog：`border-left: 2px` → `border: 1px`
- AppSidebar width transition easing 修正
- 反模式扫描：12 → 2（剩余 2 个 P3 可接受权衡）

---

## [v4.6.0] — 2026-05-11

### Changed — 设计系统 v5.0：Clean Neutral + Agent Review 修复

**global.css 设计系统重构：**
- 字体从 Plus Jakarta Sans（未加载）改为 Inter（Google Fonts 正确加载）
- 配色从暖色 stone 改为中性灰色（Tailwind gray scale）
- 阴影从 claymorphism 改为 clean Tailwind-style shadows
- 去除所有 clay-shadow、gradient-rainbow、gradient-brand、ease-bounce、hover-nudge 等过时变量
- Dark theme 修复：`--sre-success-soft` 从错误的 teal 改为正确的 green
- 新增 `--sre-shadow-glow` 到 `:root`（之前仅在 dark theme 定义）
- 删除 ~190 行死 CSS（与 AppRail/AppSidebar scoped 样式重复）
- 删除未使用的 `.stagger-list`（无 nth-child 延迟规则）
- 统一 surface 类：`.surface-clay`、`.surface-glass` 改用标准阴影

**Login.vue 安全修复：**
- 修复开放重定向漏洞：`redirect` 参数现在验证以 `/` 开头且不以 `//` 开头
- Mesh blob 性能优化：blur 移到静态父元素，子元素仅动画 transform
- 新增 `prefers-reduced-motion` 媒体查询保护
- `langOptions` 改为 `computed()` 以响应 locale 变化

**Dashboard 类型安全：**
- IncidentDashboard 的 4 个 `ref<any>` 改为完整接口定义
- Index.vue 的 `KpiDef.icon` 从 `any` 改为 `Component`
- Index.vue 的 `setInterval` 移入 `onMounted` 并在 `onUnmounted` 清理
- 修复硬编码英文字符串（"published"、"Triggered"、"Closed"）改用 i18n

**AppShell 安全修复：**
- localStorage JSON.parse 包装在 try/catch 中防止崩溃
- hoverTimeout 在 onUnmounted 时清理

**后端安全修复：**
- SSRF 修复：`datasource.go` 的 `LabelValues` 使用 `url.PathEscape()` 编码用户输入
- Goroutine 泄漏修复：`alert_event.go` 添加 100 容量信号量限制并发 dispatch
- 静默错误修复：`incident.go` 的 11 处 `_ = s.repo.AddTimeline(...)` 改为错误日志记录

---

## [v4.5.0] — 2026-05-11

### Changed — UI 精细化：去掉 AI 风，走精致现代路线

**设计方向调整（基于 ui-ux-pro-max 指导）：**
- Style: Soft UI Evolution — 柔和阴影、改进对比度、现代美学
- 去掉所有"AI 风"元素：彩虹渐变、弹跳动画、彩色圆形图标背景

**AppRail 重写：**
- 去掉彩色渐变圆形图标背景，改为简洁单色图标
- 新增小色点指示器（6px dot）标识当前 active app
- 去掉弹跳/浮动/jelly 动画，改为简单的背景色切换
- 用户头像去掉彩虹渐变光环

**AppSidebar 修复：**
- collapsed 且未 pin 时完全隐藏 nav（opacity: 0 + pointer-events: none）
- 解决了 collapsed 状态下菜单项堆叠的问题
- 菜单项 hover 去掉 translateX 移动，改为简单背景色变化
- 选中指示器从渐变色改为纯色
- border 从 2px 改为 1px

**AppShell 更新：**
- topbar 去掉彩虹渐变底线
- border 从 2px 改为 1px
- 新增 per-app 背景色区分：main-content 根据 data-app 应用微妙色调
  - On-Call: 微妙珊瑚色调
  - Alert: 微妙蓝色调
  - Platform: 微妙紫色调

**登录页精简：**
- 去掉彩虹渐变细线
- 登录按钮改为标准主题色（去掉彩虹渐变）
- 卡片动画从弹跳改为简单 fade + translateY
- 阴影从 claymorphism 改为柔和 shadow

**global.css 清理：**
- 删除过度动画 keyframes：bounce-hover、wiggle、float、jelly、color-breathe
- 删除工具类：.hover-bounce、.hover-wiggle、.hover-float、.jelly
- 页面过渡从 scale + translateY 改为简单 translateY
- 所有 2px border 改为 1px，16px radius 改为 12px
- 暗色主题 gradient mesh 透明度降低（更微妙）

---

## [v4.4.0] — 2026-05-11

## [v4.4.0] — 2026-05-11

### Changed — 视觉重构：Vibrant Clay 色彩丰富的 DevOps 平台

**配色系统重构：**
- 去掉单色 teal 贯穿全部，改为每个功能区独立主色
  - On-Call: 珊瑚橙 `#FF6B6B` → 暖橙 `#FFA07A`
  - Alert: 电光蓝 `#4FACFE` → 青色 `#00F2FE`
  - Platform: 梦幻紫 `#A855F7` → 粉紫 `#D946EF`
- 新增彩虹渐变 `--sre-gradient-rainbow` 用于强调元素
- 新增 Claymorphism 阴影变量（双重阴影 + 厚边框 + 大圆角）

**动画系统：**
- 新增 keyframes：`sre-bounce-hover`、`sre-wiggle`、`sre-float`、`sre-jelly`、`sre-gradient-flow`、`sre-color-breathe`
- 新增工具类：`.hover-bounce`、`.hover-wiggle`、`.hover-float`、`.jelly`、`.gradient-flow`
- 页面过渡：fade + scale(0.97) 组合

**AppRail 重写：**
- 图标包裹在彩色渐变圆形背景中（每个 app 独立颜色）
- hover 时放大 + 阴影增强
- active 时浮动动画 + 彩色光晕
- 用户头像外加彩虹渐变光环

**AppSidebar 更新：**
- 去掉 glassmorphism backdrop-filter，改为 claymorphism
- 新增 `data-app` 属性 + `--sidebar-accent` CSS 变量
- 选中菜单项左侧彩色渐变指示器
- hover 时菜单项微移 + 彩色背景
- app 名称使用对应 app 颜色

**AppShell 更新：**
- topbar 去掉 glassmorphism，改为 claymorphism + 彩虹渐变底线
- main-header 去掉 glassmorphism

**登录页重设计：**
- 背景 blob 颜色从 teal/blue/amber 改为 coral/blue/purple
- 登录卡片去掉 glassmorphism，改为 claymorphism
- 卡片顶部新增彩虹渐变细线（3px）
- 登录按钮改为彩虹渐变背景 + hover 弹跳动画
- 表单入场动画改为弹跳效果
- 卡片入场改为弹跳动画

**品牌更新：**
- tagline 从"一站式可观测性与事件响应平台"改为"一站式 DevOps 平台"

---

## [v4.3.0] — 2026-05-11

### Changed — 视觉系统升级：Glassmorphism + OLED 深色主题

**Dark Theme 全面升级：**
- 背景色从暖石色调（#0c0a09）切换到深邃 OLED 蓝黑色（#050a14）
- 新增 `--sre-gradient-mesh` 多色径向渐变用于页面背景
- 新增 `--sre-shadow-glow` 品牌色辉光阴影
- 文本色从暖白（#fafaf9）切换到冷白（#f1f5f9），层次更分明

**Glassmorphism 全面应用：**
- 所有 surface（glass/clay/card）加入 `backdrop-filter: blur()` 毛玻璃效果
- 卡片、行卡、通知卡、配置区块全部升级为半透明毛玻璃
- hover 时增加微妙的 `box-shadow` 提升层次感

**组件视觉升级：**
- AppRail：毛玻璃背景，active 图标带辉光效果（`box-shadow: 0 0 16px -4px`）
- AppSidebar：毛玻璃背景 + 玻璃边框
- AppShell topbar：毛玻璃背景
- main-content 区域叠加 mesh 渐变背景
- 登录页：完全重设计 — 全屏动画渐变 mesh 背景 + 居中 glass card

**登录页重设计：**
- 去掉 60/40 分栏布局，改为全屏居中 glass card
- 三个动画渐变色球（teal/blue/amber）做 mesh 背景
- card 带 24px blur + 品牌色辉光阴影
- logo + 品牌名移入 card 顶部
- 底部 footer 显示版本号 + 系统状态

---

### Fixed — 头像裂图 + 登录页清理

- 头像图片加载失败时自动 fallback 到首字母显示
- 登录后跳转从 `/dashboard` 修正为 `/oncall/overview`
- 移除登录页暴露的默认凭据（admin/admin123）
- 提示文字改为低调灰色，不再用警告框

---

## [v4.2.0] — 2026-05-11

### Changed — 布局重构：修复 6 个用户反馈问题

**左侧图标栏：**
- 去掉 MascotFox，改为用户头像（NAvatar）+ 点击弹出菜单
- 弹出菜单包含：个人信息、修改密码、退出登录
- 图标加大（20→22px，按钮 36→40px），active 状态加左侧 accent bar
- Tooltip 增强：显示名称+描述

**侧边栏：**
- 去掉底部用户信息区（头像、名字、角色、折叠按钮、版本号）
- 顶部加应用名 + 折叠按钮（`|<`）
- 鼠标移入导航区自动展开，移出 300ms 后自动收起
- 支持「固定」模式（点击折叠按钮切换固定/自动）

**个人信息页：**
- 加 NTabs：基本信息 + 修改密码
- 底部加退出登录按钮（危险区）

**新增组件：**
- `ChangePasswordModal.vue` — 密码修改弹窗

**CSS 增强：**
- 图标栏 active 状态 inset shadow、hover shadow
- 用户弹出菜单完整样式
- nav-zone flex 布局

---

## [v4.1.1] — 2026-05-11

### Fixed — 内网访问页面卡住

- 移除 `index.html` 中 Google Fonts CDN 外链（fonts.googleapis.com）
- 内网环境下 DNS 解析/连接超时会阻塞页面渲染
- 改用系统字体栈 fallback，PingFang SC / Microsoft YaHei 等

---

## [v4.1.0] — 2026-05-11

### Added — UI 视觉美化层

**交互效果：**
- `v-ripple` 全局点击涟漪指令
- 菜单 hover 微位移动画 + 激活条入场动画
- 导轨图标按压缩放 + 弹性回弹 + 激活脉冲
- 页面切换 fade-slide 过渡动画

**视觉细节：**
- hover-reveal 滚动条（5px，平滑过渡）
- CSS 自定义属性扩展（--sre-ripple-color, --sre-hover-nudge-x 等）

**萌宠组件：**
- MascotFox 狐狸萌宠（挥手/发呆/睡觉三状态 + 点击彩蛋）
- 集成至左侧导轨底部

---

## [v4.0.0] — 2026-05-11

### Changed — 应用架构重构：三栏布局 + 三应用分区

**导航重构：**
- 顶部 Tab（Monitor/Incident/System）改为左侧图标栏（On-Call/Alert/Platform）
- 新增三栏布局：图标栏（48px）+ 菜单栏（220px）+ 内容区
- 菜单栏支持多级折叠
- 个人中心从弹窗改为独立页面

**路由重组：**
- 新增 `/oncall/` 前缀：故障响应相关页面
- 新增 `/alert/` 前缀：告警引擎 + 通知管道页面
- 新增 `/platform/` 前缀：平台管理页面
- 所有旧路由添加向后兼容重定向

**新增页面：**
- 状态页面（On-Call）— 即将上线
- 角色权限（Platform）— 即将上线
- 个人中心（Platform）— 独立页面替代弹窗

**文件变更：**
- 新增：`useAppNav.ts`, `AppShell.vue`, `AppRail.vue`, `AppSidebar.vue`
- 删除：`MainLayout.vue`（750行）
- 新增 11 个页面组件（3 stub + 1 profile + 7 wrapper）
- 修改：路由、命令面板、i18n

---

## [v3.1.0] — 2026-05-11

### Changed — UI 重构：Soft Warm 清新温暖设计风格

**设计系统重写 (global.css)：**
- 主色调从 green-500 `#22c55e` 改为 teal-600 `#0d9488`
- 强调色从 indigo-500 `#6366f1` 改为 amber-500 `#f59e0b`
- 默认主题从 Dark 切换为 Light（暖白 `#fafaf9`）
- Dark 模式改为 opt-in，使用 stone 色系暖灰（`#1c1917`/`#292524`/`#44403c`）
- 字体从 Inter 改为 Plus Jakarta Sans（Google Fonts 加载）
- 圆角 lg 从 10px 增大到 12px，xl 从 14px 到 16px，2xl 从 20px 到 24px
- 阴影改为更柔和的低透明度版本（适配浅色背景）

**主题覆盖 (App.vue)：**
- Naive UI `darkOverrides` / `lightOverrides` 全面对齐新色板
- 默认 `isDark` 从 `true` 改为 `false`
- `applyBodyClass` 切换为 `dark-theme` / `light-theme` body class

**硬编码颜色迁移（12 个文件）：**
- `notification/Subscribe.vue` + `Rules.vue`：紫色/绿色硬编码 → CSS 变量
- `dashboard/Index.vue`：ECharts 图表主题从冷灰改为暖石色
- `explore/Index.vue` + `Login.vue`：`#fff`/`#666` → CSS 变量
- `schedule/Index.vue`：box-shadow → `var(--sre-shadow-xs)`
- `CommandPalette.vue`：`#f8f9fa` → `var(--sre-bg-card)`
- `PromQLEditor.vue`：边框/焦点色 → CSS 变量
- `QueryResultChart.vue` + `TimeRangePicker.vue`：灰色文字 → CSS 变量
- `QuickSilenceModal.vue` + `PageHeader.vue`：绿色 rgba → CSS 变量

---

## [v3.0.1] — 2026-05-11

### Fixed

- 修复 `Detail.vue` 中 `severityLabel` 函数的 TypeScript TS7053 类型错误（用 `Record<string, string>` 断言解决索引签名缺失问题）

---

## [v3.0.0] — 2026-05-10

### Changed — 全面 UI 重构：Modern Dark 设计系统 (Linear/Vercel/Raycast 风格)

**设计系统重写 (global.css)：**
- 背景色从 `#070912`/`#090D1A` 改为纯黑 `#09090B`/`#0A0A0B`
- 卡片色从毛玻璃半透明改为实色 `#141416`
- 品牌主色从 emerald `#10b981` 改为 `#22C55E` (green-500)
- 强调色从 violet `#8b5cf6` 改为 `#6366F1` (indigo-500)
- 文字色：主 `#EDEDEF`、次 `#A0A0AB`、三 `#63636E`
- 圆角全面缩小：lg `16px→10px`、md `12px→8px`、sm `8px→6px`、xs `6px→4px`
- 阴影简化为单层，移除 `inset` 高光
- 移除 aurora 渐变、conic 边框、noise 纹理等装饰性 CSS 变量
- 字体改为 Inter + JetBrains Mono（本地安装优先，系统字体回退）

**主题覆盖 (App.vue)：**
- Naive UI `darkOverrides` / `lightOverrides` 全面对齐新色板
- 移除 `AuroraBackground` 组件引用
- 新增 Tooltip 主题覆盖

**布局重构 (MainLayout.vue)：**
- TopBar 高度 56px → 48px
- Sidebar 宽度 232px → 220px
- 内容区 padding 24px → 20px
- 移除所有 `backdrop-filter` / `blur` 毛玻璃效果
- 用户头像简化（移除 gradient box-shadow）

**页面级样式对齐（8 个并行 Agent）：**
- alerts-v2/Index.vue + Detail.vue：圆角值改为设计令牌引用
- channels/Index.vue + DispatchConfig.vue + NoiseConfig.vue：圆角 + 移除 translateY
- incidents/Index.vue + Detail.vue：圆角 + 移除 translateX
- datasources/Index.vue：移除 translateY 悬浮效果
- explore/Index.vue：圆角 12px → 8px
- schedule/Index.vue：圆角 + 移除 translateY
- settings/Index.vue + TeamManagement.vue：移除 translateY 动画
- Login.vue：移除 noise-overlay 和 aurora blur 伪元素
- CommandPalette.vue：移除毛玻璃，改为实色卡片背景

**ECharts 图表主题 (QueryResultChart.vue)：**
- 新增 `chartTheme` 计算属性，通过 CSS 变量自动适配深色/浅色主题
- 硬编码颜色 `#999`/`#666`/`#eee` 改为设计令牌引用

**浅色模式 (global.css body.light-theme)：**
- 背景改为 `#FAFAFA`/`#FFFFFF`，文字 `#18181B`/`#52525B`
- 阴影适配浅色背景

---

## [v2.8.2] — 2026-05-10

### Fixed — 安全修复 + Bug 修复 + 前端 i18n 全面完善

**安全修复：**
- 修复 avatar 切片越界 panic（`auth.go:83`），当 avatar 长度 < 5 时会崩溃
- 默认管理员密码改为从 `SREAGENT_ADMIN_PASSWORD` 环境变量读取，不再硬编码 `admin123`
- 虚拟用户密码改用 `crypto/rand` 生成随机值，替代可预测的 `virtual-{username}-{len}` 格式
- logger 构建错误不再静默忽略，失败时使用 fallback logger
- post_mortem handler `formatStep` 修复死代码，使用 `fmt.Sprintf` 正确格式化
- `handler.go` 新增 `GetCurrentUserIDOK` comma-ok 断言方法，支持 401 响应

**Bug 修复：**
- 修复心跳检查器使用字符串比较 `err.Error() != "record not found"` 代替 `errors.Is(err, gorm.ErrRecordNotFound)`
- 修复 Resolve/Close handler 静默忽略 `ShouldBindJSON` 错误，改为正确返回 400 响应
- V2 模型注册到 AutoMigrate（`model.V2Models()`），确保 Alert/Channel/Incident 等表自动迁移
- AlertRule severity 枚举文档化：critical/warning/info 为主，p0-p4 标记为 Legacy

**前端 i18n — 全面国际化：**
- alerts-v2/Index.vue + Detail.vue：所有硬编码文本替换为 i18n 键（30+ 处）
- Login.vue：品牌标语、功能特性、欢迎语等 8 处替换为 `auth.brand.*` 键
- MainLayout.vue：语言选项、角色文本替换为 i18n 键
- notification/Rules.vue + Subscribe.vue + Media.vue：严重等级、表单标签、占位符等替换为 i18n 键
- settings/AuditLog.vue + OIDCConfig.vue + AIConfig.vue：筛选条件、配置项标签等 37 处替换为 i18n 键
- components/RuleFormModal.vue + QuickSilenceModal.vue + QueryResultChart.vue：表单标签、按钮文本等替换为 i18n 键
- 新增 `severity.*` 顶层 i18n 命名空间（critical/warning/info/p0-p4），供严重等级标签统一使用
- 新增 `notifyRule.repeatDisplay` / `namePlaceholder` / `invalidJson` 等键
- 新增 `subscribe.namePlaceholder` 等键
- 新增 `settings.auditTimeToday` / `oidcIssuerRequired` / `aiSectionProvider` 等 30+ 键

**前端改进：**
- SecurityConfig 组件在设置页导航中注册，用户可访问安全配置
- 抽取 `relTime()` 到 `utils/format.ts`，alerts-v2/Index.vue 和 dashboard-v2/Index.vue 复用
- DispatchConfig.vue NTag `size="tiny"` 修正为 `size="small"`（Naive UI 有效值）
- 严重等级标签颜色统一使用 CSS 变量（`--sre-critical-soft` 等），从全局 `.sev-chip` 类派生
- 路由权限守卫：`/schedule`、`/integrations` 限 admin+team_lead，`/channels` 限 admin+team_lead+member

**文档更新：**
- `docs/api.md` 从 87 个端点扩充至 164 个，覆盖所有 v2 模块

---

## [v2.8.1] — 2026-05-10

### Changed — 仪表盘 UI 升级: 高级 KPI 卡片 + 玻璃-粘土图表容器

**仪表盘重设计 (Dashboard + IncidentDashboard)：**
- KPI 卡片增加图标 (PulseOutline, TimerOutline, CheckmarkCircleOutline 等)，彩色顶部色条去除底部条纹
- 图表容器统一使用 `.surface-clay` 风格，标题 + 操作栏分区清晰
- Top Noisy Rules 增加排名徽章 (#1 主色, #2 强调色, #3 信息色)
- IncidentDashboard KPI 行统一 5 列网格 + hover 交互
- KPI 图标容器按 tone 着色 (critical/红, success/绿, info/蓝)

**MainLayout 视觉打磨：**
- 侧边栏分组标签增加小圆点指示器
- 顶部 Tab 激活态改为发光阴影 (box-shadow glow) 替代 inset ring
- Tab 点击添加 scale 反馈
- 侧边栏用户 Pill 增加 border 过渡
- 折叠按钮按钮态更平滑

**设计系统：**
- 新增 `.text-critical` / `.text-warning` / `.text-success` / `.text-info` 语义颜色工具类

---

## [v2.8.0] — 2026-05-09

### Changed — 顶部 Tab + 侧边栏导航架构重构 (FlashCat 风格)

**交互架构升级：**
- 顶部 Tab 一级模块切换: 监控告警 / 故障管理 / 系统配置
- 左侧二级导航随 Tab 动态切换，每个 Tab 独立菜单项
- Tab 切换自动导航到模块首页（/dashboard / /incident-dashboard / /notification）
- 路由变化自动同步 Tab 激活状态

**视觉打磨：**
- 全局 deepest dark 背景 (#070912)，更专业深邃
- 卡片透明度微调，暗色模式更可见
- 数据表格表头加粗 (font-weight: 600)，圆角统一 12px
- 菜单选中态增强，亮色模式更高对比度
- 新增 `.content-card`、`.surface-glass-interactive`、`.page-container` 语义类
- 新增 `--sre-shadow-card-hover` 悬停发光阴影令牌

**代码清理：**
- MainLayout.vue 去 sidebar 可折叠父菜单，改顶部 Tab 导航
- 移除未使用的 icon 导入 (AlertCircleOutline, ShieldCheckmarkOutline 等)
- 修复 inject 类型标注

**i18n：**
- 新增 `monitorAlert` 翻译键 (zh-CN: 监控告警, en: Monitor & Alert)

---

## [v2.7.0] — 2026-05-09

### Changed — UI 全局重设计: "Vibrant Glass + Clay" 融合风格

**设计系统升级：**
- 色板: 午夜石板 (#090D1A) 替代纯黑，主色翠绿 #10B981 + 紫罗兰强调 #8B5CF6
- 融合 4 种设计风格: Vibrant & Block-based + Glassmorphism + Claymorphism + Motion-Driven
- 玻璃效果 (backdrop-filter blur) 应用于侧边栏、顶栏、模态框
- 软土风格 (Claymorphism) 双阴影卡片，圆角 12-16px
- 弹性动画 (spring physics)，200-340ms 过渡
- 暗色模式: 半透明玻璃背景，柔和午夜底色
- 亮色模式: 暖灰底 #F1F4F9 + 半透白卡片

**侧边栏重构 (FlashCat 风格):**
- 5 个平铺 `type:'group'` → 3 个可折叠父菜单: 监控告警、故障管理、系统配置
- 用户头像/名称从顶栏移至侧边栏底部
- 玻璃材质侧边栏，24px 模糊 + 饱和度增强
- 顶部导航栏简化: 仅保留面包屑、时钟、搜索、语言、主题

**删除死代码:**
- SpotlightCursor (已在 v1.8.1 移除引用，清理残留)
- 废弃的 CSS 动画和注释代码
- 未使用的 page-enter/leave 过渡组件

### Changed — 文件

| 文件 | 变更 |
|------|------|
| `web/src/styles/global.css` | 重写: 新色板、玻璃/土质效果、块布局、弹性动效 |
| `web/src/App.vue` | Naive UI 主题覆盖全面更新对齐新色板 |
| `web/src/layouts/MainLayout.vue` | 3层可折叠菜单 + 玻璃侧边栏 + 用户区移至侧边栏底部 |
| `web/src/i18n/zh-CN.ts` | 新增 menu.alertCenter/incidentMgmt/systemConfig |
| `web/src/i18n/en.ts` | 新增 menu.alertCenter/incidentMgmt/systemConfig |
| `web/package.json` | 版本 2.6.2 → 2.7.0 |
| `CLAUDE.md` | 版本 2.6.2 → 2.7.0 |

---
## [Unreleased]

### Changed — UX Accessibility & Quality Sweep

**CRITICAL - Reduced Motion:**
- `global.css`: 增强 `@media (prefers-reduced-motion)` 查询，显式禁用 `.sre-stagger`、`.sre-lift`、`.sre-row-card`、`.fade-in`、`.slide-up`、`.scale-in`、`.pulse` 等所有动画类，以及 hover lift 变换效果和 conic-border 旋转动画

**CRITICAL - Focus States:**
- `global.css`: 新增 `.sre-row-card`、`.sre-lift`、`.surface-card--interactive`、`.ds-card`、`.sre-notify-card` 的 `:focus-visible` 样式（2px `var(--sre-primary)` 描边）
- `CommandPalette.vue`: `.cp-input` 添加 `:focus-visible` 内描边替代 `outline: none`
- `incidents/Detail.vue`: `.comment-input` 和 `.pm-title-input` 添加 `:focus-visible` 样式

**CRITICAL - Touch Targets:**
- `global.css`: `.sre-icon-btn` 从 24x24 扩大到 36x36（符合 WCAG 2.5.5），添加 hover/focus-visible 过渡
- `global.css`: Naive UI `n-button--tiny-type.n-button--circle-type` 最小尺寸设为 32x32

**CRITICAL - Dark/Light Contrast:**
- `BizGroupManagement.vue`: `.bg-chip` / `.bg-count` / `.bg-role-member` 硬编码 `rgba(255,255,255,*)` 替换为语义化 `var(--sre-bg-hover)` 令牌
- `BizGroupManagement.vue`: `.bg-count` 修复无效 `--sre-text` 变量为 `--sre-text-primary`
- `BizGroupManagement.vue`: `.bg-role-member .sre-dot` 硬编码 `rgba(255,255,255,0.4)` 替换为 `var(--sre-text-tertiary)`
- `BizGroupManagement.vue`: 清理 linter 发现的不一致 fallback 值（`.bg-chip-info` border-color 等）

**HIGH - Loading Feedback:**
- `alerts-v2/Detail.vue`: 初始加载添加 `LoadingSkeleton`，避免空白闪烁

**HIGH - Cursor Pointer:**
- 验证所有可点击元素（`sre-row-card`、`sre-lift`、`surface-card--interactive`、`ds-card`、`stat-card`、`dash-card`）已有 `cursor: pointer`

**MEDIUM - Error States:**
- `datasources/Index.vue`: 3 处 `message.error(err.message)` 添加 fallback 为 `t('common.loadFailed')`

**Global CSS 改进:**
- `.sre-label-eyebrow` 颜色由 `--sre-text-tertiary` 提升为 `--sre-text-secondary`（确保浅色主题 4.5:1 对比度）
- 新增 "Accessibility" 注释段，集中管理焦点样式和触摸目标

## [v2.6.2] - 2026-05-09

### Fixed — P4 全站质量修复完成

**P4 - incidents 模块 (FlashCat 9/10)：**
- `incidents/Index.vue`: 移除 3 处硬编码 CSS fallback 颜色，添加 `font-family`，i18n-ify `durationText()`，2 处 `var(--sre-hairline)` 替换 `1px solid`，添加 i18n 表单 placeholder
- `incidents/Detail.vue`: 2 处 `n-empty` → `EmptyState` 组件，`.incident-id` 添加 monospace 字体，修复硬编码 "by" 使用 i18n，MdEditor language/theme 改为计算属性，初始加载添加 `LoadingSkeleton`，timeline action 映射到 i18n 标签
- 提取 3 个子组件：`SnoozeModal.vue` / `MergeModal.vue` / `ReassignModal.vue`

**P4 - alerts 模块 (FlashCat 9/10)：**
- `events/Index.vue`: i18n-ify severityOptions/timePresetOptions/refreshOptions/NRadioButton/placeholder，修复 color-mix 边框，移除死 CSS
- `events/Detail.vue`: 11 处硬编码 `rgba()` 替换为 `var(--sre-*)` 令牌，`n-spin` → `LoadingSkeleton`，移除非法 `font-feature-settings`
- `mute/Index.vue`: i18n-ify statusText()/dayMap/typeOptions 及全部模板字符串，添加 font-family
- `inhibition/Index.vue`: i18n-ify status 标签/relTime/eyebrow labels/footer 字符串，添加 font-family
- `history/Index.vue`: 全 i18n（11 处），`NEmpty` → `EmptyState`，`NSpin` → `LoadingSkeleton`，4 处 inline style → CSS class

**P4 - 共享组件修复：**
- `LoadingSkeleton.vue`: `rgba()` 替换为 `var(--sre-overlay-subtle)`
- `EmptyState.vue`: 4 处硬编码 `rgba()` 边框替换为 `var(--sre-*-ring/soft)` 令牌
- `global.css`: 新增 `--sre-overlay-subtle` / `--sre-bg-subtle` 令牌（dark + light）

**i18n 新增：**
- `zh-CN.ts` / `en.ts`: 新增 ~54 个 key（incident 14 + alert 20 + mute 12 + inhibition 8）

### Fixed — 全站质量修复（P0-P3）

**P0 - 构建修复：**
- `.gitignore` 添加 `.superpowers/` 排除
- `md-editor-v3` 依赖安装（构建修复）

**P1 - 后端错误处理修复（15 处）：**
- `alert_v2_pipeline.go`: 修复 3 处 `_ =` 静默丢弃 DB 错误
- `schedule.go`: 修复 DeleteEscalationPolicy 中 `steps, _ :=` 和 `_ = stepRepo.Delete` 错误忽略
- `system_setting.go`: `parseBool` 不再静默丢弃解析错误
- `user.go`, `datasource.go`, `team.go`, `alert_event.go`, `message_template.go`, `biz_group.go`, `seed.go`, `integration.go`, `notification.go`: 修复 `existing, _ :=` 模式，区分 `gorm.ErrRecordNotFound` 与真正的 DB 错误

**P2 - i18n 国际化修复：**
- 新增 ~200 个 i18n key（zh-CN + en）
- 修复 6 个严重未国际化文件：`RoutingRules.vue`（完全未翻译）、`QuickSilenceModal.vue`（完全未翻译）、`incidents/Detail.vue`（~35 处）、`explore/Index.vue`（~13 处）、`DispatchConfig.vue`（~11 处）、`IncidentDashboard.vue`（~8 处）
- 修复 `alerts/events/Index.vue`、`channels/Index.vue`、`Login.vue` 中的硬编码字符串

**P3 - UI 设计系统对齐：**
- 字体标准化：22 个文件中的硬编码 `font-family` 替换为 `var(--sre-font-*)` CSS 变量
- 颜色系统：6 个文件中的硬编码颜色表替换为 `var(--sre-*)` 设计令牌
- CSS 去重：将 notification（Rules/Subscribe/Media）和 settings config（AI/Lark/OIDC/SMTP）中 ~280 行重复 CSS 提取到 `global.css` 共享样式

---



## [v2.6.0] - 2026-05-08

### Changed — UI 重构 Phase 5（共享组件 + 设计系统文档 + 浅色审计）

**共享组件抽取：**
- `components/common/EmptyState.vue` 新增（替代散落的 NEmpty）：支持 size sm/md/lg、variant default/success/warning/critical/info、primary/secondary 操作按钮
- `components/common/LoadingSkeleton.vue` 新增（替代 NSpin 大圈圈）：3 种 variant（row / card-grid / kpi），shimmer 动效

**6 个高频页面接入新组件：**
- incidents/Index、channels/Index、alerts/rules/Index、alerts/events/Index、alerts-v2/Index、integrations/Index
- 加载态用 LoadingSkeleton 骨架屏（首次加载，避免空白闪烁）
- 空状态用 EmptyState（统一图标+文案+CTA 按钮）

**浅色主题对比度修复：**
- `global.css` 两处 `.sre-row-card:hover` / `.sre-lift:hover` 边框色：硬编码 rgba(255,255,255,0.x) → var(--sre-border-strong)
- `dashboard/Index.vue` ECharts 浅色模式不可读：新增 isLightTheme 监听 body.classList，chartPalette computed 提供 8 个动态颜色值（tooltipBg/legend/axis/grid/pieCenter）
- `channels/Index.vue` `.card-star:hover` 背景：rgba(0,0,0,0.05) → var(--sre-bg-hover)

**设计系统文档：**
- `docs/design-system.md` 新建（~220 行）
- 内容：设计哲学 / Typography (Geist + 拒绝清单) / Color (主色 + severity + surfaces + text + WCAG) / Spacing / Radius / Component patterns / Page anatomy / Anti-patterns 10 项 / Skill 应用 / Migration notes / File index / Decision log

**v2.x UI 重构总结**（v2.1.0 → v2.6.0）：
- 35+ 文件全站对齐 FlashCat "Operational Refinement" 美学
- 字体 Geist + JetBrains Mono 替代 Inter/system
- 4px severity stripe + sre-dot 圆点 + hairline borders 全站统一
- 自定义 sre-row-card div 列表 取代 NDataTable（除审计外）
- Skills 全程加持：frontend-design / vue 3.5 / web-design-guidelines

vue-tsc ✅

---

## [v2.5.0] - 2026-05-08

### Changed — UI 重构 Phase 4（Settings 子页 + Login）

10 个 settings 子页 + Login 全部对齐 FlashCat 设计语言。

**Platform Config（4 个表单页统一模板）：**
- `AIConfig.vue`：Provider + Behavior 两段卡片，新增 temperature/max_tokens/system_prompt 字段
- `LarkBotConfig.vue`：App Credentials + Defaults 两段
- `SMTPConfig.vue`：Server + Sender + Test Delivery 三段，新增 from_name
- `OIDCConfig.vue`：Provider + Claim Mapping + Behavior 三段，新增 username_claim/email_claim

共享模板：紧凑 header（18px 600 + 描述）+ 状态 banner（success/error tone）+ 段落卡片（hairline 边框）+ 2 列 form-grid + uppercase eyebrow labels + 内联 Test/Save 按钮

**Organization Pages：**
- `UserManagement.vue`：NDataTable → sre-row-card（avatar 圆形 + role chip 颜色映射 + 状态点 + last_login relTime）
- `TeamManagement.vue`：NDataTable → 卡片网格（成员头像组叠层 + member count + incidents count）
- `VirtualUsers.vue`：sre-row-card（type icon tile + type chip mono + truncate notify_target）
- `BizGroupManagement.vue`：300px 自定义递归树 + 右侧详情面板（PARENT/PATH/CHILDREN/CREATED 元信息卡 + MEMBERS 列表）

**Audit：**
- `AuditLog.vue`：NDataTable → 竖向时间线（hairline rail + 8px 圆点按 action tone 着色 + mono 时间戳 + action chip + resource 跳转链接）

**Login 重设计**：
- "Mission Control" 双栏布局（60% brand / 40% form）
- 品牌侧：Geist 800 -2px tracking 56px 标题 + gradient text-clip + 极细噪点 + 一处低饱和极光斑（左下）
- 表单侧：mono uppercase eyebrow labels + stagger 浮现（60ms 增量）+ 主色按钮 hover ring + 警告 banner（默认密码提醒）
- 拒绝紫渐变 / 卡通 illustration / 立体阴影；接受 hairline + 噪点 + 字体层次

修复 6 处 TS 错误：AIConfig/OIDCConfig/SMTPConfig 新增字段类型断言。

vue-tsc ✅

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
