# SREAgent 产品对齐计划 — 参照 FlashCat/Flashduty

> 目标：将 SREAgent 从"告警引擎 + 基础 OnCall"升级为接近 Flashduty 能力的一站式告警响应平台。
> 参照文档：https://docs.flashcat.cloud/zh/home
> 范围约束：数据源仅 Prometheus/VM/VLogs，通知渠道仅飞书+邮件，不做 RUM。

---

## 一、范围说明

### 做什么
- On-Call 告警响应核心能力（协作空间、故障模型、降噪、路由、分派）
- Monitors 告警引擎增强（无数据告警、规则文件夹）
- 前端交互全面改造（参照 FlashCat UI 模式）
- Webhook 告警接入通道（Prometheus AlertManager / Grafana 兼容）
- 故障复盘 + AI 辅助

### 不做什么
- RUM（真实用户监控）
- 额外数据源（ES/ClickHouse/Loki/MySQL/Oracle/PostgreSQL/SLS）
- 额外通知渠道（钉钉/企微/Slack/Teams/电话/短信）
- 同比/环比告警、复合告警
- 状态页 (Status Page)
- 服务日历
- 作战室 (War Room)
- SSO SAML/LDAP（已有 OIDC 够用）

---

## 二、核心 Gap 分析

### 2.1 模型层（最大差距）

FlashCat 的核心是三层事件模型 + 协作空间隔离：

```
Channel (协作空间)
  └── Incident (故障) ─ 1:N ─ Alert (告警) ─ 1:N ─ Event (原始事件)
```

SREAgent 现状是扁平模型：
```
AlertRule ─ 1:N ─ AlertEvent (兼当告警和故障)
```

**需要新增**：Channel、Incident、Event 三个模型，AlertEvent 重构为 Alert。

### 2.2 On-Call 能力

| FlashCat 能力 | SREAgent 现状 | 是否实现 |
|-------------|------------|--------|
| 协作空间 (Channel) | 无 | **Phase 1** |
| 告警路由规则 | 无 | **Phase 1** |
| 故障 (Incident) 独立模型 | 无 | **Phase 1** |
| 告警聚合（规则聚合） | 仅指纹去重 | **Phase 2** |
| 抖动检测 | 无 | **Phase 2** |
| 静默增强（周期模式） | 基础 MuteRule | **Phase 2** |
| 抑制策略 | 有 InhibitionRule | 已有 |
| 分派策略绑定 Channel | 全局 Escalation | **Phase 3** |
| 标签增强 | 无 | **Phase 3** |
| 故障复盘 (Post-Mortem) | 无 | **Phase 5** |
| AI 辅助复盘/总结 | 有 AI 模块 | **Phase 5** |

### 2.3 告警引擎

| FlashCat 能力 | SREAgent 现状 | 是否实现 |
|-------------|------------|--------|
| 阈值告警 | 有 | 已有 |
| 无数据告警 | 无 | **Phase 4** |
| 规则文件夹/树形分类 | 平铺 group_name | **Phase 4** |
| Webhook 告警接入 | 无 | **Phase 4** |

---

## 三、前端交互改造方案

参照 FlashCat 的 UI 设计模式，SREAgent 前端需要做以下重点改造：

### 3.1 全局导航重构

**现状**：平铺菜单（仪表盘 / 数据查询 / 数据源 / 告警管理 / 通知管理 / 值班管理 / 设置）

**目标**（参照 FlashCat 侧边栏）：
```
协作空间          ← 新增，核心入口
故障管理          ← 新增，全局故障视图
  ├── 故障列表
  └── 告警管理
告警配置
  ├── 告警规则
  ├── 规则模板
  └── 数据源
值班排班
通知管理
数据查询          ← 已有
仪表盘            ← 已有
集成中心          ← 新增（Webhook 接入管理）
系统设置
```

### 3.2 协作空间页面（新增）

参照 FlashCat 的协作空间卡片列表 + 详情页：

**空间列表页**：
- 卡片式展示，每张卡片显示：空间名称、所属团队、活跃故障数、MTTA/MTTR
- 支持收藏（星标）、按团队筛选、排序
- 创建空间入口

**空间详情页**（Tab 结构）：
- **故障列表**：该空间下的故障，支持处理进度筛选、聚合视图
- **统计概览**：MTTA/MTTR/故障数/告警分组，4 张统计卡片
- **配置**：侧边栏菜单
  - 集成数据：专属 Webhook 端点 + 排除规则
  - 降噪处理：告警聚合 + 抖动检测 + 静默策略 + 抑制策略
  - 通知分派：分派策略（绑定到空间）
  - 设置：基础信息 + 高级配置（超时自动关闭等）

### 3.3 故障管理页面（新增）

参照 FlashCat 的故障列表 + 详情页：

