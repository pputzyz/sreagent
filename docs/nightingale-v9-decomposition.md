# Nightingale V9 代码库拆解索引

> 本文件是 Nightingale V9（夜莺监控）的前后端代码级拆解文档，供 agent 会话快速定位模块位置。
> 前端源码：`C:\project\fe\src\` | 后端源码：`C:\project\nightingale\`
> SREAgent 源码：`c:\project\sreagent\`

---

## 目录

1. [前端目录结构](#1-前端目录结构)
2. [前端路由表](#2-前端路由表)
3. [前端页面清单](#3-前端页面清单)
4. [前端组件清单](#4-前端组件清单)
5. [前端 API/Services 层](#5-前端-apiservices-层)
6. [前端状态管理](#6-前端状态管理)
7. [前端样式系统](#7-前端样式系统)
8. [Explore 页面深度拆解](#8-explore-页面深度拆解)
9. [Dashboard 架构](#9-dashboard-架构)
10. [插件架构](#10-插件架构)
11. [后端目录结构](#11-后端目录结构)
12. [后端 API 端点](#12-后端-api-端点)
13. [后端数据模型](#13-后端数据模型)
14. [后端告警引擎](#14-后端告警引擎)
15. [后端数据源代理](#15-后端数据源代理)
16. [后端认证授权](#16-后端认证授权)
17. [SREAgent 对齐状态](#17-sreagent-对齐状态)

---

## 1. 前端目录结构

```
C:\project\fe\src\
  assets/          -- 静态资源（图片）
  components/      -- 共享/可复用 UI 组件
  locales/         -- 全局 i18n locale 文件
  pages/           -- 页面级组件（路由目标）
  plugins/         -- 数据源类型插件（clickHouse, elasticsearch, mysql 等）
  routers/         -- 路由定义
  services/        -- API 服务层（HTTP 调用）
  store/           -- 类型定义（接口/类型，非实际 store 实例）
  theme/           -- 主题：CSS 变量、less 文件、深色/浅色模式
  types/           -- 全局 TypeScript 声明
  utils/           -- 工具函数、请求客户端、常量
