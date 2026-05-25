# Nightingale V9 vs SREAgent 结构对比分析

> v1.0 | 2026-05-25 | 用于指导后续模块级优化

## 一、整体架构对比

### 1.1 后端架构

| 维度 | Nightingale V9 | SREAgent |
|------|---------------|----------|
| **语言/框架** | Go + Gin + GORM | Go + Gin + 原生 SQL |
| **架构模式** | 微服务可拆分（center/alert/pushgw 三独立进程） | 单体（handler→service→repository→model） |
| **DI 方式** | 无框架，`center.Initialize()` 手动接线 | 同样无框架，`initDependencies()` 手动接线 |
| **依赖载体** | `ctx.Context`（GORM DB + CenterApi 配置） | 各 handler 持有 service 引用 |
| **配置加载** | YAML 文件目录扫描 (`cfg.LoadConfigByDir`) | 单 YAML 文件 + 环境变量覆盖 |
| **缓存层** | `memsto/` 包，16 种内存缓存 + Redis pub/sub | Redis 为主缓存，无独立内存缓存层 |
| **数据源管理** | `dscache/` 二级缓存 + 异步 InitClient | 硬编码 client 创建，无缓存层 |
| **中间件** | stat / language / Recovery / auth / perm / bgro | JWT / CORS / Logger / RBAC |
| **路由组织** | 按领域拆文件（center/router 下 20+ 文件） | 按职责拆文件（9 个 route 文件） |

### 1.2 前端架构

| 维度 | Nightingale V9 | SREAgent |
|------|---------------|----------|
| **框架** | React 17 + TypeScript | Vue 3 + TypeScript |
| **UI 库** | Ant Design 4 + Tailwind CSS | Naive UI + 自定义 CSS |
| **构建工具** | Vite | Vite |
| **状态管理** | React Context（单一 CommonStateContext） | Pinia stores |
| **路由** | React Router DOM v5 | Vue Router 4 |
| **API 层** | `services/` 目录，umi-request | `api/` 目录，axios |
| **i18n** | 每个组件自带 `locale/` 目录（6 语言） | 集中式 `i18n/` 目录 |
| **页面数** | 30+ 页面模块 | 57 个页面文件 |
| **插件系统** | `plugins/` 目录，数据源插件动态加载 | 无插件系统 |

---

## 二、核心模块对比

### 2.1 数据源模块

#### Nightingale V9 设计意图

Nightingale 的数据源抽象是一个**插件化架构**：

```
datasource/
├── datasource.go      # 核心接口定义 + 工厂注册表
├── prom/              # Prometheus 实现
├── es/                # Elasticsearch 实现
├── ck/                # ClickHouse 实现
├── mysql/             # MySQL 实现
├── postgresql/        # PostgreSQL 实现
├── tdengine/          # TDengine 实现
├── doris/             # Apache Doris 实现
├── victorialogs/      # VictoriaLogs 实现
└── opensearch/        # OpenSearch 实现
```

**关键设计决策**：
1. **接口即契约**：`Datasource` 接口有 9 个方法，覆盖初始化、验证、查询（数据/日志/映射）、缓存去重
2. **工厂+注册表**：`RegisterDatasource(typ, creator)` + `init()` 自注册，新增数据源只需加一个子包
3. **异步初始化**：`InitClient()` 不阻塞启动，首次查询时才真正建立连接
4. **缓存去重**：`Equal()` 方法让 `dscache` 能判断配置是否变更，避免重建 client
5. **能力断言**：可选能力通过接口断言检查（`if ds, ok := ds.(LogQuerier); ok {}`）

#### SREAgent 现状

```
internal/pkg/datasource/
├── client.go          # DatasourceClient 接口 + 创建逻辑
├── prom.go            # Prometheus 查询
├── vm.go              # VictoriaMetrics 查询
├── zabbix.go          # Zabbix 查询
└── vlogs.go           # VictoriaLogs 查询
```

**差距**：
- 接口方法少（只有 QueryData/QueryLog），缺少 Init/Validate/Equal
- 无工厂注册表，新增类型需改 `client.go` 的 switch
- 无缓存层，每次请求可能重复创建 client
- 无异步初始化

