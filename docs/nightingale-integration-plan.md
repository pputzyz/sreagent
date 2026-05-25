# Nightingale V9 → SREAgent 集成方案

> v1.0 | 2026-05-25 | 状态：待审批

## 背景

SREAgent 是一个自建的 SRE 平台（Go 1.25 + Gin + Vue 3 + Naive UI + MySQL 8 + Redis 7），目前在告警引擎、通知管道、值班排班、AI Agent 等方面已有较完整的实现。但在数据源抽象、通知渠道扩展性、数据查询 UI、自定义仪表盘等方面与 Nightingale V9 存在显著差距。

Nightingale V9 是一个成熟的开源可观测性平台（Go + GORM + React），采用微服务架构，支持 10+ 数据源类型、20+ 通知渠道、完整的 PromQL/LogsQL 查询编辑器、Grafana 级别的仪表盘系统。

**目标**：将 Nightingale V9 的核心能力适配到 SREAgent 的技术栈中，而不是照搬代码。后端 Go 代码可做 GORM→原生 SQL 的转换后复用；前端 React 代码不可直接复制，需用 Vue 3 + Naive UI 重新实现。

---

## 一、差距分析

| 能力维度 | SREAgent 现状 | Nightingale V9 | 差距等级 |
|----------|--------------|----------------|----------|
| **数据源抽象** | 4 种（Prometheus/VM/Zabbix/VLogs），硬编码 client | 10+ 种，工厂+注册表模式，异步初始化，缓存去重 | 🔴 高 |
| **通知渠道** | 4 种（Lark/Email/HTTP/Script） | 20+ 种，Provider 接口+注册表，DAG 管道引擎 | 🟡 中 |
| **数据查询 UI** | 无前端查询界面，仅有后端 PromQL client | 完整查询编辑器，变量插值，多数据源叠加，自动补全 | 🔴 高 |
| **自定义仪表盘** | 3 个固定视图，无用户自定义 | Grafana 级别：面板编辑器、模板变量、共享、导入导出 | 🔴 高 |
| **AI Agent** | 已有完整实现（ReAct/Plan+ReAct/Direct，7 种 LLM） | 类似架构，skill 系统，MCP+A2A | ✅ 已对齐 |
| **告警引擎** | 完整（规则/事件/分组/抑制/静默/升级/心跳） | 类似架构 | ✅ 已对齐 |
| **值班排班** | 完整（轮转/替班/升级策略） | 类似架构 | ✅ 已对齐 |
| **日志查询** | 有 VLogs 数据源但无查询 UI | 完整 LogsQL 编辑器 | 🟡 中 |
| **录制规则** | 无 | 支持 recording rules | 🟡 中 |

---

## 二、集成策略

### 2.1 总体原则

1. **后端适配，前端重建**：Go 代码可复用核心逻辑（接口定义、算法、业务流程），但需将 GORM 替换为原生 SQL。React 代码不可直接复制，需用 Vue 3 + Naive UI 重新实现 UI 和交互。
2. **渐进式集成**：按优先级分 4 个阶段，每阶段独立可用，不阻塞现有功能。
3. **保持 SREAgent 架构**：不引入 GORM，不改 monolith 架构，不换前端技术栈。
4. **接口兼容**：Nightingale 的 Datasource 接口、NotifyChannelProvider 接口等可直接借鉴，但实现层用 SREAgent 的 DB 层。

### 2.2 不做的事

- ❌ 不引入 GORM（保持原生 SQL + sqlx 风格）
- ❌ 不迁移 React 代码（用 Vue 3 重写）
- ❌ 不改 monolith 为微服务
- ❌ 不引入 Nightingale 的前端构建体系（保持 Vite + Vue 3）
- ❌ 不照搬 Nightingale 的数据库 schema（用迁移文件演进）

---

## 三、分阶段实施计划

### Phase 1：数据源抽象层重构（优先级最高）

**目标**：将 SREAgent 的硬编码数据源 client 改为 Nightingale 的工厂+注册表模式，支持 10+ 数据源类型。

#### 1.1 接口定义（借鉴 Nightingale）

```go
// internal/pkg/datasource/datasource.go
type Datasource interface {
    Init() error                    // 初始化连接
    InitClient() error              // 异步初始化 HTTP client
    Validate() error                // 验证配置有效性
    Equal(other Datasource) bool    // 缓存去重比较
    QueryData(ctx context.Context, query string, start, end int64, step int) ([]QueryResult, error)
    QueryLog(ctx context.Context, query string, limit int) ([]LogResult, error)
    MakeLogQuery(query string, limit int) (string, error)
    MakeTSQuery(query string) (string, error)
    QueryMapData(ctx context.Context, queries []string, start, end int64, step int) (map[string][]QueryResult, error)
}
```