**故障列表页**：
- 筛选栏：分派给我 / 全部 | 处理进度 | 时间范围 | 严重程度 | 协作空间
- 列表每行：故障标题、严重程度标签、处理进度（待处理/已认领/已关闭）、关联告警数、持续时间、处理人
- 支持批量操作：批量认领、批量关闭
- 聚合视图：按严重程度/空间/标签 Group By

**故障详情页**：
- **顶部**：标题 + 严重程度 + 处理进度 + 操作按钮（认领/关闭/暂缓/升级）
- **左侧内容区**（Tab 切换）：
  - 故障概览：描述、标签、AI 总结按钮
  - 关联告警：聚合的告警列表，可展开看事件
  - 时间线：完整生命周期记录 + Markdown 评论框
  - 故障复盘：内嵌复盘编辑器
- **右侧信息栏**：
  - 属性面板：协作空间、触发时间、告警数量
  - 关键时间节点：触发 → 认领 → 关闭
  - 处理人员列表 + 认领状态
  - 关联链接

### 3.4 告警管理页面（改造现有）

**现状**：AlertEvents 页面（活跃告警 + 历史告警）
**目标**：拆分为独立的告警管理页面

- 告警列表：关联到故障的告警，点击可查看事件时间线
- 筛选：严重程度、数据源、协作空间、处理状态
- 列表每行：告警标题、严重程度、归属故障、首次/最后触发时间、事件数

### 3.5 告警规则页面（改造现有）

- 新增：树形文件夹 + 拖拽排序
- 新增：规则关联协作空间（产生的告警路由到哪个空间）
- 保留现有 CRUD + 状态切换

### 3.6 集成中心页面（新增）

- 列表展示已创建的 Webhook 集成端点
- 每个集成显示：名称、类型（AlertManager/Grafana/通用）、归属空间、接收告警数
- 创建集成：选择类型 → 自动生成 Webhook URL → 配置告警处理管道

---

## 四、实施计划（精简版）

### Phase 1：核心模型重构（2-3 周）

建立 Channel → Incident → Alert → Event 四层模型。

| # | 任务 | 说明 | 优先级 |
|---|-----|------|--------|
| 1.1 | Channel 模型 + CRUD | 协作空间：name, description, team_id, config (JSON) | P0 |
| 1.2 | Incident 模型 + CRUD | 故障：title, severity, status(triggered/acknowledged/resolved/closed), channel_id | P0 |
| 1.3 | 重构 AlertEvent → Alert + Event | Alert 关联 Incident；Event 为原始上报事件 | P0 |
| 1.4 | RoutingRule 模型 | 告警路由：match_labels → channel_id | P0 |
| 1.5 | 告警引擎适配 | rule_eval 产出 Event → 创建/合入 Alert → 触发 Incident | P0 |
| 1.6 | 数据迁移 | 创建"默认空间"，现有 AlertEvent 迁移为 Alert | P0 |
| 1.7 | 前端：侧边栏导航重构 | 新菜单结构 | P0 |
| 1.8 | 前端：协作空间列表 + 详情页 | 卡片列表、配置 Tab | P0 |
| 1.9 | 前端：故障列表 + 详情页 | 核心交互页面 | P0 |

### Phase 2：智能降噪（1-2 周）

| # | 任务 | 说明 | 优先级 |
|---|-----|------|--------|
| 2.1 | 规则聚合引擎 | 按标签维度将 Alert 聚合到同一 Incident | P0 |
| 2.2 | 聚合窗口 | 可配置时间窗口，窗口外创建新故障 | P0 |
| 2.3 | 风暴预警 | 合入告警数达阈值时触发预警通知 | P1 |
| 2.4 | 抖动检测 | 频繁触发/恢复标记为抖动，可配置静默 | P1 |
| 2.5 | 静默增强 | 周期静默（按星期模式）、快速静默 | P1 |
| 2.6 | 前端：空间降噪配置页 | 聚合 + 抖动 + 静默配置 UI | P0 |

### Phase 3：分派增强（1 周）

| # | 任务 | 说明 | 优先级 |
|---|-----|------|--------|
| 3.1 | 分派策略绑定 Channel | 每个空间独立配置分派策略 | P0 |
| 3.2 | Incident 自动分派 | 故障创建时按策略通知值班人 | P0 |
| 3.3 | 标签增强 | 告警接入时自动补全标签（正则提取、映射） | P1 |
| 3.4 | 前端：空间分派配置页 | 分派策略关联空间 UI | P0 |

### Phase 4：告警引擎增强 + Webhook 接入（2 周）

