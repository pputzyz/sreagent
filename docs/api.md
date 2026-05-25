# SREAgent REST API 参考手册

> 基于源码自动生成。最后更新：2026-05-21。

## 目录

- [约定](#约定)
- [认证](#1-认证)
- [OIDC 单点登录](#2-oidc-单点登录)
- [数据源](#3-数据源)
- [告警规则](#4-告警规则)
- [告警事件](#5-告警事件)
- [静默规则](#6-静默规则)
- [通知规则（v2）](#7-通知规则v2)
- [通知媒介](#8-通知媒介)
- [消息模板](#9-消息模板)
- [订阅规则](#10-订阅规则)
- [业务分组](#11-业务分组)
- [告警通道](#12-告警通道)
- [用户通知配置](#13-用户通知配置)
- [通知渠道（v1）](#14-通知渠道v1)
- [通知策略（v1）](#15-通知策略v1)
- [用户](#16-用户)
- [团队](#17-团队)
- [值班管理](#18-值班管理)
- [升级策略](#19-升级策略)
- [AI](#18-ai)
- [飞书机器人](#19-飞书机器人)
- [诊断工作流](#20-诊断工作流)
- [变更事件](#21-变更事件)
- [通知中心](#22-通知中心)
- [RBAC 权限](#23-rbac-权限)
- [引擎](#24-引擎)
- [仪表盘](#25-仪表盘)
- [Event Pipeline](#26-event-pipeline可编程告警处理链)
- [Webhook 与心跳](#27-webhook-与心跳)
- [告警操作页面](#28-告警操作页面)
- [告警规则模板](#29-告警规则模板)
- [抑制规则](#30-抑制规则)
- [标签注册表](#31-标签注册表)
- [协作空间](#32-协作空间)
- [排除规则](#33-排除规则)
- [分派策略](#34-分派策略)
- [集成中心](#35-集成中心)
- [路由规则](#36-路由规则)
- [故障](#37-故障)
- [故障复盘](#38-故障复盘)
- [告警（v2）](#39-告警v2)
- [批量操作与导出](#40-批量操作与导出)
- [宠物系统](#43-宠物系统)
- [状态页面](#44-状态页面)
- [预设规则](#45-预设规则)
- [Alertmanager 导入](#46-alertmanager-导入)

---

## 约定

### 基础 URL

所有 API 路由均以 `/api/v1` 为前缀，另有说明除外。

### 统一响应格式

所有 JSON 端点返回统一的响应信封：

```json
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}
```

- `code = 0` — 成功
- `code != 0` — 错误（message 字段包含可读的错误描述）

### 分页

分页列表端点接受以下参数：

| 参数 | 类型 | 默认值 | 范围 | 说明 |
|------|------|--------|------|------|
| `page` | int | 1 | >= 1 | 页码 |
| `page_size` | int | 20 | 1–100 | 每页条数 |

分页响应的 `data` 结构如下：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [ ... ],
    "total": 128,
    "page": 1,
    "page_size": 20
  }
}
```

### 认证

受保护的路由需要在 `Authorization` 请求头中携带 JWT 令牌：

```
Authorization: Bearer <token>
```

令牌通过 `POST /api/v1/auth/login` 或 OIDC 回调流程获取。

### RBAC 角色

五种角色，按权限从高到低排列：

| 角色 | 说明 |
|------|------|
| `admin` | 拥有所有资源的完全访问权限 |
| `team_lead` | 管理配置对象（规则、渠道、排班、团队） |
| `member` | 执行操作（确认、解决、订阅） |
| `viewer` | 对分配的资源拥有只读权限 |
| `global_viewer` | 对所有资源拥有只读权限 |

下文引用的路由访问级别说明：
- **公开** — 无需认证
- **已认证** — 任何已认证用户
- **操作权限** — `admin`、`team_lead` 或 `member`
- **管理权限** — `admin` 或 `team_lead`
- **仅管理员** — 仅 `admin`

### 通用模型字段

所有实体均包含以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | uint | 自增主键 |
| `created_at` | datetime | ISO 8601 格式 |
| `updated_at` | datetime | ISO 8601 格式 |

---

## 1. 认证

### POST `/api/v1/auth/login` — 登录

**访问级别：** 公开

**请求体：**

```json
{
  "username": "admin",
  "password": "secret123"
}
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOi...",
    "expires_in": 86400
  }
}
```

### GET `/api/v1/auth/profile` — 获取当前用户信息

**访问级别：** 已认证

**响应：** 用户对象（详见 [用户](#16-用户) 模型）。响应中不包含密码字段。

### PUT `/api/v1/me/profile` — 更新个人资料

**访问级别：** 已认证

| 字段 | 类型 | 说明 |
|------|------|------|
| `display_name` | string | 显示名称 |
| `email` | string | 邮箱 |
| `phone` | string | 手机号 |
| `avatar` | string | Base64 data URL 或预设头像标识 |

### POST `/api/v1/me/password` — 修改个人密码

**访问级别：** 已认证

| 字段 | 类型 | 必填 | 校验规则 |
|------|------|------|----------|
| `old_password` | string | 是 | |
| `new_password` | string | 是 | 至少 6 个字符 |

### PUT `/api/v1/me/lark-bind` — 绑定飞书账号

**访问级别：** 已认证

绑定当前用户的飞书账号，用于接收飞书个人通知。

---

## 2. OIDC 单点登录

### GET `/api/v1/auth/oidc/config` — OIDC 状态

**访问级别：** 公开

**响应：**

```json
{
  "code": 0,
  "data": {
    "enabled": true,
    "login_url": "/api/v1/auth/oidc/login"
  }
}
```

未配置 OIDC 时返回 `{"enabled": false}`。

### GET `/api/v1/auth/oidc/login` — 发起 OIDC 登录

**访问级别：** 公开

重定向（302）到已配置的身份提供商授权端点。设置 `oidc_state` Cookie 用于 CSRF 防护。

### GET `/api/v1/auth/oidc/callback` — OIDC 回调

**访问级别：** 公开

由身份提供商在认证完成后调用。成功后将浏览器重定向到：

```
/?oidc_token=<jwt>&expires_in=<seconds>
```

前端路由守卫拦截 `oidc_token` 查询参数并存储令牌。

**查询参数：**

| 参数 | 说明 |
|------|------|
| `code` | 身份提供商返回的授权码 |
| `state` | CSRF 状态值（与 Cookie 中的值进行校验） |
| `error` | 错误码（可选） |
| `error_description` | 错误详情（可选） |

### POST `/api/v1/auth/oidc/token` — 用授权码换取令牌（JSON）

**访问级别：** 公开

适用于偏好 JSON 流程而非重定向的 SPA 客户端。

**请求体：**

```json
{ "code": "abc123" }
```

**响应：** 与登录响应相同（`token`、`expires_in`）。

---

## 3. 数据源

管理 Prometheus、VictoriaMetrics、VictoriaLogs 和 Zabbix 数据源。

**模型字段：** `name`、`type`（prometheus | victoriametrics | zabbix | victorialogs）、`endpoint`、`description`、`labels`（map）、`status`（healthy | unhealthy | unknown）、`auth_type`（none | basic | bearer | api_key）、`auth_config`（JSON）、`health_check_interval`、`is_enabled`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/datasources` | 已认证 | 列表（分页）。筛选：`?type=prometheus` |
| GET | `/datasources/:id` | 已认证 | 按 ID 获取 |
| POST | `/datasources` | 仅管理员 | 创建 |
| PUT | `/datasources/:id` | 仅管理员 | 更新 |
| DELETE | `/datasources/:id` | 仅管理员 | 删除 |
| POST | `/datasources/:id/health-check` | 管理权限 | 触发健康检查 |
| POST | `/datasources/:id/query` | 管理权限 | Instant Query 测试 |
| POST | `/datasources/:id/query-range` | 管理权限 | Range Query（时间序列查询） |
| POST | `/datasources/:id/log-query` | 管理权限 | 日志查询（VictoriaLogs LogsQL） |
| GET | `/datasources/:id/labels/keys` | 已认证 | 获取 label name 列表（自动补全） |
| GET | `/datasources/:id/labels/values?key=job` | 已认证 | 获取 label value 列表 |
| GET | `/datasources/:id/metrics?search=&limit=` | 已认证 | 获取 metric 名称列表 |
| POST | `/datasources/:id/log-histogram` | 管理权限 | 日志直方图（时间桶计数） |
| ANY | `/datasources/:id/proxy/*path` | 管理权限 | 通用代理（透明转发到数据源 API） |
| POST | `/ds-query` | 已认证 | 统一查询（多数据源并发） |

**通用代理 (proxy)：**

透传任意 HTTP 请求到目标数据源 endpoint，路径参数 `*path` 为数据源 API 的相对路径，query 参数原样转发。

**统一查询 (ds-query) 请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `queries` | array | 是 | 查询数组 |
| `queries[].datasource_id` | uint | 是 | 数据源 ID |
| `queries[].expression` | string | 是 | PromQL 表达式 |
| `queries[].start` | int64 | 否 | 起始时间（Unix 秒），0 = 即时查询 |
| `queries[].end` | int64 | 否 | 结束时间（Unix 秒） |
| `queries[].step` | string | 否 | 步长，默认 `15s` |

**Range Query 请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `expression` | string | 是 | PromQL 表达式 |
| `start` | number | 是 | 起始时间（Unix 秒） |
| `end` | number | 是 | 结束时间（Unix 秒） |
| `step` | string | 是 | 步长（如 `15s`、`1m`、`5m`） |

**Range Query 响应：**

```json
{
  "code": 0,
  "data": {
    "result_type": "matrix",
    "series": [
      {
        "labels": {"__name__": "http_requests_total", "job": "api-server"},
        "values": [{"ts": 1714300800000, "value": 123.45}]
      }
    ]
  }
}
```

**Log Query 请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `expression` | string | 是 | LogsQL 查询表达式 |
| `start` | number | 是 | 起始时间（Unix 秒） |
| `end` | number | 是 | 结束时间（Unix 秒） |
| `limit` | number | 否 | 最大返回条数，默认 100，上限 10000 |

**Log Query 响应：**

```json
{
  "code": 0,
  "data": {
    "entries": [
      {
        "timestamp": "2026-04-29T10:00:00Z",
        "message": "ERROR: connection refused",
        "labels": {"job": "api-server", "level": "error", "instance": "10.0.0.1:9090"}
      }
    ],
    "total": 1,
    "truncated": false
  }
}
```

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 唯一名称 |
| `type` | string | 是 | 支持的类型之一 |
| `endpoint` | string | 是 | URL 地址 |
| `description` | string | 否 | 描述 |
| `labels` | map[string]string | 否 | 键值对元数据 |
| `auth_type` | string | 否 | 默认：`none` |
| `auth_config` | string (JSON) | 否 | 认证相关配置 |
| `health_check_interval` | int | 否 | 健康检查间隔（秒） |

**健康检查响应：**

```json
{ "code": 0, "data": { "status": "healthy" } }
```

---

## 4. 告警规则

使用 PromQL、LogsQL 或其他查询表达式定义评估规则。

**模型字段：** `name`、`display_name`、`description`、`datasource_id`、`expression`、`for_duration`、`severity`（critical | warning | info）、`labels`（map）、`annotations`（map）、`status`（enabled | disabled | muted）、`group_name`、`version`、`eval_interval`、`recovery_hold`、`nodata_enabled`、`nodata_duration`、`suppress_enabled`、`biz_group_id`、`created_by`、`updated_by`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/alert-rules` | 已认证 | 列表（分页）。筛选：`?severity=critical&status=enabled&group_name=infra` |
| GET | `/alert-rules/:id` | 已认证 | 按 ID 获取 |
| GET | `/alert-rules/categories` | 已认证 | 获取所有规则分类列表 |
| GET | `/alert-rules/export` | 已认证 | 导出为 YAML。筛选：`?group_name=infra` |
| POST | `/alert-rules` | 管理权限 | 创建 |
| PUT | `/alert-rules/:id` | 管理权限 | 更新 |
| DELETE | `/alert-rules/:id` | 管理权限 | 删除 |
| PATCH | `/alert-rules/:id/status` | 管理权限 | 切换状态 |
| POST | `/alert-rules/import` | 管理权限 | 从 YAML/JSON 文件导入 |
| POST | `/alert-rules/batch/enable` | 管理权限 | 批量启用 |
| POST | `/alert-rules/batch/disable` | 管理权限 | 批量禁用 |
| POST | `/alert-rules/batch/delete` | 管理权限 | 批量删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 规则标识符 |
| `display_name` | string | 否 | 人类可读的名称 |
| `description` | string | 否 | 描述 |
| `datasource_id` | uint | 是 | 关联数据源 ID |
| `expression` | string | 是 | PromQL / LogsQL 表达式 |
| `for_duration` | string | 否 | 例如 `"5m"` |
| `severity` | string | 是 | critical、warning、info |
| `labels` | map[string]string | 否 | 附加标签 |
| `annotations` | map[string]string | 否 | 注解（摘要、描述） |
| `group_name` | string | 否 | 规则分组 |
| `eval_interval` | int | 否 | 评估间隔（秒） |
| `recovery_hold` | int | 否 | 自动恢复前的等待时间（秒） |
| `nodata_enabled` | bool | 否 | 数据缺失时是否触发告警 |
| `nodata_duration` | int | 否 | 数据缺失阈值（秒） |
| `suppress_enabled` | bool | 否 | 启用基于级别的抑制 |
| `biz_group_id` | *uint | 否 | 所属业务分组 |

**切换状态请求体：**

```json
{ "status": "enabled" }
```

**导入** — `multipart/form-data`：

| 字段 | 类型 | 说明 |
|------|------|------|
| `file` | file | `.yaml` / `.yml` / `.json`（Prometheus 规则文件格式） |
| `datasource_id` | string | 导入规则的默认数据源 |

**导入响应：**

```json
{ "code": 0, "data": { "total": 10, "success": 9, "failed": 1, "errors": ["..."] } }
```

**导出** — 返回 `application/x-yaml` Content-Type，带有 `Content-Disposition: attachment` 头。

---

## 5. 告警事件

由评估引擎生成或通过 Webhook 接收的实时和历史告警实例。

**模型字段：** `fingerprint`、`rule_id`、`alert_name`、`severity`、`status`（firing | acknowledged | assigned | silenced | resolved | closed）、`labels`（map）、`annotations`（map）、`source`、`generator_url`、`fired_at`、`acked_at`、`resolved_at`、`closed_at`、`acked_by`、`assigned_to`、`silenced_until`、`silence_reason`、`resolution`、`fire_count`、`oncall_user_id`、`is_dispatched`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/alert-events` | 已认证 | 列表（分页）。筛选：`?status=firing&severity=critical&view_mode=mine` |
| GET | `/alert-events/:id` | 已认证 | 按 ID 获取 |
| GET | `/alert-events/:id/timeline` | 已认证 | 获取事件时间线（状态变更、评论） |
| POST | `/alert-events/:id/acknowledge` | 操作权限 | 确认告警 |
| POST | `/alert-events/:id/assign` | 操作权限 | 指派给用户 |
| POST | `/alert-events/:id/resolve` | 操作权限 | 解决告警 |
| POST | `/alert-events/:id/close` | 操作权限 | 关闭告警 |
| POST | `/alert-events/:id/comment` | 操作权限 | 添加评论 |
| POST | `/alert-events/:id/silence` | 操作权限 | 静默告警 |
| POST | `/alert-events/batch/acknowledge` | 操作权限 | 批量确认 |
| POST | `/alert-events/batch/close` | 操作权限 | 批量关闭 |
| GET | `/alert-events/export` | 已认证 | 导出事件为 CSV。筛选参数同列表 |
| GET | `/alert-events/groups` | 已认证 | 按规则分组统计事件数量 |

**列表筛选参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `status` | string | firing、acknowledged、assigned、silenced、resolved、closed |
| `severity` | string | critical、warning、info |
| `view_mode` | string | `mine`（指派给我）、`unassigned`（未指派）、`all`（默认） |
| `user_id` | uint | 管理员可用此参数覆盖 view_mode=mine |

**指派请求体：**

```json
{ "assign_to": 5, "note": "Please investigate" }
```

**解决请求体：**

```json
{ "resolution": "Fixed the root cause by scaling the service" }
```

**关闭请求体：**

```json
{ "note": "False positive" }
```

**评论请求体：**

```json
{ "note": "Investigating the issue now" }
```

**静默请求体：**

```json
{ "duration_minutes": 60, "reason": "Maintenance window" }
```

**批量确认 / 关闭请求体：**

```json
{ "ids": [1, 2, 3] }
```

**批量操作响应：**

```json
{ "code": 0, "data": { "success": 3, "failed": 0 } }
```

---

## 6. 静默规则

在指定时间窗口内，对匹配特定条件的告警抑制通知。

**模型字段：** `name`、`description`、`match_labels`（map）、`severities`（逗号分隔）、`start_time`、`end_time`、`periodic_start`、`periodic_end`、`days_of_week`、`timezone`、`is_enabled`、`rule_ids`（逗号分隔）、`created_by`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/mute-rules` | 已认证 | 列表（分页） |
| GET | `/mute-rules/preview` | 已认证 | 预览静默效果（查看哪些告警会被静默） |
| GET | `/mute-rules/:id` | 已认证 | 按 ID 获取 |
| POST | `/mute-rules` | 管理权限 | 创建 |
| PUT | `/mute-rules/:id` | 管理权限 | 更新 |
| DELETE | `/mute-rules/:id` | 管理权限 | 删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `match_labels` | map[string]string | 否 | 标签匹配器 |
| `severities` | string | 否 | 例如 `"critical,warning"` |
| `start_time` | datetime | 否 | 一次性窗口开始时间（ISO 8601） |
| `end_time` | datetime | 否 | 一次性窗口结束时间 |
| `periodic_start` | string | 否 | 每日开始时间，例如 `"02:00"` |
| `periodic_end` | string | 否 | 每日结束时间，例如 `"06:00"` |
| `days_of_week` | string | 否 | 例如 `"1,2,3,4,5"`（周一=1） |
| `timezone` | string | 否 | 默认：`"Asia/Shanghai"` |
| `is_enabled` | bool | 否 | 是否启用 |
| `rule_ids` | string | 否 | 逗号分隔的告警规则 ID |

---

## 7. 通知规则（v2）

支持管道处理和按规则配置通知目标的高级通知规则。

**模型字段：** `name`、`description`、`is_enabled`、`severities`、`match_labels`（map）、`pipeline`（JSON）、`notify_configs`（JSON）、`repeat_interval`、`callback_url`、`created_by`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/notify-rules` | 已认证 | 列表（分页） |
| GET | `/notify-rules/:id` | 已认证 | 按 ID 获取 |
| POST | `/notify-rules` | 管理权限 | 创建 |
| PUT | `/notify-rules/:id` | 管理权限 | 更新 |
| DELETE | `/notify-rules/:id` | 管理权限 | 删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `is_enabled` | bool | 否 | 是否启用 |
| `severities` | string | 否 | 逗号分隔 |
| `match_labels` | map[string]string | 否 | 标签匹配器 |
| `pipeline` | string (JSON) | 否 | 处理管道步骤 |
| `notify_configs` | string (JSON) | 否 | 通知配置数组 |
| `repeat_interval` | int | 否 | 重复通知间隔（秒） |
| `callback_url` | string | 否 | Webhook 回调 URL |

---

## 8. 通知媒介

通知媒介（投递渠道）：飞书 Webhook、邮件、HTTP Webhook、脚本。

**模型字段：** `name`、`type`（lark_webhook | email | http | script）、`description`、`is_enabled`、`config`（JSON）、`variables`（JSON）、`is_builtin`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/notify-media` | 已认证 | 列表（分页） |
| GET | `/notify-media/:id` | 已认证 | 按 ID 获取 |
| POST | `/notify-media` | 管理权限 | 创建 |
| PUT | `/notify-media/:id` | 管理权限 | 更新 |
| DELETE | `/notify-media/:id` | 管理权限 | 删除 |
| POST | `/notify-media/:id/test` | 管理权限 | 发送测试通知 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `type` | string | 是 | 类型：lark_webhook、email、http、script |
| `description` | string | 否 | 描述 |
| `is_enabled` | bool | 否 | 是否启用 |
| `config` | string (JSON) | 是 | 类型专属配置 |
| `variables` | string (JSON) | 否 | 模板变量 |

**测试响应：**

```json
{ "code": 0, "data": { "message": "test notification sent" } }
```

---

## 9. 消息模板

基于 Go `text/template` 的通知消息模板。

**模型字段：** `name`、`description`、`content`（Go 模板字符串）、`type`（text | html | markdown | lark_card）、`is_builtin`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/message-templates` | 已认证 | 列表（分页） |
| GET | `/message-templates/:id` | 已认证 | 按 ID 获取 |
| POST | `/message-templates` | 管理权限 | 创建 |
| PUT | `/message-templates/:id` | 管理权限 | 更新 |
| DELETE | `/message-templates/:id` | 管理权限 | 删除 |
| POST | `/message-templates/preview` | 已认证 | 预览模板渲染结果 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `content` | string | 是 | Go 模板字符串 |
| `type` | string | 否 | 默认：`"text"` |

**预览请求体：**

```json
{ "content": "Alert {{ .AlertName }} is {{ .Status }}" }
```

**预览响应：**

```json
{ "code": 0, "data": { "rendered": "Alert CPUHigh is firing" } }
```

---

## 10. 订阅规则

允许用户/团队订阅匹配特定条件的告警，并将其路由到指定的通知规则。

**模型字段：** `name`、`description`、`is_enabled`、`match_labels`（map）、`severities`、`notify_rule_id`、`user_id`、`team_id`、`created_by`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/subscribe-rules` | 已认证 | 列表（分页） |
| GET | `/subscribe-rules/:id` | 已认证 | 按 ID 获取 |
| POST | `/subscribe-rules` | 操作权限 | 创建 |
| PUT | `/subscribe-rules/:id` | 操作权限 | 更新 |
| DELETE | `/subscribe-rules/:id` | 操作权限 | 删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `is_enabled` | bool | 否 | 是否启用 |
| `match_labels` | map[string]string | 否 | 标签匹配器 |
| `severities` | string | 否 | 逗号分隔 |
| `notify_rule_id` | uint | 是 | 目标通知规则 |
| `user_id` | uint | 否 | 为指定用户订阅 |
| `team_id` | uint | 否 | 为团队订阅 |

---

## 11. 业务分组

用于组织告警规则和访问控制的层级业务分组树。

**模型字段：** `name`（支持 `/` 表示层级）、`description`、`parent_id`、`labels`（map）、`members`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/biz-groups` | 已认证 | 列表（分页） |
| GET | `/biz-groups/tree` | 已认证 | 获取树形结构 |
| GET | `/biz-groups/:id` | 已认证 | 按 ID 获取 |
| GET | `/biz-groups/:id/members` | 已认证 | 列出分组成员 |
| POST | `/biz-groups` | 管理权限 | 创建 |
| PUT | `/biz-groups/:id` | 管理权限 | 更新 |
| DELETE | `/biz-groups/:id` | 管理权限 | 删除 |
| POST | `/biz-groups/:id/members` | 管理权限 | 添加成员 |
| DELETE | `/biz-groups/:id/members/:uid` | 管理权限 | 移除成员 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `parent_id` | uint | 否 | 父分组 ID |
| `labels` | map[string]string | 否 | 标签 |

**添加成员请求体：**

```json
{ "user_id": 5, "role": "admin" }
```

角色可选 `"admin"` 或 `"member"`。

---

## 12. 告警通道

虚拟告警路由通道，将通知媒介与可选的模板和标签匹配器绑定。

**模型字段：** `name`、`description`、`match_labels`（map）、`severities`、`media_id`、`template_id`、`throttle_min`、`is_enabled`、`created_by`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/alert-channels` | 已认证 | 列表（分页） |
| GET | `/alert-channels/:id` | 已认证 | 按 ID 获取 |
| POST | `/alert-channels` | 管理权限 | 创建 |
| PUT | `/alert-channels/:id` | 管理权限 | 更新 |
| DELETE | `/alert-channels/:id` | 管理权限 | 删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `match_labels` | map[string]string | 否 | 标签匹配器 |
| `severities` | string | 否 | 逗号分隔 |
| `media_id` | uint | 是 | 关联通知媒介 ID |
| `template_id` | uint | 否 | 关联消息模板 ID |
| `throttle_min` | int | 否 | 通知最小间隔（分钟） |
| `is_enabled` | bool | 否 | 是否启用 |

---

## 13. 用户通知配置

当前用户的个人通知偏好设置（多媒介）。

**模型字段：** `user_id`、`media_type`（lark_personal | email | webhook）、`config`（JSON）、`is_enabled`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/me/notify-configs` | 已认证 | 列出当前用户的配置 |
| PUT | `/me/notify-configs` | 已认证 | 创建或更新（按 media_type upsert） |
| DELETE | `/me/notify-configs/:mediaType` | 已认证 | 按媒介类型删除 |

**Upsert 请求体：**

```json
{
  "media_type": "email",
  "config": "{\"address\": \"user@example.com\"}",
  "is_enabled": true
}
```

**删除路径参数：** `:mediaType` — 例如 `email`、`lark_personal`、`webhook`。

---

## 14. 用户

用户管理。支持普通用户、机器人用户和渠道（虚拟）用户。

**模型字段：** `username`、`display_name`、`email`、`phone`、`lark_user_id`、`avatar`、`role`（admin | team_lead | member | viewer | global_viewer）、`is_active`、`user_type`（human | bot | channel）、`notify_target`（JSON）、`oidc_subject`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/users` | 已认证 | 列表（分页）。筛选：`?user_type=human` |
| GET | `/users/:id` | 已认证 | 按 ID 获取 |
| POST | `/users` | 仅管理员 | 创建普通用户 |
| POST | `/users/virtual` | 仅管理员 | 创建虚拟用户（机器人/渠道） |
| PUT | `/users/:id` | 仅管理员 | 更新用户 |
| PATCH | `/users/:id/active` | 仅管理员 | 启用 / 禁用用户 |
| PATCH | `/users/:id/password` | 仅管理员 | 管理员重置密码 |
| DELETE | `/users/:id` | 仅管理员 | 删除用户 |

**创建普通用户请求体：**

| 字段 | 类型 | 必填 | 校验规则 |
|------|------|------|----------|
| `username` | string | 是 | 唯一 |
| `password` | string | 是 | 至少 6 个字符 |
| `display_name` | string | 否 | |
| `email` | string | 否 | 邮箱格式 |
| `phone` | string | 否 | |
| `lark_user_id` | string | 否 | |
| `avatar` | string | 否 | |
| `role` | string | 否 | 默认：`"member"` |

**创建虚拟用户请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 用户名 |
| `display_name` | string | 否 | 显示名称 |
| `user_type` | string | 是 | `"bot"` 或 `"channel"` |
| `notify_target` | string | 否 | JSON 通知目标配置 |
| `description` | string | 否 | 描述 |
| `role` | string | 否 | 角色 |

**更新请求体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `display_name` | string | 显示名称 |
| `email` | string | 邮箱 |
| `phone` | string | 手机号 |
| `lark_user_id` | string | 飞书用户 ID |
| `avatar` | string | 头像 |
| `role` | string | 角色 |

**切换启用状态请求体：**

```json
{ "is_active": true }
```

**修改密码请求体：**

| 字段 | 类型 | 必填 | 校验规则 |
|------|------|------|----------|
| `old_password` | string | 是 | |
| `new_password` | string | 是 | 至少 6 个字符 |

---

## 15. 团队

团队管理，支持成员角色设置。

**模型字段：** `name`、`description`、`labels`（map）。成员通过关联表管理，角色为 `role`（lead | member）。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/teams` | 已认证 | 列表（分页） |
| GET | `/teams/:id` | 已认证 | 按 ID 获取 |
| GET | `/teams/:id/members` | 已认证 | 列出团队成员 |
| POST | `/teams` | 管理权限 | 创建 |
| PUT | `/teams/:id` | 管理权限 | 更新 |
| DELETE | `/teams/:id` | 管理权限 | 删除 |
| POST | `/teams/:id/members` | 管理权限 | 添加成员 |
| DELETE | `/teams/:id/members/:uid` | 管理权限 | 移除成员 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 |
|------|------|------|
| `name` | string | 是 |
| `description` | string | 否 |
| `labels` | map[string]string | 否 |

**添加成员请求体：**

```json
{ "user_id": 5, "role": "lead" }
```

角色可选 `"lead"` 或 `"member"`。

---

## 16. 值班管理

值班排班管理，支持轮转、班次、替班和参与人。

### 排班 CRUD

**模型字段：** `name`、`team_id`、`description`、`rotation_type`（daily | weekly | custom）、`timezone`、`handoff_time`、`handoff_day`、`is_enabled`、`severity_filter`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/schedules` | 已认证 | 列表（分页）。筛选：`?team_id=1` |
| GET | `/schedules/:id` | 已认证 | 按 ID 获取 |
| GET | `/schedules/:id/oncall` | 已认证 | 获取当前值班用户 |
| GET | `/schedules/:id/participants` | 已认证 | 获取轮转参与人列表 |
| GET | `/schedules/:id/overrides` | 已认证 | 获取替班列表 |
| POST | `/schedules` | 管理权限 | 创建 |
| PUT | `/schedules/:id` | 管理权限 | 更新 |
| DELETE | `/schedules/:id` | 管理权限 | 删除 |
| PUT | `/schedules/:id/participants` | 管理权限 | 设置轮转参与人 |
| POST | `/schedules/:id/overrides` | 管理权限 | 创建替班 |
| DELETE | `/schedules/:id/overrides/:oid` | 管理权限 | 删除替班 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 默认值 |
|------|------|------|--------|
| `name` | string | 是 | |
| `team_id` | uint | 否 | |
| `description` | string | 否 | |
| `rotation_type` | string | 是 | |
| `timezone` | string | 否 | `"Asia/Shanghai"` |
| `handoff_time` | string | 否 | `"09:00"` |
| `handoff_day` | int | 否 | |
| `is_enabled` | bool | 否 | true |

**设置参与人请求体：**

```json
{ "user_ids": [1, 2, 3] }
```

**创建替班请求体：**

```json
{
  "user_id": 5,
  "start_time": "2026-04-05T00:00:00Z",
  "end_time": "2026-04-06T00:00:00Z",
  "reason": "Coverage swap"
}
```

### 值班班次

**模型字段：** `schedule_id`、`user_id`、`start_time`、`end_time`、`severity_filter`、`source`（manual | rotation）、`note`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/schedules/:id/shifts` | 已认证 | 列出班次。筛选：`?start=<RFC3339>&end=<RFC3339>` |
| POST | `/schedules/:id/shifts` | 管理权限 | 创建班次 |
| PUT | `/schedules/:id/shifts/:shiftId` | 管理权限 | 更新班次 |
| DELETE | `/schedules/:id/shifts/:shiftId` | 管理权限 | 删除班次 |
| POST | `/schedules/:id/generate-shifts` | 管理权限 | 根据轮转自动生成班次 |
| GET | `/schedules/:id/ical` | 已认证 | 导出排班为 iCal 格式 |

**创建 / 更新班次请求体：**

| 字段 | 类型 | 必填 |
|------|------|------|
| `user_id` | uint | 是 |
| `start_time` | datetime (RFC 3339) | 是 |
| `end_time` | datetime (RFC 3339) | 是 |
| `severity_filter` | string | 否 |
| `note` | string | 否 |

**自动生成班次请求体：**

```json
{ "weeks": 4 }
```

校验范围：1–52 周。

**iCal 导出** — 返回 `text/calendar` Content-Type，带有 `Content-Disposition: attachment` 头。可用于将排班同步到 Outlook、Google Calendar 等日历应用。

**生成响应：**

```json
{ "code": 0, "data": { "message": "shifts generated", "weeks": 4 } }
```

---

## 17. 升级策略

多步骤升级策略，定义通知目标和延迟间隔。

### 策略 CRUD

**模型字段：** `name`、`team_id`、`is_enabled`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/escalation-policies` | 已认证 | 列表。筛选：`?team_id=1` |
| GET | `/escalation-policies/:id` | 已认证 | 获取策略及其步骤 |
| POST | `/escalation-policies` | 管理权限 | 创建 |
| PUT | `/escalation-policies/:id` | 管理权限 | 更新 |
| DELETE | `/escalation-policies/:id` | 管理权限 | 删除 |

**创建 / 更新请求体：**

```json
{ "name": "Critical P1", "team_id": 1, "is_enabled": true }
```

**获取响应：**

```json
{
  "code": 0,
  "data": {
    "policy": { "id": 1, "name": "Critical P1", "team_id": 1, "is_enabled": true },
    "steps": [ { "id": 1, "step_order": 1, "delay_minutes": 0, "target_type": "schedule", "target_id": 1 }, ... ]
  }
}
```

---

## 18. AI

AI 驱动的告警分析。支持 LLM 生成的告警报告和 SOP 建议。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/ai/alert-report` | 已认证 | 生成 AI 告警分析报告 |
| POST | `/ai/suggest-sop` | 已认证 | AI 推荐的告警 SOP |
| POST | `/ai/test` | 管理权限 | 测试 AI 提供商连通性 |
| GET | `/ai/config` | 仅管理员 | 获取 AI 配置（API key 已脱敏） |
| PUT | `/ai/config` | 仅管理员 | 更新 AI 配置 |

**生成报告 / 推荐 SOP 请求体：**

```json
{ "event_id": 42 }
```

**报告响应：**

```json
{ "code": 0, "data": { "report": "## Analysis\n...", "event_id": 42 } }
```

**SOP 响应：**

```json
{ "code": 0, "data": { "sop": "1. Check CPU usage\n2. ...", "event_id": 42 } }
```

**测试响应：**

```json
{ "code": 0, "data": { "message": "AI connection successful" } }
```

### 18.1 AI 规则生成

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/ai/rules/generate` | 操作权限 | 自然语言生成告警规则 |
| POST | `/ai/rules/dry-run` | 操作权限 | 生成 + 自动验证 PromQL |
| POST | `/ai/rules/validate` | 操作权限 | 验证 PromQL 表达式 |
| POST | `/ai/rules/suggest-labels` | 操作权限 | AI 推荐标签 |
| POST | `/ai/rules/generate-inhibition` | 操作权限 | 生成抑制规则 |
| POST | `/ai/rules/generate-mute` | 操作权限 | 生成静默规则 |
| POST | `/ai/rules/improve` | 操作权限 | 基于反馈优化规则 |

**生成请求体：**

```json
{ "description": "CPU 使用率超过 90% 持续 5 分钟", "datasource_id": 1, "rule_type": "alert" }
```

**Dry-Run 响应：** 返回生成的规则 + PromQL 验证结果（`rule` + `validation`）。

### 18.2 AI 模块配置

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/ai/modules` | 已认证 | 获取 AI 模块开关状态 |
| PUT | `/ai/modules` | 仅管理员 | 更新 AI 模块配置 |

### 18.3 AI 多 Provider 配置

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/ai/providers` | 仅管理员 | 获取多 Provider 配置 |
| PUT | `/ai/providers` | 仅管理员 | 保存多 Provider 配置 |
| POST | `/ai/test-provider` | 仅管理员 | 测试指定 Provider 连通性 |

### 18.4 AI 全局配置

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/ai/global` | 仅管理员 | 获取 AI 全局配置 |
| PUT | `/ai/global` | 仅管理员 | 更新 AI 全局配置 |

全局配置字段: `retry_max`, `context_max_chars`, `default_temperature`, `default_max_tokens`, `monthly_token_budget`, `data_masking_enabled`

### 18.5 AI Chat

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/ai/chat` | 已认证 | AI 对话（支持多轮） |
| GET | `/ai/history` | 已认证 | 获取对话历史 |
| DELETE | `/ai/history` | 已认证 | 清空对话历史 |

### 18.6 AI Agent

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/ai/agent/run` | 已认证 | 启动 Agent 任务（异步） |
| GET | `/ai/agent/tasks/:id` | 已认证 | 查询 Agent 任务状态 |
| GET | `/ai/agent/conversations` | 已认证 | 列出会话 |
| GET | `/ai/agent/conversations/:id` | 已认证 | 获取会话详情 |
| DELETE | `/ai/agent/conversations/:id` | 已认证 | 删除会话 |
| GET | `/ai/agent/conversations/:id/tool-calls` | 已认证 | 列出工具调用记录 |

### 18.7 知识库

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/knowledge` | 已认证 | 列出知识文档 |
| GET | `/knowledge/:id` | 已认证 | 获取知识文档 |
| POST | `/knowledge` | 管理权限 | 创建知识文档 |
| PUT | `/knowledge/:id` | 管理权限 | 更新知识文档 |
| DELETE | `/knowledge/:id` | 管理权限 | 删除知识文档 |
| POST | `/knowledge/search` | 已认证 | 全文检索知识库 |
| POST | `/knowledge/:id/helpful` | 操作权限 | 标记文档有帮助 |

---

## 19. 飞书机器人

飞书（Lark）机器人集成，用于交互式告警通知。

### POST `/lark/event` — 飞书事件回调

**访问级别：** 公开（通过飞书验证令牌校验）

接收飞书事件订阅回调，包括 URL 验证挑战和消息事件。返回原始 JSON 以兼容飞书协议。

### GET `/api/v1/lark/bot/config` — 获取飞书配置

**访问级别：** 仅管理员

返回飞书机器人配置（App ID、Webhook URL 等）。

### PUT `/api/v1/lark/bot/config` — 更新飞书配置

**访问级别：** 仅管理员

更新飞书机器人配置。敏感字段（app_secret、verification_token、encrypt_key）使用 AES-GCM 加密存储。

### POST `/api/v1/lark/bot/test` — 测试 Bot API 连通性

**访问级别：** 仅管理员

使用当前配置的凭据测试飞书 Bot API 连接。

### GET `/api/v1/lark/bot/status` — 获取 Bot 状态

**访问级别：** 仅管理员

返回 Bot 运行状态：configured、app_id、webhook_set、commands_enabled、natural_language_enabled、debug_mode。

---

## 20. 诊断工作流

AIOps 诊断 SOP 编排引擎，支持多步骤自动化诊断。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/diagnostic-workflows` | 已认证 | 列出诊断工作流（支持 category/enabled 筛选） |
| GET | `/diagnostic-workflows/:id` | 已认证 | 获取工作流详情（含步骤） |
| POST | `/diagnostic-workflows` | 管理权限 | 创建工作流（可含步骤） |
| PUT | `/diagnostic-workflows/:id` | 管理权限 | 更新工作流 |
| DELETE | `/diagnostic-workflows/:id` | 管理权限 | 删除工作流 |
| PUT | `/diagnostic-workflows/:id/steps` | 管理权限 | 替换工作流步骤（原子操作） |
| POST | `/diagnostic-workflows/:id/run` | 操作权限 | 启动诊断运行 |
| POST | `/diagnostic-workflows/match` | 操作权限 | 按标签+严重等级匹配工作流 |
| GET | `/diagnostic-runs` | 已认证 | 列出诊断运行（支持 workflow_id/incident_id/status 筛选） |
| GET | `/diagnostic-runs/:id` | 已认证 | 获取运行详情（含步骤结果） |

## 21. 变更事件

CI/CD 变更事件接入，用于告警关联分析。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/change-events` | 已认证 | 列出变更事件（支持 service/environment/source 筛选） |
| GET | `/change-events/:id` | 已认证 | 获取变更事件详情 |
| POST | `/change-events` | 管理权限 | 接入变更事件 |
| DELETE | `/change-events/:id` | 管理权限 | 删除变更事件 |

## 22. 通知中心

用户个人通知管理。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/notifications` | 已认证 | 列出通知 |
| GET | `/notifications/unread-count` | 已认证 | 获取未读数 |
| POST | `/notifications/read-all` | 已认证 | 全部标为已读 |
| PATCH | `/notifications/:id/read` | 已认证 | 标为已读 |
| DELETE | `/notifications/:id` | 已认证 | 删除通知 |

## 23. RBAC 权限

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/me/permissions` | 已认证 | 获取当前用户权限列表 |

---

## 24. 引擎

### GET `/api/v1/engine/status` — 引擎状态

**访问级别：** 已认证

返回告警评估引擎状态，包括活跃规则数量、评估指标和状态存储连通性。

### GET `/metrics` — Prometheus 指标

**访问级别：** 公开

返回 Go 运行时和应用自定义指标，格式为 Prometheus exposition format。可用于 Prometheus 抓取配置。

---

## 25. 仪表盘

### GET `/api/v1/dashboard/stats` — 仪表盘统计

**访问级别：** 已认证

**响应：**

```json
{
  "code": 0,
  "data": {
    "total_datasources": 3,
    "total_rules": 45,
    "active_alerts": 12,
    "resolved_today": 8,
    "total_users": 20,
    "total_teams": 4
  }
}
```

### Dashboard V2（面板仪表盘）

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/dashboards` | 已认证 | 列表（分页）。筛选：`?search=xxx` |
| GET | `/dashboards/:id` | 已认证 | 按 ID 获取 |
| POST | `/dashboards` | 管理权限 | 创建 |
| PUT | `/dashboards/:id` | 管理权限 | 更新 |
| DELETE | `/dashboards/:id` | 管理权限 | 删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 仪表盘名称 |
| `description` | string | 否 | 描述 |
| `tags` | map[string]string | 否 | 标签 |
| `config` | string (JSON) | 否 | 面板布局和变量配置 |
| `is_public` | bool | 否 | 是否公开 |

---

## 26. Event Pipeline（可编程告警处理链）

### GET `/event-pipelines` — 列表

**访问级别：** 已认证

查询参数：`?page=1&page_size=20&query=xxx&disabled=false`

### GET `/event-pipelines/:id` — 详情

**访问级别：** 已认证

### POST `/event-pipelines` — 创建

**访问级别：** 管理权限

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | Pipeline 名称 |
| `description` | string | 否 | 描述 |
| `disabled` | bool | 否 | 是否禁用 |
| `filter_enable` | bool | 否 | 是否启用前置标签过滤 |
| `label_filters` | []TagFilter | 否 | 前置过滤条件 |
| `processors` | []ProcessorConfig | 否 | 处理器配置列表（按顺序执行） |

**ProcessorConfig 结构：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `typ` | string | 处理器类型：`relabel`, `callback`, `event_drop`, `ai_summary` |
| `config` | object | 处理器专属配置 |

**relabel 配置：** `source_labels`, `separator`, `regex`, `target_label`, `replacement`, `action` (replace/keep/drop/labelmap/hashmod)

**callback 配置：** `url`, `method`, `headers`, `timeout`, `skip_ssl_verify`

**event_drop 配置：** `condition` (Go template，结果为 "true" 则丢弃)

**ai_summary 配置：** `only_critical` (bool)

### PUT `/event-pipelines/:id` — 更新

**访问级别：** 管理权限（请求体同创建）

### DELETE `/event-pipelines/:id` — 删除

**访问级别：** 管理权限

### POST `/event-pipelines/:id/tryrun` — 测试运行

**访问级别：** 管理权限

使用最近一条 firing 告警事件测试管道效果。

### GET `/event-pipelines/:id/executions` — 执行记录

**访问级别：** 已认证

查询参数：`?page=1&page_size=20`

### GET `/event-pipelines/processor-types` — 处理器类型列表

**访问级别：** 已认证

### GET `/event-pipeline-executions/:id` — 执行详情

**访问级别：** 已认证

### POST `/event-pipeline-executions/clean` — 清理旧记录

**访问级别：** 仅管理员

查询参数：`?days=30`（清理 N 天前的记录）

---

## 27. Webhook 与心跳

### POST `/webhooks/alertmanager` — Alertmanager Webhook

**访问级别：** 公开（通过共享密钥或网络层面的源 IP 进行认证）

接收 [Alertmanager webhook 格式](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config) 的告警负载。

**请求体：**

```json
{
  "version": "4",
  "status": "firing",
  "receiver": "sreagent",
  "alerts": [
    {
      "status": "firing",
      "labels": { "alertname": "HighCPU", "severity": "critical", "instance": "node1:9090" },
      "annotations": { "summary": "CPU usage above 90%", "description": "..." },
      "startsAt": "2026-04-04T10:00:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph?...",
      "fingerprint": "abc123"
    }
  ],
  "groupLabels": { "alertname": "HighCPU" },
  "commonLabels": { "alertname": "HighCPU", "severity": "critical" },
  "commonAnnotations": { "summary": "CPU usage above 90%" },
  "externalURL": "http://alertmanager:9093"
}
```

### POST `/heartbeat/:token` — 心跳 Ping

**访问级别：** 公开（通过 URL 中的 token 认证）

心跳探活端点，外部系统定期调用以报告存活状态。如超时未收到 ping，将触发心跳丢失告警。

**路径参数：** `:token` — 心跳监控点的唯一令牌。

**响应：**

```json
{ "code": 0, "data": { "message": "pong" } }
```

---

## 28. 告警操作页面

通过令牌认证的 HTML 页面，从飞书通知卡片中链接。允许一键执行告警操作，无需访问完整 UI。

### GET `/alert-action/:token` — 渲染操作页面

**访问级别：** URL 路径中的令牌（JWT）

| 查询参数 | 说明 |
|----------|------|
| `action` | 预选操作：acknowledge、silence、resolve、close |
| `duration` | 预填静默时长（分钟） |

返回包含操作表单的 HTML 页面。

### POST `/alert-action/:token` — 执行操作

**访问级别：** URL 路径中的令牌（JWT）

**表单字段**（`application/x-www-form-urlencoded`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `action` | string | acknowledge、silence、resolve、close |
| `operator_name` | string | 操作人 |
| `note` | string | 备注（可选） |
| `duration` | string | 静默时长（分钟），仅用于 silence 操作 |

返回 HTML 结果页面（成功或错误）。

---

## 29. 告警规则模板

可复用的告警规则配置模板，支持按分类管理和一键应用创建规则。

**模型字段：** `name`、`category`、`description`、`datasource_type`（prometheus | victoriametrics | zabbix | victorialogs）、`expression`、`for_duration`、`severity`（critical | warning | info）、`labels`（map）、`annotations`（map）、`group_name`、`eval_interval`、`is_builtin`、`usage_count`、`nodata_enabled`、`nodata_duration`、`ack_sla_minutes`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/alert-rule-templates` | 已认证 | 列表（分页）。筛选：`?category=infra&search=cpu` |
| GET | `/alert-rule-templates/categories` | 已认证 | 获取所有分类列表 |
| GET | `/alert-rule-templates/:id` | 已认证 | 按 ID 获取 |
| POST | `/alert-rule-templates` | 管理权限 | 创建 |
| PUT | `/alert-rule-templates/:id` | 管理权限 | 更新 |
| DELETE | `/alert-rule-templates/:id` | 管理权限 | 删除 |
| POST | `/alert-rule-templates/:id/apply` | 管理权限 | 应用模板创建告警规则 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 模板名称 |
| `category` | string | 否 | 分类（如 `infra`、`app`、`db`） |
| `description` | string | 否 | 描述 |
| `datasource_type` | string | 是 | 数据源类型 |
| `expression` | string | 是 | PromQL / LogsQL 表达式 |
| `for_duration` | string | 否 | 持续时间，例如 `"5m"` |
| `severity` | string | 是 | critical、warning、info |
| `labels` | map[string]string | 否 | 附加标签 |
| `annotations` | map[string]string | 否 | 注解 |
| `group_name` | string | 否 | 规则分组 |
| `eval_interval` | int | 否 | 评估间隔（秒），默认 60 |
| `nodata_enabled` | bool | 否 | 数据缺失时是否触发 |
| `nodata_duration` | string | 否 | 数据缺失阈值，默认 `"5m"` |
| `ack_sla_minutes` | int | 否 | 确认 SLA（分钟） |

**应用模板请求体：** 与告警规则创建请求体相同（参见 [告警规则](#4-告警规则)），用于覆盖模板中的默认值。

**应用响应：** 返回新创建的告警规则对象。

---

## 30. 抑制规则

当源告警处于 firing 状态时，自动抑制目标告警的通知。常用于抑制由同一根因引起的下游告警。

**模型字段：** `name`、`description`、`source_match`（map，源告警标签匹配器）、`target_match`（map，目标告警标签匹配器）、`equal_labels`（逗号分隔，源和目标必须相等的标签键）、`is_enabled`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/inhibition-rules` | 已认证 | 列表（分页） |
| GET | `/inhibition-rules/:id` | 已认证 | 按 ID 获取 |
| POST | `/inhibition-rules` | 管理权限 | 创建 |
| PUT | `/inhibition-rules/:id` | 管理权限 | 更新 |
| DELETE | `/inhibition-rules/:id` | 管理权限 | 删除 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称 |
| `description` | string | 否 | 描述 |
| `source_match` | map[string]string | 否 | 源告警标签匹配器 |
| `target_match` | map[string]string | 否 | 目标告警标签匹配器 |
| `equal_labels` | string | 否 | 逗号分隔的相等标签键，例如 `"cluster,namespace"` |
| `is_enabled` | bool | 否 | 是否启用 |

---

## 31. 标签注册表

全局标签键值自动补全服务，从所有数据源聚合标签数据，用于规则配置时的标签选择。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/label-registry/keys` | 已认证 | 获取所有标签键。筛选：`?datasource_id=1,2` |
| GET | `/label-registry/values?key=X` | 已认证 | 获取指定键的值。筛选：`?datasource_id=1,2` |
| POST | `/label-registry/sync` | 仅管理员 | 触发全量标签同步 |

**GetKeys 响应：**

```json
{ "code": 0, "data": ["job", "instance", "namespace", "severity"] }
```

**GetValues 响应：**

```json
{ "code": 0, "data": ["api-server", "web-server", "worker"] }
```

**Sync 响应：**

```json
{ "code": 0, "data": { "message": "sync triggered" } }
```

---

## 32. 协作空间

协作空间（Channel）是故障管理和告警路由的核心组织单元。每个空间可配置降噪规则、分派策略和自动关闭策略。

**模型字段：** `name`、`description`、`team_id`、`status`（active | disabled）、`access_level`（public | private）、`aggregation_config`（JSON）、`flapping_config`（JSON）、`auto_close_enabled`、`auto_close_origin`（triggered | last_alert）、`auto_close_minutes`、`follow_alert_close`、`active_incident_count`、`sort_order`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/channels` | 已认证 | 列表（分页）。筛选：`?query=xxx&status=active` |
| GET | `/channels/:id` | 已认证 | 按 ID 获取 |
| POST | `/channels` | 管理权限 | 创建 |
| PUT | `/channels/:id` | 管理权限 | 更新 |
| DELETE | `/channels/:id` | 管理权限 | 删除 |
| POST | `/channels/:id/star` | 已认证 | 收藏 |
| DELETE | `/channels/:id/star` | 已认证 | 取消收藏 |
| GET | `/channels/:id/exclusion-rules` | 已认证 | 排除规则列表 |
| POST | `/channels/:id/exclusion-rules` | 管理权限 | 创建排除规则 |
| GET | `/channels/:id/dispatch-policies` | 已认证 | 分派策略列表 |
| POST | `/channels/:id/dispatch-policies` | 管理权限 | 创建分派策略 |

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 空间名称（唯一） |
| `description` | string | 否 | 描述 |
| `team_id` | uint | 否 | 关联团队 ID |
| `status` | string | 否 | 默认 `"active"` |
| `access_level` | string | 否 | 默认 `"public"` |
| `aggregation_config` | string (JSON) | 否 | 聚合降噪配置 |
| `flapping_config` | string (JSON) | 否 | 抖动检测配置 |
| `auto_close_enabled` | bool | 否 | 是否启用自动关闭 |
| `auto_close_origin` | string | 否 | 计时起点：`"triggered"` 或 `"last_alert"` |
| `auto_close_minutes` | int | 否 | 自动关闭时间（分钟） |
| `follow_alert_close` | bool | 否 | 告警全部恢复时自动关闭故障 |
| `sort_order` | int | 否 | 排序权重 |

**列表响应增强：** 列表中每个对象额外包含 `is_starred` 字段，标识当前用户是否已收藏。

---

## 33. 排除规则

协作空间级别的告警过滤规则，在告警进入故障流程前进行匹配和丢弃。

**模型字段：** `channel_id`、`name`、`description`、`conditions`（JSON 数组，FilterCondition 格式）、`is_enabled`、`priority`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| PUT | `/exclusion-rules/:id` | 管理权限 | 更新 |
| DELETE | `/exclusion-rules/:id` | 管理权限 | 删除 |

> 列表和创建端点挂载在协作空间下：`GET /channels/:id/exclusion-rules` 和 `POST /channels/:id/exclusion-rules`（参见 [协作空间](#30-协作空间)）。

**创建请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 规则名称 |
| `description` | string | 否 | 描述 |
| `conditions` | string (JSON) | 否 | FilterCondition 数组，例如 `[{"field":"severity","operator":"eq","value":"info"}]` |
| `is_enabled` | bool | 否 | 是否启用 |
| `priority` | int | 否 | 优先级（值越小越先评估） |

**更新请求体：** 同创建，所有字段可选。

---

## 34. 分派策略

协作空间级别的告警分派配置，控制故障如何被通知、升级和重复提醒。

**模型字段：** `channel_id`、`name`、`description`、`is_enabled`、`priority`、`match_conditions`（JSON）、`active_time_config`（JSON）、`delay_seconds`、`escalation_policy_id`、`repeat_interval_seconds`、`max_repeats`、`notify_mode`（personal_preference | unified）、`unified_media_id`、`label_enhancement_rules`（JSON）。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/dispatch-policies/:id` | 已认证 | 按 ID 获取 |
| PUT | `/dispatch-policies/:id` | 管理权限 | 更新 |
| DELETE | `/dispatch-policies/:id` | 管理权限 | 删除 |
| GET | `/incidents/:id/dispatch-logs` | 已认证 | 故障的分派日志 |

> 列表和创建端点挂载在协作空间下：`GET /channels/:id/dispatch-policies` 和 `POST /channels/:id/dispatch-policies`（参见 [协作空间](#30-协作空间)）。
> 分派日志端点挂载在故障下：`GET /incidents/:id/dispatch-logs`（参见 [故障管理](#28-故障管理)）。

**创建 / 更新请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 策略名称 |
| `description` | string | 否 | 描述 |
| `is_enabled` | bool | 否 | 是否启用 |
| `priority` | int | 否 | 优先级（值越小越先评估） |
| `match_conditions` | string (JSON) | 否 | FilterCondition 数组，匹配时才生效 |
| `active_time_config` | string (JSON) | 否 | 生效时间窗口配置（时区、星期、时间段） |
| `delay_seconds` | int | 否 | 分派延迟（秒），0 = 立即。延迟期间如已确认则跳过 |
| `escalation_policy_id` | uint | 否 | 关联升级策略 ID |
| `repeat_interval_seconds` | int | 否 | 重复通知间隔（秒），0 = 不重复 |
| `max_repeats` | int | 否 | 最大重复次数，0 = 无限 |
| `notify_mode` | string | 否 | `"personal_preference"`（用户偏好）或 `"unified"`（统一媒介） |
| `unified_media_id` | uint | 否 | 统一模式下的通知媒介 ID |
| `label_enhancement_rules` | string (JSON) | 否 | 标签增强规则数组 |

---

## 35. 集成中心

Webhook 集成管理，支持 Alertmanager、Grafana 和标准 JSON 格式的告警接入。每个集成生成唯一的 webhook token。

**模型字段：** `name`、`description`、`type`（standard | alertmanager | grafana）、`mode`（exclusive | shared）、`channel_id`、`webhook_token`、`pipeline_config`（JSON）、`label_enhancement_config`（JSON）、`is_enabled`、`total_alerts`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/integrations` | 已认证 | 列表（分页）。筛选：`?channel_id=1` |
| GET | `/integrations/:id` | 已认证 | 按 ID 获取 |
| POST | `/integrations` | 管理权限 | 创建 |
| PUT | `/integrations/:id` | 管理权限 | 更新 |
| DELETE | `/integrations/:id` | 管理权限 | 删除 |
| POST | `/integrations/:token/alerts` | 公开（token 认证） | 接收告警 Webhook |

**创建请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 集成名称 |
| `description` | string | 否 | 描述 |
| `type` | string | 是 | 类型：`standard`、`alertmanager`、`grafana` |
| `mode` | string | 否 | 默认 `"exclusive"`（专属）或 `"shared"`（共享，需配置路由规则） |
| `channel_id` | uint | 否 | 专属模式下的目标协作空间 ID |
| `pipeline_config` | string (JSON) | 否 | 告警处理管道配置 |
| `label_enhancement_config` | string (JSON) | 否 | 标签增强配置 |
| `is_enabled` | bool | 否 | 是否启用 |

**更新请求体：** 同创建，`type` 和 `mode` 不可修改。

**接收告警 — `POST /integrations/:token/alerts`：**

无需 JWT 认证，通过 URL 中的 token 标识集成。请求体格式取决于集成类型：

- `alertmanager`：标准 Alertmanager webhook 格式
- `grafana`：Grafana webhook 格式
- `standard`：通用 JSON 格式

**响应：**

```json
{ "code": 0, "data": { "received": true } }
```

---

## 36. 路由规则

用于共享集成的告警路由，根据告警属性将告警分发到不同的协作空间。

**模型字段：** `integration_id`、`target_channel_id`、`conditions`（JSON，FilterCondition 数组）、`priority`、`is_enabled`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/routing-rules?integration_id=X` | 已认证 | 按集成查询路由规则列表 |
| POST | `/routing-rules` | 管理权限 | 创建 |
| PUT | `/routing-rules/:id` | 管理权限 | 更新 |
| DELETE | `/routing-rules/:id` | 管理权限 | 删除 |

**创建请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `integration_id` | uint | 是 | 所属集成 ID |
| `target_channel_id` | uint | 是 | 目标协作空间 ID |
| `conditions` | string (JSON) | 否 | FilterCondition 数组 |
| `priority` | int | 否 | 优先级（值越小越先匹配） |
| `is_enabled` | bool | 否 | 是否启用 |

**更新请求体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `target_channel_id` | uint | 目标协作空间 ID |
| `conditions` | string (JSON) | FilterCondition 数组 |
| `priority` | int | 优先级 |
| `is_enabled` | *bool | 是否启用 |

---

## 37. 故障

故障（Incident）是告警响应的核心工作单元，可聚合多个告警事件，支持认领、暂缓、转派、合并和升级。

**模型字段：** `title`、`description`、`severity`（critical | warning | info）、`status`（triggered | processing | closed）、`channel_id`、`labels`（map）、`assigned_to`、`triggered_at`、`acknowledged_at`、`resolved_at`、`closed_at`、`snoozed_until`、`alert_count`、`event_count`、`is_recovered`、`escalation_policy_id`、`current_escalation_step`、`merged_into_id`。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/incidents` | 已认证 | 列表（分页）。筛选：`?channel_id=&status=&severity=&query=&assigned_to=` |
| GET | `/incidents/:id` | 已认证 | 按 ID 获取 |
| POST | `/incidents` | 管理权限 | 手动创建 |
| GET | `/incidents/:id/timeline` | 已认证 | 获取故障时间线 |
| POST | `/incidents/:id/acknowledge` | 操作权限 | 认领 |
| POST | `/incidents/:id/close` | 操作权限 | 关闭 |
| POST | `/incidents/:id/reopen` | 操作权限 | 重开 |
| POST | `/incidents/:id/snooze` | 操作权限 | 暂缓 |
| POST | `/incidents/:id/reassign` | 操作权限 | 转派 |
| POST | `/incidents/:id/merge` | 操作权限 | 合并到目标故障 |
| POST | `/incidents/:id/escalate` | 操作权限 | 手动触发升级 |
| POST | `/incidents/:id/comment` | 操作权限 | 添加评论 |

**创建请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 故障标题 |
| `description` | string | 否 | 描述 |
| `severity` | string | 否 | 默认 `"warning"` |
| `channel_id` | uint | 是 | 所属协作空间 ID |
| `assigned_to` | uint | 否 | 初始指派用户 ID |

**暂缓请求体：**

```json
{ "until": "2026-05-11T10:00:00+08:00" }
```

**转派请求体：**

```json
{ "user_id": 5 }
```

**合并请求体：**

```json
{ "target_id": 42 }
```

**评论请求体：**

```json
{ "content": "正在排查中" }
```

**时间线响应：** 返回时间线条目数组，每条包含 `action`、`actor_id`、`content`、`extra`、`created_at`。

---

## 38. 故障复盘

故障复盘（Post-Mortem）用于记录故障根因分析、影响范围和改进措施。支持 AI 自动生成初稿。

**模型字段：** `incident_id`、`title`、`content`（Markdown）、`status`（draft | published）、`author_id`、`published_at`。

### 故障关联端点

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/incidents/:id/post-mortem` | 已认证 | 获取复盘（不存在时自动创建草稿） |
| PUT | `/incidents/:id/post-mortem` | 操作权限 | 更新复盘内容 |
| POST | `/incidents/:id/post-mortem/publish` | 管理权限 | 发布复盘 |
| POST | `/incidents/:id/post-mortem/ai-generate` | 操作权限 | AI 生成复盘初稿 |
| POST | `/incidents/:id/post-mortem/ai-summary` | 操作权限 | AI 摘要（仅预览，不保存） |

### 全局列表

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/post-mortems` | 已认证 | 复盘列表（分页）。筛选：`?channel_id=&status=` |

**更新请求体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `title` | string | 复盘标题 |
| `content` | string | Markdown 内容 |
| `status` | string | 状态 |

**AI 生成响应：** 返回更新后的复盘对象，`content` 字段包含 AI 生成的 Markdown 初稿（含故障概述、影响、根因分析、解决建议、预防措施）。

**AI 摘要响应：** 返回 AI 分析结果（不保存到复盘），可用于预览。

---

## 39. 告警（v2）

v2 告警模型，与协作空间和故障关联。每个告警可关联多个事件。

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/alerts` | 已认证 | 列表（分页）。筛选：`?channel_id=&incident_id=&status=&severity=&query=` |
| GET | `/alerts/:id` | 已认证 | 按 ID 获取 |
| GET | `/alerts/:id/events` | 已认证 | 获取告警事件列表（分页） |

---

## 40. 批量操作与导出

跨模块的批量操作和数据导出端点。

### 告警规则批量操作

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/alert-rules/batch/enable` | 管理权限 | 批量启用规则 |
| POST | `/alert-rules/batch/disable` | 管理权限 | 批量禁用规则 |
| POST | `/alert-rules/batch/delete` | 管理权限 | 批量删除规则 |
| GET | `/alert-rules/export` | 已认证 | 导出规则为 YAML |
| POST | `/alert-rules/import` | 管理权限 | 从 YAML/JSON 导入规则 |

**批量操作请求体：**

```json
{ "ids": [1, 2, 3] }
```

**批量操作响应：**

```json
{ "code": 0, "data": { "success": 3, "failed": 0 } }
```

### 告警事件批量操作与导出

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/alert-events/batch/acknowledge` | 操作权限 | 批量确认 |
| POST | `/alert-events/batch/close` | 操作权限 | 批量关闭 |
| GET | `/alert-events/export` | 已认证 | 导出事件为 CSV。筛选参数同列表 |
| GET | `/alert-events/groups` | 已认证 | 按规则分组统计 |

**事件分组响应：**

```json
{
  "code": 0,
  "data": [
    { "rule_id": 1, "rule_name": "HighCPU", "severity": "critical", "count": 5 },
    { "rule_id": 2, "rule_name": "DiskFull", "severity": "warning", "count": 3 }
  ]
}
```

### 其他导出

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/schedules/:id/ical` | 已认证 | 导出排班为 iCal 格式 |
| GET | `/metrics` | 公开 | Prometheus 指标（exposition format） |

---

## 43. 宠物系统

### 模型字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | uint | 所属用户 ID |
| `name` | string | 宠物名称，默认"小狐" |
| `species` | string | 物种，默认"fox" |
| `level` | int | 等级，默认 1 |
| `exp` | int | 经验值，默认 0 |
| `hunger` | int | 饥饿度（0=饱, 100=饥饿），默认 30 |
| `mood` | int | 心情（0=悲伤, 100=开心），默认 70 |

### 端点

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/pet` | 已认证 | 获取当前用户宠物（不存在则自动创建） |
| PUT | `/pet` | 已认证 | 更新宠物名称 |
| POST | `/pet/feed` | 已认证 | 喂食宠物（降低饥饿度） |
| POST | `/pet/play` | 已认证 | 和宠物玩耍（提升心情） |
| GET | `/pet/interactions` | 已认证 | 获取互动历史（支持 `?limit=N`，默认 20） |

**更新名称请求体：**

```json
{ "name": "小火狐" }
```

**宠物响应：**

```json
{
  "code": 0,
  "data": {
    "id": 1, "user_id": 5, "name": "小狐", "species": "fox",
    "level": 3, "exp": 120, "hunger": 15, "mood": 85
  }
}
```

---

## 44. 状态页面

### 模型字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 服务名称（必填，最长 128） |
| `status` | string | 状态：`operational` / `degraded` / `outage` / `maintenance` |
| `description` | string | 描述（最长 512） |
| `url` | string | 服务 URL（最长 512） |
| `icon` | string | 图标标识（最长 64） |
| `sort_order` | int | 排序权重，默认 0 |

### 端点

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/status-services` | 已认证 | 获取所有状态服务列表 |
| GET | `/status-services/:id` | 已认证 | 获取单个状态服务详情 |
| POST | `/status-services` | 仅管理员 | 创建状态服务 |
| PUT | `/status-services/:id` | 仅管理员 | 更新状态服务（部分更新） |
| DELETE | `/status-services/:id` | 仅管理员 | 删除状态服务 |

**创建请求体：**

```json
{
  "name": "API Gateway",
  "status": "operational",
  "description": "主网关服务",
  "url": "https://api.example.com/health",
  "icon": "server",
  "sort_order": 1
}
```

---

## 45. 预设规则

### 模型字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 规则标识名（必填，最长 200） |
| `display_name` | string | 显示名称（最长 200） |
| `category` | string | 分类（最长 50） |
| `sub_category` | string | 子分类（最长 50） |
| `component` | string | 组件（最长 50） |
| `expression` | string | PromQL 表达式（必填） |
| `for_duration` | string | 持续时间（最长 32） |
| `severity` | string | 严重等级（最长 20） |
| `alert_type` | string | 告警类型（最长 50） |
| `labels` | json | 标签键值对 |
| `annotations` | json | 注解键值对 |
| `source` | string | 来源（最长 100） |
| `is_builtin` | bool | 是否内置规则 |
| `usage_count` | int | 使用次数 |
| `description` | string | 描述 |

### 端点

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| GET | `/preset-rules` | 已认证 | 分页列表（支持 `?category=&search=&page=&page_size=`） |
| GET | `/preset-rules/categories` | 已认证 | 获取所有分类列表 |
| GET | `/preset-rules/:id` | 已认证 | 获取单条预设规则详情 |
| POST | `/preset-rules/:id/apply` | 管理权限 | 应用预设规则创建 AlertRule（支持覆盖字段） |
| POST | `/preset-rules/import` | 管理权限 | 从 YAML 导入预设规则 |
| DELETE | `/preset-rules/:id` | 管理权限 | 删除预设规则（仅非内置） |

**应用请求体（可选覆盖）：**

```json
{
  "channel_id": 1,
  "labels": {"team": "sre"},
  "for_duration": "10m"
}
```

**应用响应：**

```json
{
  "code": 0,
  "data": { "id": 42, "name": "HighCPU_applied", "rule_id": 1, ... }
}
```

---

## 46. Alertmanager 导入

### 模型字段

此模块无独立数据模型，导入结果写入 Channels 和 InhibitionRules 表。

**导入结果字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `channels_created` | int | 创建的 Channel 数量 |
| `inhibitions_created` | int | 创建的 InhibitionRule 数量 |
| `warnings` | []string | 警告信息 |
| `errors` | []string | 错误信息 |

### 端点

| 方法 | 路由 | 访问级别 | 说明 |
|------|------|----------|------|
| POST | `/integrations/import-alertmanager` | 管理权限 | 导入 Alertmanager YAML 配置 |

**请求方式 1 — JSON body：**

```json
{
  "yaml": "global:\n  resolve_timeout: 5m\nreceivers:\n  - name: email-team\n    email_configs:\n      - to: team@example.com\n..."
}
```

**请求方式 2 — Multipart file upload：**

```
Content-Type: multipart/form-data
file: alertmanager.yml
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "channels_created": 3,
    "inhibitions_created": 1,
    "warnings": [],
    "errors": []
  }
}
```

---

## 路由汇总

| 类别 | 数量 | 访问级别 |
|------|------|----------|
| 公开（无需认证） | 13 | 健康检查、登录、OIDC、Webhook、集成接收、飞书回调、操作页面、Prometheus 指标 |
| 只读（已认证） | 62 | 所有 GET/列表端点（含宠物/状态页面/预设规则） |
| 操作权限（member 及以上） | 22 | 告警操作、故障操作、订阅规则、复盘编辑 |
| 管理权限（team_lead 及以上） | 68 | 配置 CRUD、渠道、规则、排班、团队、Pipeline、集成、路由、预设规则应用/导入、Alertmanager 导入、Event Pipeline |
| 仅管理员 | 16 | 用户 CRUD、系统设置、AI/飞书配置、标签同步、状态页面 CRUD、Pipeline 执行清理 |
| **合计** | **~183** | |