#### 代码层面可复用的 Nightingale 片段

| 片段 | 来源 | 适配工作 |
|------|------|----------|
| `Datasource` 接口定义 | `datasource/datasource.go:83` | 直接复用，扩展 SREAgent 现有接口 |
| 工厂注册表 `RegisterDatasource` | `datasource/datasource.go` | 直接复用，纯 Go 代码无依赖 |
| ES 查询实现 | `datasource/es/es.go` | 需替换 `*gorm.DB` 为 SREAgent 的 DB 层 |
| ClickHouse 查询实现 | `datasource/ck/clickhouse.go` | 同上 |
| 缓存层 | `dscache/cache.go` | 需适配 SREAgent 的数据源 model |
| `Equal()` 比较逻辑 | 各实现的 `Equal` 方法 | 直接复用 |

---

### 2.2 通知模块

#### Nightingale V9 设计意图

Nightingale 的通知是**三层解耦**：

```
models/
├── notify_channel.go     # 渠道配置（Provider 接口）
├── notify_rule.go        # 路由规则（匹配条件 → 渠道）
├── message_tpl.go        # 消息模板（Go template 渲染）
├── event_pipeline.go     # 事件管道（DAG 编排）

alert/
├── dispatch/             # 事件分发引擎
├── sender/               # 消息发送器
│   └── provider/         # 各渠道 Provider 实现
│       ├── provider.go   # NotifyChannelProvider 接口
│       ├── email.go
│       ├── dingtalk.go
│       ├── feishu.go
│       ├── slack.go
│       ├── webhook.go
│       └── ...
```

**关键设计决策**：
1. **Provider 接口+注册表**：`NotifyChannelProvider` 接口定义发送契约，各渠道通过 `init()` 自注册
2. **DAG 管道引擎**：事件经过 relabel → enrichment → AI summary → throttle → dispatch 的有向无环图
3. **模板引擎**：Go `text/template` 渲染，支持变量、条件、循环
4. **20+ 内置渠道**：邮件、钉钉、飞书、企微、Slack、Telegram、Discord、PagerDuty、Webhook 等
5. **渠道与规则分离**：渠道只管"怎么发"，规则管"发给谁、什么时候发"

#### SREAgent 现状

```
internal/model/
├── notify_media.go       # 4 种媒体类型
├── notify_rule.go        # 路由规则
├── message_template.go   # 消息模板
├── notification.go       # 通知记录

internal/service/
├── notify_service.go     # 通知发送逻辑
├── lark_service.go       # 飞书具体实现
├── smtp_service.go       # 邮件具体实现
```

**差距**：
- 渠道硬编码在 service 层，无 Provider 接口
- 无 DAG 管道，事件→通知是线性流程
- 4 种渠道 vs 20+ 种
- 模板渲染能力有限

#### 代码层面可复用的 Nightingale 片段

| 片段 | 来源 | 适配工作 |
|------|------|----------|
| `NotifyChannelProvider` 接口 | `alert/sender/provider/provider.go` | 直接复用 |
| 钉钉 Provider | `pkg/dingtalk/` | 直接复用，无 GORM 依赖 |
| Slack Provider | `alert/sender/provider/slack.go` | 需适配 HTTP client |
| Webhook Provider | `alert/sender/provider/webhook.go` | 直接复用 |
| 模板渲染引擎 | `pkg/tplx/` | 直接复用，纯 Go template |
| DAG 管道引擎 | `alert/pipeline/` | 需适配 SREAgent 的事件 model |

---

### 2.3 AI Agent 模块

#### Nightingale V9 设计意图

```
aiagent/
├── agent.go             # Agent 核心（Run 入口）
├── types.go             # 类型定义（ReAct/Plan+ReAct/Direct）
├── react.go             # ReAct 循环实现
├── plan_react.go        # Plan+ReAct 实现
├── direct.go            # Direct 模式实现
├── stream.go            # 流式输出
├── llm/                 # 统一 LLM 接口（7 种 provider）
├── tools/               # 工具定义 + 内置工具
│   └── defs/            # 内置工具定义（alert/dashboard/datasource/query 等）
├── mcp/                 # MCP 协议客户端
├── skill/               # 技能系统（自动选择 + 注册）
├── a2a/                 # Agent-to-Agent 协议
├── chat/                # 对话管理（意图检测、动作路由）
└── prompts/             # 提示词模板
```

