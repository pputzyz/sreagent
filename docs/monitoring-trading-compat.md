# monitoring-trading 兼容方案

> 最后更新: 2026-05-19

## 概述

SREAgent 兼容传统 monitoring-trading 平台的告警规则体系，支持：
- 299 条 Prometheus/VMAlert 告警规则一键导入为预置规则
- 16 条 Alertmanager 抑制规则内置模板
- 多租户标签联想（biz_project/tenant/project/maintainer）
- AI 辅助规则生成（基于预置规则模板库）

## 1. 告警规则导入

### 数据来源

monitoring-trading 项目包含 299 条告警规则，分布在 7 大类 31 个 YAML 文件中：

| 类别 | 文件数 | 规则数 | 子类 |
|------|--------|--------|------|
| database | 4 | 64 | clickhouse, elasticsearch, mongodb, redis |
| kubernetes | 9 | 81 | apiserver, container, coredns, etcd, scheduler, kube-state-metrics, kubelet, resource-reservation |
| middleware | 5 | 64 | etcd, kafka, nacos, rabbitmq, rocketmq |
| node-exporter | 6 | 51 | cpu, disk, filesystem, memory, network, system |
| probe | 2 | 11 | blackbox-http, blackbox-tcp |
| windows-exporter | 5 | 28 | cpu, disk, memory, network, system |

### 严重等级分布

| 等级 | 数量 | 映射 |
|------|------|------|
| P0 | 66 | critical |
| P1 | 123 | warning |
| P2 | 85 | info |
| P3 | 32 | info |

### 导入方式

#### 方式一：启动时自动 seed（内置 45 条常用规则）

SREAgent 启动时自动 seed 45 条内置预置规则，覆盖主机/容器/中间件/网络/数据库等常见场景。无需手动操作。

#### 方式二：全量导入脚本（299 条）

```bash
# Dry-run 预览
go run scripts/import-presets/main.go \
  --dir=/path/to/monitoring-trading/alerts \
  --dry-run

# 实际导入
go run scripts/import-presets/main.go \
  --dir=/path/to/monitoring-trading/alerts \
  --dsn="user:pass@tcp(127.0.0.1:3306)/sreagent?parseTime=true"
```

脚本自动完成：
- 递归扫描所有 YAML 文件
- 跳过 recording rules，只导入 alert rules
- 严重等级映射：P0→critical, P1→warning, P2/P3→info
- 提取 category、alert_type 等标签
- 按 name 去重，已存在的规则自动跳过

#### 方式三：前端 YAML 导入

在「预置规则库」页面点击「导入 YAML」，粘贴单个或多个 Prometheus 规则文件内容。

### 标签映射

monitoring-trading 的标签体系：

| 标签 | 用途 | 示例值 |
|------|------|--------|
| `biz_project` | 业务线 | ts, cc, mdc, cpp, metatradertools, product-center, infra |
| `tenant` | 租户/环境 | public-dfid-trading-system, cc-au-pri-v2, mdc-01pri |
| `project` | K8s 集群名 | public-monitor, business-public, win-k8s, flink |
| `maintainer` | 负责团队 | sre-t4 |
| `severity` | 严重等级 | P0, P1, P2, P3 |
| `category` | 组件类别 | redis, kafka, container, node, cpu, disk |
| `alert_type` | 功能类型 | threshold, availability, latency, saturation |
| `instance` | 实例地址 | host:port |
| `namespace` | K8s 命名空间 | default, kube-system |
| `pod` | Pod 名 | my-app-xxx |
| `container` | 容器名 | main, sidecar |
| `node` | K8s 节点 | k8s-node-01 |
| `cluster` | 中间件集群 | mdc-01pri-kafka |

## 2. 抑制规则

SREAgent 内置 16 条抑制规则预置模板，与 Alertmanager inhibit_rules 完全对齐：

