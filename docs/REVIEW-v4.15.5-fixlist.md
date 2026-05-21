# v4.15.5 Full Review Fix List

> Generated 2026-05-21 | 3 parallel audits: alert engine, diagnostic+AI+Lark, frontend
> Total: **6 P0 · 23 P1 · 28 P2**

---

## P0 — Must Fix

| # | File:Line | Category | Description | Fix |
|---|-----------|----------|-------------|-----|
| 1 | `internal/engine/leader_election.go:41,88-89,117,133` | concurrency | `isLeader` bool read/written from multiple goroutines with zero synchronization — data race | Use `atomic.Bool` or `sync.RWMutex` |
| 2 | `internal/engine/leader_election.go:70-81` | concurrency | `TryAcquire` re-acquisition: `Get`+`Set` not atomic, TTL can expire between them causing split-brain | Replace with Lua script (check-and-extend atomically) |
| 3 | `internal/service/diagnostic_workflow.go:55-67` | data-integrity | `ReplaceSteps` non-atomic delete-then-insert, crash leaves partial/zero steps | Wrap in DB transaction |
| 4 | `internal/pkg/lark/bot_api.go:40-46` | availability | Token TTL `expire-60` can be ≤0 when API returns `expire<=60`, causing refresh storm | Clamp: `if effective < 30 { effective = 30 }` |
| 5 | `internal/service/lark.go:165,191,218,253,301` | performance | Every bot call creates new `BotClient`, each fetches fresh token — hits Lark rate limits | Cache single `BotClient` on `LarkService` |
| 6 | `web/src/composables/useAppNav.ts:300` | bug | Menu key `ai-settings` has no matching route, causes 404 | Change key to `/platform/settings/ai` |

---

## P1 — Should Fix