**关键设计决策**：
1. **三种运行模式**：ReAct（默认）、Plan+ReAct（复杂任务分步规划）、Direct（纯生成无工具）
2. **统一 LLM 接口**：`LLM` 接口抽象 7 种 provider（OpenAI/Claude/Gemini/Ollama/Bedrock/Vertex/Kimi）
3. **工具系统**：内置工具（alert 查询、dashboard 操作、PromQL 查询等）+ MCP 工具 + Skill 工具
4. **技能自动选择**：LLM 根据任务上下文自动选择相关技能
5. **流式输出**：SSE 流式推送 thinking/tool_call/text/done 等事件

#### SREAgent 现状

SREAgent 的 AI 实现已经相当完整：
- `AIService`（告警分析：摘要、根因、影响、建议）
- `AgentService`（多步规划执行、工具注册、对话持久化）
- 支持多种 LLM（OpenAI/Claude/自定义）
- 知识库、诊断工作流、巡检调度

**差距很小**，主要差异：
- Nightingale 有 MCP+A2A 协议支持
- Nightingale 有 7 种 LLM provider（SREAgent 可能少几种）
- Nightingale 的内置工具更丰富（dashboard 操作、host 管理等）

#### 评估

AI 模块**不需要大规模迁移**，只需：
1. 补齐 MCP 协议支持（如需要）
2. 对齐 LLM provider 列表
3. 补充缺失的内置工具

---

### 2.4 告警引擎模块

#### Nightingale V9 设计意图

```
alert/
├── eval/          # 规则评估（PromQL 执行 + 条件判断）
├── dispatch/      # 事件分发（路由匹配 → 通知触发）
├── sender/        # 消息发送（多渠道并行）
├── mute/          # 静默规则匹配
├── naming/        # 引擎集群选主（Redis 分布式锁）
├── queue/         # 事件队列
├── record/        # 录制规则
├── process/       # 事件处理（去重、分组、relabel）
└── pipeline/      # 事件管道（DAG 编排）
```

**关键设计决策**：
1. **引擎可独立部署**：`alert/` 有自己的 `Initialize()`，支持 edge 模式（无本地 DB）
2. **内存缓存驱动**：`memsto/AlertRuleCache` 等缓存从 DB 或 center API 同步规则，评估器只读缓存
3. **集群选主**：`naming/` 包用 Redis 分布式锁实现多实例选主，避免重复评估
4. **录制规则**：支持将 PromQL 查询结果写回时序数据库（recording rules）

#### SREAgent 现状

```
internal/engine/
├── evaluator.go         # 规则评估
├── rule_eval.go         # 规则执行
├── suppression.go       # 抑制
├── heartbeat.go         # 心跳检测
├── escalation.go        # 升级策略
└── ...                  # 19 个文件
```

**SREAgent 已有完整的告警引擎**，核心功能对齐。差异：
- 无引擎集群选主（单实例）
- 无录制规则
- 无事件管道 DAG 编排
- 评估器直接读 DB（Nightingale 读内存缓存，性能更好）

#### 评估

告警引擎**不需要大规模迁移**，可选择性优化：
1. 引入内存缓存层提升评估性能（借鉴 `memsto/` 模式）
2. 补充录制规则（如需要）
3. 事件管道 DAG（Phase 2 通知渠道扩展时一并引入）

---

### 2.5 仪表盘模块

#### Nightingale V9 设计意图

```
pages/dashboard/
├── DashboardList.vue    # 仪表盘列表
├── DashboardView.vue    # 查看模式
├── DashboardEdit.vue    # 编辑模式
├── PanelEditor.vue      # 面板编辑器
├── chart/               # 图表组件
└── variableConfigs/     # 模板变量

models/
├── dashboard.go         # 仪表盘数据模型
├── board.go             # 面板数据模型
```

