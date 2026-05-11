# SREAgent v4.0 应用架构重构设计

> **定位：** SREAgent = DevOps + AIOps + AgentOps 一站式可观测性与故障响应平台
> 由多个子模块组成的完整解决方案，每个子模块专精对应领域，SaaS 化思维，互相解耦又互相关联。

---

## 1. 产品定位

### 1.1 核心理念

SREAgent 不是单一的值班平台或告警工具，而是一个**平台级产品**：

- **大而全**：覆盖监控告警 → 故障响应 → 通知分发 → 值班排班 → 复盘改进的完整链路
- **专而精**：每个子模块专精对应领域，可以独立演进
- **可扩展**：未来可聚合吸收新的子模块（如日志分析、APM、混沌工程等）
- **SaaS 化**：模块间通过标准接口通信，支持独立部署和水平扩展

### 1.2 与开源方案的定位差异

| 能力 | Alertmanager | VMAlert | PagerDuty | SREAgent |
|------|-------------|---------|-----------|----------|
| 告警评估 | ✅ 路由 | ✅ 评估 | ❌ | ✅ 自研 + 接入外部 |
| 事件管理 | ❌ | ❌ | ✅ | ✅ |
| 故障协作 | ❌ | ❌ | ✅ | ✅ |
| 值班排班 | ❌ | ❌ | ✅ | ✅ |
| 通知管道 | ✅ 路由 | ❌ | ✅ | ✅ |
| 复盘改进 | ❌ | ❌ | ✅ | ✅ AI 驱动 |
| 数据查询 | ❌ | ❌ | ❌ | ✅ PromQL/LogsQL |
| 状态页面 | ❌ | ❌ | ✅ | ✅（规划中） |

**SREAgent 的差异化**：自研告警引擎 + AI 故障分析 + 完整故障生命周期管理，同时兼容开源生态。

---

## 2. 应用架构

### 2.1 三应用 + 平台管理

```
┌─────────────────────────────────────────────────────────────┐
│                    SREAgent Platform                        │
├──────────────┬──────────────┬──────────────┬───────────────┤
│   On-Call    │    Alert     │   Platform   │               │
│  故障响应     │  告警引擎     │   平台管理    │               │
│              │  通知管道     │              │               │
├──────────────┼──────────────┼──────────────┤               │
│ 协作空间      │ 概览          │ 个人中心     │               │
│ 故障列表      │ 告警规则       │ 组织管理     │               │
│ 状态页面      │ 告警事件       │ 审计日志     │               │
│ 故障复盘      │ 静默/抑制      │ 系统设置     │               │
│ 集成中心      │ 数据源         │              │               │
│ 值班管理      │ 数据查询       │              │               │
│ 配置中心      │ 仪表盘        │              │               │
│              │ 通知策略       │              │               │
│              │ 消息模板       │              │               │
│              │ 通知渠道       │              │               │
│              │ 订阅管理       │              │               │
└──────────────┴──────────────┴──────────────┴───────────────┘
```

### 2.2 导航交互设计

**三栏布局：**

```
┌──────────────────────────────────────────────────────────────┐
│  [Logo] SREAgent                              ⌘K  🕐 🌐 👤 │
├─────┬──────────────┬─────────────────────────────────────────┤
│     │              │                                         │
│  ◉  │  协作空间     │                                         │
│On-  │  故障列表     │           页面内容区                      │
│Call │  状态页面     │                                         │
│     │  故障复盘     │                                         │
│  ○  │  ────────── │                                         │
│    │  集成中心     │                                         │
│Al-  │  值班管理     │                                         │
│ert  │   ├ 排班计划  │                                         │
│     │   ├ 升级策略  │                                         │
│  ○  │   └ 响应剧本  │                                         │
│Pl-  │  ────────── │                                         │
│at-  │  配置中心  ▸  │                                         │
│form │   ├ 通知规则  │                                         │
│     │   ├ 路由规则  │                                         │
│     │   ├ 业务分组  │                                         │
│     │   ├ 订阅规则  │                                         │
│     │   └ 影响矩阵  │                                         │
│     │  ────────── │                                         │
│  ⚙  │  ▼ 张三      │                                         │
│     │  v3.1.0     │                                         │
└─────┴──────────────┴─────────────────────────────────────────┘
```