| # | File:Line | Category | Description | Fix |
|---|-----------|----------|-------------|-----|
| 7 | `internal/engine/evaluator.go:634-649,655-666` | concurrency | `GetFiringEvents` returns mutable `[]*AlertState` pointers, data race after unlock | Deep-copy while holding lock |
| 8 | `internal/engine/evaluator.go:547-575,93-103` | concurrency | Legacy multi-DS: `e.evaluators[rule.ID]` overwrite leaks old goroutine | Key by `(ruleID, datasourceID)` |
| 9 | `internal/engine/evaluator.go:547-575` | concurrency | `PerDS.AddRule` calls `Stop()` then immediately starts new — old goroutine still mid-evaluate | Wait for old goroutine to finish (WaitGroup/done channel) |
| 10 | `internal/engine/rule_eval.go:364-434` | logic-bug | `createAlertEvent`: DB timeout after commit → `EventID=0` → next cycle creates duplicate | Query existing event by `(rule_id, fingerprint, status=firing)` on error |
| 11 | `internal/engine/rule_eval.go:255-278` | logic-bug | Resolution path: `resolveAlertEvent` failure + `sl.state=nil` silently loses alert | Check error before clearing state |
| 12 | `internal/engine/escalation_executor.go:132-133` | timeout | Single 55s timeout for up to 10k events, partial escalation with no retry | Process in smaller batches with per-batch timeout |
| 13 | `internal/engine/escalation_executor.go:288-292` | logic-bug | `break` on `now.Before(dueAt)` assumes steps sorted by delay; wrong order → step never fires | Sort by `DelayMinutes` not `StepOrder`, or use `continue` |
| 14 | `internal/engine/heartbeat_checker.go:234-236` | concurrency | `onAlert` callback blocks heartbeat loop — no other rules checked until it returns | Dispatch via goroutine or worker pool |
| 15 | `internal/service/diagnostic_workflow.go:96` | goroutine-leak | `context.Background()` in `executeRun` goroutine — never cancelled on shutdown | Derive from server lifecycle context with timeout |
| 16 | `internal/handler/larkbot.go:25` | security | Unbounded `io.ReadAll(c.Request.Body)` — OOM on large payload | Use `io.LimitReader(body, 1MB)` |
| 17 | `internal/service/ai.go:435,634,718,801,892` | security | Unbounded `io.ReadAll(resp.Body)` on LLM responses — OOM risk | Wrap with `io.LimitReader(body, 10MB)` |
| 18 | `internal/service/larkbot.go:452` | security | Unbounded `io.ReadAll(resp.Body)` on Lark webhook response | Use `io.LimitReader(body, 1MB)` |
| 19 | `internal/service/alert_event.go:431,441` | concurrency | Notification dispatch goroutines use `context.Background()` — leak on shutdown | Store server-level context, derive from it |
| 20 | `internal/service/alert_event.go:478,484` | concurrency | `triggerLarkCardUpdate` uses `context.Background()` — same leak | Derive from server-level context |
| 21 | `internal/service/larkbot.go:163,222` | performance | `HandleEvent` loads config, then `handleMessageEvent` loads it again — double DB fetch | Pass loaded cfg to `handleMessageEvent` |
| 22 | `internal/service/alert_event.go:119-143` | race | `Acknowledge/Assign/Resolve/Close/Silence`: TOCTOU read-check-update without locking | Use conditional `UPDATE ... WHERE status='firing'`, check `rows_affected` |
| 23 | `internal/service/lark.go:318-337` | logic | `isWithinBusinessHours` uses `time.Now()` (server TZ), ignores parse errors | Accept `*time.Location` param, validate parsed values |
| 24 | `internal/service/alert_event.go:260-272` | logic | `BatchAcknowledge/Close` create timeline for ALL input IDs, not just updated ones | Only insert for actually-updated IDs |
| 25 | `web/src/components/alert-rule/AIGenerateModal.vue:122,139,171,186,219` | i18n | 5 `message.warning/success` calls with hardcoded Chinese strings | Replace with `t()` calls |
| 26 | `web/src/components/alert-rule/AIGenerateModal.vue:367-419` | i18n | ~12 hardcoded Chinese strings in template (dry-run, buttons, labels) | Extract to `t()` with new i18n keys |
| 27 | `web/src/pages/oncall/EscalationPolicies.vue:33-37` | i18n | `targetTypeOptions` hardcoded English labels | Use `t('escalation.user/team/schedule')` |
| 28 | `web/src/pages/oncall/EscalationPolicies.vue:116` | bug | Uses `t('common.saveSuccess')` but key is `common.savedSuccess` — renders raw key | Fix to `t('common.savedSuccess')` |
| 29 | `web/src/pages/oncall/EscalationPolicies.vue:282` | missing-loading | Submit button has no `:loading` binding, no feedback during save | Add `saving` ref + `:loading="saving"` |
| 30 | `web/src/pages/alerts/events/Index.vue:680` | bug | Page change doesn't `clearSelection()`, stale IDs in batch ops | Add `clearSelection()` before `fetchList()` |
| 31 | `web/src/layouts/AppRail.vue:83-94,113,160-170` | a11y | Icon-only buttons lack `aria-label`, screen readers can't identify them | Add `:aria-label` to all icon-only buttons |

---

## P2 — Nice to Fix

