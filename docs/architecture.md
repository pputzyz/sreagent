# 架构设计

> 最后更新：2026-05-08（v2.0.2）

## 系统架构

```
                                ┌─────────────────────────────────────────┐
                                │           Vue 3 Frontend                │
                                │   Naive UI + TypeScript + Pinia         │
                                └──────────────────┬──────────────────────┘
                                                   │ HTTP REST
                                ┌──────────────────▼──────────────────────┐
                                │           API Layer (Gin)               │
                                │   JWT Auth / OIDC / RBAC / CORS         │
                                └──────────────────┬──────────────────────┘
              ┌──────────────────────┬─────────────┴─────────┬────────────────────────┐
              │                      │                       │                        │
  ┌───────────▼─────────┐  ┌─────────▼──────────┐  ┌────────▼────────┐  ┌────────────▼──────────┐
  │   DataSource Svc    │  │   Alert Engine      │  │  OnCall Svc     │  │  Integration Webhook  │
  │   Prom/VM/VLogs     │  │  Evaluator + FSM    │  │ Schedule/Escal  │  │  Standard / AlertMgr  │
  │   Zabbix            │  │  Heartbeat/Suppress │  │                 │  │  Grafana 格式归一化    │
  └───────────┬─────────┘  └─────────┬──────────┘  └────────┬────────┘  └────────────┬──────────┘
              │                      │                       │                        │
              │           ┌──────────▼──────────────┐        │            ┌───────────▼──────────┐
              │           │  WrapOnAlert Hook        │        │            │  RoutingRule 路由     │
              │           │  AlertV2Pipeline         │        │            │  (优先级匹配/目标空间)│
              │           └──────────┬──────────────┘        │            └───────────┬──────────┘
              │                      │                        │                        │
              │           ┌──────────▼──────────────────────────────────────────────────▼──────────┐
              │           │                    AlertV2 处理管道                                     │
              │           │  normalise → rate_limit → applyPipeline → label_enhance                │
              │           │    → upsertAlert → ensureIncident → NoiseReducer → DispatchService     │
              │           └──────────────────────────────────┬─────────────────────────────────────┘
              │                                              │
              │           ┌──────────────────────────────────▼─────────────────────────────────────┐
              │           │                  Channel / Incident / Alert 三层事件模型               │
              │           │   Event（原始）→ Alert（按 alert_key 去重）→ Incident（故障聚合）       │
              │           │                   └── Channel（协作空间）                              │
              │           └──────────────────────────────────┬─────────────────────────────────────┘
              │                                              │
  ┌───────────▼──────────────────────────────────────────────▼─────────────────────────────────────┐
  │                                         Redis 7                                                │
  │   引擎状态持久化 (Hash per rule) · 风暴预警滚动计数器 · 节流                                   │
  └──────────────────────────────────────────────┬─────────────────────────────────────────────────┘
                                                 │
  ┌──────────────────────────────────────────────▼─────────────────────────────────────────────────┐
  │                                        MySQL 8.0                                               │
  │   33 张表 · golang-migrate 000001-000033 管理 · GORM v2 ORM                                   │
  │   v2 新增: channels / incidents / alerts / integrations / routing_rules / dispatch_policies    │
  └────────────────────────────────────────────────────────────────────────────────────────────────┘

              ┌────────────────────────────────────────────────────────────┐
              │                       外部集成                              │
              │  Lark Bot+Hook · LLM API · Email SMTP · Custom Webhooks   │
              │  Keycloak OIDC · Prometheus / VictoriaMetrics / Zabbix    │
              └────────────────────────────────────────────────────────────┘
```

## 四层事件模型

```
Event（原始事件）
  → Alert（去重聚合，按 alert_key，关联 Channel）
    → Incident（故障，多 Alert 聚合，完整生命周期）
      → Channel（协作空间，多 Incident，排除/分派/降噪配置）
```

- **Event**：从引擎或 Integration Webhook 接收的原始告警信号
- **Alert**：按 `alert_key`（规则ID + 指纹）去重，持续更新 `last_fired_at`，不产生重复噪音
- **Incident**：关联一批同类 Alert 的故障对象，支持 ack / close / reopen / snooze / merge / reassign / escalate
- **Channel**：协作空间，一个 Channel 关联多个 Incident，含降噪规则、排除规则、分派策略

## 关键架构决策

