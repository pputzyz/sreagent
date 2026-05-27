# Alert/Oncall 模块重构设计

> SREAgent v4.42.0–v4.44.0 | 2026-05-26

## 背景

SREAgent 经过 7 轮 Nightingale 移植后，Alert 和 Oncall 模块存在严重的架构问题：

1. **模块边界模糊**：Oncall Config Center 引用 `/alert/notify/*` 路由，通知子系统归属不清
2. **路由机制重叠**：AlertChannel（简单路由）与 DispatchPolicy（复杂路由）功能重复
3. **数据源绑定过死**：AlertRule 必须绑定 DataSourceID，灵活性差
4. **团队通知缺失**：NotifyMedia 是全局的，没有团队级渠道配置
5. **菜单结构混乱**：两个模块的功能菜单交叉引用，用户找不到功能

## 设计决策

| 问题 | 决策 | 理由 |
|------|------|------|
| 告警路由机制 | 统一为 DispatchPolicy | 能力更强（延迟、重复、标签增强、升级），与 Nightingale 对齐 |
| 通知子系统归属 | Oncall 独占 | 告警引擎产出事件，通知管道由 Oncall 独立管理 |
| 数据源绑定 | 可选 | 空时使用同类型默认数据源，多查询规则每个 Query 独立指定 |
| 团队通知渠道 | 团队级配置 + 个人覆盖 | 团队统一管理渠道，成员可覆盖个人偏好 |

## 核心链路

```
告警规则 → 告警事件 → 分派策略(DispatchPolicy) → 升级策略(EscalationPolicy) → 通知渠道(NotifyMedia)
                         ↑                              ↑
                    标签匹配 + 团队                 排班(Schedule)
                    + 时间窗口 + 优先级            + 延迟 + 重复
```

## 模块职责划分

### Alert 模块 — 告警引擎 + 数据

- 数据源管理、数据查询、指标视图、ES 探索
- 告警规则（阈值/心跳/多查询/预置规则/录制规则）
- 告警事件（活跃/历史）
- 静默规则、抑制规则
- 事件管道（EventPipeline）

### Oncall 模块 — 响应 + 通知

- 协作空间（Channel）+ 事件管理（Incident）
- **分派策略（DispatchPolicy）** — 告警→渠道的核心路由
- **通知策略（NotifyRule）** — 定义谁收什么通知
- **通知渠道（NotifyMedia）** — 17 种渠道 + 团队级配置
- **消息模板（MessageTemplate）** — 通知内容模板
- **订阅规则（SubscribeRule）** — 个人/团队订阅
- 排班（Schedule）+ 升级策略（EscalationPolicy）
- 状态页 + 复盘

### Platform 模块 — 系统管理

- 用户/团队/权限
- 系统设置（SMTP/OIDC/LDAP/AI）
- 审计日志

---

## 数据模型变更

### 1. 废弃 AlertChannel → 迁移为 DispatchPolicy

现有 `alert_channels` 数据迁移为 `dispatch_policies`：

| AlertChannel 字段 | → DispatchPolicy 字段 | 转换逻辑 |
|---|---|---|
| `name` | `name`（新字段） | 直接复制 |
| `match_labels` | `match_conditions` | JSON 格式转换：`{labels: {...}}` → `[{field, op, value}]` |
| `datasource_id` | `datasource_id` | 直接复制 |
| `severities` | `match_conditions` | 追加 severity 条件 |
| `media_id` | `unified_media_id` | 直接复制 + `notify_mode='unified'` |
| `template_id` | `unified_template_id`（新字段） | 直接复制 |
| `throttle_min` | `repeat_interval_seconds` | ×60 转换 |

`dispatch_policies` 表新增字段：
- `name VARCHAR(255) NOT NULL DEFAULT ''` — 策略名称
- `unified_template_id BIGINT NULL` — 统一模板 ID

### 2. Team 通知渠道配置

新建 `team_notify_channels` 表：