| # | File:Line | Category | Description | Fix |
|---|-----------|----------|-------------|-----|
| 32 | `internal/engine/leader_election.go:94-97` | concurrency | `Start()` overwrites `l.cancel` without stopping previous loop — goroutine leak | Guard with `sync.Once` |
| 33 | `internal/engine/evaluator.go:631` | tech-debt | `GetFiringEvents` holds RLock while iterating ALL states — O(rules*fingerprints) | Implement firing index cache |
| 34 | `internal/engine/rule_eval.go:75` | timeout | `evaluate()` derives timeout from `context.Background()` not `re.ctx` | Use `context.WithTimeout(re.ctx, 30s)` |
| 35 | `internal/engine/suppression.go:26` | code-quality | `severityRank` uses `log.Printf` instead of structured zap logger | Accept logger parameter |
| 36 | `internal/engine/suppression.go:179-209` | logic | `RemoveSeverity` requires exact severity match, entry lingers if severity changed | Remove regardless of severity |
| 37 | `internal/engine/escalation_executor.go:87-107` | security | `sendViaChannel` passes unsanitized labels to template renderer | Escape label/annotation values |
| 38 | `internal/engine/escalation_executor.go:105` | code-quality | Notification body hardcoded `"[severity] name - status"`, ignores template config | Use media's template engine |
| 39 | `internal/engine/escalation_executor.go:233-249` | error-handling | `checkSLABreach`: `recordTimeline` fails silently after `UpdateSLAEscalated` succeeds | Same transaction or retry |
| 40 | `internal/engine/heartbeat_checker.go:149-163` | performance | `checkRule` fallback: N+1 `GetByFingerprint` per rule | Retry batch query before fallback |
| 41 | `internal/engine/state_store.go:41-55,58-71` | code-quality | `toStateEntry/fromStateEntry` shallow-copy maps, fragile for future callers | Deep-copy maps |
| 42 | `internal/service/larkbot.go:47-57` | security | `resolveUserID` falls back to user ID 1 silently — unauth Lark user can ack as admin | Log warning, consider rejecting |
| 43 | `internal/service/lark.go:82-93` | error-handling | `GenerateAlertActionToken` failure only logged, card sent without action link | Propagate error or note in card |
| 44 | `internal/service/alert_event.go:336-356` | logic | Duplicate resolve webhook overwrites `ResolvedAt` timestamp | Make idempotent: only set if nil |
| 45 | `internal/service/alert_event.go:359-363` | logic | Firing re-fire updates `FireCount` but not `Labels/Annotations` | Merge incoming labels |
| 46 | `internal/service/system_setting.go:227-279` | cache | Two-layer cache TOCTOU on `GetAIConfig` — acceptable with 30s TTL | Document behavior |
| 47 | `internal/handler/larkbot.go:32-33` | error-handling | All `HandleEvent` errors wrapped as `ErrMissingParam` (400), masks real error type | Map error types to proper codes |
| 48 | `internal/service/larkbot.go:276-329` | performance | `cmdHealth` loads up to 1000 full AlertEvent rows just to count | Push aggregation to DB |
| 49 | `internal/service/larkbot.go:372-393` | performance | `cmdStatus` 4 separate DB round-trips loading full rows for counts | Use `SELECT COUNT(*)` queries |
| 50 | `internal/service/lark.go:326-328` | input-validation | `isWithinBusinessHours` doesn't validate `fmt.Sscanf` return or hour/minute range | Check return, validate 0-23/0-59 |
| 51 | `internal/service/alert_event.go:246-273` | error-handling | `BatchAcknowledge/Close` swallow timeline insert errors | Return partial-success indicator |
| 52 | `internal/service/lark.go:160-162` | error-handling | `SendEnrichedAlertNotificationViaBot` discards specific DB error | Return actual error when `err!=nil` |
| 53 | `internal/handler/larkbot.go:25` | security | No HMAC signature verification, only plaintext token comparison | Implement Lark event signature verification |
| 54 | `web/src/stores/preferences.ts:39-41` | error-handling | `catch {}` silently swallows preference-update errors | Add `console.warn` or toast |
| 55 | `web/src/api/request.ts:14-21` | error-handling | `errorCodeMap` missing 50xxx codes, falls through to raw message | Add `50001/50003` mappings |
| 56 | `web/src/pages/settings/AISettings.vue:412` | i18n | Save button uses past-tense `providerSaved`, should be action label `saveProviders` | Change i18n key |
| 57 | `web/src/pages/settings/SMTPConfig.vue:55` | i18n | Uses `common.success` instead of `common.savedSuccess` — inconsistent | Align to `common.savedSuccess` |
| 58 | `web/src/pages/settings/OIDCConfig.vue:99-101` | code-quality | `testConnection` uses fragile `as Record<string, unknown>` cast | Define properly on API module |
| 59 | `web/src/pages/oncall/EscalationPolicies.vue:239` | i18n | Step tag hardcoded `"Step"` prefix | Use `t('escalation.step', { n })` |