**关键设计决策**：
1. **Grafana 级别的自由度**：用户可自定义面板类型、布局、查询、样式
2. **模板变量**：支持 `$instance`, `$job` 等变量，修改变量自动刷新所有面板
3. **多数据源叠加**：同一图表可叠加多个数据源的查询结果
4. **导入导出**：JSON 格式的仪表盘模板
5. **权限控制**：按团队/用户控制查看/编辑权限

#### SREAgent 现状

```
web/src/pages/dashboard/
├── Index.vue              # 统计概览（固定布局）
├── IncidentDashboard.vue  # 事件仪表盘
└── UnifiedDashboard.vue   # 统一仪表盘
```

**差距最大**：SREAgent 只有固定视图，无自定义能力。

#### 评估

仪表盘是**工作量最大的迁移项**，建议：
1. 先实现数据查询 UI（Phase 3）
2. 再实现自定义仪表盘（Phase 4）
3. 前端用 Vue 3 重建，不搬 React 代码

---

## 三、架构模式差异总结

### 3.1 SREAgent 可以从 Nightingale 学到的

| 模式 | 说明 | 优先级 |
|------|------|--------|
| **工厂+注册表** | 数据源、通知渠道等通过接口+注册表实现插件化 | P0 |
| **内存缓存层** | `memsto/` 模式，规则/用户/数据源等热数据缓存在内存 | P1 |
| **异步初始化** | 数据源 client 不阻塞启动，首次查询时才连接 | P1 |
| **接口断言** | 可选能力通过接口断言检查，而非类型判断 | P2 |
| **Provider 模式** | 通知渠道通过 Provider 接口解耦 | P0 |
| **DAG 管道** | 事件处理通过 DAG 编排，支持插件化节点 | P2 |

### 3.2 SREAgent 已经做得好的

| 方面 | 说明 |
|------|------|
| **分层清晰** | handler→service→repository→model 严格单向 |
| **迁移管理** | golang-migrate 版本化迁移，有 up/down |
| **RBAC** | 完整的角色权限控制 |
| **AI Agent** | 完整的多模式 Agent + 工具系统 |
| **告警引擎** | 完整的规则评估 + 事件处理 + 升级策略 |
| **值班排班** | 完整的轮转 + 替班 + 升级 |

### 3.3 不需要迁移的

| 模块 | 原因 |
|------|------|
| AI Agent | SREAgent 已完整实现，差距很小 |
| 告警引擎 | 核心功能对齐，只需选择性优化 |
| 值班排班 | SREAgent 已完整实现 |
| 用户管理 | SREAgent 已完整实现 |
| 认证系统 | SREAgent 已有 JWT + OIDC + RBAC |

---

## 四、建议的优化顺序

基于对比分析，按**底层→上层**的依赖关系：

### 第一批：基础设施优化（1-2 周）

1. **数据源抽象层重构**
   - 引入 `Datasource` 接口（扩展到 9 个方法）
   - 实现工厂+注册表
   - 实现内存缓存层
   - 保持现有 4 种数据源向后兼容

2. **通知渠道 Provider 模式**
   - 引入 `NotifyChannelProvider` 接口
   - 将现有 4 种渠道重构为 Provider 实现
   - 为后续扩展打基础

### 第二批：能力扩展（2-3 周）

3. **新增数据源类型**
   - Elasticsearch（日志查询）
   - ClickHouse（日志/指标）
   - 从 Nightingale 复用查询实现，适配 DB 层

4. **新增通知渠道**
   - 钉钉、企微、Slack
   - 从 Nightingale 复用 Provider 实现

### 第三批：前端增强（3-4 周）

5. **数据查询 UI**
   - PromQL/LogsQL 查询编辑器
   - 查询结果展示（表格 + 图表）

6. **自定义仪表盘**
   - 面板编辑器
   - 拖拽布局
   - 模板变量

---

## 五、等待用户输入

用户将提供 Nightingale V9 的功能文档，用于：
1. 补充设计意图层面的理解
2. 确认 UI 交互层面的规格
3. 识别文档中提到但代码中未实现的功能
4. 确定哪些功能是用户真正需要的

---

*本文档基于 C:\project\nightingale（后端）和 C:\project\fe（前端）的源码分析。*