| ADR | 决策 | 原因 |
|-----|------|------|
| ADR-1 | AI/Lark 配置存 DB（system_settings），AES-256-GCM 加密 | 密钥不出现在 ConfigMap/Secret |
| ADR-2 | golang-migrate 是 schema 唯一来源，GORM AutoMigrate 只作安全网 | 迁移可审计可回滚 |
| ADR-3 | Redis Hash 持久化引擎状态 | 重启后恢复飞行中告警，Redis 不可用时降级到纯内存 |
| ADR-4 | OIDC 配置存 DB，启动时合并 configmap | 运行时配置无需重启 |
| ADR-5 | RBAC 三级权限（adminOnly/manage/operate） | 精细权限控制 |
| ADR-6 | AlertV2Pipeline 非侵入式 hook（WrapOnAlert） | 原引擎无需修改，v1/v2 路径并行，向前兼容 |
| ADR-7 | Channel/Incident/Alert 三层事件模型（参照 FlashCat） | 结构化故障协作，减少告警噪音，支持根因聚合 |
| ADR-8 | 共享集成路由规则（RoutingRule 表，优先级匹配） | 一个 Integration 可路由到多个 Channel，减少重复配置 |
| ADR-9 | 分派策略独立于 Channel，多策略优先级排序匹配 | 灵活配置不同条件/时间的通知策略，降低耦合 |
| ADR-10 | NoiseReducer in-memory flapState + Redis 滚动计数器 | 风暴预警不依赖 DB 写入，抖动检测低延迟 |

## 数据库迁移文件

迁移文件路径：`internal/pkg/dbmigrate/migrations/`，当前共 **33 个**（000001-000033）

v2 新增（000019-000033）：

| 序号 | 文件 | 创建的表 |
|------|------|----------|
| 000019 | create_channels | channels |
| 000020 | create_channel_stars | channel_stars |
| 000021 | create_channel_exclusion_rules | channel_exclusion_rules |
| 000022 | create_incidents | incidents |
| 000023 | create_incident_assignees | incident_assignees |
| 000024 | create_incident_timelines | incident_timelines |
| 000025 | create_post_mortems | post_mortems |
| 000026 | create_alerts_v2 | alerts（v2） |
| 000027 | create_alert_events_v2 | alert_events_v2 |
| 000028 | create_integrations | integrations |
| 000029 | create_routing_rules | routing_rules |
| 000030 | seed_default_channel | INSERT default channel |
| 000031 | create_dispatch_policies | dispatch_policies |
| 000032 | create_dispatch_logs | dispatch_logs |
| 000033 | alert_rule_channel | ALTER alert_rules ADD channel_id |

## 告警引擎状态机

```
inactive → pending（for_duration）→ firing → recovery_hold → resolved
                                        └── nodata（数据缺失）
```

- 每规则一个 goroutine，Evaluator 管理协程池
- LevelSuppressor 基于严重级别去重
- HeartbeatChecker 心跳超时检测
- EscalationExecutor SLA 超时自动升级
- AlertGroupManager group_wait/interval 通知分组

## AlertV2 处理管道

```
Integration Webhook
  → normalise（格式归一化：Standard / AlertManager / Grafana）
  → rate limit（令牌桶 100/s + 1000/min）
  → applyPipeline（rewrite_severity / title / desc / drop）
  → DispatchService.ApplyLabelEnhancements（标签增强：extract / combine / map / delete）
  → upsertAlert（按 alert_key 去重合入 Alert 表）
  → ensureIncident（复用/新建 Incident）
  → NoiseReducer（聚合 / 风暴预警 / 抖动检测）
  → DispatchService.FindMatchingPolicy（分派策略匹配：条件 / 时间 / 延迟）
  → sendNotification（按分派策略通知）
```

原有告警引擎通过 **WrapOnAlert hook** 注入，产生的告警同样经过上述 v2 管道处理。

## 通知管道

```
v1 路径（保留兼容）：
Engine fires
  → AlertGroupManager → Inhibition → Mute → RouteAlert
    → v1 策略管道（NotifyChannel + NotifyPolicy）
    → v2 规则管道（NotifyRule → NotifyMedia）
    → 订阅管道（SubscribeRule → NotifyRule）
    → SendNotification（lark_webhook / lark_bot / email / webhook / script）

v2 路径（并行）：
Engine fires / Integration Webhook
  → AlertV2Pipeline.WrapOnAlert / Receive
    → normalise → rate_limit → applyPipeline → label_enhance
    → upsertAlert → ensureIncident
    → NoiseReducer → DispatchService
    → SendNotification（lark_webhook / lark_bot / email / webhook）
```

v1 路径和 v2 路径并行运行，互不影响。WrapOnAlert 在原有 `SetOnAlert` 回调完成后触发，零侵入原有引擎逻辑。
