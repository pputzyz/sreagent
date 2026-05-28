# SREAgent 架构修复计划

> 2026-05-27 | 基于深度代码分析

## 问题全景

当前告警→通知→升级链路存在 **6 个断裂点**，导致：
- 通知可能重复发送（NotifyRule + Escalation 各发一次）
- 降噪在通知之后执行（噪音告警已经通知出去了）
- DispatchPolicy 的升级策略关联是死代码
- 前端菜单混乱，用户不知道该用哪个功能

## 当前事件流（有问题的）

```
告警引擎评估触发
    │
    ▼
onAlertFn (wire.go:462)
    ├─ ① 抑制检查 (同步)
    ├─ ② 静默检查 (同步)
    ├─ ③ 业务分组标注 (同步)
    ├─ ④ AlertGroupManager.ProcessEvent (同步)
    │       │
    │       ▼ 缓冲后 flush
    │   NotificationService.RouteAlert
    │       ├─ NotifyRule 匹配 → Pipeline → 模板 → 发送通知  ← 路径 A
    │       └─ 订阅规则匹配 → 发送通知
    │
    └─ ⑤ AlertV2Pipeline.WrapOnAlert (异步 goroutine)
            ├─ 降噪检查  ← 太晚了，路径 A 已经发了
            ├─ DispatchPolicy 匹配 → 标签增强  ← 增强了但没人用
            ├─ Upsert Alert 记录
            └─ 创建/关联 Incident

    ┌─────────────────────────────────────────┐
    │  EscalationExecutor (独立后台 goroutine) │
    │  每 60s 轮询所有 firing 事件              │
    │  匹配升级策略 → 执行升级步骤              │ ← 路径 C，完全独立
    │  不知道路径 A 是否已经通知过              │
    └─────────────────────────────────────────┘
```

**问题**：
1. 路径 A（NotifyRule）和路径 C（Escalation）互不感知，可能重复通知
2. 降噪在路径 A 之后执行，无法阻止噪音通知
3. DispatchPolicy 的标签增强结果不传递给 NotifyRule
4. DispatchPolicy.EscalationPolicyID 字段从未被读取

---

## 修复方案：统一事件处理管线

### 目标事件流

```
告警引擎评估触发
    │
    ▼
onAlertFn (统一入口)
    ├─ ① 抑制检查
    ├─ ② 静默检查
    ├─ ③ 降噪检查  ← 移到这里，同步执行
    ├─ ④ 业务分组标注
    ├─ ⑤ Channel 路由（基于规则的 channel_id 或标签匹配）
    │
    ▼
AlertPipeline.ProcessEvent (统一管线)
    ├─ DispatchPolicy 匹配 → 标签增强
    ├─ 创建 AlertEvent + Alert 记录
    ├─ 创建/关联 Incident
    │
    ▼
NotificationRouter (统一路由)
    ├─ NotifyRule 匹配 → Pipeline → 模板 → 发送  ← 即时通知
    ├─ 订阅规则匹配 → 发送
    │
    ▼
EscalationScheduler (统一调度)
    ├─ 如果 DispatchPolicy 关联了 EscalationPolicy → 注册升级计划
    ├─ 如果没有关联 → 使用团队默认升级策略
    └─ 升级执行器检查"是否已通知"再决定是否升级
```

---

## 分阶段修复

### Phase 1: 降噪前置 + 去重复通知（紧急）

**目标**: 解决噪音通知和重复通知

#### 1.1 降噪移到同步路径

**修改**: `internal/engine/wire.go`

在 `onAlertFn` 中，将降噪检查从 AlertV2Pipeline 移到同步路径：

```
当前顺序: 抑制 → 静默 → 分组标注 → ProcessEvent(发通知) → [异步]降噪
目标顺序: 抑制 → 静默 → 降噪 → 分组标注 → ProcessEvent(发通知)
```

**文件变更**:
- `internal/engine/wire.go` — 在调用 `alertGroupMgr.ProcessEvent` 之前加入降噪检查
- `internal/service/noise_reducer.go` — 暴露同步检查接口 `ShouldSuppress(ctx, event) bool`

#### 1.2 EscalationExecutor 检查通知状态

**修改**: `internal/engine/escalation_executor.go`

在执行升级步骤前，检查该事件是否已经通过 NotifyRule 路径发送过通知：

**文件变更**:
- `internal/engine/escalation_executor.go` — `executeStep` 方法中增加通知状态检查
- `internal/model/alert_timeline.go` — 增加 `notification_sent` 事件类型
- `internal/service/notification.go` — 发送通知后记录 timeline 事件

#### 1.3 统一通知去重

**问题**: NotifyRule 用 `routeDedup.TrySend`，Escalation 用 `EscalationStepExecution`，两套去重。

**方案**: 引入统一的 `NotificationDedupService`

**文件变更**:
- `internal/service/notify_dedup.go` — 新建，基于 Redis 的去重服务
- `internal/service/notify_rule.go` — 使用统一去重替代 `routeDedup`
- `internal/engine/escalation_executor.go` — 使用统一去重替代 `EscalationStepExecution`

---

### Phase 2: 统一通知路由（核心）

**目标**: 合并 NotifyRule 和 Escalation 为统一管线

#### 2.1 DispatchPolicy 连接 EscalationPolicy

**问题**: `DispatchPolicy.EscalationPolicyID` 是死代码。

**修改**: 让 AlertV2Pipeline 在匹配 DispatchPolicy 后，注册升级计划

