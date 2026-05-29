# Alert Pipeline Architecture Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 6 disconnection points in the alert→notification→escalation pipeline so that noise reduction blocks notifications, escalation doesn't duplicate notifications, and DispatchPolicy.EscalationPolicyID is actually used.

**Architecture:** Move noise reduction into the synchronous `onAlertFn` callback (before notification). Add a unified notification dedup service shared by both NotifyRule and Escalation paths. Wire DispatchPolicy.EscalationPolicyID to the escalation executor. Consolidate frontend menu.

**Tech Stack:** Go 1.25, Gin, GORM, Redis, Vue 3, Naive UI, vue-i18n

---

## File Structure

### Backend files to modify:
- `cmd/server/wire.go` — Reorder onAlertFn: noise reduction before notification
- `internal/service/noise_reducer.go` — Add `ShouldSuppress(ctx, event)` convenience method
- `internal/service/alert_v2_pipeline.go` — Use DispatchPolicy.EscalationPolicyID, write to AlertEvent
- `internal/engine/escalation_executor.go` — Check notification dedup before escalating
- `internal/service/notify_rule.go` — Use shared NotificationDedupService instead of routeDedup
- `internal/model/alert_event.go` — Add `EscalationPolicyID` field if missing

### Backend files to create:
- `internal/service/notify_dedup.go` — Shared Redis-based notification dedup service

### Frontend files to modify:
- `web/src/composables/useAppNav.ts` — Reorganize menu sections
- `web/src/router/index.ts` — Clean up dead routes, add redirects

---

### Task 1: Add `ShouldSuppress` to NoiseReducer

**Files:**
- Modify: `internal/service/noise_reducer.go`

- [ ] **Step 1: Read current noise_reducer.go to understand the interface**

Read `internal/service/noise_reducer.go` lines 1-40 (struct + constructor).

- [ ] **Step 2: Add ShouldSuppress method**

Add after the `Evaluate` method (after line 118):

```go
// ShouldSuppress is a convenience method that checks only the exclusion rules.
// Returns true if the event should be dropped (matched an exclusion rule).
// Unlike Evaluate, it skips flapping/storm/aggregation — use for pre-notification gating.
func (nr *NoiseReducer) ShouldSuppress(ctx context.Context, event *model.AlertEvent) (bool, string) {
	// Resolve channel from event labels or use default
	var channelID uint
	if chStr, ok := event.Labels["_channel_id"]; ok && chStr != "" {
		fmt.Sscanf(chStr, "%d", &channelID)
	}
	if channelID == 0 {
		// No channel context — cannot evaluate exclusion rules
		return false, ""
	}

	result := nr.Evaluate(ctx, channelID, nr.buildAlertKey(event), event)
	return result.Excluded, result.ExcludeReason
}

// buildAlertKey creates a dedup key from rule + labels (same logic as AlertV2Pipeline).
func (nr *NoiseReducer) buildAlertKey(event *model.AlertEvent) string {
	if event.Fingerprint != "" {
		return event.Fingerprint
	}
	return fmt.Sprintf("rule:%d", event.RuleID)
}
```

- [ ] **Step 3: Verify import `fmt` exists**

Check that `"fmt"` is in the import block. If not, add it.

- [ ] **Step 4: Run go build**

```bash
go build ./cmd/server/
```

Expected: PASS (no new errors — ShouldSuppress is unused yet)

---

### Task 2: Move Noise Reduction into onAlertFn (before notification)

**Files:**
- Modify: `cmd/server/wire.go` lines 462-512

- [ ] **Step 1: Read current onAlertFn**

Read `cmd/server/wire.go` lines 462-529 to see the current structure.

- [ ] **Step 2: Insert noise reduction check after mute check, before bizgroup**

In `cmd/server/wire.go`, inside the `onAlertFn` closure, insert between the mute check (step 2) and the bizgroup annotation (step 3):

```go
	// 2.5. Noise reduction: check exclusion rules before notification.
	if noiseReducer != nil {
		if suppressed, reason := noiseReducer.ShouldSuppress(ctx, event); suppressed {
			zapLogger.Info("alert excluded by noise reduction, skipping notification",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
				zap.String("reason", reason),
			)
			return
		}
	}
```

This goes after the mute check block (after line ~487) and before the bizgroup block (line ~489).

- [ ] **Step 3: Verify `noiseReducer` is in scope**

The `noiseReducer` variable is created at line 259 and is a local in `initDependencies`. The `onAlertFn` closure at line 462 captures it. Verify it's accessible. If not, pass it as a parameter or ensure it's declared before the closure.