#### 1.2 工厂+注册表

```go
// internal/pkg/datasource/registry.go
var (
    creators   = make(map[string]func(*DatasourceConfig) Datasource)
    dsCache    = &Cache{data: make(map[string]map[int64]Datasource)}
)

func Register(dsType string, creator func(*DatasourceConfig) Datasource) {
    creators[dsType] = creator
}

// 各数据源实现通过 init() 自注册
func init() {
    datasource.Register("prometheus", func(cfg *DatasourceConfig) Datasource {
        return &PromDatasource{cfg: cfg}
    })
}
```

#### 1.3 需新增的数据源实现

| 数据源 | 优先级 | 复杂度 | 说明 |
|--------|--------|--------|------|
| Elasticsearch | P0 | 中 | 日志查询核心，借鉴 Nightingale 的 es/ 包 |
| ClickHouse | P0 | 中 | 日志/指标双用途，借鉴 ck/ 包 |
| InfluxDB | P1 | 低 | 时序数据库，HTTP API 查询 |
| MySQL/Postgres | P1 | 中 | SQL 数据源，用于自定义指标 |
| CloudWatch | P2 | 高 | AWS 云监控，需要 AWS SDK |
| Stackdriver | P2 | 高 | GCP 云监控 |

#### 1.4 数据源缓存层

借鉴 Nightingale 的 `dscache/cache.go`：
- `map[type]map[id]Datasource` 二级缓存
- 异步 `InitClient()`，避免阻塞启动
- `Equal()` 比较实现配置变更时的增量更新
- 定时刷新（60s）+ 主动推送双通道

#### 1.5 前端：数据源管理页重构

**现状**：SREAgent 的数据源管理页只支持 4 种类型的基本配置。
**目标**：支持 10+ 类型的完整配置，包括：
- 数据源类型选择器（带图标和说明）
- 动态表单（根据类型显示不同配置项）
- 连接测试（异步，显示详细结果）
- 数据源健康状态监控

**实现**：用 Vue 3 + Naive UI 重新实现，参考 Nightingale 的 React 组件逻辑。

---

### Phase 2：通知渠道扩展（优先级高）

**目标**：将 4 种通知渠道扩展到 15+，引入 Provider 接口+注册表模式。

#### 2.1 Provider 接口

```go
// internal/pkg/notify/provider.go
type NotifyChannelProvider interface {
    Name() string
    Send(ctx context.Context, params SendParams) error
    ValidateConfig(cfg map[string]interface{}) error
    GetConfigSchema() ConfigSchema  // 用于前端动态表单
}

var providers = make(map[string]NotifyChannelProvider)

func RegisterProvider(p NotifyChannelProvider) {
    providers[p.Name()] = p
}
```

#### 2.2 需新增的渠道

| 渠道 | 优先级 | 实现方式 |
|------|--------|----------|
| 钉钉（DingTalk） | P0 | Webhook + 签名验证 |
| 企业微信（WeCom） | P0 | Webhook + Markdown |
| Slack | P0 | Webhook + Block Kit |
| Telegram | P1 | Bot API |
| Discord | P1 | Webhook |
| 短信（SMS） | P1 | 阿里云/腾讯云 SDK |
| PagerDuty | P2 | Events API v2 |
| Opsgenie | P2 | REST API |
| 飞书机器人增强 | P1 | 交互式卡片 + 消息更新 |
| Webhook 增强 | P1 | 自定义 Header/Body/Method |

#### 2.3 DAG 管道引擎

借鉴 Nightingale 的 dispatch 层，引入 DAG 管道：
- 事件 → 路由匹配 → 模板渲染 → 渠道分发 → 回调追踪
- 支持管道节点插件化（relabel、AI summary、enrichment、throttle）
- 可视化管道编辑器（Phase 2 的前端部分）

#### 2.4 通知模板引擎

引入 Go template 渲染：
```go
// 模板支持变量：{{.AlertName}}, {{.Severity}}, {{.Value}}, {{.Labels.xxx}}
// 支持条件：{{if eq .Severity "critical"}}...{{end}}
// 支持循环：{{range .Labels}}...{{end}}
```

前端模板编辑器：Monaco Editor + 模板变量自动补全。

---

### Phase 3：数据查询 UI（优先级高）

**目标**：实现完整的 PromQL/LogsQL 查询编辑器。

#### 3.1 查询编辑器组件

```
web/src/components/query/
├── QueryEditor.vue          # 主编辑器（Monaco Editor）
├── QueryAutocomplete.ts     # PromQL 自动补全（指标名 + 标签）
├── DatasourceSelector.vue   # 数据源选择器
├── TimeRangePicker.vue      # 时间范围选择
├── QueryResult.vue          # 查询结果展示（表格 + 图表）
└── QueryHistory.vue         # 查询历史
```