**文件变更**:
- `internal/service/alert_v2_pipeline.go` — `process()` 方法中读取 `policy.EscalationPolicyID`，写入 `AlertEvent.EscalationPolicyID`
- `internal/model/alert_event.go` — 确认 `EscalationPolicyID` 字段存在（或新增）
- `internal/engine/escalation_executor.go` — 优先使用事件关联的升级策略，而非全局匹配

#### 2.2 NotificationService 统一入口

**目标**: EscalationExecutor 的通知也通过 NotificationService 发送，共享模板和渠道配置。

**文件变更**:
- `internal/service/notification.go` — 新增 `SendEscalationNotify(ctx, event, step)` 方法
- `internal/engine/escalation_executor.go` — `executeStep` 调用 `NotificationService.SendEscalationNotify` 替代直接调用 `NotifyMediaService`

#### 2.3 前端菜单整合

**问题**: NotifyRule 和 DispatchPolicy 是两套并行机制，前端菜单让人困惑。

**方案**: 统一为"通知策略"概念

**菜单重组**:

```
On-Call
├── 概览
├── 我的告警
│
├── 协作空间
│   ├── 协作空间列表
│   ├── 故障管理
│   ├── 状态页面
│   └── 复盘
│
├── 值班管理
│   ├── 排班管理
│   └── 升级策略
│
├── 通知配置
│   ├── 通知策略（合并 NotifyRule + DispatchPolicy）
│   ├── 通知渠道（合并 NotifyMedia + AlertChannel）
│   ├── 消息模板
│   └── 我的订阅
│
├── 集成
│   ├── 集成中心
│   └── 路由规则
│
└── 配置
    └── 业务分组
```

**文件变更**:
- `web/src/composables/useAppNav.ts` — 重组菜单结构
- `web/src/router/index.ts` — 调整路由
- i18n 文件 — 更新菜单标签

---

### Phase 3: Channel 枢纽（增强）

**目标**: 让 Channel（协作空间）成为告警路由的核心枢纽

#### 3.1 告警规则绑定 Channel

**现状**: 告警规则有 `channel_id` 字段，但引擎不使用它路由。

**修改**: 引擎触发告警时，基于规则的 `channel_id` 路由到对应 Channel

**文件变更**:
- `internal/engine/rule_eval_actions.go` — `onAlert` 回调中传递 `channel_id`
- `internal/service/alert_v2_pipeline.go` — 使用 `channel_id` 查找 DispatchPolicy

#### 3.2 Channel 级别配置

**现状**: DispatchPolicy 绑定 Channel，但降噪、排除规则等配置分散。

**目标**: Channel 成为配置聚合点

**文件变更**:
- `internal/model/channel.go` — 确认关联字段完整
- 前端 Channel 详情页 — 展示关联的 DispatchPolicy、降噪规则、排除规则

---

### Phase 4: 前端体验优化（完善）

#### 4.1 统一通知策略页面

**问题**: 当前有"通知策略"（NotifyRule）和"通知规则"（Rules.vue）两个页面。

**方案**: 合并为一个页面，统一管理

#### 4.2 菜单图标去重

**问题**: 多个菜单使用相同图标，用户难以区分。

**文件变更**:
- `web/src/composables/useAppNav.ts` — 为重复图标的菜单分配不同图标

#### 4.3 清理死路由

**问题**: `oncall/config/notify-rules` 路由存在但不在菜单中。

**方案**: 删除或重定向

---

## 实施优先级

| 优先级 | Phase | 内容 | 影响 | 工作量 |
|--------|-------|------|------|--------|
| P0 | 1.1 | 降噪前置 | 修复噪音通知 | 小 |
| P0 | 1.2 | 升级检查通知状态 | 修复重复通知 | 中 |
| P0 | 1.3 | 统一去重 | 防止重复发送 | 中 |
| P1 | 2.1 | DispatchPolicy→Escalation 连通 | 修复死代码 | 中 |
| P1 | 2.2 | 统一通知入口 | 简化架构 | 大 |
| P1 | 2.3 | 前端菜单整合 | 改善 UX | 中 |
| P2 | 3.1 | Channel 枢纽 | 完善路由 | 中 |
| P2 | 3.2 | Channel 配置聚合 | 完善管理 | 小 |
| P3 | 4.1-4.3 | 前端优化 | 体验提升 | 中 |

---

## 验证标准

每个 Phase 完成后：

1. **go build** 通过
2. **go test** 通过
3. **npx vue-tsc --noEmit** 零错误（前端变更时）
4. **端到端测试**: 创建规则 → 触发告警 → 验证只收到一次通知 → 验证升级在指定延迟后触发
5. **降噪测试**: 创建降噪规则 → 触发匹配告警 → 验证不发送通知
6. **菜单验证**: 所有菜单项可点击，无空白页，无重复项

---

## 附录：关键文件索引

| 文件 | 作用 |
|------|------|
| `internal/engine/wire.go` | DI 组装，onAlertFn 定义 |
| `internal/engine/rule_eval_actions.go` | 告警触发回调 |
| `internal/engine/escalation_executor.go` | 升级执行器 |
| `internal/service/notification.go` | 通知路由 |
| `internal/service/notify_rule.go` | NotifyRule 匹配和处理 |
| `internal/service/dispatch.go` | DispatchPolicy 匹配 |
| `internal/service/alert_v2_pipeline.go` | V2 管线（降噪+Dispatch+Incident） |
| `internal/service/noise_reducer.go` | 降噪服务 |
| `web/src/composables/useAppNav.ts` | 前端菜单定义 |
| `web/src/router/index.ts` | 前端路由 |
