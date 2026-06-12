# 告警转发器 (Alert Forwarder)

> v4.72.0 新增

## 概述

告警转发器是一个独立的模块，用于在不同系统之间转发告警。它支持：

- **入站转发**：接收外部系统的告警（如 Alertmanager、Grafana、Prometheus），并接入平台能力
- **出站转发**：将平台告警转发到外部系统（通过通知媒体或直接 HTTP）
- **双向转发**：同时支持入站和出站

## 核心特性

### 1. 数据源格式支持

| 格式 | 说明 |
|------|------|
| `alertmanager` | Prometheus Alertmanager 标准格式 |
| `grafana` | Grafana 告警格式（兼容 Alertmanager） |
| `prometheus` | Prometheus Remote Write 格式 |
| `generic` | 通用 JSON 格式 |

### 2. 入站认证

| 类型 | 说明 |
|------|------|
| `none` | 无认证 |
| `bearer` | Bearer Token 认证 |
| `basic` | Basic Auth 认证 |
| `hmac` | HMAC 签名认证（支持 SHA-256、SHA-1） |

### 3. 等级映射

支持将外部系统的告警等级映射到平台等级：

```json
{
  "severity_mapping": {
    "enabled": true,
    "direction": "both",
    "mapping": {
      "critical": "P0",
      "error": "P1",
      "warning": "P2",
      "info": "P3"
    },
    "default_severity": "P2"
  }
}
```

- `direction`：映射方向
  - `inbound`：仅入站映射
  - `outbound`：仅出站映射
  - `both`：双向映射

### 4. 平台能力接入

转发器可以选择性接入以下平台能力：

| 能力 | 说明 |
|------|------|
| `enable_notification` | 接入通知管道，触发通知分发 |
| `enable_escalation` | 接入升级策略，自动升级 |
| `enable_mute` | 接入静默规则，免打扰 |
| `enable_inhibition` | 接入抑制规则，告警抑制 |
| `enable_ai_analysis` | 接入 AI 分析，智能分析 |
| `pipeline_id` | 接入自定义事件处理管道 |

### 5. 标签匹配

支持通过标签过滤告警：

```json
{
  "match_labels": {
    "env": "production",
    "team": "sre"
  }
}
```

只有匹配标签的告警才会被转发。

## API 端点

### 认证端点（需要 JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/alert-forwarders` | 创建转发器 |
| GET | `/api/v1/alert-forwarders` | 列表（分页、筛选） |
| GET | `/api/v1/alert-forwarders/:id` | 详情 |
| PUT | `/api/v1/alert-forwarders/:id` | 更新 |
| DELETE | `/api/v1/alert-forwarders/:id` | 删除 |
| POST | `/api/v1/alert-forwarders/:id/enable` | 启用 |
| POST | `/api/v1/alert-forwarders/:id/disable` | 禁用 |
| POST | `/api/v1/alert-forwarders/batch/enable` | 批量启用 |
| POST | `/api/v1/alert-forwarders/batch/disable` | 批量禁用 |
| POST | `/api/v1/alert-forwarders/batch/delete` | 批量删除 |
| POST | `/api/v1/alert-forwarders/:id/test` | 测试转发器 |
| GET | `/api/v1/alert-forwarders/stats` | 统计信息 |

### 公开端点（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/alert-forwarders/:id/inbound` | 入站端点 |

入站端点的认证由转发器自身的认证配置控制。

## 使用示例

### 1. 创建入站转发器

```bash
curl -X POST http://localhost:8080/api/v1/alert-forwarders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alertmanager-inbound",
    "description": "接收 Alertmanager 告警",
    "direction": "inbound",
    "enabled": true,
    "inbound_config": {
      "source_format": "alertmanager",
      "auth_type": "bearer",
      "auth_config": {
        "token": "my-secret-token"
      }
    },
    "severity_mapping": {
      "enabled": true,
      "direction": "inbound",
      "mapping": {
        "critical": "P0",
        "warning": "P2"
      }
    },
    "platform_capabilities": {
      "enable_notification": true,
      "enable_mute": true,
      "enable_inhibition": true
    }
  }'
```

### 2. 发送告警到入站端点

```bash
curl -X POST http://localhost:8080/api/v1/alert-forwarders/1/inbound \
  -H "Authorization: Bearer my-secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "alerts": [
      {
        "status": "firing",
        "labels": {
          "alertname": "HighCPU",
          "severity": "critical",
          "instance": "web-01"
        },
        "annotations": {
          "summary": "CPU usage > 90%"
        },
        "fingerprint": "abc123",
        "startsAt": "2026-06-12T10:00:00Z"
      }
    ]
  }'
```

### 3. 创建出站转发器

```bash
curl -X POST http://localhost:8080/api/v1/alert-forwarders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "webhook-outbound",
    "description": "转发到外部 Webhook",
    "direction": "outbound",
    "enabled": true,
    "outbound_config": {
      "target_url": "https://external-system.example.com/webhook",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer external-token"
      },
      "timeout": 30000,
      "retry_times": 3
    },
    "severity_mapping": {
      "enabled": true,
      "direction": "outbound",
      "mapping": {
        "P0": "critical",
        "P1": "error",
        "P2": "warning"
      }
    }
  }'
```

## 数据模型

```go
type AlertForwarder struct {
    ID                   uint
    Name                 string
    Description          string
    Enabled              bool
    Direction            ForwarderDirection  // inbound, outbound, bidirectional
    Priority             int
    InboundConfig        *InboundConfig
    OutboundConfig       *OutboundConfig
    SeverityMapping      *SeverityMappingConfig
    PlatformCapabilities *PlatformCapabilitiesConfig
    MatchLabels          map[string]string
}
```

## 处理流程

### 入站流程

1. 接收外部告警请求
2. 验证转发器是否启用
3. 验证认证（Bearer/Basic/HMAC）
4. 解析 payload（根据数据源格式）
5. 标签匹配（如果配置了 match_labels）
6. 等级映射（如果启用了 severity_mapping）
7. 创建 AlertEvent
8. 接入平台能力（根据 platform_capabilities 配置）

### 出站流程

1. 平台 AlertEvent 触发
2. 遍历所有启用的出站转发器
3. 标签匹配
4. 等级映射（如果启用了 severity_mapping）
5. 转发到目标（通过通知媒体或直接 HTTP）

## 与现有模块的关系

- **NotifyRule**：转发器是独立的抽象，专注于"转发"场景；NotifyRule 专注于"通知"场景
- **EventPipeline**：转发器可以配置 PipelineID 来接入自定义事件处理管线
- **NotifyMedia**：转发器的出站目标可以是现有的任何通知媒体
- **Integration**：转发器的入站端点是独立的，不复用现有的 webhook 处理