```

---

## 2. 前端路由表

路由定义文件：`C:\project\fe\src\routers\index.tsx`

| 路由 | 组件 | 源文件 |
|---|---|---|
| `/metric/explorer` | MetricExplore | `pages/explorer/Metric.tsx` |
| `/log/explorer` | LogExplore | `pages/explorer/Log.tsx` |
| `/log/index-patterns` | IndexPatterns | `pages/log/IndexPatterns/` |
| `/object/explorer` | ObjectExplore | `pages/monitor/object/` |
| `/dashboards` | Dashboard 列表 | `pages/dashboard/List/` |
| `/dashboard/:id` | Dashboard 详情 | `pages/dashboard/Detail/` |
| `/dashboards/share/:id` | Dashboard 分享 | `pages/dashboard/Share/` |
| `/alert-rules` | 告警规则列表 | `pages/alertRules/` |
| `/alert-rules/add/:bgid` | 创建告警规则 | `pages/alertRules/` |
| `/alert-rules/edit/:id` | 编辑告警规则 | `pages/alertRules/` |
| `/alert-mutes` | 静默规则 | `pages/warning/shield/` |
| `/alert-subscribes` | 告警订阅 | `pages/warning/subscribe/` |
| `/recording-rules/:id?` | 记录规则 | `pages/recordingRules/` |
| `/alert-his-events` | 历史告警事件 | `pages/historyEvents/` |
| `/alert-cur-events/:eventId` | 当前事件详情 | `pages/event/detail/` |
| `/alert-his-events/:eventId` | 历史事件详情 | `pages/event/detail/` |
| `/job-tpls` | 任务模板 | `pages/taskTpl/` |
| `/job-tasks` | 任务执行 | `pages/task/` |
| `/datasources` | 数据源管理 | `pages/datasource/` |
| `/trace/explorer` | Trace 探索 | `pages/traceCpt/Explorer/` |
| `/trace/dependencies` | Trace 依赖 | `pages/traceCpt/Dependencies/` |
| `/roles` | 权限管理 | `pages/permissions/` |
| `/system/site-settings` | 站点设置 | `pages/siteSettings/` |
| `/system/sso-settings` | SSO 配置 | `pages/help/SSOConfigs/` |
| `/system/variable-settings` | 变量配置 | `pages/variableConfigs/` |
| `/system/alerting-engines` | 告警引擎 | `pages/help/servers/` |
| `/help/notification-tpls` | 通知模板 | `pages/help/NotificationTpls/` |
| `/help/notification-settings` | 通知设置 | `pages/help/NotificationSettings/` |
| `/login` | 登录 | `pages/login/` |
| `/account/profile/:tab` | 个人资料 | `pages/account/profile/` |
| `/busi-groups` | 业务组 | `pages/user/business/` |
| `/users` | 用户管理 | `pages/user/users/` |
| `/user-groups` | 用户组 | `pages/user/groups/` |

---

## 3. 前端页面清单

### 3.1 探索（Explorer）

| 页面 | 路径 | 说明 |
|---|---|---|
| Metric Explorer | `pages/explorer/Metric.tsx` | 指标探索，多面板模式 |
| Log Explorer | `pages/explorer/Log.tsx` | 日志探索，tab 模式 |
| Explorer 调度器 | `pages/explorer/Explorer.tsx` | 根据数据源类型分发到对应子组件 |
| Prometheus Explorer | `pages/explorer/Prometheus/index.tsx` | PromQL 查询 + PromGraphCpt |
| Elasticsearch Explorer | `pages/explorer/Elasticsearch/index.tsx` | ES 日志查询（736行） |
| Loki Explorer | `pages/explorer/Loki/index.tsx` | Loki LogQL 查询 |

### 3.2 告警

| 页面 | 路径 | 说明 |
|---|---|---|
| AlertRules | `pages/alertRules/` | 规则列表 + 表单编辑器 |
| AlertRules/Form/ | `pages/alertRules/Form/` | 复杂表单：Triggers, Severity, Inhibit, Effective, Notify, Rule |
| AlertRules/List/ | `pages/alertRules/List/` | 列表视图：CloneToBgids, CloneToHosts, EventsDrawer, Import |
| AlertCurEvent | `pages/alertCurEvent/` | 当前活跃告警事件 |
| HistoryEvents | `pages/historyEvents/` | 历史告警事件 |
| Event Detail | `pages/event/detail/` | 事件详情（Detail/, DetailNG/, EventNotifyRecords/, TaskTpls/） |
| Shield (Mute) | `pages/warning/shield/` | 静默规则 |
| Subscribe | `pages/warning/subscribe/` | 告警订阅 |
| EventPipeline | `pages/eventPipeline/` | 事件管道（Form, Executions, List） |

### 3.3 通知

| 页面 | 路径 | 说明 |
|---|---|---|
| NotificationChannels | `pages/notificationChannels/` | 通知渠道管理 |
| NotificationRules | `pages/notificationRules/` | 通知规则 |
| NotificationTemplates | `pages/notificationTemplates/` | 通知模板 |

### 3.4 Dashboard

| 页面 | 路径 | 说明 |
|---|---|---|
| Dashboard List | `pages/dashboard/List/` | 仪表盘列表 |
| Dashboard Detail | `pages/dashboard/Detail/` | 仪表盘详情 |
| Dashboard Share | `pages/dashboard/Share/` | 公开分享仪表盘 |
| Chart | `pages/chart/` | 单图表查看器 |
| EmbeddedDashboards | `pages/embeddedDashboards/` | 嵌入式仪表盘 iframe |

### 3.5 配置管理

| 页面 | 路径 | 说明 |
|---|---|---|
| Datasource | `pages/datasource/` | 数据源管理 |
| SiteSettings | `pages/siteSettings/` | 站点设置 |
| VariableConfigs | `pages/variableConfigs/` | 全局变量配置 |
| Permissions | `pages/permissions/` | 角色/权限管理 |
| SSOConfigs | `pages/help/SSOConfigs/` | SSO 配置 |
| AI Config | `pages/aiConfig/` | AI 配置（agents, llmConfigs, mcpServers, skills） |

---

## 4. 前端组件清单

### 4.1 查询/输入组件

| 组件 | 路径 | 说明 |
|---|---|---|
| **PromQLInputNG** | `components/PromQLInputNG/` | Monaco 编辑器，支持 PromQL 自动补全、变量插值 |
| **PromQLInput** | `components/PromQLInput/` | 旧版 CodeMirror 编辑器 |
| **LogQLInput** | `components/LogQLInput/` | CodeMirror LogQL 编辑器 |
| **KQLInput** | `components/KQLInput/` | KQL 输入（Kibana 查询语言） |
| **PromQueryBuilder** | `components/PromQueryBuilder/` | 可视化 PromQL 构建器 |
| **PromGraphCpt** | `components/PromGraphCpt/` | 完整 Prometheus 图表组件，含 Table + Graph tabs |

**PromGraphCpt 子组件：**
- `Table.tsx` -- 即时查询（`/api/v1/query`），支持时间戳选择、单位选择、CSV 导出
- `Graph.tsx` -- 范围查询（`/api/v1/query_range`），支持线/面积切换、标准选项
- `components/QueryStatsView` -- 查询统计
- `components/Panel` -- 面板容器
- `components/GraphStandardOptions` -- 图表标准选项
- `components/MetricsExplorer` -- 指标浏览器

### 4.2 数据/可视化组件

| 组件 | 路径 | 说明 |
|---|---|---|
| UPlotChart | `components/UPlotChart/` | uPlot 图表库 |
| G2PieChart | `components/G2PieChart/` | G2 饼图 |
| CodeMirror | `components/CodeMirror/` | 通用 CodeMirror 包装器 |

### 4.3 表单/输入组件

| 组件 | 路径 | 说明 |
|---|---|---|
| TimeRangePicker | `components/TimeRangePicker/` | 时间范围选择器（含 AutoRefresh, TimeZonePicker） |
| DatasourceSelect | `components/DatasourceSelect/` | 数据源选择器（V2, V3 变体） |
| BusinessGroup | `components/BusinessGroup/` | 业务组树选择器 |
| KVTagSelect | `components/KVTagSelect/` | KV 标签选择器 |
| ViewSelect | `components/ViewSelect/` | 保存的视图选择器 |
| Resolution | `components/Resolution/` | 分辨率选择器 |
| CronPattern | `components/CronPattern/` | Cron 表达式编辑器 |

### 4.4 布局/导航组件

| 组件 | 路径 | 说明 |
|---|---|---|
| pageLayout | `components/pageLayout/` | 主页面布局（侧边栏、头部） |
| menu | `components/menu/` | 菜单系统 |
| Splitter | `components/Splitter/` | 可调整大小的分割面板 |
| BreadCrumb | `components/BreadCrumb/` | 面包屑导航 |

### 4.5 AI/Chat 组件

| 组件 | 路径 | 说明 |
|---|---|---|
| AiChat | `components/AiChat/` | AI 聊天面板（旧版） |
| AiChatNG | `components/AiChatNG/` | 新一代 AI 聊天（FloatingPanel, ContentRenderer） |

### 4.6 工具组件

| 组件 | 路径 | 说明 |
|---|---|---|
| HistoricalRecords | `components/HistoricalRecords/` | 查询历史 |
| FullscreenButton | `components/FullscreenButton/` | 全屏切换 |
| FieldsList | `components/FieldsList/` | 字段列表（日志用） |
| LogsViewer | `components/LogsViewer/` | 日志查看器 |
| RenderValue | `components/RenderValue/` | 字段值渲染 |
| Tags | `components/Tags/` | 标签管理 |
| VirtualTable | `components/VirtualTable/` | 虚拟化表格 |
| ExportImport | `components/ExportImport/` | 数据导出/导入 |

---

## 5. 前端 API/Services 层

请求客户端：`C:\project\fe\src\utils\request.tsx`（248行，基于 umi-request）

基础 URL 模式：`/api/n9e/...`（社区版）/ `/api/n9e-plus/...`（企业版）

### 5.1 核心服务文件

| 文件 | 说明 |
|---|---|
| `services/common.ts` | 数据源列表、业务组、权限 |
| `services/dashboardV2.ts` | Dashboard CRUD、批量查询、标签查询 |
| `services/warning.ts` | 告警规则、事件、指标查询 |
| `services/login.ts` | 认证、SSO、JWT |
| `services/manage.ts` | 用户、团队、业务组、角色管理 |
| `services/targets.ts` | 监控目标管理 |
| `services/metric.ts` | 指标查询 |
| `services/metricViews.ts` | 指标视图 CRUD |
| `services/shield.ts` | 静默规则 CRUD |
| `services/subscribe.ts` | 告警订阅 CRUD |
| `services/recording.ts` | 记录规则 CRUD |
| `services/account.ts` | 个人资料、Token 管理 |

### 5.2 关键 API 端点映射

**数据源代理查询：**
```
GET  /api/n9e/proxy/{dsId}/api/v1/labels          -- 标签名列表
GET  /api/n9e/proxy/{dsId}/api/v1/label/{label}/values  -- 标签值列表
GET  /api/n9e/proxy/{dsId}/api/v1/series           -- 序列查询
GET  /api/n9e/proxy/{dsId}/api/v1/query            -- 即时查询
GET  /api/n9e/proxy/{dsId}/api/v1/query_range      -- 范围查询
GET  /api/n9e/proxy/{dsId}/api/v1/label/__name__/values  -- 指标名列表
POST /api/n9e/proxy/{dsId}/_msearch                -- ES 日志查询
GET  /api/n9e/proxy/{dsId}/_cat/indices            -- ES 索引列表
```

**批量查询：**
```
POST /api/n9e/query-range-batch    -- 批量 Prometheus 范围查询
POST /api/n9e/query-instant-batch  -- 批量 Prometheus 即时查询
POST /api/n9e/ds-query             -- 统一数据源查询（任意插件类型）
POST /api/n9e/logs-query           -- 统一日志查询
```

**告警相关：**
```
GET  /api/n9e/busi-group/{id}/alert-rules     -- 告警规则列表
POST /api/n9e/busi-group/{id}/alert-rules     -- 创建告警规则
GET  /api/n9e/alert-cur-events/list           -- 当前事件列表
GET  /api/n9e/alert-his-events/list           -- 历史事件列表
GET  /api/n9e/alert-cur-event/{id}            -- 事件详情
GET  /api/n9e/alert-cur-events/card           -- 事件卡片视图
POST /api/n9e/alert-cur-events/card/details   -- 卡片详情
```

---

## 6. 前端状态管理

使用 **React Context**（`CommonStateContext`），非 Redux/MobX。

`store/` 目录只定义 TypeScript 接口和类型：
- `store/common.ts` -- 根状态类型、RequestMethod 枚举
- `store/commonInterface/` -- BusiGroupItem, CommonStoreState
- `store/dashboardInterface/` -- Dashboard, ChartConfig, Layout 接口
- `store/warningInterface/strategy.ts` -- strategyItem, Expression, Metric 类型
- `store/eventInterface/` -- warningEventItem 类型

部分组件使用 `react-hooks-global-state`（如 `pages/explorer/globalState.ts`）。

---

## 7. 前端样式系统

**技术栈：** Tailwind CSS + Less + CSS 自定义属性（变量）

**主题入口：** `C:\project\fe\src\theme\index.less`
**变量定义：** `C:\project\fe\src\theme\variable.css`（503行）

**CSS 变量前缀：** `--fc-`

关键变量：
```css
--fc-fill-1 ~ --fc-fill-7     /* 填充色 */
--fc-text-1 ~ --fc-text-5     /* 文本色 */
--fc-border-color              /* 边框色 */
--fc-primary-color             /* 主色（#6C53B1） */
--fc-success-color             /* 成功色 */
--fc-warning-color             /* 警告色 */
--fc-error-color               /* 错误色 */
```

**4 种主题：** light（默认）、dark、light-gold、light-blue

**字体：** Inter（自托管 woff2）、Noto Sans SC（中文）、Monda-Regular

---

## 8. Explore 页面深度拆解

### 8.1 Prometheus Explorer 数据流

```
用户输入 PromQL
    ↓