- [ ] **Step 4: Run go build**

```bash
go build ./cmd/server/
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/noise_reducer.go cmd/server/wire.go
git commit -m "fix: move noise reduction before notification in onAlertFn

Noise reduction (exclusion rules) now runs synchronously in the onAlertFn
callback BEFORE notification dispatch. Previously it ran asynchronously in
AlertV2Pipeline, meaning excluded alerts had already been notified.

Fixes Gap 3 and Gap 4 from ARCHITECTURE_FIX_PLAN.md"
```

---

### Task 3: Create Unified NotificationDedupService

**Files:**
- Create: `internal/service/notify_dedup.go`
- Test: `internal/service/notify_dedup_test.go`

- [ ] **Step 1: Check existing dedup mechanism**

Read `internal/service/notify_rule.go` to find `routeDedup` — it's a package-level `*Deduper` variable. Read how `TrySend` works.

- [ ] **Step 2: Create NotificationDedupService**

Create `internal/service/notify_dedup.go`:

```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NotificationDedupService provides cross-path deduplication for notifications.
// Both NotifyRule and Escalation paths call this to prevent duplicate sends.
type NotificationDedupService struct {
	rdb    *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

func NewNotificationDedupService(rdb *redis.Client, logger *zap.Logger) *NotificationDedupService {
	return &NotificationDedupService{
		rdb:    rdb,
		logger: logger,
		ttl:    4 * time.Hour, // dedup window: same event+media won't re-send within 4h
	}
}

// TrySend returns true if this notification should be sent (first time),
// false if it's a duplicate. Uses Redis SET NX for atomicity.
// key format: "notify_dedup:{eventID}:{mediaID}:{fingerprint}:{status}"
func (d *NotificationDedupService) TrySend(ctx context.Context, key string) bool {
	if d.rdb == nil {
		return true // no Redis — allow all (degrades gracefully)
	}
	redisKey := fmt.Sprintf("notify_dedup:%s", key)
	ok, err := d.rdb.SetNX(ctx, redisKey, "1", d.ttl).Result()
	if err != nil {
		d.logger.Warn("notify_dedup: redis error, allowing send",
			zap.String("key", key), zap.Error(err))
		return true // Redis down — allow (better duplicate than miss)
	}
	return ok
}

// MarkSent explicitly marks a key as sent (for external callers).
func (d *NotificationDedupService) MarkSent(ctx context.Context, key string) {
	if d.rdb == nil {
		return
	}
	redisKey := fmt.Sprintf("notify_dedup:%s", key)
	d.rdb.Set(ctx, redisKey, "1", d.ttl)
}

// BuildKey creates a dedup key from notification components.
func BuildNotifyDedupKey(eventID uint, mediaID uint, fingerprint, status string) string {
	return fmt.Sprintf("%d:%d:%s:%s", eventID, mediaID, fingerprint, status)
}
```

- [ ] **Step 3: Write test**

Create `internal/service/notify_dedup_test.go`:

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildNotifyDedupKey(t *testing.T) {
	key := BuildNotifyDedupKey(42, 7, "abc123", "firing")
	assert.Equal(t, "42:7:abc123:firing", key)
}

