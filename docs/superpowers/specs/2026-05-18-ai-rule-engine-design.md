# AI 智能规则引擎设计文档

> 2026-05-18 | SREAgent v4.10.18+

## 背景

传统告警平台（monitoring-trading）有 315 条 Prometheus/VM 规则、13 条抑制规则、丰富的飞书卡片模板，但创建规则需要手动编写 YAML + PromQL。SREAgent 已有 AI 能力（Chat/分析/SOP）和完整的规则 CRUD，但缺少：

1. **AI 规则生成**：口述需求 → 自动生成规则
2. **预置规则库**：315 条行业标准规则
3. **传统平台兼容**：标签体系、路由、飞书卡片格式适配
4. **多数据源上下文**：label_registry 跨数据源感知

## 设计目标

- 用户口述"监控 Redis 内存超过 80%" → AI 生成完整告警规则（含 PromQL、标签、路由）
- 表单编辑时 AI 实时补全表达式、推荐阈值
- 315 条预置规则按组件分类，一键应用
- 传统平台的飞书卡片格式可导入/导出
- 多数据源场景下 AI 自动感知 label 上下文

---

## 架构总览

```
┌─────────────────────────────────────────────────┐
│                    Frontend                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ AI Chat  │  │ RuleForm │  │ PresetLibrary │  │
│  │ (对话式)  │  │ (表单+AI)│  │ (预置规则库)  │  │
│  └────┬─────┘  └────┬─────┘  └──────┬───────┘  │
│       │              │               │           │
│       └──────────────┼───────────────┘           │
│                      ▼                           │
│              /api/v1/ai/rules/*                   │
└──────────────────────┬───────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────┐
│              AI Rule Engine (Backend)             │
│                                                  │
│  ┌─────────────────┐  ┌────────────────────┐    │
│  │ RuleGeneratorSvc │  │ PresetRuleSvc      │    │
│  │ (LLM 规则生成)    │  │ (预置规则管理)      │    │
│  └────────┬────────┘  └────────┬───────────┘    │
│           │                     │                │
│  ┌────────▼────────────────────▼───────────┐    │
│  │         Context Builder                  │    │
│  │  - LabelRegistry (跨数据源)              │    │
│  │  - Datasource metrics                    │    │
│  │  - Existing rules (避免重复)             │    │
│  │  - Traditional rules (知识库)            │    │
│  └──────────────────────────────────────────┘    │
└──────────────────────────────────────────────────┘
```

---

## 模块 1：AI 规则生成引擎

### 1.1 对话式创建（Chat Mode）

在现有 AI Chat 的 `mode` 中新增 `rule_creation`：

**System Prompt 核心约束：**
```
你是 SRE 告警规则生成助手。用户会用自然语言描述告警需求，你需要：
1. 理解用户意图（监控什么组件、什么指标、什么阈值）
2. 查询 label_registry 获取可用标签和值
3. 生成标准 Prometheus/VM 告警规则
4. 输出结构化 JSON，包含：expression, for_duration, severity, labels, annotations

必须遵守：
- 生成的 PromQL 必须基于真实存在的 metric name（从数据源查询确认）
- 标签必须使用 label_registry 中存在的 key/value
- severity 必须是 critical/warning/info 之一
- 必须包含 runbook_url（如果用户提供了的话）
- 如果信息不足，必须追问（数据源、环境、阈值单位等）
```

**多轮对话流程：**
```
用户: "监控 Redis 内存使用率超过 80% 告警"
AI: "我需要确认几个信息：
     1. 数据源是哪个？（检测到你有 2 个 Prometheus 数据源）
     2. 环境是 prod 还是全部？
     3. 80% 持续多久触发？（建议 5 分钟）
     4. 严重等级？（建议 warning）"
用户: "prod 环境，5 分钟，warning"
AI: 生成规则 JSON + PromQL 预览 + 标签建议
用户: "确认创建"
AI: 调用 AlertRuleService.Create 创建规则
```

### 1.2 表单 + AI 辅助（Form Assist）

在现有 RuleFormModal 中增加 AI 辅助能力：

**触发点：**
- 表达式输入框旁的"AI 生成"按钮
- 用户输入自然语言描述后，AI 转换为 PromQL
- 标签区域的"AI 推荐"按钮，基于 label_registry 推荐标签

