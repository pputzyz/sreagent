// Package muterule provides shared mute-rule matching logic used by both
// the engine (suppression) and the service layer. Extracting this code
// eliminates the duplication that previously existed between
// internal/engine/suppression.go and internal/service/mute_rule.go.
package muterule

import (
	"strconv"
	"strings"
	"time"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// LoadMuteTimezone loads a timezone with a consistent fallback to Asia/Shanghai.
// Previously lived in internal/service; moved here so both service and engine
// can use it without an import cycle.
func LoadMuteTimezone(name string) *time.Location {
	if name == "" {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		if loc != nil {
			return loc
		}
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Shanghai")
		if loc != nil {
			return loc
		}
		return time.Local
	}
	return loc
}

// IsTimeWindowMuted checks whether now falls inside the mute rule's time window.
// Supports one-time windows (StartTime/EndTime), periodic windows
// (PeriodicStart/PeriodicEnd), optional day-of-week filtering, and timezone
// awareness. When no time fields are set it returns true (always muted).
func IsTimeWindowMuted(muteRule *model.MuteRule, now time.Time) bool {
	loc := LoadMuteTimezone(muteRule.Timezone)
	nowLocal := now.In(loc)

	// One-time window: if both StartTime and EndTime are set, check them.
	if muteRule.StartTime != nil && muteRule.EndTime != nil {
		if nowLocal.Before(*muteRule.StartTime) || nowLocal.After(*muteRule.EndTime) {
			return false
		}
		return true
	}

	// Periodic window: PeriodicStart + PeriodicEnd + optional DaysOfWeek.
	if muteRule.PeriodicStart != "" && muteRule.PeriodicEnd != "" {
		// Day-of-week filter (1=Mon ... 7=Sun, matching ISO 8601).
		if muteRule.DaysOfWeek != "" {
			weekday := int(nowLocal.Weekday())
			if weekday == 0 {
				weekday = 7 // Sunday = 7
			}
			days := strings.Split(muteRule.DaysOfWeek, ",")
			dayMatch := false
			for _, d := range days {
				if dayNum, err := strconv.Atoi(strings.TrimSpace(d)); err == nil {
					if dayNum == weekday {
						dayMatch = true
						break
					}
				}
			}
			if !dayMatch {
				return false
			}
		}

		// Parse periodic times (HH:MM format).
		start, errS := time.Parse("15:04", muteRule.PeriodicStart)
		end, errE := time.Parse("15:04", muteRule.PeriodicEnd)
		if errS != nil || errE != nil {
			return false
		}

		currentMinutes := nowLocal.Hour()*60 + nowLocal.Minute()
		startMinutes := start.Hour()*60 + start.Minute()
		endMinutes := end.Hour()*60 + end.Minute()

		if startMinutes <= endMinutes {
			// Normal range: e.g., 02:00-06:00 (left-closed, right-open).
			return currentMinutes >= startMinutes && currentMinutes < endMinutes
		}
		// Overnight range: e.g., 22:00-06:00.
		return currentMinutes >= startMinutes || currentMinutes < endMinutes
	}

	// No time restriction — always muted (label/severity already matched).
	return true
}

// IsMutedByRule checks whether an alert event matches a single mute rule.
// The check combines four criteria (all must pass):
//  1. Rule ID filter — if the mute rule targets specific rule IDs, event's rule must be among them.
//  2. Label matching — event must carry ALL labels specified in the mute rule.
//  3. Severity filter — event severity must be listed (if the filter is non-empty).
//  4. Time window — current time must be inside the rule's time window.
func IsMutedByRule(rule *model.MuteRule, eventLabels map[string]string, eventSeverity string, ruleID *uint, now time.Time) bool {
	// 1. Check specific rule IDs if set.
	if rule.RuleIDs != "" && ruleID != nil {
		ruleIDs := strings.Split(rule.RuleIDs, ",")
		matched := false
		for _, idStr := range ruleIDs {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
				if uint(id) == *ruleID {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	// 2. Label matching — alert must match ALL labels in the mute rule.
	if !labelmatch.Match(eventLabels, map[string]string(rule.MatchLabels)) {
		return false
	}

	// 3. Severity filter.
	if rule.Severities != "" {
		sevs := strings.Split(rule.Severities, ",")
		matched := false
		for _, sev := range sevs {
			if strings.TrimSpace(sev) == eventSeverity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 4. Time window check (day-of-week + time range + timezone).
	if !IsTimeWindowMuted(rule, now) {
		return false
	}

	return true
}
