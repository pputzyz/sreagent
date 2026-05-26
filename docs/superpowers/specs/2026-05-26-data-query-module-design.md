# 数据查询模块对齐设计 — v4.32.0

## Context

SREAgent 需要对齐 Nightingale V9 的数据查询模块，补齐 4 个子模块：
- **A: Recording Rules Enhancement** — 引擎执行 + API 文档 + 审计日志
- **B: Saved Views Persistence** — 后端 CRUD + 前端持久化
- **C: Metric Views** — 全新三面板探索器
- **D: Instant Query Enhancement** — PromQL 编辑器增强

## A. Recording Rules Enhancement

### 现状
- 后端完整：model → repo → service → handler → routes
- 前端完整：Index.vue 含 CRUD、批量操作、导入导出、克隆、启用/禁用
- 缺失：引擎执行、API 文档、审计日志

### 新增：引擎执行器
**文件**: `internal/engine/recording_rule.go`

```
RecordingRuleEngine {
  repo    repository.RecordingRuleRepository
  dsSvc   service.DataSourceService
  evalSvc service.EvaluatorService
}

func (e *RecordingRuleEngine) Run(ctx context.Context, rule *model.RecordingRule) error
```

- 解析 CronPattern，按调度执行
- 调用数据源查询 PromQL → 写入目标数据源（作为 recording 规则输出）
- 记录执行状态到 `recording_rule_executions` 表

**新表**: `recording_rule_executions`
```sql
CREATE TABLE recording_rule_executions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  rule_id BIGINT NOT NULL,
  status VARCHAR(20) NOT NULL,  -- success/error
  error_message TEXT,
  duration_ms INT,
  samples_written INT,
  executed_at DATETIME NOT NULL,
  INDEX idx_rule_executed (rule_id, executed_at)
);
```

### API 文档补充
在 `docs/api.md` 中补充 recording-rules 端点的完整文档。

### 审计日志
在 handler 的 Create/Update/Delete 操作中添加 `h.auditSvc.Log()` 调用。

## B. Saved Views Persistence

### 现状
- `ViewSelect.vue` 使用 localStorage（key: `sre-saved-views`）
- 无后端 API

### 新增后端
**Model**: `internal/model/saved_view.go`
```go
type SavedView struct {
    BaseModel
    Name        string `json:"name" gorm:"size:200;not null"`
    Description string `json:"description" gorm:"size:500"`
    Tab         string `json:"tab" gorm:"size:20;not null"`  // metric/log/trace
    DatasourceID uint  `json:"datasourceId"`
    Expression  string `json:"expression" gorm:"type:text"`
    QueryConfig  string `json:"queryConfig" gorm:"type:text"` // JSON
    IsPublic    bool   `json:"isPublic" gorm:"default:false"`
    CreatedBy   uint   `json:"createdBy"`
    UpdatedBy   uint   `json:"updatedBy"`
}
```

**API 端点**:
- `GET /api/v1/saved-views` — 列表（分页 + 过滤 tab/isPublic）
- `POST /api/v1/saved-views` — 创建
- `PUT /api/v1/saved-views/:id` — 更新
- `DELETE /api/v1/saved-views/:id` — 删除
- `POST /api/v1/saved-views/:id/copy` — 克隆

### 前端改造
- `ViewSelect.vue`: localStorage → API 调用，保留 localStorage 作为 fallback
- 新增 `web/src/api/saved-views.ts` API 模块

## C. Metric Views（新功能）

### 设计
独立页面 `/explore/metrics`，三面板布局：

```
┌──────────────┬────────────────────────────────────────┐
│  视图列表     │  标签筛选 + 指标列表                      │
│  (左侧栏)     │  (中部)                                  │
│              │                                         │
│  - 收藏视图   │  ┌─────────────────────────────────┐    │
│  - 最近使用   │  │  标签选择器 (级联)                │    │
│  - 搜索      │  │  __name__ = go_goroutines        │    │
│              │  │  instance = localhost:9090        │    │
│              │  └─────────────────────────────────┘    │
│              │                                         │
│              │  ┌─────────────────────────────────┐    │
│              │  │  指标分类列表                      │    │
│              │  │  go_* (12)                        │    │
│              │  │  node_* (8)                       │    │
│              │  │  process_* (5)                    │    │
│              │  └─────────────────────────────────┘    │
├──────────────┴────────────────────────────────────────┤
│  图表区域 (底部)                                        │
│  ┌─────────────────────────────────────────────┐      │
│  │  go_goroutines{instance="localhost:9090"}    │      │
│  │  ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁                            │      │
│  └─────────────────────────────────────────────┘      │
└───────────────────────────────────────────────────────┘
```