**API 设计：**

```
POST /api/v1/ai/rules/generate
Body: {
  "description": "监控 Redis 内存使用率超过 80%",
  "datasource_id": 1,           // 可选，AI 可自动选择
  "context": {
    "existing_rules": true,     // 是否参考已有规则避免重复
    "include_labels": true,     // 是否推荐标签
    "include_routing": true,    // 是否推荐路由
    "target_channel_id": null   // 可选，指定目标通道
  }
}
Response: {
  "expression": "redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {
    "service": "redis",
    "env": "prod",
    "component": "cache",
    "category": "memory",
    "alert_type": "threshold"
  },
  "annotations": {
    "summary": "Redis 内存使用率超过 80%",
    "description": "实例 {{ $labels.instance }} 内存使用率 {{ $value }}%",
    "runbook_url": ""
  },
  "suggested_channel": {
    "id": 3,
    "name": "Redis 告警通道",
    "reason": "匹配 labels.service=redis"
  },
  "confidence": 0.92,
  "warnings": []
}
```

```
POST /api/v1/ai/rules/validate
Body: {
  "expression": "redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80",
  "datasource_id": 1
}
Response: {
  "valid": true,
  "result_type": "vector",
  "sample_count": 12,
  "sample_labels": ["instance-1:6379", "instance-2:6379"],
  "warnings": []
}
```

```
POST /api/v1/ai/rules/suggest-labels
Body: {
  "expression": "redis_memory_used_bytes / ...",
  "datasource_id": 1
}
Response: {
  "detected_metrics": ["redis_memory_used_bytes", "redis_memory_max_bytes"],
  "suggested_labels": {
    "service": { "value": "redis", "confidence": 0.95, "source": "metric_name" },
    "env": { "value": "prod", "confidence": 0.8, "source": "label_registry" },
    "component": { "value": "cache", "confidence": 0.9, "source": "metric_pattern" }
  },
  "available_instances": [
    { "labels": {"instance": "10.1.1.1:6379", "job": "redis"}, "value": 72.5 }
  ]
}
```

### 1.3 后端实现

**新增文件：**

| 文件 | 职责 |
|------|------|
| `service/rule_generator.go` | AI 规则生成核心逻辑 |
| `handler/ai_rule.go` | AI 规则生成 API handler |
| `model/preset_rule.go` | 预置规则模型 |

**RuleGeneratorService 核心结构：**

```go
type RuleGeneratorService struct {
    aiSvc        *AIService
    labelRegSvc  *LabelRegistryService
    dsSvc        *DataSourceService
    ruleSvc      *AlertRuleService
    presetRepo   *PresetRuleRepository
}

type RuleGenerateRequest struct {
    Description string           `json:"description"`
    DatasourceID *uint           `json:"datasource_id"`
    Context     GenerateContext  `json:"context"`
}

type GenerateContext struct {
    ExistingRules  bool `json:"existing_rules"`
    IncludeLabels  bool `json:"include_labels"`
    IncludeRouting bool `json:"include_routing"`
    TargetChannelID *uint `json:"target_channel_id"`
}

type RuleGenerateResult struct {
    Expression     string            `json:"expression"`
    ForDuration    string            `json:"for_duration"`
    Severity       string            `json:"severity"`
    Labels         map[string]string `json:"labels"`
    Annotations    map[string]string `json:"annotations"`
    SuggestedChannel *ChannelSuggestion `json:"suggested_channel"`
    Confidence     float64           `json:"confidence"`
    Warnings       []string          `json:"warnings"`
}
```

**Context Builder 逻辑：**
1. 从 `LabelRegistry` 获取目标数据源的所有 label keys/values
2. 从 `DataSourceService` 执行 `GET /api/v1/labels` 获取实时指标列表
3. 从 `AlertRuleService.List` 检查是否已有类似规则（避免重复）
4. 从 `PresetRuleService` 搜索匹配的预置规则作为参考
5. 组装成结构化 context 注入 LLM system prompt

---

## 模块 2：预置规则库

### 2.1 数据模型