### 严重等级级联（4 条）
- 主机 P0 抑制 P1/P2/P3（equal: biz_project+category+instance+project）
- 主机 P1 抑制 P2/P3
- 容器 P0 抑制 P1/P2/P3（equal: biz_project+namespace+pod+container+project）
- 容器 P1 抑制 P2/P3

### 主机/节点宕机级联（3 条）
- NodeExporterDown 抑制所有严重等级
- KubeNodeNotReady 抑制容器告警
- KubeNodeNotReady 抑制 Pod 告警

### 中间件/数据库宕机级联（7 条）
- KafkaExporterDown → 抑制 category=kafka
- RedisDown → 抑制 category=redis
- ElasticsearchClusterRed → 抑制 ElasticsearchClusterYellow
- MongoDBDown → 抑制 category=mongodb
- RabbitMQDown → 抑制 category=rabbitmq
- NacosDown → 抑制 category=nacos
- RocketMQExporterDown → 抑制 category=rocketmq

### 探测失败级联（2 条）
- BlackboxHttpProbeFailed → 抑制延迟/状态码/DNS 延迟告警
- BlackboxTcpProbeFailed → 抑制 TCP 延迟告警

所有抑制规则使用 `equal_labels` 包含 `biz_project`，防止跨业务线误抑制。

## 3. 多租户标签联想

### 工作原理

1. **数据源同步**：label_registry 每 10 分钟自动从 VictoriaMetrics 拉取所有标签 key/value
2. **被动记录**：告警事件触发时，从 labels 中提取 key/value 写入 registry（source=event）
3. **前端联想**：LabelMatcherEditor 组件在用户输入时调用 `/label-registry/keys` 和 `/label-registry/values` 获取候选值

### 联想范围

当用户在规则编辑器中输入标签时：
- 输入 `biz_project` → 联想出 ts, cc, mdc, cpp, metatradertools, product-center, infra
- 输入 `tenant` → 联想出 public-dfid-trading-system, cc-au-pri-v2, mdc-01pri 等
- 输入 `project` → 联想出 public-monitor, business-public, win-k8s, flink
- 输入 `category` → 联想出 redis, kafka, container, node, cpu, disk 等

### 跨数据源支持

label_registry 按 datasource_id 隔离。多个数据源可以有相同的标签 key 但不同的 value 集合。前端通过 `datasourceId` 参数过滤。

## 4. 通知路由兼容

### 传统平台路由结构

monitoring-trading 使用 Alertmanager 按 `biz_project` 路由到不同飞书群：

| biz_project | 飞书群 |
|-------------|--------|
| ts | feishu-ts |
| cc | feishu-cc |
| mdc | feishu-mdc |
| cpp | feishu-cpp-rps (按 tenant 细分) |
| metatradertools | feishu-metatradertools |
| product-center | feishu-product-center |
| infra | feishu-infra |

### SREAgent 映射方式

在 SREAgent 中，可以通过以下方式实现等效路由：
1. **告警通道（AlertChannel）**：每个 biz_project 创建一个告警通道
2. **匹配条件（match_labels）**：使用 `biz_project=ts` 等条件筛选
3. **通知规则（NotifyRule）**：组合多个通道，按严重等级/标签分发

## 5. AI 辅助规则生成

SREAgent 的 AI 规则生成功能可以基于预置规则模板库：
- 用户描述需求（如 "帮我创建 Redis 内存告警"）
- AI 从预置规则库中匹配最接近的模板
- 自动调整阈值、持续时间、严重等级
- 填充 biz_project/tenant 等多租户标签

## 6. 飞书卡片模板

monitoring-trading 的飞书卡片模板位于 `alerts/templates/feishu-card.tmpl`，支持：
- 颜色编码（红=P0, 橙=P1, 绿=resolved）
- 交互按钮（runbook 链接、告警中心链接）
- 丰富的上下文标签展示

SREAgent 的飞书集成支持等效的卡片格式，可通过消息模板（MessageTemplate）自定义。
