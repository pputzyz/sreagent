package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// NoiseReduceResult is the outcome of noise reduction for one alert event.
type NoiseReduceResult struct {
	// Excluded: the alert matched an exclusion rule and should be dropped.
	Excluded bool
	ExcludeReason string

	// AggregationKey: computed from channel's aggregation dimensions.
	// Empty string means no aggregation — treat the alert individually.
	AggregationKey string

	// StormWarning: storm threshold hit; level is the threshold value exceeded.
	StormWarning bool
	StormLevel   int

	// Flapping: alert is flapping; action depends on channel's flapping mode.
	Flapping     bool
	FlappingMode string // off | notify_only | notify_then_silence
}

// NoiseReducer applies a channel's noise reduction configuration to an
// incoming alert event, returning a NoiseReduceResult that the caller
// (AlertV2Pipeline) uses to decide what to do with the event.
type NoiseReducer struct {
	channelRepo      *repository.ChannelRepository
	exclusionRepo    *repository.ExclusionRuleRepository
	logger           *zap.Logger

	// In-memory flapping tracker: key = "channelID:alertKey", value = flap state
	flapMu     sync.Mutex
	flapStates map[string]*flapState

	// In-memory storm tracker: key = "channelID", value = storm counter
	stormMu      sync.Mutex
	stormCounters map[string]*stormCounter
}

type flapState struct {
	Changes    []time.Time // timestamps of state changes in the observation window
	Silenced   bool
	SilentUntil time.Time
}

type stormCounter struct {
	Count      int
	WindowStart time.Time
	Notified   map[int]bool // which thresholds have already been notified
}

func NewNoiseReducer(
	channelRepo *repository.ChannelRepository,
	exclusionRepo *repository.ExclusionRuleRepository,
	logger *zap.Logger,
) *NoiseReducer {
	return &NoiseReducer{
		channelRepo:   channelRepo,
		exclusionRepo: exclusionRepo,
		logger:        logger,
		flapStates:    make(map[string]*flapState),
		stormCounters: make(map[string]*stormCounter),
	}
}

// Evaluate runs all noise reduction checks for an alert event in the given channel.
func (nr *NoiseReducer) Evaluate(
	ctx context.Context,
	channelID uint,
	alertKey string,
	event *model.AlertEvent,
) NoiseReduceResult {
	result := NoiseReduceResult{}

	// 1. Load channel config
	ch, err := nr.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		nr.logger.Warn("noise_reducer: failed to load channel", zap.Uint("channel_id", channelID), zap.Error(err))
		return result
	}

	// 2. Check exclusion rules
	if excluded, reason := nr.checkExclusion(ctx, channelID, event); excluded {
		result.Excluded = true
		result.ExcludeReason = reason
		return result
	}

	// 3. Compute aggregation key
	result.AggregationKey = nr.computeAggregationKey(ch, event)

	// Only proceed with flapping/storm for firing events
	if event.Status != model.EventStatusFiring {
		return result
	}

	// 4. Storm warning
	result.StormWarning, result.StormLevel = nr.checkStorm(ch, channelID)

	// 5. Flapping detection
	result.Flapping, result.FlappingMode = nr.checkFlapping(ch, channelID, alertKey, event)

	return result
}

// RecordResolution records a resolution event for flapping detection.
func (nr *NoiseReducer) RecordResolution(channelID uint, alertKey string) {
	nr.recordStateChange(channelID, alertKey)
}

// --- Exclusion rules ---

func (nr *NoiseReducer) checkExclusion(ctx context.Context, channelID uint, event *model.AlertEvent) (bool, string) {
	rules, err := nr.exclusionRepo.ListEnabledByChannel(ctx, channelID)
	if err != nil {
		return false, ""
	}
	for _, rule := range rules {
		var conditions []model.FilterCondition
		if err := json.Unmarshal([]byte(rule.Conditions), &conditions); err != nil {
			continue
		}
		if matchAllConditions(conditions, event) {
			return true, rule.Name
		}
	}
	return false, ""
}

// matchAllConditions returns true when all conditions match the alert event.
func matchAllConditions(conditions []model.FilterCondition, event *model.AlertEvent) bool {
	for _, c := range conditions {
		if !matchCondition(c, event) {
			return false
		}
	}
	return true
}

func matchCondition(c model.FilterCondition, event *model.AlertEvent) bool {
	var actual string
	switch {
	case c.Field == "severity":
		actual = string(event.Severity)
	case c.Field == "alertname", c.Field == "title":
		actual = event.AlertName
	case strings.HasPrefix(c.Field, "labels."):
		labelKey := strings.TrimPrefix(c.Field, "labels.")
		actual = event.Labels[labelKey]
	default:
		return false
	}

	switch c.Operator {
	case "eq":
		return actual == c.Value
	case "ne":
		return actual != c.Value
	case "contains":
		return strings.Contains(actual, c.Value)
	case "not_contains":
		return !strings.Contains(actual, c.Value)
	case "regex":
		matched, _ := regexp.MatchString(c.Value, actual)
		return matched
	case "in":
		for _, v := range strings.Split(c.Value, ",") {
			if strings.TrimSpace(v) == actual {
				return true
			}
		}
		return false
	case "not_in":
		for _, v := range strings.Split(c.Value, ",") {
			if strings.TrimSpace(v) == actual {
				return false
			}
		}
		return true
	}
	return false
}