```go
type PresetRule struct {
    ID           uint   `gorm:"primaryKey"`
    Name         string `gorm:"size:200;not null"`      // "NodeCpuCriticallyHigh"
    DisplayName  string `gorm:"size:200"`                // "节点 CPU 使用率极高"
    Category     string `gorm:"size:50;index"`           // "node-exporter", "kubernetes", "middleware"
    SubCategory  string `gorm:"size:50"`                 // "cpu", "memory", "disk"
    Component    string `gorm:"size:50"`                 // "node", "redis", "kafka"
    Expression   string `gorm:"type:text;not null"`      // PromQL
    ForDuration  string `gorm:"size:20"`                 // "5m"
    Severity     string `gorm:"size:20"`                 // "critical", "warning", "info"
    AlertType    string `gorm:"size:50"`                 // "threshold", "availability", etc.
    Labels       JSONLabels                               // 默认标签
    Annotations  JSONLabels                               // summary, description, runbook_url
    Source       string `gorm:"size:100"`                // "monitoring-trading", "awesome-prometheus-alerts"
    IsBuiltin    bool   `gorm:"default:true"`
    UsageCount   uint   `gorm:"default:0"`
    Description  string `gorm:"type:text"`               // 规则说明
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 2.2 分类体系

从 monitoring-trading 的目录结构映射：

| 目录 | Category | SubCategory | 规则数 |
|------|----------|-------------|--------|
| node-exporter/ | infrastructure | cpu/memory/disk/filesystem/network/system | ~60 |
| kubernetes/ | kubernetes | container/apiserver/coredns/etcd/scheduler/... | ~90 |
| middleware/ | middleware | kafka/etcd/nacos/rabbitmq/rocketmq | ~50 |
| database/ | database | redis/elasticsearch/mongodb/clickhouse | ~40 |
| probe/ | probe | http/tcp | ~15 |
| windows-exporter/ | windows | cpu/disk/memory/network/system | ~60 |

### 2.3 导入工具

新增 `scripts/import-preset-rules.go`：

1. 读取 `monitoring-trading/alerts/` 下所有 YAML 文件
2. 解析 Prometheus rule groups → `PresetRule` 模型
3. 映射 severity: P0→critical, P1→warning, P2→info, P3→info
4. 提取 labels (severity/category/alert_type) 和 annotations
5. 批量写入 `preset_rules` 表
6. 支持增量更新（按 Name 去重）

### 2.4 API

```
GET  /api/v1/preset-rules                    # 列表（分页+搜索+分类筛选）
GET  /api/v1/preset-rules/:id                # 详情
POST /api/v1/preset-rules/:id/apply          # 从预置规则创建 AlertRule
POST /api/v1/preset-rules/import             # 批量导入 YAML
GET  /api/v1/preset-rules/categories         # 分类树
GET  /api/v1/preset-rules/search?q=redis     # 全文搜索
POST /api/v1/preset-rules/ai-suggest         # AI 推荐相关预置规则
```

---

## 模块 3：传统平台兼容

### 3.1 标签体系适配

SREAgent 现有标签：service, env, component, owner, severity, business_impact

传统平台标签：biz_project, tenant, maintainer, service, project, hostname, cluster

**兼容方案：** 在 label_registry 中扩展语义映射：

```go
type LabelMapping struct {
    ID           uint
    SourceLabel  string  // "biz_project"
    TargetLabel  string  // "business_impact" 或自定义
    DefaultValue string
}
```

前端 RuleFormModal 中：
- 自动检测数据源中的 label keys
- 如果检测到传统标签（biz_project/tenant），显示映射提示
- AI 生成规则时自动处理标签映射

### 3.2 路由规则导入

**Alertmanager config → SREAgent 映射：**

| Alertmanager | SREAgent |
|-------------|----------|
| route | Channel |
| route.routes[] | DispatchPolicy |
| receiver (webhook path) | Channel webhook URL |
| inhibit_rules | InhibitionRule |
| group_by | Channel.AggregationConfig |
| group_wait/interval | Channel.GroupWaitSeconds/GroupIntervalSeconds |

新增 `POST /api/v1/integrations/import-alertmanager` 端点：
1. 解析 alertmanager YAML
2. 为每个 receiver 创建 Channel
3. 为每个 route 创建 DispatchPolicy（match conditions）
4. 为每个 inhibit_rule 创建 InhibitionRule
5. 返回导入结果 + 冲突报告

### 3.3 飞书卡片模板兼容

**传统格式 vs SREAgent 格式对比：**

| 元素 | 传统 feishu-card.tmpl | SREAgent BuildEnrichedAlertCard |
|------|----------------------|--------------------------------|
| Header | alertname + severity 颜色 | alertname + severity 颜色 ✅ 一致 |
| 标签展示 | emoji + 中文标签名 | 结构化字段 | 
| 操作按钮 | 操作手册 + 告警中心 | 查看详情 + 确认 + 静默 |
| AI 分析 | 无 | summary + causes + steps |
| @提及 | OpenID mentions | 已有 |

**兼容方案：** 新增 `MessageTemplate.Type = "lark_card_v2"` 类型，支持传统格式的 Go template 语法。用户可以在模板库中选择：
- **SREAgent 增强版**：含 AI 分析 + 操作按钮
- **传统兼容版**：emoji 标签 + 操作手册按钮 + @提及
- **自定义**：用户自己编写

---

## 模块 4：AI 抑制规则生成

### 4.1 抑制规则 AI 能力

在 AI Chat 的 `rule_creation` 模式中，支持抑制规则生成：

**口述示例：**
- "当 NodeExporterDown 时，抑制同实例的所有告警"
- "P0 告警触发时，抑制同组件的 P1/P2/P3"
- "KafkaExporterDown 抑制所有 kafka 类别的告警"

**AI 输出结构：**
```json
{
  "type": "inhibition",
  "name": "NodeExporterDown 抑制规则",
  "source_labels": ["alertname"],
  "source_value": "NodeExporterDown",
  "target_labels": ["biz_project", "instance", "project"],
  "equal_labels": ["biz_project", "instance", "project"],
  "description": "当 NodeExporter 挂掉时，抑制同实例的所有告警（避免告警风暴）"
}
```

**API：**
```
POST /api/v1/ai/rules/generate-inhibition
Body: {
  "description": "当 NodeExporterDown 时抑制同实例所有告警",
  "datasource_id": 1
}
```

### 4.2 预置抑制规则模板

从 monitoring-trading 的 13 条 inhibit_rules 导入为预置模板：

| 模板名 | 类型 | 说明 |
|--------|------|------|
| Host Severity Cascade | 级联 | P0 抑制同实例 P1/P2/P3 |
| Container Severity Cascade | 级联 | P0 抑制同 Pod P1/P2/P3 |
| Exporter Down Suppression | 级联 | Exporter 挂掉时抑制该组件所有告警 |
| KubeNodeNotReady Cascade | 级联 | 节点 NotReady 抑制该节点所有 Pod 告警 |
| Cluster State Cascade | 级联 | ES Cluster Red 抑制 Cluster Yellow |
| Probe Failure Cascade | 级联 | HTTP 探测失败抑制延迟/状态/DNS 告警 |

---

## 模块 5：AI 模块配置系统

### 5.1 AI 能力分类

将平台所有 AI 能力按类别分组，每类独立启停：

| 类别 | 配置键 | 功能 | 默认 |
|------|--------|------|------|
| **平台辅助** | `ai.platform` | 配置优化建议、标签推荐、路由建议 | 关闭 |
| **AI 对话** | `ai.chat` | 通用 SRE 问答、SOP 建议 | 关闭 |
| **规则生成** | `ai.rule_gen` | 口述生成告警/抑制规则、PromQL 生成 | 关闭 |
| **告警分析** | `ai.analysis` | 告警根因分析、AI 摘要注入通知 | 关闭 |
| **Agent** | `ai.agent` | 自动诊断工作流、自动修复（Phase 2-3） | 关闭 |

### 5.2 数据模型

扩展现有 `SystemSetting`（group="ai"）：

```go
// 现有 AIConfig 保持不变，新增 AI 模块配置
type AIModuleConfig struct {
    Platform  AIModule `json:"platform"`   // 平台辅助
    Chat      AIModule `json:"chat"`       // AI 对话
    RuleGen   AIModule `json:"rule_gen"`   // 规则生成
    Analysis  AIModule `json:"analysis"`   // 告警分析
    Agent     AIModule `json:"agent"`      // Agent
}