- **图标栏**（48px）：3 个应用图标 + 底部齿轮（平台管理）
- **菜单栏**（220px）：当前应用的子菜单，支持多级折叠
- **内容区**（自适应）：页面内容

---

## 3. On-Call 应用

### 3.1 定位

一站式故障响应平台。从告警落地到复盘完成的完整闭环。

### 3.2 子模块

| 子模块 | 页面 | 现有路由 | 状态 |
|--------|------|---------|------|
| **概览** | 故障响应仪表盘 | `/incident-dashboard` | 迁移（见 3.6） |
| **协作空间** | 列表 + 详情 | `/channels`, `/channels/:id` | 迁移 |
| **故障列表** | 列表 + 详情 | `/incidents`, `/incidents/:id` | 迁移 |
| **状态页面** | 状态页管理 | — | **新建** |
| **故障复盘** | 复盘列表 | 从 incident detail 拆出 | 重构 |
| **集成中心** | 集成列表 + 配置 | `/integrations` | 迁移 |
| **值班管理** | 排班 + 升级 + 剧本 | `/schedule` | 迁移+扩展 |
| **配置中心** | 通知/路由/分组/订阅/影响矩阵 | 多处 | 整合 |

### 3.3 状态页面（新建模块）

参考 Atlassian Statuspage / Cachet：

- **公开状态页**：面向外部用户，展示服务健康状态
- **内部状态页**：面向内部团队，展示故障公告和进展
- **组件管理**：定义服务组件（API、Web、数据库等），每个组件有四种状态：
  - `operational` — 正常运行
  - `degraded_performance` — 性能下降
  - `partial_outage` — 部分中断
  - `major_outage` — 严重中断
- **自动关联**：故障列表（Incident）自动更新对应组件状态
- **公告管理**：手动发布公告、维护通知

### 3.4 未来增强功能（迭代实现）

| 功能 | 优先级 | 说明 |
|------|--------|------|
| AI 故障副驾驶 | P1 | 故障详情页内嵌 AI 面板：根因分析、SOP 推荐、相似故障匹配 |
| 战时指挥室 | P1 | 高严重级故障自动创建指挥室，汇聚时间线、参与者、沟通记录 |
| 响应剧本 | P2 | 按故障类型/服务预定义响应流程（Runbook） |
| 故障影响矩阵 | P2 | 定义各服务的严重等级 + 预期响应时间 + 影响范围 |
| 干系人通知 | P2 | P0/P1 故障自动通知管理层/客户 |
| SLA 看板 | P3 | 故障响应时间、恢复时间 vs SLA 目标的实时追踪 |
| 故障时间线可视化 | P3 | 时间线改为可交互的甘特图/时间轴 |

### 3.5 菜单结构

```
On-Call
├── 概览
├── ──────────
├── 协作空间
├── 故障列表
├── 状态页面
├── 故障复盘
├── ──────────
├── 集成中心
├── 值班管理
│   ├── 排班计划
│   ├── 升级策略
│   └── 响应剧本（未来）
├── ──────────
├── 配置中心
│   ├── 通道通知规则        ← 通道（Channel）级别的通知路由
│   ├── 路由规则
│   ├── 业务分组
│   ├── 订阅规则
│   └── 影响矩阵（未来）
└──
```

### 3.6 概览页面内容

On-Call 概览（原 `/incident-dashboard`）展示故障响应核心指标：

| 区域 | 内容 |
|------|------|
| KPI 卡片 | 活跃故障数、MTTA（平均响应）、MTTR（平均恢复）、今日新增 |
| 故障趋势 | 近 7/30 天故障趋势折线图 |
| 严重等级分布 | 按 P0-P4 / critical-warning-info 分布的饼图 |
| 值班人 | 当前值班人 + 下一班值班人 |
| 活跃故障列表 | 最近 5 条活跃故障（快速入口） |
| 团队排行 | 按故障响应时间的团队排行 |

---

## 4. Alert 应用

### 4.1 定位

告警引擎 + 通知管道的完整链路：数据 → 规则 → 事件 → 通知。

### 4.2 告警引擎混合模式

**核心策略：** 保留自研引擎作为默认，同时支持接入外部 Alertmanager/VMAlert。

| 模式 | 规则评估 | 事件管理 | 通知 |
|------|---------|---------|------|
| **自研模式**（默认） | SREAgent Engine | SREAgent | SREAgent |
| **外部模式** | Alertmanager/VMAlert | SREAgent | SREAgent |
| **混合模式** | 两者并存 | SREAgent 统一管理 | SREAgent |

