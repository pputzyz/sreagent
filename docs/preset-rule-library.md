# 预置规则库

> **v4.13.0** | 45 条告警规则 + 16 条抑制模板

## 目录

- [概述](#概述)
- [数据模型](#数据模型)
- [内置规则分类](#内置规则分类)
- [抑制规则模板](#抑制规则模板)
- [从 YAML 导入](#从-yaml-导入)
- [Apply 流程](#apply-流程)
- [API 端点](#api-端点)
- [使用指南](#使用指南)

---

## 概述

预置规则（PresetRule）是告警规则的**模板**，来源于社区最佳实践、厂商推荐或从 Prometheus 规则 YAML 文件导入。用户可以从预置规则库中选择模板，快速创建实际的告警规则（AlertRule），而无需从零编写 PromQL 表达式。

**核心特性**：
- 内置 45 条告警规则，覆盖数据库、Kubernetes、中间件、节点、探测、Windows 等分类
- 内置 16 条抑制规则模板，减少告警风暴
- 支持从 monitoring-trading YAML 批量导入
- 应用时可覆盖数据源、通道、标签、严重等级
- 内置规则不可删除，用户自定义规则可自由管理

---

## 数据模型

### PresetRule 结构

```go
type PresetRule struct {
    BaseModel
    Name        string     `json:"name" gorm:"size:200;not null;index"`
    DisplayName string     `json:"display_name" gorm:"size:200"`
    Category    string     `json:"category" gorm:"size:50;index"`
    SubCategory string     `json:"sub_category" gorm:"size:50"`
    Component   string     `json:"component" gorm:"size:50"`
    Expression  string     `json:"expression" gorm:"type:text;not null"`
    ForDuration string     `json:"for_duration" gorm:"size:32"`
    Severity    string     `json:"severity" gorm:"size:20;index"`
    AlertType   string     `json:"alert_type" gorm:"size:50"`
    Labels      JSONLabels `json:"labels" gorm:"type:json"`
    Annotations JSONLabels `json:"annotations" gorm:"type:json"`
    Source      string     `json:"source" gorm:"size:100"`
    IsBuiltin   bool       `json:"is_builtin" gorm:"default:true"`
    UsageCount  int        `json:"usage_count" gorm:"default:0"`
    Description string     `json:"description" gorm:"type:text"`
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `Name` | string | 规则唯一标识（如 `RedisDown`） |
| `DisplayName` | string | 显示名称（如 "Redis Down"） |
| `Category` | string | 分类（database / kubernetes / middleware / node-exporter / probe / inhibition） |
| `SubCategory` | string | 子分类（通常与 Component 相同） |
| `Component` | string | 组件名（如 redis / mysql / kafka） |
| `Expression` | string | PromQL 表达式或抑制规则 JSON |
| `ForDuration` | string | 持续时间（如 "5m"） |
| `Severity` | string | 严重等级（critical / warning / info） |
| `AlertType` | string | 告警类型（threshold / heartbeat） |
| `Labels` | JSON | 附加标签（如 `{"category": "redis"}`） |
| `Annotations` | JSON | 注解（如 `{"summary": "...", "description": "..."}`） |
| `Source` | string | 来源（monitoring-trading / preset_inhibition / yaml_import） |
| `IsBuiltin` | bool | 是否内置（内置规则不可删除） |
| `UsageCount` | int | 被应用次数（用于排序热门规则） |
| `Description` | string | 规则描述 |

---

## 内置规则分类

### database — 数据库

数据库相关的告警规则，覆盖 MySQL、Redis、MongoDB、Elasticsearch、PostgreSQL 等。

**典型规则**：

| 规则名 | 严重等级 | 说明 |
|--------|---------|------|
| `MySQLDown` | critical | MySQL 实例不可用 |
| `MySQLHighConnections` | warning | MySQL 连接数过高 |
| `MySQLSlowQueries` | warning | MySQL 慢查询过多 |
| `RedisDown` | critical | Redis 实例不可用 |
| `RedisHighMemoryUsage` | warning | Redis 内存使用率过高 |
| `MongoDBDown` | critical | MongoDB 实例不可用 |
| `ElasticsearchClusterRed` | critical | ES 集群状态为 Red |
| `ElasticsearchClusterYellow` | warning | ES 集群状态为 Yellow |

### kubernetes — 容器编排

Kubernetes 集群和容器相关的告警规则。

**典型规则**：

| 规则名 | 严重等级 | 说明 |
|--------|---------|------|
| `KubeNodeNotReady` | critical | K8s 节点 NotReady |
| `KubePodCrashLooping` | critical | Pod 频繁重启 |
| `KubePodPending` | warning | Pod 长时间 Pending |
| `KubeContainerOOMKilled` | critical | 容器 OOM Killed |
| `KubeDeploymentReplicasMismatch` | warning | Deployment 副本数不匹配 |
| `KubePVUsageHigh` | warning | 持久卷使用率过高 |

### middleware — 中间件

中间件相关的告警规则，覆盖 Kafka、RabbitMQ、Nginx 等。

**典型规则**：

| 规则名 | 严重等级 | 说明 |
|--------|---------|------|
| `KafkaExporterDown` | critical | Kafka Exporter 不可用 |
| `KafkaConsumerLagHigh` | warning | Kafka 消费延迟过高 |
| `RabbitMQDown` | critical | RabbitMQ 不可用 |
| `RabbitMQQueueDepthHigh` | warning | RabbitMQ 队列积压 |
| `NginxHighErrorRate` | warning | Nginx 5xx 错误率过高 |

### node-exporter — 节点监控

主机节点相关的告警规则。

**典型规则**：

| 规则名 | 严重等级 | 说明 |
|--------|---------|------|
| `NodeExporterDown` | critical | Node Exporter 不可用 |
| `HighCPUUsage` | warning | CPU 使用率过高 |
| `HighMemoryUsage` | warning | 内存使用率过高 |
| `DiskSpaceRunningLow` | warning | 磁盘空间不足 |
| `DiskIONoise` | warning | 磁盘 I/O 过高 |
| `NetworkErrors` | warning | 网络错误率过高 |
| `HighSystemLoad` | warning | 系统负载过高 |

### probe — 黑盒探测

HTTP/TCP 探测相关的告警规则。

**典型规则**：

| 规则名 | 严重等级 | 说明 |
|--------|---------|------|
| `BlackboxHttpProbeFailed` | critical | HTTP 探测失败 |
| `BlackboxHttpProbeLatencyHigh` | warning | HTTP 探测延迟过高 |
| `BlackboxHttpStatus5xx` | warning | HTTP 探测返回 5xx |
| `BlackboxTcpProbeFailed` | critical | TCP 探测失败 |

### windows-exporter — Windows 监控

Windows 主机相关的告警规则。

**典型规则**：

| 规则名 | 严重等级 | 说明 |
|--------|---------|------|
| `WindowsExporterDown` | critical | Windows Exporter 不可用 |
| `WindowsHighCPUUsage` | warning | Windows CPU 使用率过高 |
| `WindowsHighMemoryUsage` | warning | Windows 内存使用率过高 |
| `WindowsDiskSpaceLow` | warning | Windows 磁盘空间不足 |

---

## 抑制规则模板

### 概述

抑制规则模板定义了告警之间的级联抑制关系，当高优先级告警触发时，自动抑制相关的低优先级告警，减少告警风暴。

### 严重等级级联

| 模板名 | 源告警 | 被抑制告警 | 等标签 |
|--------|--------|-----------|--------|
| `host-severity-p0-cascade` | severity=P0 | severity=P1/P2/P3 | biz_project, category, instance, project |
| `host-severity-p1-cascade` | severity=P1 | severity=P2/P3 | biz_project, category, instance, project |
| `container-severity-p0-cascade` | severity=P0, category=container | severity=P1/P2/P3, category=container | biz_project, namespace, pod, container, project |
| `container-severity-p1-cascade` | severity=P1, category=container | severity=P2/P3, category=container | biz_project, namespace, pod, container, project |

### 组件 Down 级联

| 模板名 | 源告警 | 被抑制告警 | 等标签 |
|--------|--------|-----------|--------|
| `node-exporter-down-cascade` | NodeExporterDown | 所有严重等级 | biz_project, instance, project |
| `kafka-down-cascade` | KafkaExporterDown | kafka 类告警 | biz_project, instance, project |
| `redis-down-cascade` | RedisDown | redis 类告警 | biz_project, instance, project |
| `mongodb-down-cascade` | MongoDBDown | mongodb 类告警 | biz_project, instance, project |
| `rabbitmq-down-cascade` | RabbitMQDown | rabbitmq 类告警 | biz_project, instance, project |

### K8s 节点级联

| 模板名 | 源告警 | 被抑制告警 | 等标签 |
|--------|--------|-----------|--------|
| `kube-node-notready-container` | KubeNodeNotReady | container 类告警 | biz_project, node, project |
| `kube-node-notready-pod` | KubeNodeNotReady | pod 类告警 | biz_project, node, project |

### 其他级联

| 模板名 | 源告警 | 被抑制告警 | 等标签 |
|--------|--------|-----------|--------|
| `es-cluster-red-cascade` | ElasticsearchClusterRed | ElasticsearchClusterYellow | biz_project, instance, project |
| `probe-failed-cascade` | BlackboxHttpProbeFailed | 延迟/状态码/DNS 告警 | biz_project, instance, project |

### 抑制规则 JSON 结构

```json
{
  "source_match": {"severity": "P0"},
  "target_match": {"severity": "~P1|P2|P3"},
  "equal_labels": ["biz_project", "category", "instance", "project"]
}
```

- `source_match`：源告警需要匹配的标签
- `target_match`：被抑制告警需要匹配的标签（支持正则前缀 `~`）
- `equal_labels`：源和目标告警需要相等的标签列表

---

## 从 YAML 导入

### ImportPresets 脚本

`scripts/import-presets/main.go` 从 monitoring-trading 项目的 YAML 文件批量导入预置规则。

**用法**：

```bash
# 预览模式（不写入数据库）
go run scripts/import-presets/main.go \
  --dir=/path/to/monitoring-trading/alerts \
  --dry-run

# 实际导入
go run scripts/import-presets/main.go \
  --dir=/path/to/monitoring-trading/alerts \
  --dsn="user:pass@tcp(127.0.0.1:3306)/sreagent?parseTime=true"
```

### 严重等级映射

| YAML 中的 severity | 映射到 |
|-------------------|--------|
| P0 | critical |
| P1 | warning |
| P2, P3 | info |
| （空） | warning |
| 其他 | 保持原值（小写） |

### 目录结构

脚本扫描以下分类目录：

```
monitoring-trading/alerts/
├── database/          → category: database
│   ├── mysql.yaml
│   ├── redis.yaml
│   └── ...
├── kubernetes/        → category: kubernetes
├── middleware/        → category: middleware
├── node-exporter/     → category: node-exporter
├── probe/             → category: probe
└── windows-exporter/  → category: windows-exporter
```

### 导入逻辑

1. 遍历每个分类目录下的所有 `.yaml` 文件
2. 解析 Prometheus 规则文件格式（`groups[].rules[]`）
3. 跳过 recording rules（只有 `alert` 字段非空的规则才会导入）
4. 从文件路径推导 category 和 component
5. 按规则名去重（已存在的规则跳过）
6. 批量插入数据库

### API 导入

也可以通过 API 导入自定义 YAML：

```http
POST /api/v1/preset-rules/import
Content-Type: application/x-yaml

groups:
  - name: my-rules
    rules:
      - alert: MyCustomAlert
        expr: up == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Instance down"
```

---

## Apply 流程

### 从预置规则创建告警规则

`PresetRuleService.Apply` 将预置规则转换为实际的 AlertRule：

```go
func (s *PresetRuleService) Apply(ctx context.Context, presetID uint, override *PresetRuleOverride) (*model.AlertRule, error) {
    // 1. 加载预置规则
    preset, err := s.repo.GetByID(ctx, presetID)

    // 2. 验证数据源（如果指定了）
    if override.DatasourceID > 0 {
        s.dsRepo.GetByID(ctx, override.DatasourceID)
    }

    // 3. 创建 AlertRule
    rule := &model.AlertRule{
        Name:        preset.Name,
        DisplayName: preset.DisplayName,
        Description: preset.Description,
        Expression:  preset.Expression,
        ForDuration: preset.ForDuration,
        Severity:    preset.Severity,
        Labels:      preset.Labels,
        Annotations: preset.Annotations,
        Category:    preset.Category,
    }

    // 4. 应用覆盖
    if override != nil {
        // 覆盖数据源、通道、严重等级、标签
    }

    // 5. 保存到数据库
    s.ruleRepo.Create(ctx, rule)

    // 6. 增加使用计数
    s.repo.IncrementUsage(ctx, presetID)

    return rule, nil
}
```

### 覆盖选项 (PresetRuleOverride)

```go
type PresetRuleOverride struct {
    DatasourceID uint              `json:"datasource_id"` // 绑定到特定数据源
    ChannelID    uint              `json:"channel_id"`    // 绑定到特定通道
    Labels       map[string]string `json:"labels"`        // 合并到预置规则的标签
    Severity     string            `json:"severity"`      // 覆盖严重等级
}
```

### 典型使用流程

```
1. 用户浏览预置规则库 → GET /preset-rules?category=database
2. 选择一条规则查看详情 → GET /preset-rules/:id
3. 点击"应用" → POST /preset-rules/:id/apply
   {
     "datasource_id": 1,
     "channel_id": 2,
     "severity": "critical",
     "labels": {"env": "prod", "team": "sre"}
   }
4. 系统创建 AlertRule 并返回
5. 评估引擎自动发现新规则并开始评估
```

---

## API 端点

### GET /preset-rules

分页查询预置规则列表。

**参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码（默认 1） |
| `page_size` | int | 每页数量（默认 20，最大 100） |
| `category` | string | 按分类过滤 |
| `search` | string | 搜索关键词（匹配名称和表达式） |

**响应**：

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "RedisDown",
        "display_name": "Redis Down",
        "category": "database",
        "component": "redis",
        "expression": "redis_up == 0",
        "for_duration": "1m",
        "severity": "critical",
        "is_builtin": true,
        "usage_count": 15
      }
    ],
    "total": 45,
    "page": 1,
    "page_size": 20
  }
}
```

### GET /preset-rules/:id

获取单条预置规则详情。

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "RedisDown",
    "display_name": "Redis Down",
    "category": "database",
    "sub_category": "redis",
    "component": "redis",
    "expression": "redis_up == 0",
    "for_duration": "1m",
    "severity": "critical",
    "alert_type": "threshold",
    "labels": {"category": "redis"},
    "annotations": {"summary": "Redis instance down", "description": "..."},
    "source": "monitoring-trading",
    "is_builtin": true,
    "usage_count": 15,
    "description": "Redis 实例不可用"
  }
}
```

### GET /preset-rules/categories

获取所有分类列表。

**响应**：

```json
{
  "code": 0,
  "data": ["database", "inhibition", "kubernetes", "middleware", "node-exporter", "probe", "windows-exporter"]
}
```

### POST /preset-rules/:id/apply

应用预置规则，创建实际的 AlertRule。

**请求**：

```json
{
  "datasource_id": 1,
  "channel_id": 2,
  "severity": "critical",
  "labels": {"env": "prod", "team": "sre"}
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 100,
    "name": "RedisDown",
    "expression": "redis_up == 0",
    "severity": "critical",
    "datasource_id": 1,
    "channel_id": 2,
    "status": "enabled",
    "version": 1
  }
}
```

### POST /preset-rules/import

从 YAML 导入预置规则。

**请求**：YAML 格式的 Prometheus 规则文件

**响应**：

```json
{
  "code": 0,
  "data": {
    "imported": 25
  }
}
```

### POST /preset-rules

创建自定义预置规则。

**请求**：

```json
{
  "name": "MyCustomAlert",
  "display_name": "自定义告警",
  "category": "custom",
  "expression": "my_metric > 100",
  "severity": "warning",
  "description": "自定义告警规则"
}
```

### DELETE /preset-rules/:id

删除预置规则（内置规则不可删除）。

**错误响应**（内置规则）：

```json
{
  "code": 10002,
  "message": "built-in preset rules cannot be deleted"
}
```

---

## 使用指南

### 最佳实践

1. **先浏览后应用**：在应用预置规则前，仔细查看表达式和标签是否符合实际环境
2. **指定数据源**：应用时务必指定 `datasource_id`，避免规则在错误的数据源上评估
3. **调整阈值**：预置规则的阈值是通用建议，应根据实际负载调整
4. **添加标签**：通过 `override.labels` 添加环境、团队等标签，便于通知路由
5. **使用分类过滤**：按分类浏览可以快速找到相关规则

### 自定义预置规则

用户可以创建自定义预置规则，作为团队内部的规则模板：

```http
POST /api/v1/preset-rules
{
  "name": "InternalAPILatency",
  "display_name": "内部 API 延迟过高",
  "category": "custom",
  "component": "api",
  "expression": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 1",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"team": "backend"},
  "annotations": {"summary": "内部 API P99 延迟超过 1 秒"},
  "description": "监控内部 API 的 P99 延迟"
}
```

### 使用计数

每次应用预置规则时，`usage_count` 会自动递增。可以按使用次数排序，找到最热门的规则模板：

```http
GET /api/v1/preset-rules?page_size=10&sort=usage_count
```