type AIModule struct {
    Enabled     bool   `json:"enabled"`
    Description string `json:"description"`
}
```

存储方式：`SystemSetting{Group: "ai", Key: "modules", Value: JSON(AIModuleConfig)}`

### 5.3 API

```
GET  /api/v1/ai/modules          # 获取所有 AI 模块状态
PUT  /api/v1/ai/modules          # 更新 AI 模块配置（admin only）
GET  /api/v1/ai/modules/status   # 前端用：返回各模块启用状态 + AI 总开关
```

### 5.4 前端 AI 开关联动

**AI 设置页面（admin）：** `/platform/ai-settings`

```
┌─────────────────────────────────────────────────┐
│ AI 能力配置                                      │
│                                                  │
│ ┌─ AI 总开关 ────────────────────────────────┐  │
│ │ AI 引擎  [○ 关闭]  API: OpenAI  Model: GPT │  │
│ └────────────────────────────────────────────┘  │
│                                                  │
│ ┌─ 模块开关 ────────────────────────────────┐  │
│ │                                            │  │
│ │ 平台辅助    [○ 关闭]  配置优化、标签推荐     │  │
│ │ AI 对话     [○ 关闭]  SRE 问答、SOP 建议    │  │
│ │ 规则生成    [○ 关闭]  口述生成规则           │  │
│ │ 告警分析    [○ 关闭]  根因分析、AI 摘要      │  │
│ │ Agent       [○ 关闭]  自动诊断、自动修复     │  │
│ │                                            │  │
│ └────────────────────────────────────────────┘  │
│                                                  │
│ ┌─ 连接测试 ────────────────────────────────┐  │
│ │ [测试连接]  状态: ✅ 连接正常 (延迟 230ms)  │  │
│ └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