// --- Aggregation key ---

// computeAggregationKey returns an aggregation key if aggregation is enabled,
// otherwise returns an empty string.
func (nr *NoiseReducer) computeAggregationKey(ch *model.Channel, event *model.AlertEvent) string {
	if ch.AggregationConfig == "" {
		return ""
	}

	var cfg model.ChannelNoiseAggregation
	if err := json.Unmarshal([]byte(ch.AggregationConfig), &cfg); err != nil || !cfg.Enabled {
		return ""
	}

	// Determine dimensions to use
	dims := cfg.Dimensions // default: unified dimensions

	if cfg.Mode == "fine_grained" {
		// Check each branch in order; use first matching branch's dimensions
		for _, branch := range cfg.Branches {
			if matchAllConditions(branch.Conditions, event) {
				dims = branch.Dimensions
				break
			}
		}
		if len(dims) == 0 {
			dims = cfg.DefaultDimensions
		}
	}

	if len(dims) == 0 {
		return ""
	}

	var parts []string
	for _, dim := range dims {
		val := event.Labels[dim]
		if cfg.StrictMode || val != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", dim, val))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}

// --- Storm warning ---

// checkStorm increments the per-channel storm counter (rolling 1-minute window)
// and returns true + the threshold value if any configured threshold is exceeded.
func (nr *NoiseReducer) checkStorm(ch *model.Channel, channelID uint) (bool, int) {
	if ch.AggregationConfig == "" {
		return false, 0
	}
	var cfg model.ChannelNoiseAggregation
	if err := json.Unmarshal([]byte(ch.AggregationConfig), &cfg); err != nil || len(cfg.StormThresholds) == 0 {
		return false, 0
	}

	key := fmt.Sprintf("%d", channelID)
	nr.stormMu.Lock()
	defer nr.stormMu.Unlock()

	sc, ok := nr.stormCounters[key]
	if !ok || time.Since(sc.WindowStart) > time.Minute {
		sc = &stormCounter{
			Count:       0,
			WindowStart: time.Now(),
			Notified:    make(map[int]bool),
		}
		nr.stormCounters[key] = sc
	}
	sc.Count++

	for _, threshold := range cfg.StormThresholds {
		if sc.Count >= threshold && !sc.Notified[threshold] {
			sc.Notified[threshold] = true
			return true, threshold
		}
	}
	return false, 0
}

// --- Flapping detection ---

func (nr *NoiseReducer) checkFlapping(ch *model.Channel, channelID uint, alertKey string, event *model.AlertEvent) (bool, string) {
	if ch.FlappingConfig == "" {
		return false, "off"
	}
	var cfg model.ChannelFlappingConfig
	if err := json.Unmarshal([]byte(ch.FlappingConfig), &cfg); err != nil || cfg.Mode == "off" || cfg.Mode == "" {
		return false, "off"
	}

	// Record firing as a state change
	nr.recordStateChange(channelID, alertKey)

	key := fmt.Sprintf("%d:%s", channelID, alertKey)
	windowDur := time.Duration(cfg.WindowMinutes) * time.Minute

	nr.flapMu.Lock()
	defer nr.flapMu.Unlock()

	fs, ok := nr.flapStates[key]
	if !ok {
		fs = &flapState{}
		nr.flapStates[key] = fs
	}

	// Check if currently silenced
	if fs.Silenced && time.Now().Before(fs.SilentUntil) {
		return true, cfg.Mode
	}
	if fs.Silenced && time.Now().After(fs.SilentUntil) {
		fs.Silenced = false
		fs.Changes = nil
	}

	// Prune changes outside window
	cutoff := time.Now().Add(-windowDur)
	active := fs.Changes[:0]
	for _, t := range fs.Changes {
		if t.After(cutoff) {
			active = append(active, t)
		}
	}
	fs.Changes = active

	if len(fs.Changes) >= cfg.MaxChanges {
		if cfg.Mode == "notify_then_silence" && cfg.MuteMinutes > 0 {
			fs.Silenced = true
			fs.SilentUntil = time.Now().Add(time.Duration(cfg.MuteMinutes) * time.Minute)
			fs.Changes = nil // reset after silencing
		}
		return true, cfg.Mode
	}
	return false, cfg.Mode
}

// recordStateChange appends a state-change timestamp for flap tracking.
// Called for both firing and resolution events.
func (nr *NoiseReducer) recordStateChange(channelID uint, alertKey string) {
	key := fmt.Sprintf("%d:%s", channelID, alertKey)
	nr.flapMu.Lock()
	defer nr.flapMu.Unlock()
	fs, ok := nr.flapStates[key]
	if !ok {
		fs = &flapState{}
		nr.flapStates[key] = fs
	}
	fs.Changes = append(fs.Changes, time.Now())
}