### 前端组件
- `web/src/pages/explore/MetricViews.vue` — 主页面
- `web/src/components/query/MetricLabelSelector.vue` — 级联标签选择器
- `web/src/components/query/MetricList.vue` — 指标分类列表
- `web/src/components/query/MetricChart.vue` — 图表展示

### API
复用现有 `/api/v1/datasources/:id/query` 端点。
新增 `/api/v1/datasources/:id/label-values` 获取标签值。

## D. Instant Query Enhancement

### 改进点
1. PromQL 编辑器自动补全增强：集成指标名和标签名
2. 查询历史记录（最近 20 条）
3. 查询结果格式化：表格 + 图表双视图
4. 错误提示优化：精确定位语法错误位置

### 前端改动
- `web/src/components/query/PromQLEditor.vue` — 增强自动补全
- `web/src/pages/explore/Index.vue` — 添加查询历史、结果视图切换

## 修改文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/engine/recording_rule.go` | 新增 | 规则执行引擎 |
| `internal/engine/recording_rule_test.go` | 新增 | 测试 |
| `internal/model/saved_view.go` | 新增 | SavedView 模型 |
| `internal/repository/saved_view.go` | 新增 | SavedView 仓库 |
| `internal/service/saved_view.go` | 新增 | SavedView 服务 |
| `internal/handler/saved_view.go` | 新增 | SavedView 处理器 |
| `internal/router/saved_view_routes.go` | 新增 | 路由注册 |
| `internal/pkg/dbmigrate/migrations/000072_saved_views.up.sql` | 新增 | 迁移 |
| `internal/pkg/dbmigrate/migrations/000072_saved_views.down.sql` | 新增 | 回滚 |
| `internal/pkg/dbmigrate/migrations/000073_recording_rule_executions.up.sql` | 新增 | 迁移 |
| `internal/pkg/dbmigrate/migrations/000073_recording_rule_executions.down.sql` | 新增 | 回滚 |
| `web/src/api/saved-views.ts` | 新增 | API 模块 |
| `web/src/pages/explore/MetricViews.vue` | 新增 | 指标视图页面 |
| `web/src/components/query/MetricLabelSelector.vue` | 新增 | 标签选择器 |
| `web/src/components/query/MetricList.vue` | 新增 | 指标列表 |
| `web/src/components/query/MetricChart.vue` | 新增 | 图表组件 |
| `web/src/components/query/ViewSelect.vue` | 修改 | 后端持久化 |
| `web/src/components/query/PromQLEditor.vue` | 修改 | 增强补全 |
| `web/src/pages/explore/Index.vue` | 修改 | 查询历史+视图切换 |
| `cmd/server/wire.go` | 修改 | DI 注入 |
| `internal/router/router.go` | 修改 | 路由注册 |
| `internal/handler/recording_rule.go` | 修改 | 审计日志 |
| `web/src/i18n/zh-CN.ts` | 修改 | 翻译 |
| `web/src/i18n/en.ts` | 修改 | 翻译 |
| `CHANGELOG.md` | 修改 | 版本记录 |
| `MODULES.md` | 修改 | 模块更新 |
| `CLAUDE.md` | 修改 | 版本号 |
| `web/package.json` | 修改 | 版本号 |
| `docs/api.md` | 修改 | API 文档 |

## 验证

1. `go build ./cmd/server/` 通过
2. `cd web && npx vue-tsc --noEmit` 零错误
3. `cd web && npx vite build` 成功
4. Recording Rules: 创建规则 → 执行记录可查
5. Saved Views: 创建/编辑/删除/克隆 → 数据持久化
6. Metric Views: 选择数据源 → 级联标签 → 选择指标 → 图表展示
7. Instant Query: PromQL 补全 → 查询历史 → 结果视图切换
