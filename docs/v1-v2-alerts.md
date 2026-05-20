# v1/v2 Alert Evaluation Dual-Track

> Last updated: 2026-05-20 | SREAgent v4.14.0

## Overview

SREAgent supports two alert evaluation engines running in parallel:

| Aspect | v1 (Legacy) | v2 (Pipeline) |
|--------|-------------|---------------|
| Architecture | Per-rule goroutine (`RuleEvaluator`) | Centralized pipeline (`AlertV2Pipeline`) |
| State management | In-memory `sync.Map` per evaluator | Shared `StateStore` (Redis-backed) |
| Concurrency | 1 goroutine per rule | Worker pool with bounded concurrency |
| Datasource routing | Single datasource per rule | Multi-datasource with fallback |
| Grouping | None | Label-based group_wait / group_interval |
| Suppression | Per-evaluator `LevelSuppressor` | Centralized `SuppressionEngine` |
| Status | Stable, default | Opt-in via `rule_type=v2` |

## When to Use Which

### v1 (Default)
- Simple threshold-based alerts
- Single datasource per rule
- Low rule count (< 200)
- No grouping/silencing requirements

### v2 (Pipeline)
- High rule count (200+)
- Multi-datasource alerts
- Label-based grouping and routing
- Complex suppression requirements
- Need for centralized state management

## Migration Path

### Step 1: Enable v2 for new rules
New rules can be created with `rule_type=v2` in the alert rule configuration. Existing rules continue using v1.

### Step 2: Gradual migration
Individual rules can be migrated by updating their `rule_type` field from `threshold` (v1 default) to `v2_pipeline`.

### Step 3: Full migration (future)
A future release will provide a batch migration tool. Until then, both engines run side-by-side.

## Configuration

```yaml
engine:
  # v1 evaluator settings
  eval_interval: 60s
  max_concurrent_evals: 50

  # v2 pipeline settings (opt-in)
  v2:
    enabled: true
    worker_count: 10
    group_wait: 30s
    group_interval: 5m
```

## Key Files

| Component | v1 | v2 |
|-----------|----|----|
| Evaluator | `engine/rule_eval.go` | `service/alert_v2_pipeline.go` |
| State | In-memory `sync.Map` | `engine/state_store.go` |
| Suppression | `engine/suppression.go` | `engine/suppression.go` (shared) |
| Heartbeat | `engine/heartbeat_checker.go` | `engine/heartbeat_checker.go` (shared) |
| Escalation | `engine/escalation_executor.go` | `engine/escalation_executor.go` (shared) |

## Metrics

Both engines report to the same Prometheus metrics:

- `sreagent_alerts_evaluated_total{rule_id, result}` — evaluation count
- `sreagent_engine_last_heartbeat_timestamp` — deadman switch
- `sreagent_engine_leader_status` — leader election status
- `sreagent_heartbeat_checks_total{result}` — heartbeat check results

## FAQ

**Q: Can I switch a rule from v1 to v2 without downtime?**
A: Yes. Update the `rule_type` field. The v1 evaluator will stop picking it up, and the v2 pipeline will start.

**Q: Do v1 and v2 share the same alert events table?**
A: Yes. Both write to `alert_events` with the same schema. The `source` field distinguishes the origin.

**Q: What happens to v1 when I enable v2?**
A: v1 continues running for all rules with `rule_type=threshold`. Only rules explicitly set to `v2_pipeline` use the new engine.