**外部接入方式：**
- 已有 `/webhooks/alertmanager` 接口
- 外部告警通过 Webhook 推入，转换为 SREAgent 的 Alert 事件
- 规则列表中标注来源（自研 / 外部），外部规则为只读展示

**自研引擎优势（保留理由）：**
- 深度集成故障管理（自动关联 Incident）
- AI 辅助分析（根因、SOP）
- 灵活的升级链和值班排班
- 标签路由和业务分组

**外部引擎优势（接入理由）：**
- 生态成熟，社区支持
- 集群化高可用
- 与 Prometheus/VM 原生集成

### 4.3 子模块

| 子模块 | 页面 | 现有路由 | 状态 |
|--------|------|---------|------|
| **概览** | 告警仪表盘 | `/dashboard` | 迁移（见 4.5） |
| **告警规则** | 自研规则 + 外部规则 | `/alerts/rules` | 重构 |
| **告警事件** | 活跃 + 历史 | `/alerts/events`, `/alerts/history` | 迁移 |
| **静默与抑制** | 静默规则 + 抑制规则 | `/alerts/mute-rules`, `/alerts/inhibition-rules` | 合并 |
| **数据源** | 数据源管理 | `/datasources` | 迁移 |
| **数据查询** | Explore | `/query` | 迁移 |
| **自定义仪表盘** | Panel 仪表盘 | `/dashboards-v2` | 迁移 |
| **通知策略** | 通知管道 | `/notification` 部分 | 重构 |
| **消息模板** | 模板管理 | `/notification` 部分 | 迁移 |
| **通知渠道** | 渠道管理 | `/notification` 部分 | 迁移 |
| **订阅管理** | 用户订阅 | `/notification` 部分 | 迁移 |

### 4.4 菜单结构

```
Alert
├── 概览
├── ──────────
├── 告警规则
│   ├── 自研规则
│   └── 外部规则
├── 告警事件
│   ├── 活跃告警
│   └── 历史告警
├── 静默与抑制
│   ├── 静默规则
│   └── 抑制规则
├── ──────────
├── 数据源
├── 数据查询
├── 自定义仪表盘
├── ──────────
├── 通知策略              ← 全局通知管道策略（谁收到什么通知）
├── 消息模板
├── 通知渠道              ← 飞书/邮件/Webhook 渠道配置
└── 订阅管理              ← 用户自定义订阅规则
```

**通知规则 vs 通知策略 区分：**
- **On-Call → 配置中心 → 通道通知规则**：协作空间（Channel）级别的通知路由，决定告警进入哪个协作空间
- **Alert → 通知策略**：全局通知管道，决定通知发给谁、通过什么渠道、用什么模板

### 4.5 概览页面内容

Alert 概览（原 `/dashboard`）展示告警引擎核心指标：

| 区域 | 冄容 |
|------|------|
| KPI 卡片 | 活跃告警数、告警规则数、数据源数、今日触发数 |
| 告警趋势 | 近 7/30 天告警触发趋势折线图 |
| Top 触发规则 | 触发次数最多的 Top 10 规则 |
| 严重等级分布 | 按 severity 分布的饼图 |
| 通知统计 | 通知发送量、成功率、渠道分布 |
| 数据源健康 | 各数据源连接状态 |

---

## 5. Platform 应用

### 5.1 定位

平台管理与治理中心。左下角齿轮图标入口。

### 5.2 子模块

| 子模块 | 页面 | 现有路由 | 状态 |
|--------|------|---------|------|
| **个人中心** | 基本信息 + 消息通知 | `/me/*` (弹窗) | 重构为页面 |
| **组织管理** | 成员管理 | `/users` | 迁移 |
| | 团队管理 | `/teams` | 迁移 |
| | 角色权限 | — | **新建** |
| | 单点登录 | `/settings` OIDC 部分 | 拆出 |
| **审计日志** | 审计日志 | `/audit-logs` | 迁移 |
| **系统设置** | 邮件服务 | `/settings` SMTP 部分 | 拆出 |
| | 飞书机器人 | `/settings` Lark 部分 | 拆出 |
| | AI 配置 | `/settings` AI 部分 | 拆出 |
| | 安全设置 | `/settings` Security 部分 | 拆出 |