---

## 模块 6：前端 AI 工况设计

### 6.1 三种工况

| 工况 | 条件 | UI 表现 |
|------|------|---------|
| **AI 完全关闭** | `AIConfig.Enabled = false` | 所有 AI 按钮隐藏，纯手工模式 |
| **AI 部分开启** | `Enabled = true` 但某模块关闭 | AI 按钮显示但灰色禁用，hover 提示"该功能未启用" |
| **AI 全部开启** | `Enabled = true` 且模块开启 | AI 按钮正常可用，功能完整 |

### 6.2 各页面 AI 能力映射

| 页面 | AI 能力 | 依赖模块 | 关闭时表现 |
|------|---------|---------|-----------|
| RuleFormModal | 表达式生成、标签推荐、验证 | `ai.rule_gen` | 隐藏 ✨ 按钮 |
| AI Chat | 对话式规则创建、SOP 建议 | `ai.chat` + `ai.rule_gen` | Chat 面板隐藏"规则创建"标签 |
| PresetRules | AI 推荐相关规则 | `ai.rule_gen` | 隐藏"AI 推荐"按钮 |
| InhibitionForm | 抑制规则口述生成 | `ai.rule_gen` | 隐藏 AI 入口 |
| AlertEvents/Detail | 根因分析、AI 摘要 | `ai.analysis` | 隐藏"AI 分析"按钮 |
| NotifyRuleForm | AI 摘要注入 pipeline | `ai.analysis` | 隐藏 "ai_summary" pipeline 步骤 |
| Settings/AI | AI 配置页面 | 全局 | 始终可见 |
| Dashboard | 配置优化建议 | `ai.platform` | 隐藏"AI 优化建议" |

### 6.3 前端实现

新增 composable `useAIModule`：

```typescript
export function useAIModule() {
  const modules = ref<AIModuleConfig>({})

  async function loadModules() {
    const res = await aiApi.getModules()
    modules.value = res.data
  }

  function isEnabled(module: 'platform' | 'chat' | 'rule_gen' | 'analysis' | 'agent') {
    return modules.value[module]?.enabled ?? false
  }

  function isAIAvailable() {
    return modules.value._global_enabled ?? false
  }

  return { modules, loadModules, isEnabled, isAIAvailable }
}
```

