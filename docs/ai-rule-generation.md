# AI 规则生成

> **v4.13.0** | 自然语言 → 告警规则 / 抑制规则 / 静默规则

## 目录

- [端到端流程](#端到端流程)
- [架构设计](#架构设计)
- [Few-Shot Prompt 结构](#few-shot-prompt-结构)
- [缓存策略](#缓存策略)
- [DryRun 模式](#dryrun-模式)
- [规则改进 (Improve)](#规则改进-improve)
- [标签建议](#标签建议)
- [表达式验证](#表达式验证)
- [API 端点](#api-端点)
- [前端集成](#前端集成)
- [配置要求](#配置要求)

---

## 端到端流程

```
用户输入自然语言描述
        │
        ▼
┌───────────────────┐
│  1. 检查缓存       │  ← 相同描述 + 数据源 + 规则类型 → 命中缓存直接返回
└───────┬───────────┘
        │ 缓存未命中
        ▼
┌───────────────────┐
│  2. 检查 AI 启用   │  ← 系统设置 AI.Enabled + AI.Modules.RuleGen.Enabled
└───────┬───────────┘
        │ 已启用
        ▼
┌───────────────────┐
│  3. 构建上下文      │
│  - 标签注册表       │  ← 最多 50 个标签键 + 每个键 5 个常用值
│  - 已有规则         │  ← 最多 30 条已有规则（避免重复）
│  - 预置规则匹配     │  ← 关键词搜索匹配的预置规则模板
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  4. 构建 Prompt     │
│  - System Prompt    │  ← 角色定义 + 输出格式 + 上下文信息
│  - Few-Shot 示例    │  ← 2 个告警规则示例
│  - User Prompt      │  ← 用户描述 + 数据源信息 + 规则类型
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  5. 调用 LLM       │  ← JSON 模式输出
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  6. 后处理          │
│  - 标准化 severity  │  ← 非标准值自动修正为 warning
│  - 默认 for_duration│  ← 空值默认 "0s"
│  - 初始化 labels    │  ← 确保 map 非 nil
│  - 钳制 confidence  │  ← 限制在 [0, 1] 范围
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  7. 缓存结果        │  ← 10 分钟 TTL
└───────┬───────────┘
        │
        ▼
    返回结果给用户
```

---

## 架构设计

### 核心组件

```
RuleGeneratorService
├── AIService              — LLM 调用封装
├── LabelRegistryService   — 标签注册表查询
├── DataSourceService      — 数据源查询（表达式验证）
├── AlertRuleService       — 已有规则查询
├── PresetRuleRepository   — 预置规则搜索
├── DataSourceRepository   — 数据源信息查询
└── RuleGenCache           — 结果缓存
```

### 文件结构

| 文件 | 职责 |
|------|------|
| `internal/service/rule_generator.go` | 核心生成逻辑、DryRun、Validate、SuggestLabels |
| `internal/service/rule_gen_prompts.go` | Few-shot prompt 模板（告警/抑制/静默） |
| `internal/service/rule_gen_cache.go` | 内存 TTL 缓存 |
| `internal/handler/ai_rule.go` | HTTP 端点处理 |

---

## Few-Shot Prompt 结构

### 告警规则 Few-Shot

```go
func fewShotAlertRule(labels []string) string {
    // 根据可用标签动态生成提示
    return `
## 示例

用户需求: "监控 Redis 内存使用率超过 80% 持续 5 分钟"
输出:
{
  "name": "RedisHighMemoryUsage",
  "expression": "redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"component": "redis", "team": "infra"},
  "annotations": {
    "summary": "Redis 内存使用率超过 80%",
    "description": "实例 {{ $labels.instance }} 内存使用率 {{ $value | printf \"%.1f\" }}%"
  }
}

用户需求: "CPU 使用率超过 90% 告警"
输出:
{
  "name": "HighCPUUsage",
  "expression": "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 90",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"component": "host"},
  "annotations": {
    "summary": "CPU 使用率超过 90%",
    "description": "实例 {{ $labels.instance }} CPU 使用率 {{ $value | printf \"%.1f\" }}%"
  }
}

可用标签（从数据源同步）: [instance, job, env, service, ...]
建议在 expression 和 labels 中使用这些标签。`
}
```

### 抑制规则 Few-Shot

```go
func fewShotInhibition() string {
    return `
## 示例

用户需求: "当节点宕机时，抑制该节点上的所有告警"
输出:
{
  "name": "NodeDownSuppressesNodeAlerts",
  "source_match": {"alertname": "NodeDown"},
  "target_match_re": {"instance": "{{ $labels.instance }}.*"},
  "equal": ["instance"],
  "description": "节点宕机时抑制同实例的其他告警"
}`
}
```

### 静默规则 Few-Shot

```go
func fewShotMute() string {
    return `
## 示例

用户需求: "每天凌晨 2-4 点静默所有告警，做维护窗口"
输出:
{
  "name": "DailyMaintenanceWindow",
  "matchers": [],
  "time_periods": [{"start": "02:00", "end": "04:00", "weekdays": ["mon","tue","wed","thu","fri","sat","sun"]}],
  "description": "每日凌晨维护窗口"
}`
}
```

### System Prompt 模板

完整的 System Prompt 由三部分拼接而成：

1. **角色定义 + 输出格式**：

```
你是 SRE 告警规则生成助手。根据用户的自然语言描述，生成标准的告警规则或抑制规则。

{标签上下文}

{已有规则上下文}

{预置规则匹配}

输出格式要求（严格 JSON）：
对于告警规则：
{
  "type": "alert",
  "name": "AlertName",
  "expression": "PromQL表达式",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"service": "xxx", "env": "prod", "component": "xxx"},
  "annotations": {"summary": "中文摘要", "description": "详细描述"},
  "confidence": 0.9,
  "description": "规则说明"
}

注意：
- severity 必须是 critical/warning/info 之一
- PromQL 必须使用真实存在的指标名
- labels 必须包含 service, env, component
- for_duration 使用 Go duration 格式（如 1m, 5m, 10m）
- 如果信息不足，在 warnings 中列出需要确认的事项
- 回复中只包含 JSON，不要添加其他文本
```

2. **Few-Shot 示例**（通过 `fewShotAlertRule()` 追加）

3. **User Prompt**（用户描述 + 数据源信息）

---

## 缓存策略

### 缓存键生成

```go
func cacheKey(description string, dsID *uint, ruleType string) string {
    h := sha256.New()
    h.Write([]byte(description))
    if dsID != nil {
        h.Write([]byte(fmt.Sprintf(":%d", *dsID)))
    }
    h.Write([]byte(":" + ruleType))
    return hex.EncodeToString(h.Sum(nil))
}
```

缓存键 = SHA256(description + ":" + datasource_id + ":" + ruleType)

### 缓存配置

| 参数 | 值 | 说明 |
|------|---|------|
| TTL | 10 分钟 | 默认值，可在 `NewRuleGenCache` 中调整 |
| 清理间隔 | 5 分钟 | 后台 goroutine 定期清理过期条目 |
| 存储位置 | 内存（`sync.Map`） | 进程重启后缓存丢失 |

### 缓存命中条件

三个维度必须完全匹配：
1. `description` — 用户输入的自然语言描述（完全一致）
2. `datasource_id` — 目标数据源 ID（nil vs 非 nil）
3. `rule_type` — 规则类型（"alert" / "inhibition"）

**注意**：微小的描述差异（如多一个空格）会导致缓存未命中。

---

## DryRun 模式

### 功能说明

DryRun 将规则生成和表达式验证合并为一次调用，让用户在保存前预览规则效果。

### 执行流程

```
1. 调用 Generate() 生成规则
2. 如果有 datasource_id 且 expression 非空：
   a. 向数据源发送 PromQL 查询
   b. 获取返回的时间序列数量和标签
3. 返回规则 + 验证结果
```

### 响应结构

```json
{
  "code": 0,
  "data": {
    "rule": {
      "type": "alert",
      "name": "HighCPUUsage",
      "expression": "100 - avg by(instance)(rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100 > 90",
      "for_duration": "5m",
      "severity": "warning",
      "labels": {"component": "host"},
      "annotations": {"summary": "CPU 使用率超过 90%"},
      "confidence": 0.85,
      "warnings": []
    },
    "validation": {
      "valid": true,
      "result_type": "vector",
      "sample_count": 15,
      "sample_labels": ["instance", "job", "mode"],
      "warnings": []
    }
  }
}
```

### Validation 结果说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `valid` | bool | 表达式语法是否正确且可执行 |
| `result_type` | string | 返回类型：vector / matrix / scalar |
| `sample_count` | int | 返回的时间序列数量 |
| `sample_labels` | string[] | 返回数据中出现的标签键列表 |
| `error` | string | 错误信息（valid=false 时） |
| `warnings` | string[] | 警告信息（如 sample_count=0） |

---

## 规则改进 (Improve)

### 功能说明

用户对 AI 生成的规则不满意时，可以提供自然语言反馈，AI 会根据反馈优化规则。

### 请求示例

```json
{
  "rule": {
    "type": "alert",
    "name": "HighCPUUsage",
    "expression": "100 - avg(rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100 > 90",
    "severity": "warning"
  },
  "feedback": "表达式应该按 instance 分组，阈值改为 85%，severity 改为 critical",
  "datasource_id": 1
}
```

### 改进流程

1. 将当前规则 JSON + 用户反馈发送给 LLM
2. LLM 根据反馈修改相关字段
3. 保持未提及的字段不变
4. 在 `warnings` 中说明做了哪些修改

---

## 标签建议

### SuggestLabels 功能

根据表达式中的指标名和标签注册表，自动推荐合适的标签值。

**输入**：`datasource_id` + `expression`

**输出**：

```json
{
  "code": 0,
  "data": {
    "detected_metrics": {
      "node_cpu_seconds_total": "node_cpu_seconds_total",
      "redis_memory_used_bytes": "redis_memory_used_bytes"
    },
    "suggested_labels": {
      "instance": {
        "value": "10.0.0.1:9100",
        "confidence": 0.8,
        "source": "label_registry"
      },
      "job": {
        "value": "node-exporter",
        "confidence": 0.8,
        "source": "label_registry"
      }
    },
    "available_instances": []
  }
}
```

### 指标名提取

系统使用正则 `[a-zA-Z_:][a-zA-Z0-9_:]*` 提取表达式中的标识符，然后过滤掉 PromQL 关键字（如 `sum`, `avg`, `rate` 等），只保留包含下划线或全小写的标识符作为指标名。

---

## 表达式验证

### ValidateExpression 功能

独立验证 PromQL 表达式的正确性，不依赖规则生成。

```go
func (s *RuleGeneratorService) ValidateExpression(
    ctx context.Context,
    datasourceID uint,
    expression string,
) (*ValidationResult, error) {
    resp, err := s.dsSvc.QueryDatasource(ctx, datasourceID, expression, time.Now())
    if err != nil {
        return &ValidationResult{Valid: false, Error: err.Error()}, nil
    }
    // ... 提取 result_type, sample_count, sample_labels
}
```

**注意**：验证使用当前时间作为查询时间点，返回的是瞬时向量（instant vector）。

---

## API 端点

### POST /ai/rules/generate

从自然语言生成告警规则。

**请求**：

```json
{
  "description": "监控 Redis 内存使用率超过 80% 持续 5 分钟",
  "datasource_id": 1,
  "rule_type": "alert",
  "context": {
    "existing_rules": true,
    "include_labels": true,
    "include_routing": false
  }
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "type": "alert",
    "name": "RedisHighMemoryUsage",
    "expression": "redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80",
    "for_duration": "5m",
    "severity": "warning",
    "labels": {"component": "redis", "team": "infra"},
    "annotations": {
      "summary": "Redis 内存使用率超过 80%",
      "description": "实例 {{ $labels.instance }} 内存使用率 {{ $value | printf \"%.1f\" }}%"
    },
    "confidence": 0.9,
    "warnings": []
  }
}
```

### POST /ai/rules/dry-run

生成并验证规则。

**请求**：同 `/ai/rules/generate`

**响应**：

```json
{
  "code": 0,
  "data": {
    "rule": { ... },
    "validation": {
      "valid": true,
      "result_type": "vector",
      "sample_count": 15,
      "sample_labels": ["instance", "job"]
    }
  }
}
```

### POST /ai/rules/validate

验证 PromQL 表达式。

**请求**：

```json
{
  "datasource_id": 1,
  "expression": "up == 0"
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "valid": true,
    "result_type": "vector",
    "sample_count": 3,
    "sample_labels": ["instance", "job"]
  }
}
```

### POST /ai/rules/suggest-labels

为表达式推荐标签。

**请求**：

```json
{
  "datasource_id": 1,
  "expression": "rate(http_requests_total[5m])"
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "detected_metrics": {"http_requests_total": "http_requests_total"},
    "suggested_labels": {
      "instance": {"value": "10.0.0.1:8080", "confidence": 0.8, "source": "label_registry"},
      "job": {"value": "api-server", "confidence": 0.8, "source": "label_registry"}
    }
  }
}
```

### POST /ai/rules/generate-inhibition

生成抑制规则。

**请求**：

```json
{
  "description": "当节点宕机时，抑制该节点上的所有告警",
  "datasource_id": 1
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "type": "inhibition",
    "name": "NodeDownSuppressesNodeAlerts",
    "source_labels": ["alertname"],
    "source_value": "NodeDown",
    "target_labels": ["instance"],
    "equal_labels": ["instance"],
    "confidence": 0.9,
    "warnings": []
  }
}
```

### POST /ai/rules/generate-mute

生成静默规则。

**请求**：

```json
{
  "description": "凌晨 2 点到 6 点静默 staging 环境的告警",
  "timezone": "Asia/Shanghai"
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "type": "mute",
    "name": "staging-凌晨维护静默",
    "description": "staging 环境每日凌晨维护窗口",
    "match_labels": {"env": "staging"},
    "severities": [],
    "periodic_start": "02:00",
    "periodic_end": "06:00",
    "days_of_week": [],
    "timezone": "Asia/Shanghai",
    "confidence": 0.95,
    "warnings": []
  }
}
```

### POST /ai/rules/improve

基于反馈改进已生成的规则。

**请求**：

```json
{
  "rule": {
    "type": "alert",
    "name": "HighCPUUsage",
    "expression": "100 - avg(rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100 > 90"
  },
  "feedback": "按 instance 分组，阈值改为 85%",
  "datasource_id": 1
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "type": "alert",
    "name": "HighCPUUsage",
    "expression": "100 - avg by(instance)(rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100 > 85",
    "severity": "warning",
    "confidence": 0.9,
    "warnings": ["已添加 instance 分组", "阈值从 90% 调整为 85%"]
  }
}
```

---

## 前端集成

### save_as_draft 工作流

典型的前端使用流程：

```
1. 用户输入描述 → 调用 POST /ai/rules/generate
2. 展示生成结果，用户可编辑
3. 用户点击"验证" → 调用 POST /ai/rules/validate
4. 展示验证结果（sample_count, warnings）
5. 用户点击"改进" → 调用 POST /ai/rules/improve
6. 用户确认 → 调用 POST /alert-rules（保存为草稿或启用）
```

### 错误处理

| 错误场景 | HTTP 状态 | 错误码 | 说明 |
|---------|----------|--------|------|
| AI 未启用 | 500 | 50003 | 需要在系统设置中配置 AI |
| 规则生成模块未启用 | 500 | 50003 | 需要在 AI 模块设置中开启 |
| LLM 调用失败 | 500 | 50003 | API key 无效或网络问题 |
| 表达式验证失败 | 200 | 0 | `validation.valid = false`，附带 error 字段 |

---

## 配置要求

### 前置条件

1. **AI 配置**：系统设置中配置 AI 服务（API Key、模型等）
2. **AI 模块**：启用 `RuleGen` 模块
3. **数据源**：至少有一个已启用的数据源（用于表达式验证和标签建议）
4. **标签注册表**：建议先同步数据源的标签（提升生成质量）

### 环境变量

无需额外环境变量，AI 配置存储在数据库的系统设置中。

### 性能考虑

- LLM 调用通常需要 2-5 秒
- 缓存命中时响应时间 < 10ms
- 表达式验证取决于数据源响应速度（通常 < 1s）
- 建议前端设置 30 秒超时