#### 3.2 核心功能

| 功能 | 说明 |
|------|------|
| PromQL 编辑器 | Monaco Editor + PromQL 语法高亮 + 自动补全 |
| LogsQL 编辑器 | 同上，LogsQL 语法 |
| 变量插值 | `$instance`, `$job` 等变量，支持下拉选择 |
| 多数据源叠加 | 同一图表叠加多个数据源的查询结果 |
| 即时/范围查询 | 切换 instant 和 range 查询模式 |
| 查询历史 | 本地存储最近 50 条查询 |
| 结果导出 | CSV/JSON 导出 |

#### 3.3 后端：查询代理层

```go
// internal/handler/query.go
func (h *QueryHandler) ProxyQuery(c *gin.Context) {
    dsId := c.Param("datasource_id")
    ds, ok := h.dsCache.Get(dsId)
    if !ok { /* 404 */ }
    
    // 统一查询入口，支持 PromQL/LogsQL/SQL
    result, err := ds.QueryData(ctx, query, start, end, step)
    // ...
}
```

---

### Phase 4：自定义仪表盘（优先级中，工作量最大）

**目标**：实现 Grafana 级别的自定义仪表盘系统。

#### 4.1 数据模型

```sql
CREATE TABLE dashboards (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    tags JSON,                    -- ["linux", "redis", "mysql"]
    panels JSON,                  -- 面板配置（完整 JSON）
    variables JSON,               -- 模板变量
    layout JSON,                  -- 布局配置（grid positions）
    created_by BIGINT,
    updated_by BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE dashboard_shares (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    dashboard_id BIGINT NOT NULL,
    share_type ENUM('public', 'team', 'user') NOT NULL,
    target_id BIGINT,             -- team_id 或 user_id
    permission ENUM('view', 'edit') DEFAULT 'view',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 4.2 面板类型

| 面板类型 | 说明 | 实现方式 |
|----------|------|----------|
| 时间序列图 | 折线图、面积图 | ECharts 或 Apache ECharts |
| 表格 | 查询结果表格 | Naive UI NDataTable |
| 统计值 | 单值指标 | 自定义组件 |
| 日志面板 | 日志流 | 自定义组件 |
| 告警面板 | 告警事件列表 | 复用现有告警组件 |
| 拓扑图 | 服务拓扑 | D3.js 或 vis.js |
| Markdown | 自定义文本 | Markdown 渲染 |

#### 4.3 前端：仪表盘编辑器

```
web/src/pages/dashboard/custom/
├── DashboardList.vue          # 仪表盘列表（搜索、标签筛选、收藏）
├── DashboardView.vue          # 仪表盘查看模式
├── DashboardEdit.vue          # 仪表盘编辑模式
├── PanelEditor.vue            # 面板编辑器
├── PanelRenderer.vue          # 面板渲染器（根据类型分发）
├── VariableEditor.vue         # 模板变量编辑器
└── ImportExport.vue           # 导入导出（JSON）
```

#### 4.4 核心交互

- 拖拽布局（grid 布局，参考 react-grid-layout → vue-grid-layout）
- 面板实时预览
- 变量联动（修改变量自动刷新所有面板）
- 时间范围联动
- 全屏模式
- 自动刷新（5s/10s/30s/1m/5m）

---

## 四、前端技术方案

### 4.1 图表库选择

**推荐 Apache ECharts**：
- 与 Vue 3 集成良好（vue-echarts）
- 支持时间序列、饼图、柱状图、散点图、热力图等
- 支持大数据量渲染（sampling）
- 开源免费，社区活跃

### 4.2 查询编辑器

**推荐 Monaco Editor**：
- VS Code 同款编辑器
- 支持自定义语法高亮和自动补全
- 已有 Vue 3 wrapper（@guolao/vue-monaco-editor）

### 4.3 仪表盘布局

**推荐 vue-grid-layout**：
- react-grid-layout 的 Vue 移植版
- 支持拖拽、缩放、响应式

### 4.4 组件目录结构

```
web/src/components/
├── charts/
│   ├── TimeSeriesChart.vue    # 时间序列图
│   ├── StatPanel.vue          # 统计值面板
│   ├── TablePanel.vue         # 表格面板
│   └── LogPanel.vue           # 日志面板
├── query/
│   ├── PromQLEditor.vue       # PromQL 编辑器
│   ├── LogsQLEditor.vue       # LogsQL 编辑器
│   └── QueryBuilder.vue       # 可视化查询构建器
└── dashboard/
    ├── GridLayout.vue          # 网格布局
    ├── PanelWrapper.vue        # 面板包装器
    └── VariablePicker.vue      # 变量选择器