PromQLInputNG（Monaco 编辑器，自动补全从 /api/v1/labels 和 /api/v1/label/{label}/values 获取）
    ↓
PromGraphCpt（Table + Graph tabs）
    ├─ Table tab → 即时查询 GET /api/n9e/proxy/{dsId}/api/v1/query
    └─ Graph tab → 范围查询 GET /api/n9e/proxy/{dsId}/api/v1/query_range
    ↓
结果渲染：Table 列表 / Timeseries 图表
```

**关键文件：**
- `pages/explorer/Metric.tsx` -- 入口，多面板管理
- `pages/explorer/Explorer.tsx` -- 调度器，数据源选择
- `pages/explorer/Prometheus/index.tsx` -- Prometheus 子组件
- `components/PromGraphCpt/index.tsx` -- 核心图表组件（300行）
- `components/PromGraphCpt/Table.tsx` -- Table tab（290行）
- `components/PromGraphCpt/Graph.tsx` -- Graph tab（342行）
- `components/PromQLInputNG/index.tsx` -- Monaco PromQL 编辑器（193行）
- `components/PromQLInputNG/utils.ts` -- 变量插值（`$__from`, `$__to`, `$__interval` 等）

**变量插值：**
- `interpolateString()` -- 替换 `$__from`, `$__to`, `$__interval`, `$__rate_interval`, `$__range`
- `getRealStep()` -- 根据时间范围和 maxDataPoints 计算 step（Prometheus 最大 11000 点）

### 8.2 Elasticsearch Explorer 数据流

```
选择索引/模式 + 日期字段 + 查询语法（Lucene/KQL）
    ↓
