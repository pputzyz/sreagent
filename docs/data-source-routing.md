# 多数据源路由

> **v4.13.0** | DataSourceID 路由语义 + 标签匹配引擎

## 目录

- [架构概述](#架构概述)
- [DataSourceID 字段语义](#datasourceid-字段语义)
- [告警规则的数据源绑定](#告警规则的数据源绑定)
- [评估引擎的分发逻辑](#评估引擎的分发逻辑)
- [通知路由中的数据源过滤](#通知路由中的数据源过滤)
- [labelmatch.MatchWithSourceID](#labelmatchmatchwithsourceid)
- [AlertChannel 路由](#alertchannel-路由)
- [NotifyRule 路由](#notifyrule-路由)
- [DispatchPolicy 路由](#dispatchpolicy-路由)
- [实战示例](#实战示例)
- [配置指南](#配置指南)

---

## 架构概述

在生产环境中，通常存在多个 Prometheus/VictoriaMetrics 实例，每个实例负责不同的监控维度：

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  cc-prom      │  │  cpp-prom     │  │  vm-cluster  │
│  (容器监控)    │  │  (C++ 服务)   │  │  (全局指标)   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └────────────┬────┘─────────────────┘
                    │
            ┌───────▼────────┐
            │    SREAgent     │
            │  告警引擎 + 路由 │
            └────────────────┘
```

SREAgent 通过 `DataSourceID` 字段实现了**数据源级别的路由隔离**，确保：
- 告警规则只在指定数据源上评估
- 通知通道只接收来自指定数据源的告警
- 抑制规则在数据源维度上隔离

---

## DataSourceID 字段语义

系统中多个模型都包含 `DataSourceID` 字段，其语义统一为：

| 值 | 含义 | 行为 |
|---|------|------|
| `nil` (NULL) | 通配符 | 匹配任意数据源 |
| `非 nil` (具体 ID) | 精确匹配 | 只匹配该数据源 |

这个设计允许用户创建"全局规则"（绑定到所有数据源）或"特定规则"（只绑定到某个数据源）。

### 涉及的模型

```go
// AlertRule — 告警规则
type AlertRule struct {
    DataSourceID   *uint          `json:"datasource_id" gorm:"index"`
    DataSource     *DataSource    `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
    DatasourceType DataSourceType `json:"datasource_type" gorm:"size:32;index"`
}

// AlertChannel — 通知通道
type AlertChannel struct {
    DataSourceID *uint       `json:"datasource_id" gorm:"index"`
    DataSource   *DataSource `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
}

// NotifyRule — 通知规则
type NotifyRule struct {
    DataSourceID *uint       `json:"datasource_id" gorm:"index"`
    DataSource   *DataSource `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
}

// DispatchPolicy — 分发策略
type DispatchPolicy struct {
    DataSourceID *uint       `json:"datasource_id" gorm:"index"`
    DataSource   *DataSource `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
}
```

---

## 告警规则的数据源绑定

### 绑定方式

告警规则的数据源绑定有两种模式：

**模式 1：精确绑定**（`DataSourceID` 非 nil）

```json
{
  "name": "RedisHighMemory",
  "datasource_id": 1,
  "datasource_type": "prometheus",
  "expression": "redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80"
}
```

规则只在 ID=1 的数据源上评估。

**模式 2：类型绑定**（`DataSourceID` 为 nil，`DatasourceType` 非空）

```json
{
  "name": "HighCPUUsage",
  "datasource_type": "prometheus",
  "expression": "100 - avg by(instance)(rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100 > 90"
}
```

规则会在所有类型为 `prometheus` 的已启用数据源上评估。

### 优先级

精确绑定（`DataSourceID`）优先于类型绑定（`DatasourceType`）。当两者都存在时，以 `DataSourceID` 为准。

---

## 评估引擎的分发逻辑

`internal/engine/evaluator.go` 中的 `startRuleEvaluators` 方法实现了数据源分发：

```go
func (e *Evaluator) startRuleEvaluators(ctx context.Context, rule *model.AlertRule) {
    if rule.DataSourceID != nil {
        // 精确绑定：使用预加载的 DataSource
        ds := rule.DataSource
        if ds == nil {
            e.logger.Warn("rule has datasource_id but DataSource is nil — skipping")
            return
        }
        e.startRuleEvaluator(rule, ds)
        return
    }

    // 类型绑定：遍历所有匹配类型的已启用数据源
    if rule.DatasourceType == "" {
        e.logger.Warn("rule has no datasource_id and no datasource_type — skipping")
        return
    }

    dsList, err := e.dsRepo.ListEnabledByType(ctx, rule.DatasourceType)
    if err != nil {
        e.logger.Error("failed to list datasources by type", zap.Error(err))
        return
    }

    // 当前实现：使用第一个匹配的数据源
    // 未来计划：支持 fan-out 到多个数据源
    e.startRuleEvaluator(rule, &dsList[0])
}
```

### 规则同步机制

评估引擎每 30 秒从数据库同步规则：

1. 加载所有 `status=enabled` 的规则（Preload DataSource）
2. 对比已有的 evaluator：
   - 规则版本变化 → 停止旧 evaluator，启动新 evaluator
   - 新规则 → 启动新 evaluator
   - 已删除/禁用的规则 → 停止 evaluator
3. 每个 evaluator 在独立 goroutine 中运行

### 每个 RuleEvaluator 的生命周期

```
Start → 加载规则配置
     → 绑定数据源
     → 按 eval_interval 定期执行:
         1. 查询数据源（PromQL/LogQL）
         2. 对比历史状态
         3. 触发 firing/resolved 事件
         4. 调用 onAlert 回调
     → 监听 stopCh 退出
```

---

## 通知路由中的数据源过滤

当告警事件产生后，通知路由需要根据 `DataSourceID` 进行过滤。核心逻辑在 `NotificationService.RouteAlert` 中：

```go
func (s *NotificationService) RouteAlert(ctx context.Context, event *model.AlertEvent) error {
    // 从事件关联的规则中解析 datasource_id
    var dataSourceID *uint
    if s.ruleRepo != nil && event.RuleID != nil {
        if rule, err := s.ruleRepo.GetByID(ctx, *event.RuleID); err == nil {
            dataSourceID = rule.DataSourceID
        }
    }

    // 使用 dataSourceID 过滤匹配的通知规则
    rules, err := s.notifyRuleSvc.FindMatchingRules(ctx, event, dataSourceID)
    // ...
}
```

**关键点**：`DataSourceID` 从事件关联的告警规则中获取，然后传递给下游的匹配函数。

---

## labelmatch.MatchWithSourceID

`internal/pkg/labelmatch/matcher.go` 提供了统一的标签匹配引擎，支持数据源维度过滤。

### 函数签名

```go
func MatchWithSourceID(
    target map[string]string,  // 事件的标签
    targetDSID *uint,          // 事件的数据源 ID（来自告警规则）
    pattern map[string]string, // 匹配模式的标签
    patternDSID *uint,         // 匹配模式的数据源 ID（来自通道/规则）
) bool
```

### 匹配逻辑

```
1. 数据源维度检查：
   - 如果 patternDSID == nil → 通配，跳过数据源检查
   - 如果 patternDSID != nil：
       - targetDSID == nil → 不匹配
       - *patternDSID != *targetDSID → 不匹配
       - *patternDSID == *targetDSID → 继续标签检查

2. 标签维度检查（调用 Match 函数）：
   - 空 pattern → 通配，返回 true
   - 非空 pattern → 所有 pattern 条件必须满足（AND 逻辑）
```

### 标签操作符

`Match` 函数支持四种操作符前缀：

| 前缀 | 含义 | 示例 |
|------|------|------|
| （无） | 精确相等 | `"prod"` → target 值必须是 "prod" |
| `!=` | 不等于 | `"!=staging"` → target 值不能是 "staging" |
| `=~` | 正则匹配 | `"=~^prod-.*$"` → target 值必须匹配正则 |
| `!~` | 正则不匹配 | `"!~^test-.*$"` → target 值不能匹配正则 |

### 正则缓存

系统使用 `sync.Map` 缓存编译后的正则表达式，避免重复编译：

```go
var regexCache sync.Map // map[string]*regexp.Regexp

func getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
    if re, ok := regexCache.Load(pattern); ok {
        return re.(*regexp.Regexp), nil
    }
    // double-check locking
    regexCacheMu.Lock()
    defer regexCacheMu.Unlock()
    if re, ok := regexCache.Load(pattern); ok {
        return re.(*regexp.Regexp), nil
    }
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, err
    }
    regexCache.Store(pattern, re)
    return re, nil
}
```

---

## AlertChannel 路由

`AlertChannelService.FindMatchingChannels` 使用 `MatchWithSourceID` 过滤通道：

```go
func (s *AlertChannelService) FindMatchingChannels(
    ctx context.Context,
    event *model.AlertEvent,
    dataSourceID *uint,
) ([]model.AlertChannel, error) {
    channels, err := s.repo.ListEnabled(ctx)
    // ...

    var matched []model.AlertChannel
    for _, ch := range channels {
        // 数据源 + 标签双重过滤
        if !labelmatch.MatchWithSourceID(
            map[string]string(event.Labels), dataSourceID,
            map[string]string(ch.MatchLabels), ch.DataSourceID,
        ) {
            continue
        }
        // 严重等级过滤
        if ch.Severities != "" && !severityMatch(ch.Severities, string(event.Severity)) {
            continue
        }
        matched = append(matched, ch)
    }
    return matched, nil
}
```

**过滤链**：数据源匹配 → 标签匹配 → 严重等级匹配

---

## NotifyRule 路由

`NotifyRuleService.FindMatchingRules` 的过滤逻辑类似：

```go
func (s *NotifyRuleService) FindMatchingRules(
    ctx context.Context,
    event *model.AlertEvent,
    dataSourceID *uint,
) ([]model.NotifyRule, error) {
    return s.ruleRepo.FindMatchingRules(
        ctx,
        map[string]string(event.Labels),
        string(event.Severity),
        dataSourceID,
    )
}
```

Repository 层的 SQL 查询会同时检查 `datasource_id IS NULL OR datasource_id = ?` 条件。

---

## DispatchPolicy 路由

`DispatchPolicy` 同样支持 `DataSourceID` 过滤，但增加了更多维度：

- **MatchConditions**：JSON 格式的过滤条件数组
- **ActiveTimeConfig**：时间窗口限制（工作日/时间段）
- **Priority**：优先级排序（数字越小优先级越高）

```json
{
  "channel_id": 1,
  "name": "Critical Payment Alerts",
  "datasource_id": 2,
  "match_conditions": "[{\"field\":\"severity\",\"operator\":\"in\",\"value\":\"critical,p0\"}]",
  "priority": 10,
  "delay_seconds": 0,
  "escalation_policy_id": 1
}
```

---

## 实战示例

### 场景：双 Prometheus 实例隔离

假设环境中有两个 Prometheus 实例：

| ID | 名称 | 类型 | 用途 |
|----|------|------|------|
| 1 | cc-prom | prometheus | 容器集群监控 |
| 2 | cpp-prom | prometheus | C++ 服务监控 |

**需求**：
- cc-prom 的容器告警发送到"容器告警群"
- cpp-prom 的服务告警发送到"后端告警群"
- 全局告警（如节点宕机）发送到两个群

**配置**：

1. **告警规则**（精确绑定）：

```json
// 容器告警 — 绑定到 cc-prom
{
  "name": "PodCrashLooping",
  "datasource_id": 1,
  "expression": "rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 0",
  "severity": "critical"
}

// 服务告警 — 绑定到 cpp-prom
{
  "name": "HighErrorRate",
  "datasource_id": 2,
  "expression": "rate(http_requests_total{status=~\"5..\"}[5m]) / rate(http_requests_total[5m]) > 0.05",
  "severity": "warning"
}

// 全局告警 — 不绑定数据源
{
  "name": "NodeDown",
  "datasource_type": "prometheus",
  "expression": "up == 0",
  "severity": "critical"
}
```

2. **通知通道**（数据源过滤）：

```json
// 容器告警群 — 只接收 cc-prom 的告警
{
  "name": "容器告警群",
  "datasource_id": 1,
  "match_labels": {"category": "container"},
  "severities": "critical,warning",
  "media_id": 1
}

// 后端告警群 — 只接收 cpp-prom 的告警
{
  "name": "后端告警群",
  "datasource_id": 2,
  "match_labels": {},
  "severities": "critical,warning",
  "media_id": 2
}

// 全局告警群 — 接收所有数据源的告警（datasource_id = nil）
{
  "name": "全局告警群",
  "match_labels": {"severity": "critical"},
  "severities": "critical",
  "media_id": 3
}
```

**路由结果**：

| 事件 | 数据源 | 匹配通道 |
|------|--------|---------|
| PodCrashLooping (critical) | cc-prom (1) | 容器告警群 + 全局告警群 |
| HighErrorRate (warning) | cpp-prom (2) | 后端告警群 |
| NodeDown (critical) | cc-prom (1) | 全局告警群 |
| NodeDown (critical) | cpp-prom (2) | 全局告警群 |

---

## 配置指南

### 最佳实践

1. **优先使用精确绑定**：明确指定 `datasource_id`，避免意外的通配匹配
2. **全局规则保持通配**：如节点宕机、探测失败等通用规则不绑定数据源
3. **通知通道按团队划分**：每个团队的通道设置对应的数据源过滤
4. **抑制规则注意数据源**：确保抑制规则的数据源与被抑制的告警一致

### 常见陷阱

1. **忘记设置 `datasource_type`**：当 `datasource_id` 为 nil 且 `datasource_type` 为空时，规则会被跳过
2. **数据源被禁用**：已禁用的数据源不会被评估引擎使用
3. **通配符意外匹配**：`datasource_id = nil` 的通道会匹配所有数据源的告警

### 调试技巧

查看评估引擎状态：

```http
GET /api/v1/engine/status
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "running": true,
    "total_rules": 156,
    "active_alerts": 23,
    "uptime": "72h30m15s"
  }
}
```