### 5.3 菜单结构

```
Platform
├── 个人中心
│   ├── 基本信息
│   └── 消息通知
├── ──────────
├── 组织管理
│   ├── 成员管理
│   ├── 团队管理
│   ├── 角色权限
│   └── 单点登录
├── ──────────
├── 审计日志
├── ──────────
└── 系统设置
    ├── 邮件服务
    ├── 飞书机器人
    ├── AI 配置
    └── 安全设置
```

### 5.4 关键改动

| 改动 | 说明 |
|------|------|
| 个人中心：弹窗 → 页面 | 支持更多内容，更好的 UX |
| 设置页面拆分 | 当前大页面拆为独立子页面 |
| 新增角色权限页面 | 展示 RBAC 权限矩阵 |

---

## 6. 技术架构

### 6.1 前端架构

```
web/src/
├── apps/                      ← 新增：按应用组织
│   ├── oncall/                ← On-Call 应用
│   │   ├── spaces/            ← 协作空间
│   │   ├── incidents/         ← 故障列表
│   │   ├── status-page/       ← 状态页面（新建）
│   │   ├── postmortems/       ← 故障复盘
│   │   ├── integrations/      ← 集成中心
│   │   ├── schedule/          ← 值班管理
│   │   └── config/            ← 配置中心
│   ├── alert/                 ← Alert 应用
│   │   ├── overview/          ← 概览
│   │   ├── rules/             ← 告警规则
│   │   ├── events/            ← 告警事件
│   │   ├── suppression/       ← 静默/抑制
│   │   ├── datasource/        ← 数据源
│   │   ├── explore/           ← 数据查询
│   │   ├── dashboards/        ← 仪表盘
│   │   └── notify/            ← 通知管道
│   └── platform/              ← Platform 应用
│       ├── profile/           ← 个人中心
│       ├── org/               ← 组织管理
│       ├── audit/             ← 审计日志
│       └── settings/          ← 系统设置
├── components/                ← 共享组件
├── composables/               ← 共享 composable
├── stores/                    ← Pinia stores
├── layouts/
│   ├── AppShell.vue           ← 新增：三栏布局壳
│   ├── AppRail.vue            ← 新增：左侧图标栏
│   └── AppSidebar.vue         ← 新增：菜单栏（替代 MainLayout sidebar）
└── router/
    └── index.ts               ← 路由重组
```

### 6.2 路由结构

```ts
const routes = [
  { path: '/login', component: Login },
  {
    path: '/',
    component: AppShell,
    children: [
      // On-Call
      { path: 'oncall', redirect: '/oncall/spaces' },
      { path: 'oncall/overview', ... },
      { path: 'oncall/spaces', ... },
      { path: 'oncall/spaces/:id', ... },
      { path: 'oncall/incidents', ... },
      { path: 'oncall/incidents/:id', ... },
      { path: 'oncall/status-page', ... },
      { path: 'oncall/postmortems', ... },
      { path: 'oncall/integrations', ... },
      { path: 'oncall/schedule', ... },
      { path: 'oncall/schedule/escalation', ... },
      { path: 'oncall/config/notify-rules', ... },
      { path: 'oncall/config/routing-rules', ... },
      { path: 'oncall/config/biz-groups', ... },
      { path: 'oncall/config/subscribe-rules', ... },

      // Alert
      { path: 'alert', redirect: '/alert/overview' },
      { path: 'alert/overview', ... },
      { path: 'alert/rules', ... },
      { path: 'alert/rules/external', ... },
      { path: 'alert/events', ... },
      { path: 'alert/events/:id', ... },
      { path: 'alert/history', ... },
      { path: 'alert/suppression', ... },
      { path: 'alert/suppression/inhibition', ... },
      { path: 'alert/datasources', ... },
      { path: 'alert/explore', ... },
      { path: 'alert/dashboards', ... },
      { path: 'alert/dashboards/:id', ... },
      { path: 'alert/notify/policies', ... },
      { path: 'alert/notify/templates', ... },
      { path: 'alert/notify/channels', ... },
      { path: 'alert/notify/subscriptions', ... },

      // Platform
      { path: 'platform', redirect: '/platform/profile' },
      { path: 'platform/profile', ... },
      { path: 'platform/profile/notifications', ... },
      { path: 'platform/org/members', ... },
      { path: 'platform/org/teams', ... },
      { path: 'platform/org/roles', ... },
      { path: 'platform/org/sso', ... },
      { path: 'platform/audit', ... },
      { path: 'platform/settings/smtp', ... },
      { path: 'platform/settings/lark', ... },
      { path: 'platform/settings/ai', ... },
      { path: 'platform/settings/security', ... },

      // 兼容旧路由
      { path: 'dashboard', redirect: '/oncall/overview' },
      { path: 'channels', redirect: '/oncall/spaces' },
      { path: 'incidents', redirect: '/oncall/incidents' },
      // ... 其他重定向
    ],
  },
]
```