| # | 任务 | 说明 | 优先级 |
|---|-----|------|--------|
| 4.1 | 无数据告警 | 指标停止上报时触发告警 | P0 |
| 4.2 | 规则文件夹 | 树形分类管理告警规则 | P1 |
| 4.3 | 标准 Webhook 接入 API | POST /api/v1/integrations/:id/alerts | P0 |
| 4.4 | AlertManager 兼容 | 解析 Prometheus AlertManager webhook 格式 | P0 |
| 4.5 | Grafana 兼容 | 解析 Grafana webhook 格式 | P1 |
| 4.6 | 告警处理管道 (Pipeline) | 接入时做标签提取、格式化、过滤 | P1 |
| 4.7 | 前端：集成中心页面 | Webhook 端点管理 + Pipeline 配置 | P1 |
| 4.8 | 前端：规则文件夹树 | 告警规则树形管理 UI | P1 |

### Phase 5：故障复盘 + 分析增强（1-2 周）

| # | 任务 | 说明 | 优先级 |
|---|-----|------|--------|
| 5.1 | PostMortem 模型 | 关联 Incident，Markdown 内容 | P1 |
| 5.2 | AI 辅助复盘 | 基于故障数据 + 关联告警自动生成复盘初稿 | P2 |
| 5.3 | AI 故障总结 | 故障详情页一键生成 AI 摘要（概述/影响/建议） | P1 |
| 5.4 | 分析看板增强 | 按空间/团队维度统计 MTTA/MTTR/故障趋势 | P1 |
| 5.5 | 前端：故障详情复盘 Tab | 复盘编辑器 + AI 生成按钮 | P1 |
| 5.6 | 前端：增强仪表盘 | 空间级 + 团队级统计面板 | P1 |

---

## 五、数据模型演进

### 现有模型
```
DataSource ─1:N─ AlertRule ─1:N─ AlertEvent ─1:N─ AlertTimeline
Team ─1:N─ TeamMember ─N:1─ User
EscalationPolicy ─1:N─ EscalationStep
NotifyRule / MuteRule / InhibitionRule / SubscribeRule → NotifyMedia
```

### 目标模型
```
Channel (协作空间)
  ├── channel_id, name, description, team_id
  ├── noise_config JSON (聚合规则、抖动配置)
  ├── auto_close_config JSON
  ├── RoutingRule ─ match_labels → channel_id
  ├── EscalationPolicy (从全局改为空间级)
  ├── MuteRule (从全局改为空间级)
  ├── InhibitionRule (从全局改为空间级)
  └── Integration (Webhook 端点)
       └── Pipeline (处理管道)

Incident (故障)
  ├── incident_id, title, description, severity
  ├── status: triggered / acknowledged / resolved / closed
  ├── channel_id FK
  ├── assigned_to (处理人)
  ├── acknowledged_at, resolved_at, closed_at
  ├── alert_count, event_count
  ├── IncidentTimeline ─ 生命周期记录
  └── PostMortem ─ 复盘报告

Alert (告警, 原 AlertEvent 重构)
  ├── alert_id, title, severity, fingerprint
  ├── status: firing / resolved
  ├── incident_id FK
  ├── channel_id FK
  ├── rule_id FK (内部规则) 或 integration_id FK (外部接入)
  └── Event (原始事件) ─ 每次触发/恢复为一条 Event

DataSource ─1:N─ AlertRule ─(产生)─ Event → Alert → Incident
                  └── RuleFolder (规则文件夹)
```

---

## 六、里程碑

| 里程碑 | 内容 | 预计版本 |
|--------|------|---------|
| M1 | Phase 1 — 协作空间 + 故障模型 + 导航重构 | v2.0.0 |
| M2 | Phase 2 — 智能降噪 | v2.1.0 |
| M3 | Phase 3 — 分派增强 | v2.2.0 |
| M4 | Phase 4 — 无数据告警 + Webhook 接入 + 集成中心 | v2.3.0 |
| M5 | Phase 5 — 故障复盘 + 分析增强 | v2.4.0 |

预计总工期：8-10 周

---

## 七、技术注意事项

1. **向后兼容**：Phase 1 的模型重构必须提供平滑迁移路径，现有 AlertEvent 数据不能丢失，创建"默认协作空间"归入所有现有数据
2. **性能**：告警聚合引擎用 Redis 缓存活跃 Incident 的聚合维度 key，匹配复杂度 O(1)
3. **前端**：重度交互页面（故障详情、协作空间配置）使用 Naive UI 的 NTabs + NDataTable + VirtualScroll
4. **API**：新 API 遵循 `/api/v1/` 前缀，Webhook 接入用 `/api/v1/integrations/:id/alerts`
5. **迁移文件**：每个 Phase 的 DB 变更严格遵循 up/down 迁移规范
6. **通知渠道**：仅飞书（Lark）+ 邮件（SMTP），不扩展其他渠道
7. **数据源**：仅 Prometheus / VictoriaMetrics / VictoriaLogs，不扩展其他数据源