```

---

## 五、版本兼容策略

### 5.1 与 Nightingale 的关系

SREAgent 不追求与 Nightingale 版本号保持一致。两者是独立产品，但核心设计理念和接口模式保持对齐：

- **接口兼容**：Datasource 接口、NotifyChannelProvider 接口的设计理念与 Nightingale 对齐
- **配置格式**：数据源配置、通知渠道配置的 JSON schema 尽量兼容
- **数据格式**：QueryResult、StreamChunk 等数据结构保持语义一致
- **独立迭代**：SREAgent 按自己的节奏发版，不追 Nightingale 的 tag

### 5.2 迁移路径

对于从 Nightingale 迁移到 SREAgent 的用户：
- 提供数据源配置导入工具（JSON → SREAgent DB）
- 提供告警规则导入工具
- 提供仪表盘导入工具（Nightingale JSON → SREAgent JSON）

---

## 六、依赖引入

### 6.1 后端新增依赖

| 依赖 | 用途 | 阶段 |
|------|------|------|
| `github.com/elastic/go-elasticsearch/v8` | Elasticsearch 客户端 | Phase 1 |
| `github.com/ClickHouse/clickhouse-go/v2` | ClickHouse 客户端 | Phase 1 |
| `github.com/influxdata/influxdb-client-go/v2` | InfluxDB 客户端 | Phase 1 |
| `github.com/aws/aws-sdk-go-v2` | CloudWatch 客户端 | Phase 2 |

### 6.2 前端新增依赖

| 依赖 | 用途 | 阶段 |
|------|------|------|
| `echarts` + `vue-echarts` | 图表渲染 | Phase 3-4 |
| `@guolao/vue-monaco-editor` | 查询编辑器 | Phase 3 |
| `vue-grid-layout` | 仪表盘布局 | Phase 4 |
| `monaco-editor` | 编辑器核心 | Phase 3 |

---

## 七、工作量估算

| 阶段 | 后端工作量 | 前端工作量 | 总工作量 | 建议周期 |
|------|-----------|-----------|---------|----------|
| Phase 1：数据源抽象 | 5-7 天 | 3-4 天 | 8-11 天 | 2 周 |
| Phase 2：通知渠道 | 3-4 天 | 4-5 天 | 7-9 天 | 1.5 周 |
| Phase 3：数据查询 UI | 2-3 天 | 5-7 天 | 7-10 天 | 1.5 周 |
| Phase 4：自定义仪表盘 | 3-4 天 | 10-14 天 | 13-18 天 | 3 周 |
| **总计** | **13-18 天** | **22-30 天** | **35-48 天** | **8 周** |

---

## 八、风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 数据源抽象层改造破坏现有功能 | 高 | 保持现有 4 种数据源的向后兼容，新接口用 adapter 模式包装旧实现 |
| 仪表盘工作量超预期 | 中 | Phase 4 可拆分为 MVP（固定面板类型）+ 完整版（可扩展面板） |
| ECharts 性能问题（大数据量） | 中 | 启用 sampling，限制单面板最大数据点数 |
| Monaco Editor 包体积大 | 低 | 按需加载，CDN 分发 |

---

## 九、验收标准

### Phase 1 验收
- [ ] 新增 Elasticsearch 数据源，可在数据源管理页配置并测试连接
- [ ] 新增 ClickHouse 数据源，同上
- [ ] 现有 4 种数据源功能不受影响
- [ ] 数据源缓存层正常工作（启动不阻塞，配置变更自动刷新）

### Phase 2 验收
- [ ] 新增钉钉、企业微信、Slack 通知渠道
- [ ] 通知模板支持 Go template 语法
- [ ] 现有 4 种渠道功能不受影响
- [ ] 通知发送记录可追踪

### Phase 3 验收
- [ ] PromQL 查询编辑器可用（语法高亮、自动补全、执行查询）
- [ ] 查询结果以表格和图表形式展示
- [ ] 支持即时查询和范围查询切换
- [ ] 查询历史可查看

### Phase 4 验收
- [ ] 可创建、编辑、删除自定义仪表盘
- [ ] 支持至少 4 种面板类型
- [ ] 面板支持拖拽布局
- [ ] 模板变量可联动刷新
- [ ] 仪表盘可分享给团队

---

## 十、实施决策点

在正式开始实施前，需要确认以下决策：

1. **图表库**：ECharts（推荐） vs Chart.js vs D3.js
2. **查询编辑器**：Monaco Editor（推荐） vs CodeMirror
3. **仪表盘布局**：vue-grid-layout（推荐） vs 自研 grid
4. **Phase 4 时机**：是否在 Phase 1-3 完成后再开始
5. **数据源优先级**：先做 ES/ClickHouse 还是先做 InfluxDB/SQL

---

*本文档基于对 Nightingale V9 源码（C:\project\nightingale）的深度分析编写。*
