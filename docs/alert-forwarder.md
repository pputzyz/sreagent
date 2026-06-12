# 告警转发器 (Alert Forwarder)

> v4.74.0 重构

## 概述

告警转发器是平台的告警上下游兼容层，支持三种核心场景：

1. **平台告警 → 外部**：出站转发，可选等级映射
2. **外部告警 → 平台核心**：入站集成模式，走完整生命周期
3. **外部告警 → 外部**：入站代理模式，纯转发，可选等级映射

## 入站模式

### 集成模式 (Integrate)

外部告警进入平台核心，走完整的告警管理流程：

```
外部告警 → 入站端点 → 认证 → 解析 → 等级映射(可选) → 保存到数据库
    → 抑制检查 → 静默检查 → 通知路由 → 升级策略
```

与平台自身告警引擎产生的告警享有相同的生命周期管理能力。

**适用场景**：
- 第三方监控系统（Zabbix、Nagios 等）告警接入
- 多云平台告警统一管理
- 需要复用平台通知、升级、静默能力

### 代理模式 (Proxy)

外部告警不进入平台核心，仅做等级映射后转发到外部目标：

```
外部告警 → 入站端点 → 认证 → 解析 → 等级映射(可选) → 转发到外部目标
```

**适用场景**：
- 告警网关/代理
- 等级标准转换（如 P0/P1/P2 → critical/warning/info）
- 多系统间告警转发

## 出站转发

平台自身产生的告警，在通知管道之外，还可以通过出站转发器发送到外部系统：

```
平台告警事件 → 出站转发器 → 等级映射(可选) → 发送到外部系统
```

**适用场景**：
- 将平台告警同步到外部工单系统
- 多平台告警聚合
- 与第三方 Incident Management 集成

## 等级映射

入站和出站等级映射是**独立配置**的：

- **入站等级映射**：将外部系统的等级映射到平台等级
  - 例：`critical → P0`、`warning → P2`
- **出站等级映射**：将平台等级映射到外部系统期望的格式
  - 例：`P0 → critical`、`P2 → warning`

```json
{
  "inbound_severity_mapping": {
    "enabled": true,
    "mapping": {
      "critical": "P0",
      "error": "P1",
      "warning": "P2",
      "info": "P3"
    },
    "default_severity": "P2"
  },
  "outbound_severity_mapping": {
    "enabled": true,
    "mapping": {
      "P0": "critical",
      "P1": "error",
      "P2": "warning",
      "P3": "info"
    }
  }
}
```

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

### 公开端点（无需认证，转发器自身认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/alert-forwarders/:id/inbound` | 入站端点 |

## 使用示例

### 1. 集成模式 - 接收 Alertmanager 告警进入平台

```json
{
  "name": "alertmanager-integrate",
  "direction": "inbound",
  "inbound_config": {
    "source_format": "alertmanager",
    "mode": "integrate",
    "auth_type": "bearer",
    "auth_config": { "token": "my-secret" }
  },
  "inbound_severity_mapping": {
    "enabled": true,
    "mapping": { "critical": "P0", "warning": "P2" }
  },
  "platform_capabilities": {
    "enable_notification": true,
    "enable_mute": true,
    "enable_inhibition": true
  }
}
```

### 2. 代理模式 - 转发到另一个外部系统

```json
{
  "name": "alertmanager-proxy",
  "direction": "inbound",
  "inbound_config": {
    "source_format": "alertmanager",
    "mode": "proxy",
    "auth_type": "bearer",
    "auth_config": { "token": "my-secret" },
    "proxy_target": {
      "target_url": "https://other-system.example.com/webhook",
      "method": "POST",
      "headers": { "Authorization": "Bearer other-token" }
    }
  },
  "inbound_severity_mapping": {
    "enabled": true,
    "mapping": { "critical": "P0", "warning": "P2" }
  }
}
```

### 3. 出站转发 - 平台告警同步到外部系统

```json
{
  "name": "outbound-webhook",
  "direction": "outbound",
  "outbound_config": {
    "target_url": "https://ticket-system.example.com/api/alerts",
    "method": "POST",
    "headers": { "Authorization": "Bearer token" }
  },
  "outbound_severity_mapping": {
    "enabled": true,
    "mapping": { "P0": "critical", "P1": "error", "P2": "warning" }
  }
}
```

## 数据模型

```go
type AlertForwarder struct {
    ID                     uint
    Name                   string
    Description            string
    Enabled                bool
    Direction              ForwarderDirection  // inbound, outbound, bidirectional
    Priority               int
    InboundConfig          *InboundConfig
    OutboundConfig         *OutboundConfig
    InboundSeverityMapping  *SeverityMappingConfig
    OutboundSeverityMapping *SeverityMappingConfig
    PlatformCapabilities   *PlatformCapabilitiesConfig  // integrate mode only
    MatchLabels            map[string]string
}

type InboundConfig struct {
    SourceFormat ForwarderSourceFormat  // alertmanager, grafana, prometheus, generic
    Mode         InboundMode            // integrate, proxy
    AuthType     ForwarderAuthType      // none, bearer, basic, hmac
    AuthConfig   *AuthConfig
    ProxyTarget  *OutboundConfig        // proxy mode only
}
```
