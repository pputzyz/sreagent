# 通知管道

> **v4.13.0** | 告警事件 → 抑制检查 → 静默检查 → 分组等待 → 通知分发

## 目录

- [管道概述](#管道概述)
- [Pipeline 架构](#pipeline-架构)
- [AlertChannel vs NotifyRule](#alertchannel-vs-notifyrule)
- [分组等待机制 (group_wait / group_interval)](#分组等待机制)
- [节流与去重](#节流与去重)
- [模板渲染](#模板渲染)
- [升级策略集成](#升级策略集成)
- [回调通知](#回调通知)
- [DispatchPolicy 分发策略](#dispatchpolicy-分发策略)
- [v1 到 v2 迁移历史](#v1-到-v2-迁移历史)
- [API 端点](#api-端点)

---

## 管道概述

告警事件从产生到最终送达用户，经过以下管道阶段：

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
│ 业务分组      │  BizGroup — 按业务维度归类
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
│  ├ NotifyRule │  → 标签匹配 + 严重等级匹配 + 数据源过滤
│  └ Subscribe  │  → 用户/团队订阅规则
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 管道处理      │  Pipeline: relabel → AI 分析
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 模板渲染      │  MessageTemplate → 渲染通知内容
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 媒体分发      │  NotifyMediaService → 飞书/邮件/Webhook
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 升级策略      │  EscalationExecutor — SLA 超时升级
└──────────────┘
```

---

## Pipeline 架构

### 核心组件

```
NotificationService
├── SubscribeRuleService    — 用户/团队订阅规则
├── NotifyRuleService       — 通知规则 CRUD + 事件处理
└── AlertRuleRepository     — 查询规则的 datasource_id

AlertGroupManager
├── groups map[string]*alertGroup  — 按 group_key 缓冲事件
├── routeFunc                      — 下游通知分发函数
└── ruleRepo                       — 查询 group_wait/group_interval

EscalationExecutor
├── EscalationPolicyRepository  — 升级策略
├── EscalationStepRepository    — 升级步骤
├── AlertEventRepository        — 事件查询
├── AlertTimelineRepository     — 时间线记录
└── NotifyMediaService          — 通知分发
```

### 文件结构

| 文件 | 职责 |
|------|------|
| `internal/service/notification.go` | 主路由函数 `RouteAlert` |
| `internal/service/notify_rule.go` | NotifyRule CRUD + `ProcessEvent` |
| `internal/service/alert_channel.go` | AlertChannel CRUD + `FindMatchingChannels` |
| `internal/service/alert_group.go` | `AlertGroupManager` 分组缓冲 |
| `internal/engine/escalation_executor.go` | 升级策略执行器 |

---

## AlertChannel vs NotifyRule

系统中有两套通知路由机制，各有侧重：

### AlertChannel（通知通道）

**定位**：简单的标签匹配路由，适合"按标签分类发送到不同群"的场景。

| 特性 | 说明 |
|------|------|
| 匹配维度 | 标签 + 严重等级 + 数据源 |
| 通知目标 | 单个 NotifyMedia |
| 节流控制 | `ThrottleMin`（分钟级） |
| 模板 | 可选的 TemplateID 覆盖 |
| 管道处理 | 无 |

```go
type AlertChannel struct {
    Name         string      // 通道名称
    MatchLabels  JSONLabels  // 标签匹配条件
    DataSourceID *uint       // 数据源过滤（nil=通配）
    Severities   string      // 严重等级过滤（空=全部）
    MediaID      uint        // 通知媒体 ID
    TemplateID   *uint       // 可选模板覆盖
    ThrottleMin  int         // 节流（分钟）
    IsEnabled    bool        // 是否启用
}
```

### NotifyRule（通知规则）

**定位**：灵活的事件处理管道，支持 AI 分析、relabel、多严重等级分发。

| 特性 | 说明 |
|------|------|
| 匹配维度 | 标签 + 严重等级 + 数据源 |
| 通知目标 | 多个 NotifyConfig（按严重等级分发） |
| 节流控制 | `RepeatInterval`（秒级） |
| 模板 | 每个 NotifyConfig 独立模板 |
| 管道处理 | 支持 relabel、AI summary 等步骤 |
| 回调 | 可配置 CallbackURL |

```go
type NotifyRule struct {
    Name           string        // 规则名称
    Severities     string        // 严重等级过滤
    MatchLabels    JSONLabels    // 标签匹配条件
    DataSourceID   *uint         // 数据源过滤（nil=通配）
    Pipeline       string        // 事件处理管道 JSON
    NotifyConfigs  string        // 通知配置 JSON
    RepeatInterval int           // 重复间隔（秒）
    CallbackURL    string        // 回调 URL
}
```

### 选择建议

| 场景 | 推荐使用 |
|------|---------|
| 简单的标签→群组路由 | AlertChannel |
| 需要 AI 分析告警 | NotifyRule |
| 不同严重等级发到不同媒体 | NotifyRule |
| 需要 relabel 处理 | NotifyRule |
| 需要回调通知 | NotifyRule |

---

## 分组等待机制

### AlertGroupManager

`AlertGroupManager` 实现了 Alertmanager 风格的告警分组，避免同一组告警在短时间内多次触发通知。

### 分组键 (Group Key)

```
格式: "{GroupName}:{RuleID}" 或 "rule:{RuleID}"
```

- 如果规则设置了 `GroupName`，使用 `{GroupName}:{RuleID}`
- 否则使用 `rule:{RuleID}`（每条规则独立分组）

### 分组时机

| 参数 | 含义 | 默认值 |
|------|------|--------|
| `GroupWaitSeconds` | 首次告警的缓冲时间 | 0（禁用） |
| `GroupIntervalSeconds` | 后续通知的最小间隔 | 0（禁用） |

### 处理流程

```
事件进入 ProcessEvent
    │
    ├─ 状态是 resolved → 直接发送（不分组）
    │
    ├─ group_wait=0 且 group_interval=0 → 直接发送（分组禁用）
    │
    └─ 启用分组:
        │
        ├─ 首次通知（lastFlush 为零值）:
        │   启动 timer = group_wait
        │   timer 到期 → flushGroup → 发送所有缓冲事件
        │
        └─ 后续通知（lastFlush 非零值）:
            启动 timer = group_interval
            timer 到期 → flushGroup → 发送所有缓冲事件
```

### 代码示例

```go
func (m *AlertGroupManager) ProcessEvent(ctx context.Context, event *model.AlertEvent) error {
    // 解析事件直接发送
    if event.Status == model.EventStatusResolved {
        return m.routeFunc(ctx, event)
    }

    // 获取分组配置
    groupWait, groupInterval := m.getGroupTiming(event)
    if groupWait == 0 && groupInterval == 0 {
        return m.routeFunc(ctx, event) // 分组禁用
    }

    // 获取或创建分组
    groupKey := m.getGroupKey(event)
    g := m.getOrCreateGroup(groupKey, groupWait, groupInterval)

    // 添加事件到分组
    g.events = append(g.events, event)

    // 启动定时器（如果还没有）
    if g.timer == nil {
        delay := g.groupWait // 首次
        if !g.lastFlush.IsZero() {
            delay = g.groupInterval // 后续
        }
        g.timer = time.AfterFunc(delay, func() {
            m.flushGroup(groupKey)
        })
    }

    return nil
}
```

---

## 节流与去重

### 节流 (Throttle)

NotifyRule 使用 `RepeatInterval` 控制重复通知的间隔：

```go
func (s *NotifyRuleService) isThrottled(ctx context.Context, rule *model.NotifyRule, nc *model.NotifyConfig) bool {
    if rule.RepeatInterval <= 0 {
        return false
    }

    // 查询该 media + rule 的最后发送记录
    lastRecord, err := s.recordRepo.GetLastSentRecord(ctx, nc.MediaID, rule.ID)
    if err != nil {
        return false // 无历史记录，不节流
    }

    elapsed := time.Since(lastRecord.CreatedAt)
    throttleDuration := time.Duration(rule.RepeatInterval) * time.Second
    return elapsed < throttleDuration
}
```

**节流粒度**：基于 `media_id + rule_id` 的组合，不同媒体独立节流。

### 去重 (Dedup)

系统使用全局去重防止同一事件通过不同路径重复发送：

```go
// 去重键格式: "v2:{rule_id}:{media_id}:{fingerprint}"
dedupKey := fmt.Sprintf("v2:%d:%d:%s", rule.ID, nc.MediaID, event.Fingerprint)
if !routeDedup.TrySend(dedupKey) {
    // 已发送过，跳过
    continue
}
```

`routeDedup` 是一个进程内的去重器，基于 fingerprint（标签集合的哈希）判断是否已发送。

### AlertChannel 节流

AlertChannel 使用 `ThrottleMin`（分钟级）：

```go
// 节流检查基于最后发送时间
if ch.ThrottleMin > 0 {
    lastSent := getLastSentTime(ch.ID)
    if time.Since(lastSent) < time.Duration(ch.ThrottleMin)*time.Minute {
        continue // 节流中
    }
}
```

---

## 模板渲染

### 模板数据结构

```go
type TemplateData struct {
    AlertName   string            // 告警名称
    Severity    string            // 严重等级
    Status      string            // 状态（firing/resolved）
    Labels      map[string]string // 告警标签
    Annotations map[string]string // 注解
    FiredAt     time.Time         // 触发时间
    EventID     uint              // 事件 ID
    Source      string            // 来源
}
```

### 渲染流程

```go
// 1. 构建模板数据
templateData := EventToTemplateData(event, analysis)

// 2. 渲染模板
if nc.TemplateID > 0 {
    rendered, err := s.templateSvc.RenderTemplate(ctx, nc.TemplateID, templateData)
    if err != nil {
        // 降级到基本消息
        renderedContent = fmt.Sprintf("[%s] %s - %s", event.Severity, event.AlertName, event.Status)
    }
} else {
    // 无模板，使用基本格式
    renderedContent = fmt.Sprintf("[%s] %s - %s", event.Severity, event.AlertName, event.Status)
}
```

### 默认消息格式

当没有配置模板时，使用基本格式：

```
[{severity}] {alertName} - {status}
```

示例：`[critical] NodeDown - firing`

### AI 增强

如果 NotifyRule 的 Pipeline 中配置了 `ai_summary` 步骤，系统会调用 AI 服务生成告警分析：

```json
{
  "pipeline": [
    {"type": "ai_summary", "config": {"only_critical": true}}
  ]
}
```

AI 分析结果会附加到模板数据中，可用于生成更丰富的通知内容。

---

## 升级策略集成

### EscalationExecutor

`EscalationExecutor` 是后台守护进程，每 60 秒检查一次 firing 状态的告警事件，执行升级策略。

### 升级流程

```
1. 查询所有 firing 状态的事件（最多 10000 条）
2. 批量加载关联的规则（用于 SLA 检查）
3. 对每个事件：
   a. 检查升级策略 → 执行到期的升级步骤
   b. 检查 SLA → 超时未确认则触发 SLA 升级
```

### 升级步骤执行

```go
func (e *EscalationExecutor) escalateEvent(ctx context.Context, event *model.AlertEvent, now time.Time) {
    // 获取已执行的步骤（用于去重）
    executedSteps := e.executedStepOrders(ctx, event.ID)

    // 遍历所有启用的升级策略
    for _, policy := range policies {
        steps := e.stepRepo.ListByPolicyID(ctx, policy.ID)
        sort.Slice(steps, func(i, j int) bool {
            return steps[i].StepOrder < steps[j].StepOrder
        })

        for _, step := range steps {
            // 去重：检查是否已执行
            stepKey := fmt.Sprintf("step:%d", step.ID)
            if executedSteps[stepKey] {
                continue
            }

            // 检查是否到期
            dueAt := event.FiredAt.Add(time.Duration(step.DelayMinutes) * time.Minute)
            if now.Before(dueAt) {
                break // 后续步骤更不会到期
            }

            // 执行步骤
            e.executeStep(ctx, event, &policy, &step)
        }
    }
}
```

### 升级目标类型

| TargetType | 行为 |
|-----------|------|
| `user` | 直接通知指定用户（通过 UserNotifyConfig） |
| `team` | 通知团队所有成员 |
| `schedule` | 通知当前值班人（从 OnCallShift 查询） |

### 个人通知渠道

升级步骤支持三种个人通知渠道：

| 媒体类型 | 说明 | 配置格式 |
|---------|------|---------|
| `webhook` | 自定义 Webhook | `{"url": "https://..."}` |
| `lark_personal` | 飞书个人消息 | `{"user_id": "xxx"}` 或 `{"open_id": "ou_xxx"}` |
| `email` | 邮件（使用全局 SMTP） | `{"email": "user@example.com"}` |

### SLA 超时升级

当告警规则配置了 `AckSlaMinutes` 且事件在 SLA 窗口内未被确认时，自动触发升级：

```go
func (e *EscalationExecutor) checkSLABreach(ctx context.Context, event *model.AlertEvent, ...) {
    if rule.AckSlaMinutes <= 0 || event.SlaEscalatedAt != nil {
        return // SLA 未配置或已升级
    }

    slaDeadline := event.FiredAt.Add(time.Duration(rule.AckSlaMinutes) * time.Minute)
    if now.Before(slaDeadline) {
        return // 仍在 SLA 窗口内
    }

    // 标记 SLA 升级（防止重复触发）
    e.eventRepo.UpdateSLAEscalated(ctx, event.ID, now)

    // 记录时间线
    note := fmt.Sprintf("SLA breach: event not acknowledged within %d minutes", rule.AckSlaMinutes)
    e.recordTimeline(ctx, event.ID, note, nil)
}
```

---

## 回调通知

NotifyRule 支持配置 `CallbackURL`，当事件被处理后自动发送 HTTP POST 回调：

```go
func (s *NotifyRuleService) fireCallback(ctx context.Context, callbackURL string, event *model.AlertEvent, analysis *AlertAnalysis) {
    payload := map[string]interface{}{
        "event_id":   event.ID,
        "alert_name": event.AlertName,
        "severity":   event.Severity,
        "status":     event.Status,
        "labels":     event.Labels,
        "fired_at":   event.FiredAt,
    }
    if analysis != nil {
        payload["ai_analysis"] = analysis
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", callbackURL, strings.NewReader(string(body)))
    req.Header.Set("Content-Type", "application/json")

    client := safehttp.NewSafeClient(10 * time.Second)
    resp, err := client.Do(req)
    // ...
}
```

**回调用途**：
- 与外部系统集成（如 Jira、PagerDuty）
- 触发自动化修复流程
- 记录到外部日志系统

---

## DispatchPolicy 分发策略

### 概述

DispatchPolicy 是 v2 引入的通道级分发配置，提供更精细的控制：

```go
type DispatchPolicy struct {
    ChannelID           uint   // 关联通道
    Name                string
    Priority            int    // 优先级（数字越小越高）
    MatchConditions     string // JSON 过滤条件
    DataSourceID        *uint  // 数据源过滤
    ActiveTimeConfig    string // 时间窗口限制
    DelaySeconds        int    // 延迟窗口（秒）
    EscalationPolicyID  *uint  // 关联升级策略
    RepeatIntervalSeconds int  // 重复通知间隔
    MaxRepeats          int    // 最大重复次数
    NotifyMode          string // "personal_preference" | "unified"
    UnifiedMediaID      *uint  // 统一媒体 ID
    LabelEnhancementRules string // 标签增强规则
}
```

### 分发模式

| 模式 | 说明 |
|------|------|
| `personal_preference` | 尊重用户的个人通知配置 |
| `unified` | 使用统一的媒体配置 |

### 标签增强

DispatchPolicy 支持在分发前对告警标签进行增强处理：

| 操作类型 | 说明 |
|---------|------|
| `set` | 直接设置标签值 |
| `extract` | 从源字段正则提取 |
| `combine` | 模板拼接多个标签 |
| `map` | 查找表映射 |
| `delete` | 删除标签 |

---

## v1 到 v2 迁移历史

### v1 架构（NotifyPolicy）

```
AlertEvent → NotifyPolicy（简单匹配）→ NotifyChannel → 发送
```

- NotifyPolicy：简单的标签匹配 + 严重等级过滤
- NotifyChannel：通知渠道配置（飞书 Webhook、邮件等）
- 升级通过 NotifyChannel 的内嵌配置实现

### v2 架构（NotifyRule）

```
AlertEvent → NotifyRule（管道处理）→ NotifyConfig → NotifyMedia → 发送
```

- NotifyRule：灵活的事件处理管道
- NotifyConfig：按严重等级分发到不同媒体
- NotifyMedia：独立的通知媒体配置
- AlertChannel：轻量级标签路由（保留 v1 风格）
- DispatchPolicy：通道级精细控制

### 迁移要点

1. NotifyPolicy → NotifyRule：配置格式从简单匹配升级为管道模型
2. NotifyChannel → NotifyMedia：媒体配置独立管理
3. 新增 AlertChannel：作为 NotifyRule 的简化替代
4. 新增 DispatchPolicy：通道级分发策略
5. 新增 SubscribeRule：用户/团队订阅机制

### 兼容性

- v1 的 NotifyChannel 通过 `sendViaChannel` 适配器兼容 v2 的 NotifyMediaService
- 升级执行器同时支持 v1 NotifyChannel 和 v2 NotifyMedia
- 时间线记录使用 `EscalationStepID` 进行稳定去重（兼容旧版 Note 文本去重）

---

## API 端点

### NotifyRule

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/notify-rules` | 分页查询通知规则 |
| GET | `/notify-rules/:id` | 获取单条通知规则 |
| POST | `/notify-rules` | 创建通知规则 |
| PUT | `/notify-rules/:id` | 更新通知规则 |
| DELETE | `/notify-rules/:id` | 删除通知规则 |
| POST | `/notify-rules/batch-enable` | 批量启用 |
| POST | `/notify-rules/batch-disable` | 批量禁用 |
| POST | `/notify-rules/batch-delete` | 批量删除 |

### AlertChannel

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/alert-channels` | 分页查询通知通道 |
| GET | `/alert-channels/:id` | 获取单条通知通道 |
| POST | `/alert-channels` | 创建通知通道 |
| PUT | `/alert-channels/:id` | 更新通知通道 |
| DELETE | `/alert-channels/:id` | 删除通知通道 |
| POST | `/alert-channels/:id/test` | 测试通道配置 |

### DispatchPolicy

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/channels/:id/dispatch-policies` | 查询通道的分发策略 |
| POST | `/channels/:id/dispatch-policies` | 创建分发策略 |
| PUT | `/dispatch-policies/:id` | 更新分发策略 |
| DELETE | `/dispatch-policies/:id` | 删除分发策略 |

### EscalationPolicy

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/escalation-policies` | 查询升级策略 |
| POST | `/escalation-policies` | 创建升级策略 |
| PUT | `/escalation-policies/:id` | 更新升级策略 |
| DELETE | `/escalation-policies/:id` | 删除升级策略 |
| GET | `/escalation-policies/:id/steps` | 查询升级步骤 |
| POST | `/escalation-policies/:id/steps` | 创建升级步骤 |
