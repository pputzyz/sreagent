# RBAC 权限体系

> **v4.13.0** | 基于角色的访问控制（Role-Based Access Control）

## 目录

- [权限模型概述](#权限模型概述)
- [权限点总表](#权限点总表)
- [全局角色定义](#全局角色定义)
- [团队角色与权限提升](#团队角色与权限提升)
- [权限合并算法](#权限合并算法)
- [后端实现](#后端实现)
- [前端实现](#前端实现)
- [API 接口](#api-接口)
- [扩展指南](#扩展指南)

---

## 权限模型概述

SREAgent 采用 **全局角色 + 团队角色** 双层权限模型：

```
┌─────────────────────────────────────────────┐
│                 用户 (User)                  │
├─────────────────────────────────────────────┤
│  全局角色 (Global Role)                      │
│  admin / team_lead / member / viewer         │
├─────────────────────────────────────────────┤
│  团队角色 (Team Roles) — 0..N 个团队         │
│  Team A: team_lead                           │
│  Team B: member                              │
│  Team C: admin                               │
├─────────────────────────────────────────────┤
│  有效权限 = Global Perms ∪ max(Team Perms)  │
└─────────────────────────────────────────────┘
```

**核心原则**：
- 全局角色决定基础权限集
- 团队角色只能**提升**权限（不能限制）
- 取所有团队角色中的最高等级，与全局角色的权限取并集
- 前端和后端共用同一套权限字符串定义

---

## 权限点总表

### 规则管理 (rules)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `rules.view` | 查看告警规则 | -- | -- | ✓ | ✓ |
| `rules.create` | 创建告警规则 | ✓ | ✓ | ✓ | -- |
| `rules.edit` | 编辑告警规则 | ✓ | ✓ | -- | -- |
| `rules.delete` | 删除告警规则 | ✓ | -- | -- | -- |
| `rules.manage` | 规则完整管理（创建+编辑+删除） | ✓ | ✓ | -- | -- |

### 事件管理 (events)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `events.view` | 查看告警事件 | -- | -- | -- | ✓ |
| `events.ack` | 确认告警事件 | ✓ | ✓ | ✓ | -- |
| `events.assign` | 分配告警事件 | ✓ | ✓ | ✓ | -- |
| `events.manage` | 事件完整管理 | ✓ | ✓ | -- | -- |

### 事件管理 (incidents)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `incidents.view` | 查看事件 | ✓ | -- | ✓ | ✓ |
| `incidents.create` | 创建事件 | ✓ | ✓ | ✓ | -- |
| `incidents.manage` | 事件完整管理 | ✓ | ✓ | -- | -- |

### 值班排班 (schedules)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `schedules.view` | 查看排班 | -- | -- | ✓ | ✓ |
| `schedules.manage` | 管理排班 | ✓ | ✓ | -- | -- |

### 通知通道 (channels)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `channels.view` | 查看通知通道 | -- | -- | ✓ | ✓ |
| `channels.manage` | 管理通知通道 | ✓ | ✓ | -- | -- |

### 数据源 (datasources)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `datasources.view` | 查看数据源 | -- | ✓ | ✓ | ✓ |
| `datasources.manage` | 管理数据源 | ✓ | -- | -- | -- |

### 仪表盘 (dashboards)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `dashboards.view` | 查看仪表盘 | -- | -- | ✓ | ✓ |
| `dashboards.manage` | 管理仪表盘 | ✓ | ✓ | -- | -- |

### 用户与团队 (users & teams)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `users.manage` | 管理用户 | ✓ | -- | -- | -- |
| `teams.manage` | 管理团队 | ✓ | ✓ | -- | -- |
| `roles.view` | 查看角色 | ✓ | -- | -- | -- |

### 系统设置 (settings)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `settings.manage` | 管理系统设置 | ✓ | -- | -- | -- |
| `audit.view` | 查看审计日志 | ✓ | -- | -- | -- |

### 通知与待办 (notifications & todos)

| 权限点 | 说明 | admin | team_lead | member | viewer |
|--------|------|:-----:|:---------:|:------:|:------:|
| `notifications.view` | 查看通知 | ✓ | ✓ | ✓ | ✓ |
| `todos.view` | 查看待办 | -- | -- | -- | ✓ |
| `todos.manage` | 管理待办 | ✓ | ✓ | ✓ | -- |

---

## 全局角色定义

### admin（管理员）

系统最高权限角色，拥有所有权限点。适用于 SRE 团队负责人和系统管理员。

```go
// rbac.go — PermissionsByGlobalRole("admin")
"users.manage", "teams.manage", "roles.view",
"rules.manage", "rules.create", "rules.edit", "rules.delete",
"events.manage", "events.ack", "events.assign",
"schedules.manage", "channels.manage",
"settings.manage", "audit.view",
"datasources.manage", "dashboards.manage",
"incidents.manage", "incidents.create",
"notifications.view", "todos.manage",
```

**专属能力**：用户管理、系统设置、审计日志查看、数据源管理、规则删除。

### team_lead（团队负责人）

中等权限角色，可管理规则和团队，但不能管理系统设置或用户。

```go
// rbac.go — PermissionsByGlobalRole("team_lead")
"teams.manage",
"rules.manage", "rules.create", "rules.edit",
"events.manage", "events.ack", "events.assign",
"schedules.manage", "channels.manage",
"datasources.view", "dashboards.manage",
"incidents.manage", "incidents.create",
"notifications.view", "todos.manage",
```

**专属能力**：规则编辑、排班管理、通道管理、仪表盘管理。不能删除规则或管理系统设置。

### member（成员）

基础操作角色，可查看和创建规则，确认/分配事件。

```go
// rbac.go — PermissionsByGlobalRole("member")
"rules.view", "rules.create",
"events.ack", "events.assign",
"schedules.view", "channels.view",
"datasources.view", "dashboards.view",
"incidents.view", "incidents.create",
"notifications.view", "todos.manage",
```

**专属能力**：创建规则、确认/分配事件、创建事件。不能编辑/删除规则。

### viewer（只读用户）

最低权限角色，只能查看各类资源。

```go
// rbac.go — PermissionsByGlobalRole("viewer")
"rules.view", "events.view",
"schedules.view", "channels.view",
"datasources.view", "dashboards.view",
"incidents.view",
"notifications.view", "todos.view",
```

**注意**：`viewer` 和 `global_viewer` 是等价角色别名。

### 未知角色

未识别的角色默认只授予最基本的权限：

```go
"notifications.view", "todos.view"
```

---

## 团队角色与权限提升

### 角色等级

系统为每个角色定义了数值等级，用于比较：

| 角色 | 等级 |
|------|------|
| admin | 4 |
| team_lead | 3 |
| member | 2 |
| viewer / global_viewer | 1 |
| 未知 | 0 |

### 权限提升机制

当用户在某个团队中拥有高于全局角色的团队角色时，该团队角色的权限会被**合并**到有效权限集中。

**示例**：
- 用户全局角色：`member`（等级 2）
- 用户在 Team A 的角色：`team_lead`（等级 3）
- 有效权限 = `member 的权限 ∪ team_lead 的权限`

这意味着该用户在全局范围内拥有 team_lead 级别的能力，即使其全局角色只是 member。

### HighestTeamRole

当用户属于多个团队时，系统取所有团队角色中的**最高等级**进行合并：

```go
func HighestTeamRole(teamRoles []string) string {
    best := ""
    bestLevel := 0
    for _, r := range teamRoles {
        if lvl := RoleLevel(r); lvl > bestLevel {
            bestLevel = lvl
            best = r
        }
    }
    return best
}
```

---

## 权限合并算法

```
输入:
  globalRole  — 用户的全局角色
  teamRoles   — 用户在所有团队中的角色列表

步骤:
  1. perms = PermissionsByGlobalRole(globalRole)
  2. highestTeam = HighestTeamRole(teamRoles)
  3. teamPerms = PermissionsByGlobalRole(highestTeam)
  4. for each perm in teamPerms:
       perms[perm] = true   // 只增不减

输出: perms（合并后的权限集）
```

**关键约束**：
- 团队角色只能**增加**权限，不能**撤销**全局角色已有的权限
- 如果用户没有团队角色（`teamRoles` 为空），则只使用全局角色权限
- 权限检查是**精确匹配**，不支持通配符或前缀匹配

---

## 后端实现

### RequirePerm 中间件

`internal/middleware/permission.go` 提供了 `RequirePerm` 中间件，用于在 Gin 路由上进行权限检查。

**使用方式**：

```go
// 在路由注册中使用
router.GET("/rules", middleware.RequirePerm("rules.view"), ruleHandler.List)
router.POST("/rules", middleware.RequirePerm("rules.create"), ruleHandler.Create)
router.PUT("/rules/:id", middleware.RequirePerm("rules.edit"), ruleHandler.Update)
router.DELETE("/rules/:id", middleware.RequirePerm("rules.delete"), ruleHandler.Delete)
```

**执行流程**：

```
请求进入 → JWT 认证中间件设置 user_role + user_team_roles
         → RequirePerm 检查
            1. 从 context 获取 user_role
            2. 快速路径：rbac.HasPerm(globalRole, perm) → 通过则放行
            3. 慢速路径：检查 user_team_roles → rbac.EffectivePerms 合并后检查
            4. 仍不满足 → 返回 403 {"code": 10200, "message": "insufficient permissions: xxx"}
```

**错误响应**：

```json
{
  "code": 10200,
  "message": "insufficient permissions: rules.delete"
}
```

### 权限约定

项目中使用三种权限级别注解端点：

| 注解 | 含义 | 对应权限 |
|------|------|---------|
| `adminOnly` | 仅管理员 | `settings.manage`, `users.manage` 等 |
| `manage` | 管理员 + 团队负责人 | `rules.manage`, `events.manage` 等 |
| `operate` | 管理员 + 团队负责人 + 成员 | `events.ack`, `events.assign` 等 |

---

## 前端实现

### 权限常量

`web/src/permissions.ts` 定义了与后端完全对齐的权限常量：

```typescript
// 规则
export const PERM_RULES_VIEW = 'rules.view'
export const PERM_RULES_CREATE = 'rules.create'
export const PERM_RULES_EDIT = 'rules.edit'
export const PERM_RULES_DELETE = 'rules.delete'
export const PERM_RULES_MANAGE = 'rules.manage'

// 事件
export const PERM_EVENTS_VIEW = 'events.view'
export const PERM_EVENTS_ACK = 'events.ack'
export const PERM_EVENTS_ASSIGN = 'events.assign'
export const PERM_EVENTS_MANAGE = 'events.manage'

// ... 完整列表见 permissions.ts
```

### 权限组（预设组合）

```typescript
/** 可写告警规则 */
export const PERM_RULE_WRITE = [
  PERM_RULES_MANAGE, PERM_RULES_CREATE, PERM_RULES_EDIT, PERM_RULES_DELETE
]

/** 管理员功能 */
export const PERM_ADMIN_FEATURES = [
  PERM_SETTINGS_MANAGE, PERM_USERS_MANAGE
]

/** 事件操作 */
export const PERM_INCIDENT_OPS = [
  PERM_INCIDENTS_MANAGE, PERM_EVENTS_ACK, PERM_EVENTS_ASSIGN
]
```

### v-can 指令

`web/src/directives/vCan.ts` 提供了条件渲染指令，用于根据权限控制 DOM 元素的显示。

**用法**：

```vue
<template>
  <!-- 单个权限 -->
  <n-button v-can="'rules.create'" @click="createRule">
    新建规则
  </n-button>

  <!-- 多个权限（OR 逻辑，满足任一即可） -->
  <n-button v-can="['rules.edit', 'rules.delete']" @click="editRule">
    编辑
  </n-button>

  <!-- 使用预设权限组 -->
  <n-button v-can="PERM_ADMIN_FEATURES" @click="openSettings">
    系统设置
  </n-button>
</template>
```

**行为说明**：
- `v-can="'perm'"` — 单权限检查，使用 `hasPerm()`
- `v-can="['perm1', 'perm2']"` — 多权限检查（OR），使用 `hasAnyPerm()`
- 权限不足时，元素从 DOM 中**移除**（不是隐藏），替换为不可见占位符
- 在 `mounted` 和 `updated` 生命周期钩子中执行检查

### usePermissions 组合式函数

```typescript
import { usePermissions } from '@/composables/usePermissions'

const { hasPerm, hasAnyPerm, perms } = usePermissions()

// 检查单个权限
if (hasPerm('rules.create')) {
  // 允许创建规则
}

// 检查是否有任一权限
if (hasAnyPerm('rules.edit', 'rules.delete')) {
  // 允许编辑或删除
}

// 获取所有权限列表
console.log(perms.value) // ['rules.view', 'rules.create', ...]
```

---

## API 接口

### GET /me/permissions

返回当前用户的完整权限集，包括全局角色、团队信息和合并后的权限列表。

**请求**：

```http
GET /api/v1/me/permissions
Authorization: Bearer <token>
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "role": "member",
    "perms": [
      "rules.view",
      "rules.create",
      "events.ack",
      "events.assign",
      "schedules.view",
      "channels.view",
      "datasources.view",
      "dashboards.view",
      "incidents.view",
      "incidents.create",
      "notifications.view",
      "todos.manage"
    ],
    "teams": [
      {
        "team_id": 1,
        "role": "team_lead"
      },
      {
        "team_id": 3,
        "role": "member"
      }
    ]
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `role` | string | 用户的全局角色 |
| `perms` | string[] | 合并后的有效权限列表（全局 + 团队） |
| `teams` | object[] | 用户所属的团队列表 |
| `teams[].team_id` | uint | 团队 ID |
| `teams[].role` | string | 用户在该团队中的角色 |

**注意**：`perms` 列表是**无序**的，前端应使用 `includes()` 或 `Set` 进行查找。

---

## 扩展指南

### 添加新权限点

1. **后端**：在 `internal/pkg/rbac/rbac.go` 的 `PermissionsByGlobalRole` 中为相关角色添加新权限
2. **前端**：在 `web/src/permissions.ts` 中添加对应的常量
3. **路由**：在 `internal/router/` 中使用 `RequirePerm("new.perm")` 保护端点
4. **文档**：更新本文档的权限点总表

### 权限命名规范

```
<模块>.<动作>

模块: rules, events, incidents, schedules, channels, datasources, dashboards,
      users, teams, settings, audit, notifications, todos
动作: view, create, edit, delete, manage, ack, assign
```

- `manage` 表示该模块的完整管理权限（通常是 create + edit + delete 的超集）
- `view` 是最低权限，几乎所有角色都有

### 注意事项

- 权限字符串是**大小写敏感**的
- 前端权限常量必须与后端 `rbac.go` 中的字符串**完全一致**
- 添加新角色需要更新 `RoleLevel` 函数和 `PermissionsByGlobalRole` 的 switch 分支
- 团队角色的权限提升是**全局生效**的，不是仅限于该团队的资源