各组件中使用：
```vue
<script setup>
const { isEnabled } = useAIModule()
const showAIGenerate = computed(() => isEnabled('rule_gen'))
</script>

<template>
  <NButton v-if="showAIGenerate" @click="openAIGenerate">
    ✨ AI 生成
  </NButton>
</template>
```

---

## 模块 7：前端设计

### 7.1 预置规则库页面

### 4.1 预置规则库页面

**路由：** `/alert/presets`

```
┌─────────────────────────────────────────────────┐
│ 预置规则库                            [导入 YAML] │
│ ┌─────────────────────────────────────────────┐  │
│ │ 🔍 搜索规则... │ 全部 │ 基础设施 │ K8s │ 中间件 │  │
│ └─────────────────────────────────────────────┘  │
│                                                  │
│ ┌── 基础设施 / CPU ────────────────────────────┐  │
│ │ ☐ NodeCpuCriticallyHigh    P0  [应用] [预览]  │  │
│ │ ☐ NodeCpuHigh              P1  [应用] [预览]  │  │
│ │ ☐ NodeCpuIowaitHigh        P2  [应用] [预览]  │  │
│ └──────────────────────────────────────────────┘  │
│ ┌── Kubernetes / Container ────────────────────┐  │
│ │ ☐ ContainerMemoryCritical  P0  [应用] [预览]  │  │
│ │ ☐ ContainerHighCpu         P1  [应用] [预览]  │  │
│ └──────────────────────────────────────────────┘  │
│                                                  │
│ [批量应用] [AI 推荐] [全选]                       │
└──────────────────────────────────────────────────┘
```

### 4.2 AI 规则创建入口

**对话式入口：** 在 AI Chat 面板中新增"规则创建"模式标签

**表单辅助入口：** 在 RuleFormModal 中：
- 表达式输入框旁增加 ✨ 按钮 → 弹出自然语言输入
- 标签区域增加"AI 推荐"按钮
- 保存前增加"AI 验证"按钮（验证 PromQL 语法 + 数据源中是否存在该指标）

### 4.3 路由规则导入

在 Integrations 页面增加"导入 Alertmanager 配置"按钮，支持粘贴 YAML 或上传文件。

---

## 实施计划

### Phase A — 预置规则库 + AI 配置系统（本次实现）
1. `model/preset_rule.go` + 迁移（preset_rules 表）
2. `scripts/import-preset-rules.go` — 从 monitoring-trading 导入 315 条规则 + 13 条抑制模板
3. `service/preset_rule.go` + `handler/preset_rule.go`
4. AI 模块配置系统（AIModuleConfig + API + useAIModule composable）
5. 前端 `/alert/presets` 预置规则库页面
6. 前端 `/platform/ai-settings` AI 配置页面
7. `POST /:id/apply` 从预置规则创建告警规则

### Phase B — AI 规则生成引擎（接续实现）
1. `service/rule_generator.go` — Context Builder + LLM prompt
2. `handler/ai_rule.go` — generate/validate/suggest-labels/generate-inhibition API
3. 前端 AI Chat "规则创建"模式（告警规则 + 抑制规则）
4. 前端 RuleFormModal AI 辅助按钮（✨ 生成 + AI 推荐 + AI 验证）
5. 前端 InhibitionForm AI 辅助
6. 实时 PromQL 验证（调用数据源查询确认指标存在）
7. useAIModule composable 集成到所有相关页面

### Phase C — 传统平台兼容（接续实现）
1. Alertmanager config 导入端点
2. 飞书卡片模板库（传统兼容版 + SREAgent 增强版）
3. 标签映射配置
4. 抑制规则预置模板

### Phase D — 多数据源增强（接续实现）
1. label_registry 扩展 `project` 维度
2. AI Context Builder 感知多数据源标签
3. 跨数据源 PromQL 生成（自动添加 project label filter）

---

## 技术约束

- **LLM Provider**: 复用现有 AIConfig（OpenAI/Azure/Ollama/Custom）
- **Prompt Token 限制**: Context Builder 需要裁剪 label_registry 到 Top-100 高频标签
- **PromQL 验证**: 通过数据源的 `/api/v1/query` 端点执行 dry-run
- **导入性能**: 315 条规则批量导入 < 5s
- **前端**: 复用现有 Naive UI 组件 + usePaginatedList composable