QueryBuilder 构建 DSL
    ↓
fetchData() → POST /api/n9e/proxy/{dsId}/_msearch
    ├─ 日志条目（data, total）
    └─ 直方图数据（series）
    ↓
UI 布局：
    ├─ FieldsSidebar（左侧字段过滤）
    ├─ Timeseries（日志量直方图）
    ├─ 日志表格（分页、排序、字段选择、行展开）
    └─ 过滤标签栏
```

**关键文件：**
- `pages/explorer/Elasticsearch/index.tsx`（736行）
- `pages/explorer/Elasticsearch/services.ts` -- ES API 调用
- `pages/explorer/Elasticsearch/utils.ts` -- DSL 构建、字段处理

### 8.3 Loki Explorer 数据流

```
输入 LogQL（CodeMirror 编辑器）
    ↓
fetchData() → GET /api/n9e/proxy/{dsId}/api/v1/query_range
    ├─ 流选择器 {xxx} → 并行获取日志条目 + 量直方图
    └─ 指标查询 → Timeseries 图表
    ↓
UI 布局：
    ├─ 量图表（可切换）
    ├─ 日志行（关键字高亮、JSON 格式化、换行）
    └─ 操作栏（时间、换行、JSON、排序）
```

**关键文件：**
- `pages/explorer/Loki/index.tsx`（393行）
- `pages/explorer/Loki/services.ts` -- Loki API 调用

---

## 9. Dashboard 架构

### 9.1 数据流

```
Dashboard List → GET /api/n9e/busi-group/{id}/boards
    ↓
Dashboard Detail → GET /api/n9e/board/{id} → 解析 configs JSON
    ↓
Panels 渲染（网格布局）
    ↓
每个 Panel 的 targets → 批量查询 POST /api/n9e/query-range-batch
    ↓
Transformations 管道处理
    ↓
Renderer 渲染（Timeseries/Stat/Table/Pie/BarGauge 等）
```

### 9.2 关键类型

```typescript
interface IPanel {
  id: string; name: string; type: IType;
  targets: ITarget[]; layout: ILayout;
  options: IOptions; overrides: IOverride[];
  transformations: ITransformation[];
}

interface ITarget {
  refId: string; expr: string; // PromQL
  legendFormat: string; query: object; // ES
  instant: boolean; hide: boolean;
}

type IType = 'row' | 'timeseries' | 'stat' | 'table' | 'tableNG' | 'pie' | 'hexbin' | 'barGauge' | 'text' | 'gauge' | 'iframe';
```

### 9.3 关键目录

| 目录 | 说明 |
|---|---|
| `pages/dashboard/Panels/` | 面板管理（添加、删除、排序） |
| `pages/dashboard/Editor/` | 面板编辑器 |
| `pages/dashboard/Editor/Options/` | 每种图表类型的选项 |
| `pages/dashboard/Editor/QueryEditor/` | 查询编辑器 |
| `pages/dashboard/Renderer/` | 数据获取和渲染 |
| `pages/dashboard/Renderer/Renderer/` | 图表渲染器（Timeseries, Stat, Table, Pie 等） |
| `pages/dashboard/Renderer/datasource/` | 数据源抽象层 |
| `pages/dashboard/Variables/` | 变量系统 |
| `pages/dashboard/transformations/` | 20+ 数据转换 |

---

## 10. 插件架构

每个插件位于 `src/plugins/` 下，结构统一：

```
plugins/{name}/
  index.tsx          -- 插件入口/注册
  AlertRule/         -- 告警规则表单组件
  Dashboard/         -- Dashboard 查询构建器
  Datasource/        -- 数据源配置表单
  Event/             -- 告警事件展示
  Explorer/          -- Explorer 查询构建器和可视化
  RecordingRules/    -- 记录规则查询组件
  services.ts        -- 插件特有 API
  types.ts           -- 插件特有类型
  utils.ts           -- 插件特有工具
  constants.ts       -- 插件常量
  locale/            -- 插件 i18n
  components/        -- 共享组件