```sql
CREATE TABLE team_notify_channels (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    team_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_team_media (team_id, media_id),
    INDEX idx_team_id (team_id)
);
```

### 3. NotifyMedia 增加团队归属

`notify_media` 表新增列：
```sql
ALTER TABLE notify_media ADD COLUMN team_id BIGINT NULL;
ALTER TABLE notify_media ADD INDEX idx_team_id (team_id);
```

- `team_id IS NULL` → 全局共享渠道
- `team_id IS NOT NULL` → 团队专属渠道

### 4. AlertRule.DataSourceID 改为可选

```sql
ALTER TABLE alert_rules MODIFY COLUMN datasource_id BIGINT NULL;
```

引擎评估逻辑变更：
- `datasource_id IS NOT NULL` → 使用指定数据源
- `datasource_id IS NULL` → 查找同类型的默认数据源（`datasource_type` 匹配 + `is_default=true`）
- 若无默认数据源 → 跳过评估 + 记录警告日志

### 5. 团队成员通知偏好

新建 `user_team_notify_prefs` 表：

```sql
CREATE TABLE user_team_notify_prefs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,
    is_muted BOOLEAN NOT NULL DEFAULT false,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_team_media (user_id, team_id, media_id),
    INDEX idx_user_id (user_id),
    INDEX idx_team_id (team_id)
);
```

---

## 路由变更

### 后端 API 路由

| 现有路由 | 新路由 | 变更类型 |
|---|---|---|
| `GET/POST/PUT/DELETE /alert-channels/*` | 委托给 DispatchPolicy | 兼容期保留 |
| `/notify-rules/*` | 不变 | — |
| `/notify-media/*` | 不变 | — |
| `/message-templates/*` | 不变 | — |
| `/subscribe-rules/*` | 不变 | — |
| 新增 | `/team-notify-channels/*` | 团队渠道 CRUD |
| 新增 | `/users/:id/team-notify-prefs/*` | 个人偏好 CRUD |

### 前端路由

| 现有路由 | 新路由 |
|---|---|
| `/alert/notify/policies` | `/oncall/notify/policies` |
| `/alert/notify/templates` | `/oncall/notify/templates` |
| `/alert/notify/channels` | `/oncall/notify/channels` |
| `/alert/notify/subscriptions` | `/oncall/notify/subscriptions` |
| `/alert/channels` | `/oncall/config/dispatch-policies` |

### 前端菜单结构

**Oncall 菜单（重构后）**

```
Oncall
├── Overview
├── My Alerts
├── Channels（协作空间）
├── Incidents（事件管理）
├── Status Page
├── Postmortems
├── Schedule（排班）
├── 通知中心
│   ├── 分派策略（DispatchPolicy，含原 AlertChannel 迁移数据）
│   ├── 通知策略（NotifyRule）
│   ├── 通知渠道（NotifyMedia + 团队渠道配置）
│   ├── 消息模板（MessageTemplate）
│   └── 订阅规则（SubscribeRule）
└── 配置中心
    ├── 集成（Integrations）
    ├── 路由规则（RoutingRules）
    ├── 升级策略（EscalationPolicy）
    └── 业务组（BizGroups）
```

**Alert 菜单（精简后）**

```
Alert
├── Overview
├── 规则管理
│   ├── 告警规则
│   ├── 预置规则库
│   └── 录制规则
├── 告警事件
│   ├── 活跃告警
│   └── 告警历史
├── 静默管理
│   ├── 静默规则
│   └── 抑制规则
└── 数据
    ├── 数据源
    ├── 数据查询
    ├── 指标视图
    ├── ES 探索
    ├── ES 索引模式
    ├── 事件管道
    ├── 内置指标
    ├── 仪表盘
    └── 内置仪表盘
```

---

## 通知链路详解

### 分派策略匹配流程