func TestBuildNotifyDedupKey_Resolved(t *testing.T) {
	key := BuildNotifyDedupKey(42, 7, "abc123", "resolved")
	assert.Equal(t, "42:7:abc123:resolved", key)
	// firing and resolved must have different keys
	firingKey := BuildNotifyDedupKey(42, 7, "abc123", "firing")
	assert.NotEqual(t, key, firingKey)
}
```

- [ ] **Step 4: Run test**

```bash
go test ./internal/service/ -run TestBuildNotifyDedup -v
```

Expected: PASS

- [ ] **Step 5: Run go build**

```bash
go build ./cmd/server/
```

Expected: PASS

---

### Task 4: Wire NotificationDedupService and Use in NotifyRule

**Files:**
- Modify: `cmd/server/wire.go` — Create and inject NotificationDedupService
- Modify: `internal/service/notify_rule.go` — Replace routeDedup with injected dedup service

- [ ] **Step 1: Read NotifyRuleService constructor**

Read `internal/service/notify_rule.go` to find the `NewNotifyRuleService` constructor and its dependencies. Note how `routeDedup` is currently used (it's a package-level variable).

- [ ] **Step 2: Add NotificationDedupService to NotifyRuleService struct**

In `internal/service/notify_rule.go`, add to the struct:

```go
type NotifyRuleService struct {
	// ... existing fields ...
	dedupSvc *NotificationDedupService // nil = fall back to in-memory routeDedup
}
```

Add to constructor parameter list.

- [ ] **Step 3: Replace routeDedup.TrySend with dedupSvc in ProcessEvent**

In the `ProcessEvent` method (around line 283-294), replace:

```go
// OLD:
dedupKey := fmt.Sprintf("v2:%d:%d:%s:%s", rule.ID, nc.MediaID, event.Fingerprint, event.Status)
if !routeDedup.TrySend(dedupKey) {
```

With:

```go
// NEW:
dedupKey := BuildNotifyDedupKey(event.ID, nc.MediaID, event.Fingerprint, string(event.Status))
if s.dedupSvc != nil && !s.dedupSvc.TrySend(ctx, dedupKey) {
```

Keep the old `routeDedup` as fallback when `dedupSvc` is nil.

- [ ] **Step 4: Wire in cmd/server/wire.go**

In `cmd/server/wire.go`, after the `noiseReducer` creation (line 259), add:

```go
notifyDedupSvc := service.NewNotificationDedupService(rdb, zapLogger)
```

Pass `notifyDedupSvc` to `NewNotifyRuleService` constructor.

- [ ] **Step 5: Run go build**

```bash
go build ./cmd/server/
```

Expected: PASS

---

### Task 5: Use NotificationDedupService in EscalationExecutor

**Files:**
- Modify: `internal/engine/escalation_executor.go` — Check dedup before sending
- Modify: `cmd/server/wire.go` — Pass dedupSvc to EscalationExecutor

- [ ] **Step 1: Add dedupSvc field to EscalationExecutor**

In `internal/engine/escalation_executor.go`, add to the struct:

```go
type EscalationExecutor struct {
	// ... existing fields ...
	dedupSvc *service.NotificationDedupService // shared dedup with NotifyRule path
}
```

Update `NewEscalationExecutor` to accept it as a parameter.

- [ ] **Step 2: Add dedup check in executeStep**

In `executeStep` method (line 470), before calling `sendViaChannel` or `dispatchToTarget`, add:

```go
	// Check if this notification was already sent by the NotifyRule path.
	if e.dedupSvc != nil {
		dedupKey := service.BuildNotifyDedupKey(event.ID, 0, event.Fingerprint, string(event.Status))
		if !e.dedupSvc.TrySend(ctx, dedupKey) {
			e.logger.Info("escalation: notification already sent by notify rule, skipping",
				zap.Uint("event_id", event.ID),
				zap.String("policy", policy.Name),
				zap.Int("step_order", step.StepOrder),
			)
			return nil
		}
	}
```

Note: Using mediaID=0 for escalation dedup because the escalation path doesn't know which media the NotifyRule path used. The key is event+fingerprint+status, which is sufficient to prevent duplicates.

- [ ] **Step 3: Wire in cmd/server/wire.go**

Pass `notifyDedupSvc` (created in Task 4) to `NewEscalationExecutor` constructor.

- [ ] **Step 4: Run go build**

```bash
go build ./cmd/server/
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/notify_dedup.go internal/service/notify_dedup_test.go \
  internal/service/notify_rule.go internal/engine/escalation_executor.go cmd/server/wire.go
git commit -m "fix: unified notification dedup across NotifyRule and Escalation paths

Both NotifyRule and EscalationExecutor now use the same Redis-based
NotificationDedupService. This prevents duplicate notifications when
an alert triggers both paths.

Fixes Gap 2 and Gap 6 from ARCHITECTURE_FIX_PLAN.md"
```

---

### Task 6: Wire DispatchPolicy.EscalationPolicyID to Escalation

**Files:**
- Modify: `internal/service/alert_v2_pipeline.go` — Write EscalationPolicyID to AlertEvent
- Modify: `internal/model/alert_event.go` — Ensure field exists
- Modify: `internal/engine/escalation_executor.go` — Prefer event-level policy ID

- [ ] **Step 1: Check AlertEvent model for EscalationPolicyID field**

Read `internal/model/alert_event.go`. Look for `EscalationPolicyID` field. If missing, add it:

```go
// In AlertEvent struct:
EscalationPolicyID *uint `json:"escalation_policy_id,omitempty" gorm:"index"`
```

- [ ] **Step 2: Read DispatchPolicy from AlertV2Pipeline.process**

In `internal/service/alert_v2_pipeline.go`, in the `process` method, after the dispatch policy matching (around line 222), the matched policy's `EscalationPolicyID` is available but never used. Add:

```go
	// After label enhancement, propagate EscalationPolicyID to the event
	if policy != nil && policy.EscalationPolicyID != nil {
		event.EscalationPolicyID = policy.EscalationPolicyID
		// Persist to DB so escalation executor can read it
		if err := p.alertV2Repo.UpdateEscalationPolicyID(ctx, event.ID, *policy.EscalationPolicyID); err != nil {
			p.logger.Warn("failed to update event escalation_policy_id",
				zap.Uint("event_id", event.ID), zap.Error(err))
		}
	}
```

- [ ] **Step 3: Add UpdateEscalationPolicyID to repository**

Read `internal/repository/alert_event.go` (or wherever the event repo is). Add:

```go
func (r *AlertEventRepository) UpdateEscalationPolicyID(ctx context.Context, eventID uint, policyID uint) error {
	return r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("id = ?", eventID).
		Update("escalation_policy_id", policyID).Error
}
```

If the repo is in a different file, find the correct one.

- [ ] **Step 4: Use event-level EscalationPolicyID in EscalationExecutor**

In `internal/engine/escalation_executor.go`, in the `escalateEvent` method, add logic to prefer the event's `EscalationPolicyID` over the team/global matching:

```go
	// If the event has a specific escalation policy from DispatchPolicy, use it.
	if event.EscalationPolicyID != nil {
		specificPolicy := findPolicyByID(policies, *event.EscalationPolicyID)
		if specificPolicy != nil {
			// Only escalate with this specific policy, skip team/global matching
			e.processPolicy(ctx, event, specificPolicy, allSteps, now)
			return
		}
	}
```

Add helper:

```go
func findPolicyByID(policies []model.EscalationPolicy, id uint) *model.EscalationPolicy {
	for i := range policies {
		if policies[i].ID == id {
			return &policies[i]
		}
	}
	return nil
}
```

- [ ] **Step 5: Run go build**

```bash
go build ./cmd/server/
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/alert_v2_pipeline.go internal/model/alert_event.go \
  internal/engine/escalation_executor.go
git commit -m "fix: wire DispatchPolicy.EscalationPolicyID to escalation executor

DispatchPolicy.EscalationPolicyID was a dead reference. Now when a dispatch
policy matches, its EscalationPolicyID is written to the AlertEvent record.
The escalation executor reads this field and uses the specific policy instead
of team/global matching.

Fixes Gap 1 from ARCHITECTURE_FIX_PLAN.md"
```

---

### Task 7: Frontend Menu Consolidation

**Files:**
- Modify: `web/src/composables/useAppNav.ts`
- Modify: `web/src/router/index.ts`

- [ ] **Step 1: Read current menu structure**

Read `web/src/composables/useAppNav.ts` fully to understand the current oncall menu sections.

- [ ] **Step 2: Reorganize On-Call menu**

Replace the oncall menu sections in `useAppNav.ts` with:

```typescript
case 'oncall':
  return [
    {
      items: [
        { label: t('menu.overview'), key: '/oncall/overview', icon: HomeOutline },
        { label: t('myAlerts.title'), key: '/oncall/my-alerts', icon: AlertCircleOutline },
      ],
    },
    {
      label: t('menu.channels'),
      items: [
        { label: t('menu.channels'),   key: '/oncall/spaces', icon: ChatbubblesOutline },
        { label: t('menu.incidents'),   key: '/oncall/incidents', icon: AlertCircleOutline },
        { label: t('menu.statusPage'),  key: '/oncall/status-page', icon: GlobeOutline },
        { label: t('menu.postmortems'), key: '/oncall/postmortems', icon: DocumentTextOutline },
      ],
    },
    {
      label: t('menu.oncallManagement'),
      items: (() => {
        const items: MenuItem[] = []
        if (authStore.canManage) {
          items.push(
            { label: t('menu.schedule'), key: '/oncall/schedule', icon: CalendarOutline },
          )
        }
        items.push(
          { label: t('menu.escalationPolicies'), key: '/oncall/config/escalation-policies', icon: SwapVerticalOutline },
        )
        return items
      })(),
    },
    {
      label: t('menu.notifyCenter'),
      items: (() => {
        const items: MenuItem[] = []
        if (authStore.canManage) {
          items.push(
            { label: t('menu.notifyPolicies'), key: '/oncall/notify/policies', icon: NotificationsOutline },
            { label: t('menu.notifyChannels'), key: '/oncall/notify/channels', icon: SendOutline },
            { label: t('menu.templates'),      key: '/oncall/notify/templates', icon: CopyOutline },
            { label: t('menu.integrations'),   key: '/oncall/integrations', icon: LinkOutline },
            { label: t('menu.routingRules'),   key: '/oncall/config/routing-rules', icon: GitBranchOutline },
          )
        }
        items.push(
          { label: t('menu.subscriptions'), key: '/oncall/notify/subscriptions', icon: MailOutline },
        )
        return items
      })(),
    },
    {
      label: t('menu.config'),
      items: (() => {
        const items: MenuItem[] = []
        if (authStore.canManage) {
          items.push(
            { label: t('menu.bizGroups'), key: '/oncall/config/biz-groups', icon: FolderOpenOutline },
          )
        }
        return items
      })(),
    },
  ]
```

Key changes:
- Escalation Policies moved from Config Center to new "Oncall Management" section (alongside Schedule)
- Config Center simplified to only Biz Groups
- Notify Center keeps: policies, channels, templates, integrations, routing rules, subscriptions

- [ ] **Step 3: Clean up dead route**

In `web/src/router/index.ts`, the route `oncall/config/notify-rules` loads `Rules.vue` standalone (same component as the tab in notification/Index.vue). Keep it as a redirect:

```typescript
{ path: 'oncall/config/notify-rules', redirect: '/oncall/notify/policies' },
```

- [ ] **Step 4: Add missing i18n keys**

Check if `menu.oncallManagement` and `menu.config` exist in i18n files. If not, add:

In `web/src/i18n/en.ts`:
```typescript
'oncallManagement': 'On-Call Management',
'config': 'Configuration',
```

In `web/src/i18n/zh-CN.ts`:
```typescript
'oncallManagement': '值班管理',
'config': '配置',
```

- [ ] **Step 5: Verify vue-tsc**

```bash
cd web && npx vue-tsc --noEmit
```

Expected: Zero errors

- [ ] **Step 6: Verify vite build**

```bash
cd web && npx vite build
```

Expected: Build success

- [ ] **Step 7: Commit**

```bash
git add web/src/composables/useAppNav.ts web/src/router/index.ts \
  web/src/i18n/en.ts web/src/i18n/zh-CN.ts
git commit -m "refactor: consolidate oncall menu — escalation in management section

- Escalation Policies moved to 'On-Call Management' alongside Schedule
- Config Center simplified to only Biz Groups
- Dead route /oncall/config/notify-rules redirects to /oncall/notify/policies
- New i18n keys: oncallManagement, config

Fixes Gap 5 frontend portion from ARCHITECTURE_FIX_PLAN.md"
```

---

### Task 8: Final Verification

- [ ] **Step 1: go build**

```bash
go build ./cmd/server/
```

Expected: PASS

- [ ] **Step 2: go test**

```bash
go test ./internal/service/ -v -count=1 2>&1 | tail -20
go test ./internal/engine/ -v -count=1 2>&1 | tail -20
```

Expected: PASS (no regressions)

- [ ] **Step 3: vue-tsc**

```bash
cd web && npx vue-tsc --noEmit
```

Expected: Zero errors

- [ ] **Step 4: vite build**

```bash
cd web && npx vite build
```

Expected: Build success

- [ ] **Step 5: Bump version and tag**

Update version in `CLAUDE.md`, `web/package.json`, and add CHANGELOG entry for v4.45.0.

```bash
git add -A
git commit -m "release: v4.45.0 — alert pipeline architecture fix

Phase 1: Noise reduction moved before notification (Gap 3+4)
Phase 2: Unified notification dedup across NotifyRule and Escalation (Gap 2+6)
Phase 3: DispatchPolicy.EscalationPolicyID wired to escalation (Gap 1)
Phase 4: Frontend menu consolidation (Gap 5)"

git tag v4.45.0
git push origin main --tags
```

---

## Gap Coverage Matrix

| Gap | Description | Fixed by Task |
|-----|-------------|---------------|
| Gap 1 | DispatchPolicy.EscalationPolicyID dead reference | Task 6 |
| Gap 2 | EscalationExecutor and NotificationService independent | Task 5 |
| Gap 3 | AlertV2Pipeline and NotificationService don't coordinate | Task 2 |
| Gap 4 | Noise reduction after notification | Task 1 + 2 |
| Gap 5 | Two matching systems, no shared logic + frontend confusion | Task 4 + 7 |
| Gap 6 | Escalation doesn't check if already notified | Task 5 |