```

| 插件 | 路径 | 数据源类型 |
|---|---|---|
| prometheus | `plugins/prometheus/` | Prometheus（仅 Dashboard） |
| elasticsearch | `plugins/elasticsearch/` | ES/OpenSearch |
| clickHouse | `plugins/clickHouse/` | ClickHouse |
| mysql | `plugins/mysql/` | MySQL |
| pgsql | `plugins/pgsql/` | PostgreSQL |
| TDengine | `plugins/TDengine/` | TDengine |
| doris | `plugins/doris/` | Apache Doris |
| victorialogs | `plugins/victorialogs/` | VictoriaLogs |
| opensearch | `plugins/opensearch/` | OpenSearch（仅 Datasource） |

---

## 11. 后端目录结构

```
C:\project\nightingale\
  aiagent/         -- AI Agent 子系统（LLM, MCP, A2A, Chat, Skill, Tools）
  alert/           -- 告警引擎
    aconf/         -- 告警配置结构体
    astats/        -- 告警 Prometheus 指标
    common/        -- 共享工具（key 生成、标签匹配）
    dispatch/      -- 事件分发：消费队列、发送通知、处理订阅
    eval/          -- 规则评估：Scheduler, AlertRuleWorker, cron 循环
    mute/          -- 静默逻辑
    naming/        -- 一致性哈希环（规则分发）
    pipeline/      -- 事件处理管道引擎
    pipeline/processor/  -- 管道处理器：aisummary, callback, eventdrop, eventupdate, logic, relabel
    process/       -- 核心事件处理：fire/recover 生命周期
    queue/         -- 内存事件队列（10M 容量）
    record/        -- 记录规则调度器
    router/        -- 告警引擎 HTTP API
    sender/        -- 通知发送器：email, DingTalk, WeCom, Feishu, Telegram, Mattermost, Lark
  center/          -- Center（API 服务器）子系统
    cconf/         -- Center 配置
    cstats/        -- Center Prometheus 指标
    integration/   -- 集成管理
    metas/         -- 主机元数据集
    router/        -- **主 API 路由器**（所有用户端点）
    sso/           -- SSO 客户端（LDAP, OAuth2, OIDC, CAS, DingTalk, Feishu）
  cli/             -- CLI 工具和升级脚本
  cmd/             -- 二进制入口：center, alert, pushgw, edge, cli, a2a-cli, aichat-cli
  conf/            -- 顶层配置加载
  datasource/      -- 数据源插件系统
    prom/          -- Prometheus 数据源配置
    es/            -- Elasticsearch 数据源
    ck/            -- ClickHouse 数据源
    mysql/         -- MySQL 数据源
    postgresql/    -- PostgreSQL 数据源
    tdengine/      -- TDengine 数据源
    victorialogs/  -- VictoriaLogs 数据源
    doris/         -- Doris 数据源
    opensearch/    -- OpenSearch 数据源
    commons/eslike/ -- 共享 ES-like 查询逻辑
  dscache/         -- 数据源实例缓存（运行时）
  dskit/           -- 数据源工具包：底层 DB 驱动
  memsto/          -- 内存缓存（从 center API 同步）
  models/          -- **所有数据库模型**（GORM 结构体、DB 操作）
  pkg/             -- 共享包（30+）
    ginx/          -- Gin 辅助工具
    httpx/         -- HTTP 服务器配置
    poster/        -- center API HTTP 客户端
    prom/          -- Prometheus 客户端 SDK 包装
    promql/        -- PromQL 解析
    parser/        -- 表达式评估
    secu/          -- RSA 加密
    ldapx/, oauth2x/, oidcx/, cas/ -- SSO 提供者
    hash/          -- 标签哈希
    unit/          -- 值格式化
    i18nx/         -- 国际化
    logx/          -- 日志
    ormx/          -- GORM 辅助
  prom/            -- Prometheus 客户端映射管理
  pushgw/          -- Push Gateway：接收 remote write、OpenTSDB、Falcon、Datadog
  storage/         -- Redis 和 PubSub 总线抽象