### 6.3 后端 API 重构

后端 API 按应用分组，增加 `/api/v1/` 前缀的应用层：

```
/api/v1/oncall/          ← On-Call 应用 API
  /spaces/               ← 协作空间
  /incidents/            ← 故障
  /status-page/          ← 状态页面（新建）
  /postmortems/          ← 复盘
  /integrations/         ← 集成
  /schedules/            ← 值班
  /escalation-policies/  ← 升级策略
  /config/               ← 配置中心

/api/v1/alert/           ← Alert 应用 API
  /rules/                ← 告警规则
  /events/               ← 告警事件
  /suppression/          ← 静默/抑制
  /datasources/          ← 数据源
  /explore/              ← 数据查询
  /dashboards/           ← 仪表盘
  /notify/               ← 通知管道

/api/v1/platform/        ← Platform 应用 API
  /profile/              ← 个人中心
  /org/                  ← 组织管理
  /audit/                ← 审计
  /settings/             ← 系统设置
```

**注意：** 后端 API 重构可以渐进式进行，先做前端路由重组，后端通过路由别名兼容。

**后端 API 兼容策略：**
- Phase 1（前端重构）：后端 API 不动，前端通过 API 层封装适配新路径
- Phase 2（后端重构）：新增 `/api/v1/oncall/` 等路由组，旧路由通过 Gin middleware 301 重定向
- 数据库零迁移：仅路由重组，不涉及数据模型变更

### 6.4 模块间通信

模块间通过事件总线和共享 Store 通信：

```
On-Call (故障)  ←──── alert_key ────→  Alert (告警)
     │                                      │
     │ incident_id                          │ alert_id
     │                                      │
     ↓                                      ↓
  PostMortem                          Notify Pipeline
     │                                      │
     │ AI analysis                          │ template + channel
     ↓                                      ↓
  AI Copilot                          External (Lark/Email/Webhook)
```

关键关联：
- Alert 事件通过 `alert_key` 关联 On-Call 故障
- 故障复盘通过 `incident_id` 关联故障详情
- 通知管道通过 `notify_rule` 关联告警规则

---

## 7. 迁移策略

### 7.1 渐进式迁移

**Phase 1：前端导航重构**（优先级最高）
- 新建 `AppShell` 三栏布局
- 迁移现有页面到新路由结构
- 添加路由重定向保持兼容
- 不改后端 API

**Phase 2：模块拆分**
- 按应用重组前端文件
- 拆分设置页面为独立子页面
- 个人中心从弹窗改为页面

**Phase 3：新建模块**
- 状态页面
- 角色权限页面
- 告警规则拆分自研/外部

**Phase 4：功能增强**
- AI 故障副驾驶
- 战时指挥室
- 响应剧本
- 影响矩阵

### 7.2 兼容性

- 旧路由全部添加重定向
- 后端 API 路径渐进式迁移，旧路径保持兼容
- 数据库无需迁移（只是前端路由重组）

---

## 8. 设计决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 应用数量 | 3 + 平台管理 | 告警和通知是一条链路，不拆分 |
| 告警引擎 | 混合模式 | 保留自研 + 接入外部，兼顾灵活性和生态 |
| 通知归属 | 归入 Alert | 告警→通知是同一流程 |
| 值班归属 | 归入 On-Call | 值班是故障响应的一部分 |
| 导航方式 | 左侧图标栏 + 菜单栏 | Raycast/Linear 风格，应用切换直观 |
| 个人中心 | 弹窗→页面 | 更好的 UX，支持更多内容 |
| 状态页面 | 新建 | 完善故障响应闭环 |
| 迁移策略 | 渐进式 | 降低风险，保持可用性 |
