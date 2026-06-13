# 架构设计

> 最后更新：2026-05-20（v4.13.0）

## 目录

- [系统架构](#系统架构)
- [技术栈](#技术栈)
- [RBAC 双层模型](#rbac-双层模型)
- [告警引擎状态机](#告警引擎状态机)
- [多数据源评估架构](#多数据源评估架构)
- [AI 规则生成管道](#ai-规则生成管道)
- [预置规则库](#预置规则库)
- [四层事件模型](#四层事件模型)
- [通知管道 v2](#通知管道-v2)
- [AlertV2 处理管道](#alertv2-处理管道)
- [数据模型](#数据模型)
- [关键架构决策](#关键架构决策)
- [数据库迁移文件](#数据库迁移文件)

---

## 系统架构

```
                                 ┌─────────────────────────────────────────┐
                                 │           Vue 3 Frontend                │
                                 │   Naive UI + TypeScript + Pinia         │
                                 │   RBAC v-can · PromQL Editor            │
                                 └──────────────────┬──────────────────────┘
                                                    │ HTTP REST
                                 ┌──────────────────▼──────────────────────┐
                                 │           API Layer (Gin)               │
                                 │   JWT / OIDC / RequirePerm / CORS       │
                                 │   双层 RBAC（全局角色 + 团队角色合并）    │
                                 └──────────────────┬──────────────────────┘
              ┌──────────────────────┬─────────────┴─────────┬────────────────────────┐
              │                      │                       │                        │
  ┌───────────▼─────────┐  ┌─────────▼──────────┐  ┌────────▼────────┐  ┌────────────▼──────────┐
  │   DataSource Svc    │  │   Alert Engine      │  │  OnCall Svc     │  │  Integration Webhook  │
  │   Prom/VM/VLogs     │  │  Evaluator + FSM    │  │ Schedule/Escal  │  │  Standard / AlertMgr  │
  │   Zabbix            │  │  Heartbeat/Suppress │  │                 │  │  Grafana 格式归一化    │
  │   多数据源路由       │  │  连续错误升级日志    │  │                 │  │                       │
  └───────────┬─────────┘  └─────────┬──────────┘  └────────┬────────┘  └────────────┬──────────┘
              │                      │                       │                        │
              │           ┌──────────▼──────────────┐        │            ┌───────────▼──────────┐
              │           │  WrapOnAlert Hook        │        │            │  RoutingRule 路由     │
              │           │  AlertV2Pipeline         │        │            │  DataSourceID 路由     │
              │           └──────────┬──────────────┘        │            └───────────┬──────────┘
              │                      │                        │                        │
              │           ┌──────────▼──────────────────────────────────────────────────▼──────────┐
              │           │                    AlertV2 处理管道                                     │
              │           │  normalise → rate_limit → applyPipeline → label_enhance                │
              │           │    → upsertAlert → ensureIncident → NoiseReducer → DispatchService     │
              │           └──────────────────────────────────┬─────────────────────────────────────┘
              │                                              │
              │           ┌──────────────────────────────────▼─────────────────────────────────────┐
              │           │                  Channel / Incident / Alert 三层事件模型               │
              │           │   Event（原始）→ Alert（按 alert_key 去重）→ Incident（故障聚合）       │
              │           │                   └── Channel（协作空间）                              │
              │           └──────────────────────────────────┬─────────────────────────────────────┘
              │                                              │
              │    ┌───────────────────────────┐             │
              │    │   AI 规则生成管道           │             │
              │    │   自然语言 → LLM → PromQL  │             │
              │    │   DryRun + 缓存 + FewShot  │             │
              │    └───────────────────────────┘             │
              │                                              │
  ┌───────────▼──────────────────────────────────────────────▼─────────────────────────────────────┐
  │                                         Redis 7                                                │
  │   引擎状态持久化 (Hash per rule) · 风暴预警滚动计数器 · 节流                                   │
  └──────────────────────────────────────────────┬─────────────────────────────────────────────────┘
                                                 │
  ┌──────────────────────────────────────────────▼─────────────────────────────────────────────────┐
  │                                        MySQL 8.0                                               │
  │   47+ 张表 · golang-migrate 管理 · GORM v2 ORM                                                │
  │   核心表: alert_rules · alert_events · preset_rules · notify_rules · alert_channels            │
  │   RBAC: team_members · users（global_role）                                                    │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘

              ┌────────────────────────────────────────────────────────────┐
              │                       外部集成                              │
              │  Lark Bot+Hook · LLM API · Email SMTP · Custom Webhooks   │
              │  Keycloak OIDC · Prometheus / VictoriaMetrics / Zabbix    │
              └────────────────────────────────────────────────────────────┘
```

---

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 后端语言 | Go | 1.25 |
| Web 框架 | Gin | 1.10+ |
| ORM | GORM | v2 |
| 数据库 | MySQL | 8.0 |
| 缓存 | Redis | 7 |
| 前端框架 | Vue | 3.4+ |
| UI 组件库 | Naive UI | 2.x |
| 状态管理 | Pinia | 2.x |
| 前端构建 | Vite | 5.x |
| 代码编辑器 | CodeMirror | 6（PromQL 语法高亮） |
| OIDC | Keycloak | — |
| 日志 | Zap | — |
| 迁移 | golang-migrate | 0.17+ |

---

## RBAC 双层模型

### 权限架构

SREAgent 采用 **全局角色 + 团队角色** 双层权限模型，实现细粒度的访问控制。

```
┌─────────────────────────────────────────────┐
│                 用户 (User)                  │
├─────────────────────────────────────────────┤
│  全局角色 (Global Role)                      │
│  admin / team_lead / member / viewer         │
├─────────────────────────────────────────────┤
│  团队角色 (Team Roles) — 0..N 个团队         │
│  Team A: team_lead                           │
│  Team B: member                              │
├─────────────────────────────────────────────┤
│  有效权限 = Global Perms ∪ max(Team Perms)  │
└─────────────────────────────────────────────┘
```

### 角色等级

| 角色 | 等级 | 典型权限 |
|------|------|---------|
| admin | 4 | 全部权限（用户管理、系统设置、审计日志） |
| team_lead | 3 | 规则编辑、排班管理、通道管理、仪表盘管理 |
| member | 2 | 创建规则、确认/分配事件、创建事件 |
| viewer | 1 | 只读访问 |

### 权限合并算法

```go
// internal/pkg/rbac/rbac.go
func EffectivePerms(globalRole string, teamRoles []string) map[string]bool {
    perms := PermissionsByGlobalRole(globalRole)       // 1. 全局角色基础权限
    highestTeam := HighestTeamRole(teamRoles)           // 2. 取团队最高等级
    teamPerms := PermissionsByGlobalRole(highestTeam)   // 3. 团队角色权限
    for p := range teamPerms {
        perms[p] = true                                 // 4. 只增不减
    }
    return perms
}
```

**关键约束**：团队角色只能**增加**权限，不能**撤销**全局角色已有的权限。

### RequirePerm 中间件

`internal/middleware/permission.go` 提供路由级权限检查，支持两种执行模式：

| 模式 | 行为 | 适用场景 |
|------|------|---------|
| `warn` | 权限不足时记录审计日志，仍放行请求 | 灰度上线 |
| `deny` | 权限不足时返回 403 `code:10200` | 生产强制 |

```go
// 路由注册
router.POST("/rules", middleware.RequirePerm("rules.create"), handler.Create)
router.DELETE("/rules/:id", middleware.RequirePerm("rules.delete"), handler.Delete)
```

**执行流程**：
1. JWT 认证中间件设置 `user_role` + `user_team_roles` 到 context
2. `RequirePerm` 快速路径：`rbac.HasPerm(globalRole, perm)` 直接放行
3. 慢速路径：`rbac.EffectivePerms` 合并后检查
4. 仍不满足 → `OnPermissionDenied` 审计回调 + 返回 403

### 前端权限集成

- `web/src/permissions.ts` — 50+ 权限常量，与后端 `rbac.go` 完全对齐
- `web/src/composables/usePermissions.ts` — `hasPerm` / `hasAnyPerm` / `isTeamLead`
- `web/src/directives/vCan.ts` — `v-can` 指令，条件渲染（权限不足时从 DOM 移除）

**完整权限矩阵**：→ [docs/rbac.md](rbac.md)

---

## 告警引擎状态机

```
inactive → pending（for_duration）→ firing → recovery_hold → resolved
                                        └── nodata（数据缺失）
```

- 每规则一个 goroutine，Evaluator 管理协程池
- LevelSuppressor 基于严重级别去重
- HeartbeatChecker 心跳超时检测
- EscalationExecutor SLA 超时自动升级
- AlertGroupManager group_wait/interval 通知分组
- 连续 5 次查询失败自动升级为 Error 级别日志，恢复时记录恢复日志

---

## 多数据源评估架构

### 场景

生产环境通常存在多个 Prometheus/VictoriaMetrics 实例，每个实例负责不同的监控维度。SREAgent 通过 `DataSourceID` 字段实现**数据源级别的路由隔离**。

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  cc-prom      │  │  cpp-prom     │  │  vm-cluster  │
│  (容器监控)    │  │  (C++ 服务)   │  │  (全局指标)   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └─────────────────┼─────────────────┘
                         │
                 ┌───────▼────────┐
                 │    SREAgent     │
                 │  startRule      │
                 │  Evaluators     │
                 └────────────────┘
```

### DataSourceID 语义

| 值 | 含义 | 行为 |
|---|------|------|
| `nil` (NULL) | 通配符 | 匹配任意数据源 |
| 非 nil (具体 ID) | 精确匹配 | 只匹配该数据源 |

精确绑定（`DataSourceID`）优先于类型绑定（`DatasourceType`）。

### 评估引擎分发逻辑

`internal/engine/evaluator.go` 中的 `startRuleEvaluators` 实现数据源分发：

```go
func (e *Evaluator) startRuleEvaluators(ctx context.Context, rule *model.AlertRule) {
    if rule.DataSourceID != nil {
        // 精确绑定：使用预加载的 DataSource
        e.startRuleEvaluator(rule, rule.DataSource)
        return
    }
    // 类型绑定：遍历所有匹配类型的已启用数据源
    dsList, _ := e.dsRepo.ListEnabledByType(ctx, rule.DatasourceType)
    e.startRuleEvaluator(rule, &dsList[0])
}
```

**规则同步**：引擎每 30 秒从 DB 同步规则，对比版本变化，自动启停 evaluator goroutine。

### 通知路由中的数据源过滤

`NotificationService.RouteAlert` 从事件关联的规则中解析 `DataSourceID`，传递给下游匹配：

```go
// 从 event.RuleID → AlertRule.DataSourceID
// 传递给 NotifyRuleService.FindMatchingRules(ctx, event, dataSourceID)
// 使用 labelmatch.MatchWithSourceID 过滤
```

### labelmatch 匹配引擎

`internal/pkg/labelmatch/matcher.go` 提供统一匹配：

```go
func MatchWithSourceID(target map[string]string, targetDSID *uint,
    pattern map[string]string, patternDSID *uint) bool
```

**匹配逻辑**：
1. 数据源维度：`patternDSID == nil` 通配；否则精确匹配
2. 标签维度：空 pattern 通配；非空则 AND 逻辑全部满足
3. 操作符：精确相等 / `!=` 不等于 / `=~` 正则匹配 / `!~` 正则不匹配
4. 正则缓存：`sync.Map` 缓存编译后的正则，避免重复编译

**完整文档**：→ [docs/data-source-routing.md](data-source-routing.md)

---

## AI 规则生成管道

### 端到端流程

```
用户输入自然语言描述
        │
        ▼
┌───────────────────┐
│  1. 检查缓存       │  ← SHA256(description + dsID + ruleType)
└───────┬───────────┘
        │ 缓存未命中
        ▼
┌───────────────────┐
│  2. 构建上下文      │
│  - 标签注册表       │  ← 最多 50 个标签键 + 每个键 5 个常用值
│  - 已有规则         │  ← 最多 30 条（避免重复）
│  - 预置规则匹配     │  ← 关键词搜索
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  3. 构建 Prompt     │
│  - System Prompt    │  ← 角色定义 + 输出格式 + 上下文
│  - Few-Shot 示例    │  ← 告警/抑制/静默三种模板
│  - User Prompt      │  ← 用户描述 + 数据源信息
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  4. 调用 LLM       │  ← JSON 模式输出
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  5. 后处理          │
│  - 标准化 severity  │  ← 非标准值修正为 warning
│  - 默认 for_duration│  ← 空值默认 "0s"
│  - 钳制 confidence  │  ← [0, 1]
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  6. 缓存结果        │  ← 10 分钟 TTL
└───────┬───────────┘
        │
        ▼
    返回结果（或 DryRun 含验证）
```

### 核心组件

```
RuleGeneratorService
├── AIService              — LLM 调用封装（多供应商）
├── LabelRegistryService   — 标签注册表查询
├── DataSourceService      — 数据源查询（表达式验证）
├── AlertRuleService       — 已有规则查询
├── PresetRuleRepository   — 预置规则搜索
└── RuleGenCache           — 内存 TTL 缓存（sync.Map）
```

### 文件结构

| 文件 | 职责 |
|------|------|
| `internal/service/rule_generator.go` | 核心生成逻辑、DryRun、Validate、SuggestLabels、ImproveRule |
| `internal/service/rule_gen_prompts.go` | Few-shot prompt 模板（告警/抑制/静默） |
| `internal/service/rule_gen_cache.go` | 内存 TTL 缓存（SHA256 key + 10min TTL） |
| `internal/handler/ai_rule.go` | HTTP 端点处理 |

### DryRun 模式

DryRun 将规则生成和 PromQL 验证合并为一次调用，让用户在保存前预览效果：

```json
{
  "rule": { "name": "HighCPUUsage", "expression": "...", "severity": "warning" },
  "validation": {
    "valid": true,
    "result_type": "vector",
    "sample_count": 15,
    "sample_labels": ["instance", "job"],
    "warnings": []
  }
}
```

### Save as Draft 工作流

```
1. 用户输入描述 → POST /ai/rules/generate
2. 展示结果，用户可编辑 → POST /ai/rules/validate
3. 用户点击"改进" → POST /ai/rules/improve
4. 用户确认 → POST /alert-rules（status=draft）
5. 用户手动启用 → PATCH /alert-rules/:id（status=enabled）
```

### 缓存策略

| 参数 | 值 | 说明 |
|------|---|------|
| Key | SHA256(description + dsID + ruleType) | 三维度精确匹配 |
| TTL | 10 分钟 | 默认值 |
| 清理间隔 | 5 分钟 | 后台 goroutine 定期清理 |
| 存储 | 内存 `sync.Map` | 进程重启后丢失 |

**完整文档**：→ [docs/ai-rule-generation.md](ai-rule-generation.md)

---

## 预置规则库

### 概述

PresetRule 是告警规则的**模板**，来源于社区最佳实践、厂商推荐或从 Prometheus YAML 导入。用户一键应用即可创建实际的 AlertRule。

### 数据模型

```go
type PresetRule struct {
    BaseModel
    Name        string     `gorm:"size:200;not null;index"`
    DisplayName string     `gorm:"size:200"`
    Category    string     `gorm:"size:50;index"`       // database/kubernetes/middleware/...
    Component   string     `gorm:"size:50"`              // redis/mysql/kafka/...
    Expression  string     `gorm:"type:text;not null"`
    ForDuration string     `gorm:"size:32"`
    Severity    string     `gorm:"size:20;index"`
    Labels      JSONLabels `gorm:"type:json"`
    Annotations JSONLabels `gorm:"type:json"`
    Source      string     `gorm:"size:100"`             // monitoring-trading / yaml_import
    IsBuiltin   bool       `gorm:"default:true"`         // 内置规则不可删除
    UsageCount  int        `gorm:"default:0"`            // 被应用次数
    Description string     `gorm:"type:text"`
}
```

### 内置规则覆盖

| 分类 | 数量 | 典型规则 |
|------|------|---------|
| database | 8+ | MySQLDown, RedisDown, ElasticsearchClusterRed |
| kubernetes | 6+ | KubeNodeNotReady, KubePodCrashLooping |
| middleware | 5+ | KafkaExporterDown, NginxHighErrorRate |
| node-exporter | 7+ | HighCPUUsage, HighMemoryUsage, DiskSpaceRunningLow |
| probe | 4+ | BlackboxHttpProbeFailed, BlackboxTcpProbeFailed |
| windows-exporter | 4+ | WindowsHighCPUUsage, WindowsDiskSpaceLow |
| inhibition | 16 | 严重等级级联、组件 Down 级联、K8s 节点级联 |

**总计**：45 条告警规则 + 16 条抑制模板

### Apply 流程

```
用户浏览预置规则 → GET /preset-rules?category=database
        │
        ▼
选择规则 → POST /preset-rules/:id/apply
  {
    "datasource_id": 1,
    "channel_id": 2,
    "severity": "critical",
    "labels": {"env": "prod"}
  }
        │
        ▼
PresetRuleService.Apply:
  1. 加载 PresetRule
  2. 验证数据源
  3. 创建 AlertRule（覆盖数据源/通道/标签/严重等级）
  4. 增加 usage_count
  5. 评估引擎自动发现新规则
```

### 从 monitoring-trading YAML 导入

```bash
# 预览模式
go run scripts/import-presets/main.go --dir=/path/to/monitoring-trading/alerts --dry-run

# 实际导入（299 条规则）
go run scripts/import-presets/main.go --dir=/path/to/monitoring-trading/alerts --dsn="..."
```

**严重等级映射**：P0 → critical, P1 → warning, P2/P3 → info

**完整文档**：→ [docs/preset-rule-library.md](preset-rule-library.md)

---

## 四层事件模型

```
Event（原始事件）
  → Alert（去重聚合，按 alert_key，关联 Channel）
    → Incident（故障，多 Alert 聚合，完整生命周期）
      → Channel（协作空间，多 Incident，排除/分派/降噪配置）
```

- **Event**：从引擎或 Integration Webhook 接收的原始告警信号
- **Alert**：按 `alert_key`（规则ID + 指纹）去重，持续更新 `last_fired_at`，不产生重复噪音
- **Incident**：关联一批同类 Alert 的故障对象，支持 ack / close / reopen / snooze / merge / reassign / escalate
- **Channel**：协作空间，一个 Channel 关联多个 Incident，含降噪规则、排除规则、分派策略

---

## 通知管道 v2

### 管道全景

```
┌──────────────┐
│ 告警引擎      │  评估规则 → 产生 AlertEvent
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 抑制检查      │  InhibitionRule — 高优先级告警抑制低优先级
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 静默检查      │  MuteRule — 维护窗口/免打扰时段
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 分组等待      │  AlertGroupManager — group_wait / group_interval
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 通知路由      │  NotificationService.RouteAlert
│  ├ NotifyRule │  → 标签匹配 + 严重等级 + 数据源过滤
│  └ Subscribe  │  → 用户/团队订阅规则
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 模板渲染      │  MessageTemplate → 渲染通知内容
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 媒体分发      │  NotifyMediaService → 飞书/邮件/Webhook/Bot
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 升级策略      │  EscalationExecutor — SLA 超时升级
└──────────────┘
```

### AlertChannel vs NotifyRule

| 特性 | AlertChannel（静态路由） | NotifyRule（动态管道） |
|------|------------------------|----------------------|
| 定位 | 简单标签→群组路由 | 灵活事件处理管道 |
| 匹配维度 | 标签 + 严重等级 + 数据源 | 标签 + 严重等级 + 数据源 + 管道步骤 |
| 通知目标 | 单个 NotifyMedia | 多个 NotifyConfig（按严重等级分发） |
| 节流控制 | `ThrottleMin`（分钟级） | `RepeatInterval`（秒级） |
| 模板 | 可选 TemplateID 覆盖 | 每个 NotifyConfig 独立模板 |
| 管道处理 | 无 | 支持 relabel、AI summary |
| 回调 | 无 | 可配置 CallbackURL |

### 分组等待机制

| 参数 | 含义 | 默认值 |
|------|------|--------|
| `GroupWaitSeconds` | 首次告警的缓冲时间 | 0（禁用） |
| `GroupIntervalSeconds` | 后续通知的最小间隔 | 0（禁用） |

**分组键**：`{GroupName}:{RuleID}` 或 `rule:{RuleID}`（每条规则独立分组）

### 级别抑制与 GC

严重等级抑制（LevelSuppressor）防止低级别告警在高级别告警活跃期间重复通知：

- 基于 `rule_id + severity` 维度的抑制状态
- Redis Hash 持久化引擎状态
- **GC 机制**：1 小时 ticker 扫描，24 小时过期自动清理
- 风暴预警：Redis 滚动计数器，阈值触发风暴模式

### 升级策略集成

```
EscalationExecutor（60s 轮询）
  → 查询 firing 状态事件
  → 批量加载关联规则（SLA 检查）
  → 对每个事件：
      ├─ 检查升级策略 → 执行到期步骤
      └─ 检查 SLA → 超时未确认触发升级
```

| TargetType | 行为 |
|-----------|------|
| `user` | 直接通知指定用户 |
| `team` | 通知团队所有成员 |
| `schedule` | 通知当前值班人（从 OnCallShift 查询） |

**完整文档**：→ [docs/notification-pipeline.md](notification-pipeline.md)

---

## AlertV2 处理管道

```
Integration Webhook
  → normalise（格式归一化：Standard / AlertManager / Grafana）
  → rate limit（令牌桶 100/s + 1000/min）
  → applyPipeline（rewrite_severity / title / desc / drop）
  → DispatchService.ApplyLabelEnhancements（标签增强：extract / combine / map / delete）
  → upsertAlert（按 alert_key 去重合入 Alert 表）
  → ensureIncident（复用/新建 Incident）
  → NoiseReducer（聚合 / 风暴预警 / 抖动检测）
  → DispatchService.FindMatchingPolicy（分派策略匹配：条件 / 时间 / 延迟）
  → sendNotification（按分派策略通知）
```

原有告警引擎通过 **WrapOnAlert hook** 注入，产生的告警同样经过上述 v2 管道处理。

---

## 数据模型

### 核心实体关系

```
DataSource ─1:N─ AlertRule ─1:N─ AlertEvent ─1:N─ AlertTimeline
                                                    │
PresetRule ──Apply──→ AlertRule                      │
                                                     │
AlertRule.Status: draft / enabled / disabled         │
                                                     │
Team ─1:N─ TeamMember ─N:1─ User                     │
User.global_role: admin / team_lead / member / viewer│
                                                     │
EscalationPolicy ─1:N─ EscalationStep                │
                                                     │
AlertChannel ── match_labels + DataSourceID ──→ NotifyMedia
NotifyRule ── match_labels + DataSourceID ──→ NotifyConfig ──→ NotifyMedia
DispatchPolicy ── DataSourceID + MatchConditions ──→ EscalationPolicy
SubscribeRule ── match labels ──→ NotifyMedia
```

### AlertRule 状态枚举

| 状态 | 说明 | 评估引擎行为 |
|------|------|-------------|
| `draft` | 草稿（AI 生成 / 手动创建） | 不评估 |
| `enabled` | 启用 | 正常评估 |
| `disabled` | 禁用 | 不评估，保留配置 |

**设计决策**：v4.13.0 移除了 `muted` 状态（静默由 MuteRule 独立管理），新增 `draft` 状态支持 AI 生成的 save-as-draft 工作流。

### 涉及 DataSourceID 的模型

```go
AlertRule.DataSourceID       *uint  // 告警规则绑定数据源
AlertChannel.DataSourceID    *uint  // 通知通道过滤数据源
NotifyRule.DataSourceID      *uint  // 通知规则过滤数据源
DispatchPolicy.DataSourceID  *uint  // 分派策略过滤数据源
```

`nil` = 通配（匹配任意数据源），`非 nil` = 精确匹配。

---

## 关键架构决策

| ADR | 决策 | 原因 |
|-----|------|------|
| ADR-1 | AI/Lark 配置存 DB（system_settings），AES-256-GCM 加密 | 密钥不出现在 ConfigMap/Secret |
| ADR-2 | golang-migrate 是 schema 唯一来源，GORM AutoMigrate 只作安全网 | 迁移可审计可回滚 |
| ADR-3 | Redis Hash 持久化引擎状态 | 重启后恢复飞行中告警，Redis 不可用时降级到纯内存 |
| ADR-4 | OIDC 配置存 DB，启动时合并 configmap | 运行时配置无需重启 |
| ADR-5 | RBAC 双层模型（全局角色 + 团队角色合并） | 细粒度权限控制，团队角色只能提升不能限制 |
| ADR-6 | AlertV2Pipeline 非侵入式 hook（WrapOnAlert） | 原引擎无需修改，v1/v2 路径并行，向前兼容 |
| ADR-7 | Channel/Incident/Alert 三层事件模型（参照 FlashCat） | 结构化故障协作，减少告警噪音，支持根因聚合 |
| ADR-8 | 共享集成路由规则（RoutingRule 表，优先级匹配） | 一个 Integration 可路由到多个 Channel，减少重复配置 |
| ADR-9 | 分派策略独立于 Channel，多策略优先级排序匹配 | 灵活配置不同条件/时间的通知策略，降低耦合 |
| ADR-10 | NoiseReducer in-memory flapState + Redis 滚动计数器 | 风暴预警不依赖 DB 写入，抖动检测低延迟 |
| ADR-11 | DataSourceID 通配（nil）+ 精确（非 nil）双模式 | 全局规则与特定规则共存，精确优先于类型绑定 |
| ADR-12 | AI 规则生成缓存（SHA256 key + 10min TTL） | 避免重复 LLM 调用，内存 sync.Map 轻量实现 |
| ADR-13 | AlertRule.Status 移除 muted，新增 draft | 静默由 MuteRule 独立管理；draft 支持 AI save-as-draft 工作流 |
| ADR-14 | RequirePerm 支持 warn/deny 两种模式 | 灰度上线时 warn 记录但不阻断，生产切换为 deny 强制执行 |
| ADR-15 | labelmatch 统一标签匹配引擎 + 正则缓存 | 替换各模块分散的匹配逻辑，sync.Map 缓存编译后正则 |
| ADR-16 | i18n 分层：UI 文案在前端 / API 错误"码+前端翻译" / 出站文案后端 per-recipient | 翻译归属取决于文案"给谁看、走不走前端"，详见 [docs/i18n.md](i18n.md) |

---

## 数据库迁移文件

迁移文件路径：`internal/pkg/dbmigrate/migrations/`，使用 golang-migrate 管理。

### v2 新增（000019-000033）

| 序号 | 文件 | 创建的表 |
|------|------|----------|
| 000019 | create_channels | channels |
| 000020 | create_channel_stars | channel_stars |
| 000021 | create_channel_exclusion_rules | channel_exclusion_rules |
| 000022 | create_incidents | incidents |
| 000023 | create_incident_assignees | incident_assignees |
| 000024 | create_incident_timelines | incident_timelines |
| 000025 | create_post_mortems | post_mortems |
| 000026 | create_alerts_v2 | alerts（v2） |
| 000027 | create_alert_events_v2 | alert_events_v2 |
| 000028 | create_integrations | integrations |
| 000029 | create_routing_rules | routing_rules |
| 000030 | seed_default_channel | INSERT default channel |
| 000031 | create_dispatch_policies | dispatch_policies |
| 000032 | create_dispatch_logs | dispatch_logs |
| 000033 | alert_rule_channel | ALTER alert_rules ADD channel_id |

### v4.x 新增

| 序号 | 文件 | 说明 |
|------|------|------|
| 000045 | create_notifications | 用户通知中心 |
| 000046 | create_todo_items | 个人待办事项 |
| 000047 | add_datasource_id_to_routing | alert_channels / notify_rules / dispatch_policies 新增 datasource_id 列 |