```

---

## 12. 后端 API 端点

框架：**Gin** | 用户端点前缀：`/api/n9e` | 服务端点前缀：`/v1/n9e`

认证中间件链：`rt.auth()` → `rt.user()` → `rt.perm(operation)` → `rt.bgrw()/rt.bgro()`

### 12.1 认证端点（`center/router/router_login.go`）

```
POST /auth/login           -- 登录（用户名/密码，LDAP 回退）
POST /auth/logout          -- 登出
POST /auth/refresh         -- 刷新 JWT
GET  /auth/sso-config      -- SSO 配置名
GET  /auth/redirect        -- SSO 重定向
GET  /auth/callback        -- SSO 回调
GET  /auth/perms           -- 所有权限
```

### 12.2 数据源代理与查询（`center/router/router_proxy.go`, `router_query.go`）

```
ANY  /proxy/:id/*url           -- 通用反向代理到任意数据源
POST /query-range-batch        -- 批量 Prometheus 范围查询
POST /query-instant-batch      -- 批量 Prometheus 即时查询
POST /ds-query                 -- 统一数据源查询（任意插件类型）
POST /logs-query               -- 统一日志查询
POST /log-query-batch          -- 批量日志查询
GET  /datasource/brief         -- 数据源简要列表
POST /datasource/query         -- 数据源查询
```

### 12.3 告警规则（`center/router/router_alert_rule.go`）

```
GET  /busi-group/:id/alert-rules           -- 规则列表
POST /busi-group/:id/alert-rules           -- 创建规则
PUT  /busi-group/:id/alert-rule/:arid      -- 更新规则
GET  /alert-rule/:arid                     -- 规则详情
DELETE /busi-group/:id/alert-rules         -- 删除规则
PUT  /busi-group/:id/alert-rules/fields    -- 批量更新字段
POST /busi-group/:id/alert-rules/import    -- 导入规则
POST /busi-group/:id/alert-rules/clone     -- 克隆规则
PUT  /busi-group/alert-rule/validate       -- 验证规则
```

### 12.4 告警事件（`center/router/router_alert_cur_event.go`, `router_alert_his_event.go`）

```
GET  /alert-cur-events/list      -- 当前事件列表
GET  /alert-cur-events/card      -- 当前事件卡片
POST /alert-cur-events/card/details  -- 卡片详情
GET  /alert-cur-event/:eid       -- 事件详情
GET  /alert-his-events/list      -- 历史事件列表
GET  /alert-his-event/:eid       -- 历史事件详情
DELETE /alert-cur-events         -- 删除当前事件
DELETE /alert-his-events         -- 删除历史事件
GET  /alert-cur-events/stats     -- 事件统计
GET  /event-notify-records/:eid  -- 通知记录
GET  /event-detail/:hash         -- 事件详情（hash）
GET  /alert-eval-detail/:id      -- 评估详情
```

### 12.5 静默/订阅（`center/router/router_mute.go`, `router_alert_subscribe.go`）

```
CRUD /busi-group/:id/alert-mutes      -- 静默规则
CRUD /busi-group/:id/alert-subscribes -- 告警订阅
POST /alert-mute-tryrun               -- 静默规则试运行
```

### 12.6 Dashboard（`center/router/router_dashboard.go`, `router_board.go`）

```
GET  /busi-group/:id/boards           -- Dashboard 列表
POST /busi-group/:id/boards           -- 创建 Dashboard
GET  /board/:bid                      -- Dashboard 详情
PUT  /board/:bid                      -- 更新 Dashboard
PUT  /board/:bid/configs              -- 更新配置
DELETE /boards                        -- 删除 Dashboard
POST /busi-group/:id/board/:bid/clone -- 克隆 Dashboard
```

### 12.7 通知系统（`center/router/router_notify_*.go`）

```
CRUD /notify-channel-configs     -- 通知渠道
CRUD /notify-rules               -- 通知规则
CRUD /notify-tpls                -- 通知模板
CRUD /message-templates          -- 消息模板
POST /events-message             -- 事件消息
POST /notify-rule/test           -- 通知规则测试
```

### 12.8 事件管道（`center/router/router_event_pipeline.go`）

```
CRUD /event-pipelines            -- 事件管道
POST /event-pipeline-tryrun      -- 管道试运行
POST /event-processor-tryrun     -- 处理器试运行
GET  /event-pipeline-executions  -- 执行记录
```

### 12.9 AI 子系统（`center/router/router_ai_*.go`）

```
CRUD /ai-agents                  -- AI Agent
CRUD /ai-llm-configs             -- LLM 配置
CRUD /ai-skills                  -- AI 技能
CRUD /mcp-servers                -- MCP 服务器
POST /assistant/chat/new         -- AI 聊天
POST /assistant/message/new      -- AI 消息
POST /stream                     -- SSE 流
```

### 12.10 记录规则（`center/router/router_recording_rule.go`）

```
CRUD /busi-group/:id/recording-rules  -- 记录规则
```

---

## 13. 后端数据模型

所有模型位于 `C:\project\nightingale\models\`。

### 13.1 核心模型

| 模型 | 文件 | 关键字段 |
|---|---|---|
| **AlertRule** | `alert_rule.go` | Id, GroupId, Cate, DatasourceQueries, Name, Severity, Disabled, PromQl, RuleConfig, CronPattern, NotifyRuleIds, PipelineConfigs |
| **AlertCurEvent** | `alert_cur_event.go` | Id, Cate, Cluster, DatasourceId, GroupId, Hash, RuleId, RuleName, Severity, TriggerTime, TriggerValue, Tags, Annotations, IsRecovered |
| **AlertHisEvent** | `alert_his_event.go` | 同 AlertCurEvent + RecoverTime |
| **AlertMute** | `alert_mute.go` | Id, GroupId, Disabled, Prod, Cate, DatasourceIds, Tags, Severities, Btime, Etime, PeriodicRegions |
| **AlertSubscribe** | `alert_subscribe.go` | Id, Name, Disabled, GroupId, RuleId, Severities, Tags, UserGroupIds, NotifyRuleIds |
| **User** | `user.go` | Id, Username, Nickname, Password, Phone, Email, Portrait, Roles, Contacts |
| **UserGroup** | `user_group.go` | Id, Name, UserIds |
| **BusiGroup** | `busi_group.go` | Id, Name, LabelEnable, LabelValue |
| **Target** | `target.go` | Id, GroupId, Ident, Note, Tags, HostIp, AgentVersion, OS |
| **Datasource** | `datasource.go` | Id, Name, PluginType, Category, HTTP, Auth, Status, IsDefault, Weight |
| **NotifyRule** | `notify_rule.go` | ID, Name, Enable, UserGroupIds, PipelineConfigs, NotifyConfigs |
| **NotifyChannelConfig** | `notify_channel.go` | ID, Name, Ident, Enable, RequestType, RequestConfig |
| **MessageTemplate** | `message_tpl.go` | ID, Name, Content |
| **RecordingRule** | `recording_rule.go` | Id, GroupId, DatasourceQueries, Name, PromQl, QueryConfigs, CronPattern |
| **EventPipeline** | `event_pipeline.go` | ID, Name, Typ, TriggerMode, Disabled, LabelFilters, ProcessorConfigs, Nodes, Connections |
| **Dashboard** | `dashboard.go`, `board.go` | Id, GroupId, Name, Tags, Public, Configs |

### 13.2 AI 模型

| 模型 | 说明 |
|---|---|
| AIAgent | AI Agent 配置 |
| AIAssistant | AI 助手 |
| AILlmConfig | LLM 配置 |
| AISkill / AISkillFile | AI 技能 |
| AIMcpServer | MCP 服务器 |

---

## 14. 后端告警引擎

### 14.1 初始化流程

文件：`alert/alert.go` → `Initialize()` + `Start()`

1. 加载配置，创建 Redis，初始化所有 memsto 缓存
2. 创建 `prom.PromClientMap`（Prometheus API 客户端）
3. 创建 `eval.Scheduler`（启动 `LoopSyncRules` goroutine）
4. 创建 `dispatch.Dispatch` + `dispatch.Consumer`
5. 启动 `Consumer.LoopConsume()` goroutine

### 14.2 规则评估流程

```
eval.Scheduler.syncAlertRules() -- 每 9 秒同步
    ↓
遍历 alertRuleCache 中所有规则
    ↓
根据规则类型分发：
    ├─ PROMETHEUS → 一致性哈希环分配 → AlertRuleWorker + Processor
    ├─ HOST → engine name 哈希环
    └─ 其他（ES, MySQL 等）→ ExternalProcessors
    ↓
AlertRuleWorker（每个有独立 cron 调度器，默认 @every 10s）
    ↓
Eval() 方法：
    ├─ PROMETHEUS → GetPromAnomalyPoint() → 查询 Prometheus → 转换为 AnomalyPoint
    ├─ HOST → GetHostAnomalyPoint() → target_miss, offset, pct_target_miss
    └─ 其他 → GetAnomalyPoint() → dscache 插件系统 → plug.QueryData()
    ↓
Processor.Handle(anomalyPoints)
```

### 14.3 事件生命周期

```
Processor.Handle(anomalyPoints)
    ↓
BuildEvent() → 创建 AlertCurEvent
    ↓
运行事件管道（HandleEventPipeline）
    ↓
检查静默条件（mute.IsMuted）
    ↓
按标签哈希分组
    ↓
handleEvent():
    ├─ pendings map: 事件需等待 PromForDuration
    ├─ 超时后移入 fires map
    └─ 推送到 queue.EventQueue
    ↓
HandleRecover():
    ├─ 不在 alertingKeys 中的 fires 事件 → 恢复
    └─ 尊重 RecoverDuration 和 RecoverConfig
```

### 14.4 事件消费与通知

```
Consumer.LoopConsume()
    ↓
从 EventQueue 弹出事件（批量 100）
    ↓
consumeOne():
    ├─ 解析规则名称和注解模板
    ├─ 查询恢复值（recovery_promql 注解）
    ├─ 持久化事件到 DB（models.EventPersist）
    └─ 调用 dispatch.HandleEventNotify(event)
    ↓
Dispatch.HandleEventNotify():
    ├─ HandleEventWithNotifyRule()（新通知规则系统）
    ├─ 发送全局 webhook
    ├─ 填充用户/组信息
    ├─ 构建 NotifyTarget（规则通知组、全局 webhook、回调）
    ├─ 处理告警订阅（handleSub）
    └─ Send() → 分发到渠道发送器（email, DingTalk, WeCom, Feishu 等）
```

---

## 15. 后端数据源代理

文件：`center/router/router_proxy.go`

### 15.1 通用代理：`ANY /api/n9e/proxy/:id/*url`

流程：
1. 从 URL 提取数据源 ID
2. 从 `DatasourceCache.GetById(dsId)` 查找数据源
3. 解析数据源配置的 URL（`ds.HTTPJson.ParseUrl()`，随机打乱 URL 实现负载均衡）
4. 创建 `httputil.ReverseProxy`：
   - 重写 scheme/host 到目标数据源
   - 剥离 n9e 前缀：`/api/n9e/proxy/:id/xxx` → `/xxx`
   - 合并查询字符串
   - **注入凭证**：`req.SetBasicAuth(user, password)`
   - **注入自定义 Headers**：`ds.HTTPJson.Headers`
5. 使用缓存的 `http.Transport`（含 TLS、超时配置）

### 15.2 Prometheus 批量查询：`POST /api/n9e/query-range-batch`

使用 `prom.PromClientMap.GetCli(datasourceId)` 获取预配置的 Prometheus API 客户端。

### 15.3 统一数据源查询：`POST /api/n9e/ds-query`

使用 `dscache.DsCache.Get(cate, datasourceId)` 获取插件实例，调用 `plug.QueryData(ctx, query)`。

### 15.4 凭证注入方式

- **代理路径**：在 director 函数中通过 `req.SetBasicAuth()` 注入
- **插件路径**：各插件在 `InitClient()` 时从 Datasource 模型的 HTTP 和 Auth 字段创建 HTTP 客户端

---

## 16. 后端认证授权

文件：`center/router/router_mw.go`

### 16.1 认证模式

- **Proxy Auth**：从 HTTP 头读取用户名，自动创建用户
- **Token Auth**：
  1. Fixed token：检查 `X-User-Token` 头
  2. JWT：Bearer token → HMAC-SHA256 验证 → Redis 查找

### 16.2 授权中间件

| 中间件 | 说明 |
|---|---|
| `rt.user()` | 加载用户模型，设置 context |
| `rt.admin()` | Admin 角色检查 |
| `rt.perm(operation)` | 权限检查 |
| `rt.bgrw()` | 业务组读写检查 |
| `rt.bgro()` | 业务组只读检查 |
| `rt.userGroupWrite()` | 用户组写权限检查 |

### 16.3 SSO 支持

LDAP, OAuth2, OIDC, CAS, DingTalk, Feishu

---

## 17. SREAgent 对齐状态

### 17.1 已对齐模块

| 模块 | SREAgent | Nightingale | 状态 |
|---|---|---|---|
| 数据源管理 | `internal/handler/datasource.go` | `center/router/router_datasource.go` | 已对齐 |
| 告警规则 | `internal/handler/alert_rule.go` | `center/router/router_alert_rule.go` | 已对齐 |
| 告警事件 | `internal/handler/alert_event.go` | `center/router/router_alert_cur_event.go` | 已对齐 |
| 静默规则 | `internal/handler/mute_rule.go` | `center/router/router_mute.go` | 已对齐 |
| 告警订阅 | `internal/handler/subscribe_rule.go` | `center/router/router_alert_subscribe.go` | 已对齐 |
| 用户管理 | `internal/handler/user.go` | `center/router/router_user.go` | 已对齐 |
| 团队管理 | `internal/handler/team.go` | `center/router/router_user_group.go` | 已对齐 |
| 业务组 | `internal/handler/team.go` | `center/router/router_busi_group.go` | 已对齐 |
| 认证 | `internal/handler/auth.go` | `center/router/router_login.go` | 已对齐 |
| Dashboard | `internal/handler/dashboard.go` | `center/router/router_dashboard.go` | 已对齐 |
| 数据源代理 | `internal/handler/proxy.go` | `center/router/router_proxy.go` | 已对齐 |
| 标签注册表 | `internal/handler/label_registry.go` | 无直接对应 | SREAgent 特有 |

### 17.2 Explore 页面对齐状态

| 功能 | Nightingale | SREAgent | 差距 |
|---|---|---|---|
| 数据源选择器 | DatasourceSelectV3 | NSelect（已实现） | 已对齐 |
| PromQL 编辑器 | PromQLInputNG（Monaco） | PromQLEditor（CodeMirror） | 语法高亮有差距 |
| Table tab | 即时查询 `/api/v1/query` | 范围查询 `/api/v1/query_range` | 查询方式不同 |
| Graph tab | 范围查询 `/api/v1/query_range` | 范围查询（已对齐） | 已对齐 |
| 标签自动补全 | Monaco 内置 | KVEditor NAutoComplete | 已对齐 |
| 图表交互 | uPlot | ECharts | 库不同，功能已对齐 |
| 日志探索 | ES _msearch | VLogs 查询 | 查询引擎不同 |
| 直方图 | Timeseries 组件 | LogHistogram（ECharts） | 已对齐 |

### 17.3 已移植的 Nightingale 功能

| 功能 | SREAgent 版本 | 说明 |
|---|---|---|
| 事件管道 | v4.29.0 | 可编程事件处理管道，Processor 注册表模式 |
| 指标内置规则 | v4.28.0 | PresetRule + 分类体系，299 条预置规则 |
| 记录规则引擎 | v4.32.0 | RecordingRuleEngine + cron 调度 + 执行记录 |
| 记录规则前端增强 | v4.35.0 | 数据源过滤、名称验证、PromQL 预验证、批量更新 |
| 通知渠道扩展 | v4.30.0 | 从 4 种扩展至 17 种渠道类型 |
| Saved Views | v4.32.0 | 持久化视图存储，API + 前端 |
| Metric Views | v4.35.0 | 持久化指标视图（后端 CRUD + 三栏前端），Nightingale Quick Views 移植 |
| 内置指标增强 | v4.35.0 | 单位筛选、Explorer Drawer、MetricFilter 管理 UI |
| ES 日志探索 | v4.34.0 | Elasticsearch 数据源支持 + 日志查询（Index Pattern 管理见 v4.37.0） |
| LLM 配置管理 | v4.36.0 | 独立 LLM Provider CRUD + AES-256-GCM 加密 + 连接测试 |
| MCP 服务器管理 | v4.36.0 | MCP SSE 客户端 + 工具发现 + 连接测试 |
| AI 技能管理 | v4.36.0 | SKILL.md + 辅助文件 + zip/tar.gz 导入 |
| AI Agent SSE 流式 | v4.36.0 | Agent 执行实时 token 推送（EventSource + 自动回退轮询） |
| ES Index Pattern | v4.37.0 | ES 索引模式 CRUD + 告警规则引用检查 + 前端管理页面 |

### 17.4 Nightingale 特有功能（SREAgent 无对应）

| 功能 | 说明 | 优先级 |
|---|---|---|
| Push Gateway | remote write、OpenTSDB、Falcon、Datadog | 低 |
| 任务执行 | 自愈脚本执行 | 低 |
| Trace 探索 | Jaeger/Zipkin trace 查看 | 低 |

---

## 附录：快速查找指南

### "我要找 X 功能的代码"

| 功能 | 前端文件 | 后端文件 |
|---|---|---|
| PromQL 编辑器 | `components/PromQLInputNG/` | `pkg/promql/` |
| 告警规则表单 | `pages/alertRules/Form/` | `center/router/router_alert_rule.go` |
| 告警事件列表 | `pages/historyEvents/`, `pages/alertCurEvent/` | `center/router/router_alert_cur_event.go` |
| Dashboard 编辑器 | `pages/dashboard/Editor/` | `center/router/router_dashboard.go` |
| 数据源代理 | （前端使用 proxy URL） | `center/router/router_proxy.go` |
| 通知渠道 | `pages/notificationChannels/` | `center/router/router_notify_channel.go` |
| 通知规则 | `pages/notificationRules/` | `center/router/router_notify_rule.go` |
| 事件管道 | `pages/eventPipeline/` | `center/router/router_event_pipeline.go` |
| 静默规则 | `pages/warning/shield/` | `center/router/router_mute.go` |
| 告警订阅 | `pages/warning/subscribe/` | `center/router/router_alert_subscribe.go` |
| 用户管理 | `pages/user/users/` | `center/router/router_user.go` |
| SSO 配置 | `pages/help/SSOConfigs/` | `center/router/router_login.go` |
| AI 配置 | `pages/aiConfig/` | `center/router/router_ai_*.go` |
| 日志探索 | `pages/explorer/Elasticsearch/`, `pages/explorer/Loki/` | `datasource/es/`, `datasource/victorialogs/` |
| 指标探索 | `pages/explorer/Prometheus/` | `center/router/router_proxy.go` |