```
告警事件触发
  ↓
遍历所有启用的 DispatchPolicy（按 Priority 升序）
  ↓
MatchConditions 检查：
  - severity 匹配？
  - labels 匹配？
  - datasource_id 匹配？
  - 时间窗口内？
  ↓
匹配成功 → 创建 DispatchLog(pending)
  ↓
检查 NotifyMode：
  - 'unified' → 直接使用 UnifiedMediaID + UnifiedTemplateID
  - 'personal_preference' → 获取团队成员的 UserNotifyConfig
  ↓
获取通知目标：
  - 若有 EscalationPolicyID → 走升级链路
  - 否则 → 直接发送
  ↓
发送通知 → 更新 DispatchLog(status=sent/failed)
```

### 团队通知渠道获取逻辑

```
DispatchPolicy 匹配到 TeamID
  ↓
查询 team_notify_channels WHERE team_id = ?
  ↓
对每个渠道：
  1. 检查用户是否有 user_team_notify_prefs 覆盖
  2. 若 is_muted=true → 跳过
  3. 若有 media_id 覆盖 → 使用覆盖渠道
  4. 否则 → 使用团队默认渠道
  ↓
发送通知
```

---

## 实施计划

### v4.42.0 — 后端数据模型 + 迁移

**迁移文件：**
- 000092: dispatch_policies 新增 name + unified_template_id
- 000093: alert_channels → dispatch_policies 数据迁移
- 000094: 新建 team_notify_channels
- 000095: notify_media 新增 team_id
- 000096: 新建 user_team_notify_prefs
- 000097: alert_rules.datasource_id NULLABLE

**代码变更：**
- `internal/model/dispatch.go` — 新增 Name、UnifiedTemplateID 字段
- `internal/model/team_notify_channel.go` — 新模型
- `internal/model/user_team_notify_pref.go` — 新模型
- `internal/model/notify_media.go` — 新增 TeamID 字段
- `internal/model/alert_rule.go` — DataSourceID 改为 *uint
- `internal/repository/team_notify_channel.go` — CRUD
- `internal/repository/user_team_notify_pref.go` — CRUD
- `internal/service/team_notify_channel.go` — 业务逻辑
- `internal/service/user_team_notify_pref.go` — 业务逻辑
- `internal/handler/team_notify_channel.go` — HTTP handler
- `internal/handler/user_team_notify_pref.go` — HTTP handler
- `internal/router/` — 注册新路由
- `cmd/server/wire.go` — DI 注入
- `internal/engine/evaluator.go` — DataSourceID 为空时查找默认数据源
- AlertChannel 路由委托给 DispatchPolicy（兼容层）

### v4.43.0 — 前端路由迁移 + 菜单重构

**前端变更：**
- `web/src/composables/useAppNav.ts` — 菜单重新划分
- `web/src/router/index.ts` — 路由迁移 + redirect 兼容
- 通知相关页面路由从 `/alert/notify/*` 移到 `/oncall/notify/*`
- AlertChannel 页面迁移到 DispatchPolicy
- 新增团队渠道配置 UI（在通知渠道页面中）
- 移除 Oncall Config Center 中的跨模块引用

### v4.44.0 — AlertChannel 废弃 + 引擎适配

**后端变更：**
- 移除 AlertChannel 兼容路由
- 引擎 dispatch.go 适配新的分派流程
- 个人通知偏好 UI
- 团队渠道继承逻辑

---

## 向后兼容策略

| 版本 | 兼容措施 |
|---|---|
| v4.42.0 | `/api/v1/alert-channels/*` 路由保留，内部委托给 DispatchPolicy |
| v4.43.0 | 前端 `/alert/notify/*` 路由 redirect 到 `/oncall/notify/*` |
| v4.44.0 | 移除所有兼容路由和 redirect |

## 测试策略

- 每个迁移脚本的 up/down 测试
- DispatchPolicy 匹配逻辑单元测试（覆盖原 AlertChannel 场景）
- 前端路由 redirect 测试
- 团队渠道继承 + 个人覆盖逻辑测试
- E2E: 创建规则 → 触发事件 → 验证分派 → 验证通知发送
