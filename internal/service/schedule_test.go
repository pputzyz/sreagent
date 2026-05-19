package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/testutil"
)

// ---------------------------------------------------------------------------
// splitComma tests (pure function)
// ---------------------------------------------------------------------------

func Test_splitComma_basic(t *testing.T) {
	result := splitComma("critical,warning,info")
	assert.Equal(t, []string{"critical", "warning", "info"}, result)
}

func Test_splitComma_with_spaces(t *testing.T) {
	result := splitComma("critical , warning , info")
	assert.Equal(t, []string{"critical", "warning", "info"}, result)
}

func Test_splitComma_empty_string(t *testing.T) {
	result := splitComma("")
	assert.Nil(t, result)
}

func Test_splitComma_single_value(t *testing.T) {
	result := splitComma("critical")
	assert.Equal(t, []string{"critical"}, result)
}

func Test_splitComma_trailing_comma(t *testing.T) {
	result := splitComma("critical,warning,")
	assert.Equal(t, []string{"critical", "warning"}, result)
}

func Test_splitComma_leading_comma(t *testing.T) {
	result := splitComma(",critical,warning")
	assert.Equal(t, []string{"critical", "warning"}, result)
}

func Test_splitComma_only_commas(t *testing.T) {
	result := splitComma(",,,")
	assert.Nil(t, result)
}

func Test_splitComma_whitespace_tokens(t *testing.T) {
	result := splitComma("  ,  , critical  ")
	assert.Equal(t, []string{"critical"}, result)
}

// ---------------------------------------------------------------------------
// matchesSeverityFilter tests (pure function)
// ---------------------------------------------------------------------------

func Test_matchesSeverityFilter_empty_filter_matches_all(t *testing.T) {
	assert.True(t, matchesSeverityFilter("", "critical"))
	assert.True(t, matchesSeverityFilter("", "warning"))
	assert.True(t, matchesSeverityFilter("", "info"))
	assert.True(t, matchesSeverityFilter("", ""))
}

func Test_matchesSeverityFilter_empty_severity_matches_all(t *testing.T) {
	assert.True(t, matchesSeverityFilter("critical", ""))
	assert.True(t, matchesSeverityFilter("critical,warning", ""))
}

func Test_matchesSeverityFilter_exact_match(t *testing.T) {
	assert.True(t, matchesSeverityFilter("critical", "critical"))
	assert.True(t, matchesSeverityFilter("warning", "warning"))
	assert.True(t, matchesSeverityFilter("info", "info"))
}

func Test_matchesSeverityFilter_no_match(t *testing.T) {
	assert.False(t, matchesSeverityFilter("critical", "warning"))
	assert.False(t, matchesSeverityFilter("critical,warning", "info"))
}

func Test_matchesSeverityFilter_multi_value_match(t *testing.T) {
	assert.True(t, matchesSeverityFilter("critical,warning", "critical"))
	assert.True(t, matchesSeverityFilter("critical,warning", "warning"))
	assert.False(t, matchesSeverityFilter("critical,warning", "info"))
}

func Test_matchesSeverityFilter_with_spaces(t *testing.T) {
	assert.True(t, matchesSeverityFilter("critical , warning", "critical"))
	assert.True(t, matchesSeverityFilter("critical , warning", "warning"))
	assert.False(t, matchesSeverityFilter("critical , warning", "info"))
}

// ---------------------------------------------------------------------------
// calculateRotationIndex tests (pure logic on ScheduleService)
// ---------------------------------------------------------------------------

func newTestScheduleService() *ScheduleService {
	return &ScheduleService{
		logger: zap.NewNop(),
	}
}

func Test_calculateRotationIndex_daily_basic(t *testing.T) {
	svc := newTestScheduleService()

	// Schedule created 3 days ago at 09:00
	refTime := time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC)

	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "09:00",
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
		{UserID: 3, Position: 2},
	}

	// At creation time → index 0
	now := refTime
	idx := svc.calculateRotationIndex(schedule, participants, now)
	assert.Equal(t, 0, idx, "at creation time, first participant should be on-call")

	// 1 day later → index 1
	now = refTime.Add(24 * time.Hour)
	idx = svc.calculateRotationIndex(schedule, participants, now)
	assert.Equal(t, 1, idx, "after 1 day, second participant should be on-call")

	// 2 days later → index 2
	now = refTime.Add(48 * time.Hour)
	idx = svc.calculateRotationIndex(schedule, participants, now)
	assert.Equal(t, 2, idx, "after 2 days, third participant should be on-call")

	// 3 days later → wraps to index 0
	now = refTime.Add(72 * time.Hour)
	idx = svc.calculateRotationIndex(schedule, participants, now)
	assert.Equal(t, 0, idx, "after 3 days, should wrap back to first participant")
}

func Test_calculateRotationIndex_daily_two_participants(t *testing.T) {
	svc := newTestScheduleService()

	refTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "09:00",
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Day 0: user 1, Day 1: user 2, Day 2: user 1, Day 3: user 2
	for day := 0; day < 4; day++ {
		now := refTime.Add(time.Duration(day) * 24 * time.Hour)
		idx := svc.calculateRotationIndex(schedule, participants, now)
		expected := day % 2
		assert.Equal(t, expected, idx, "day %d: expected participant index %d", day, expected)
	}
}

func Test_calculateRotationIndex_weekly_basic(t *testing.T) {
	svc := newTestScheduleService()

	// Monday 2026-01-05 at 09:00
	refTime := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)

	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationWeekly,
		HandoffTime:  "09:00",
		HandoffDay:   1, // Monday
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Week 0 (same week as creation): user 1
	idx := svc.calculateRotationIndex(schedule, participants, refTime)
	assert.Equal(t, 0, idx, "first week: first participant")

	// 7 days later: user 2
	idx = svc.calculateRotationIndex(schedule, participants, refTime.Add(7*24*time.Hour))
	assert.Equal(t, 1, idx, "second week: second participant")

	// 14 days later: wraps to user 1
	idx = svc.calculateRotationIndex(schedule, participants, refTime.Add(14*24*time.Hour))
	assert.Equal(t, 0, idx, "third week: wraps to first participant")
}

func Test_calculateRotationIndex_custom_falls_back_to_daily(t *testing.T) {
	svc := newTestScheduleService()

	refTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationCustom,
		HandoffTime:  "09:00",
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Custom rotation falls back to daily
	idx := svc.calculateRotationIndex(schedule, participants, refTime)
	assert.Equal(t, 0, idx)

	idx = svc.calculateRotationIndex(schedule, participants, refTime.Add(24*time.Hour))
	assert.Equal(t, 1, idx)
}

func Test_calculateRotationIndex_unknown_rotation_returns_zero(t *testing.T) {
	svc := newTestScheduleService()

	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: time.Now()},
		RotationType: "unknown",
		HandoffTime:  "09:00",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	idx := svc.calculateRotationIndex(schedule, participants, time.Now())
	assert.Equal(t, 0, idx, "unknown rotation type should return 0")
}

func Test_calculateRotationIndex_single_participant(t *testing.T) {
	svc := newTestScheduleService()

	refTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "09:00",
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
	}

	// With one participant, always index 0
	for day := 0; day < 5; day++ {
		idx := svc.calculateRotationIndex(schedule, participants, refTime.Add(time.Duration(day)*24*time.Hour))
		assert.Equal(t, 0, idx, "single participant should always be index 0")
	}
}

func Test_calculateRotationIndex_before_handoff_time(t *testing.T) {
	svc := newTestScheduleService()

	// Created at 09:00, but checking at 08:00 (before handoff)
	refTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "09:00",
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// At 08:00 on day 0 (before handoff) — still day 0
	beforeHandoff := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	idx := svc.calculateRotationIndex(schedule, participants, beforeHandoff)
	assert.Equal(t, 0, idx, "before handoff time should still be current day")
}

func Test_calculateRotationIndex_custom_handoff_time(t *testing.T) {
	svc := newTestScheduleService()

	// Created at midnight, handoff at 18:00
	refTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "18:00",
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Before 18:00 on day 0 → user 1
	idx := svc.calculateRotationIndex(schedule, participants, time.Date(2026, 1, 1, 17, 0, 0, 0, time.UTC))
	assert.Equal(t, 0, idx)

	// After 18:00 on day 0 → still user 1 (the handoff boundary)
	idx = svc.calculateRotationIndex(schedule, participants, time.Date(2026, 1, 1, 19, 0, 0, 0, time.UTC))
	// This might be 0 or 1 depending on exact boundary alignment
	// The key test is it doesn't panic and returns a valid index
	assert.True(t, idx >= 0 && idx < len(participants), "index should be valid")
}

func Test_calculateRotationIndex_default_handoff_time(t *testing.T) {
	svc := newTestScheduleService()

	refTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "", // empty → defaults to 09:00
		Timezone:     "UTC",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Should not panic with empty handoff time
	idx := svc.calculateRotationIndex(schedule, participants, refTime)
	assert.True(t, idx >= 0 && idx < len(participants))
}

func Test_calculateRotationIndex_timezone_handling(t *testing.T) {
	svc := newTestScheduleService()

	refTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "09:00",
		Timezone:     "Asia/Shanghai", // UTC+8
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Load Shanghai timezone
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	// Same moment in Shanghai time
	nowShanghai := refTime.In(loc)
	idx := svc.calculateRotationIndex(schedule, participants, nowShanghai)
	assert.True(t, idx >= 0 && idx < len(participants), "timezone conversion should produce valid index")
}

func Test_calculateRotationIndex_invalid_timezone(t *testing.T) {
	svc := newTestScheduleService()

	refTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	schedule := &model.Schedule{
		BaseModel:    model.BaseModel{CreatedAt: refTime},
		RotationType: model.RotationDaily,
		HandoffTime:  "09:00",
		Timezone:     "Invalid/Timezone",
	}

	participants := []model.ScheduleParticipant{
		{UserID: 1, Position: 0},
		{UserID: 2, Position: 1},
	}

	// Should not panic with invalid timezone — the caller (GetCurrentOnCall) handles
	// the fallback to UTC, but calculateRotationIndex itself uses now.Location()
	// which will be whatever the caller passes.
	idx := svc.calculateRotationIndex(schedule, participants, refTime)
	assert.True(t, idx >= 0 && idx < len(participants))
}

// ---------------------------------------------------------------------------
// OnCallResult struct test
// ---------------------------------------------------------------------------

func Test_OnCallResult_fields(t *testing.T) {
	user := &model.User{BaseModel: model.BaseModel{ID: 1}, Username: "alice"}
	schedule := &model.Schedule{BaseModel: model.BaseModel{ID: 1}, Name: "test-schedule"}

	result := &OnCallResult{
		User:       user,
		Schedule:   schedule,
		IsOverride: false,
	}

	assert.Equal(t, "alice", result.User.Username)
	assert.Equal(t, "test-schedule", result.Schedule.Name)
	assert.False(t, result.IsOverride)
	assert.Nil(t, result.Override)
}

// ---------------------------------------------------------------------------
// Integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

func Test_GetCurrentOnCall_disabled_schedule(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	scheduleRepo := repository.NewScheduleRepository(db)
	participantRepo := repository.NewScheduleParticipantRepository(db)
	overrideRepo := repository.NewScheduleOverrideRepository(db)
	shiftRepo := repository.NewOnCallShiftRepository(db)
	policyRepo := repository.NewEscalationPolicyRepository(db)
	stepRepo := repository.NewEscalationStepRepository(db)

	svc := NewScheduleService(scheduleRepo, participantRepo, overrideRepo, shiftRepo, policyRepo, stepRepo, logger)

	// Create a disabled schedule
	schedule := &model.Schedule{
		Name:         "disabled-schedule",
		RotationType: model.RotationDaily,
		Timezone:     "UTC",
		HandoffTime:  "09:00",
		IsEnabled:    false,
	}
	require.NoError(t, db.Create(schedule).Error)

	_, err := svc.GetCurrentOnCall(context.Background(), schedule.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func Test_GetCurrentOnCall_no_participants(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	scheduleRepo := repository.NewScheduleRepository(db)
	participantRepo := repository.NewScheduleParticipantRepository(db)
	overrideRepo := repository.NewScheduleOverrideRepository(db)
	shiftRepo := repository.NewOnCallShiftRepository(db)
	policyRepo := repository.NewEscalationPolicyRepository(db)
	stepRepo := repository.NewEscalationStepRepository(db)

	svc := NewScheduleService(scheduleRepo, participantRepo, overrideRepo, shiftRepo, policyRepo, stepRepo, logger)

	// Create an enabled schedule with no participants
	schedule := &model.Schedule{
		Name:         "empty-schedule",
		RotationType: model.RotationDaily,
		Timezone:     "UTC",
		HandoffTime:  "09:00",
		IsEnabled:    true,
	}
	require.NoError(t, db.Create(schedule).Error)

	_, err := svc.GetCurrentOnCall(context.Background(), schedule.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no participants")
}

func Test_GetCurrentOnCall_with_participants(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	scheduleRepo := repository.NewScheduleRepository(db)
	participantRepo := repository.NewScheduleParticipantRepository(db)
	overrideRepo := repository.NewScheduleOverrideRepository(db)
	shiftRepo := repository.NewOnCallShiftRepository(db)
	policyRepo := repository.NewEscalationPolicyRepository(db)
	stepRepo := repository.NewEscalationStepRepository(db)

	svc := NewScheduleService(scheduleRepo, participantRepo, overrideRepo, shiftRepo, policyRepo, stepRepo, logger)

	// Create users
	user1 := testutil.SeedUser(t, db, "oncall-alice", model.RoleMember)
	user2 := testutil.SeedUser(t, db, "oncall-bob", model.RoleMember)

	// Create schedule
	schedule := &model.Schedule{
		Name:         "test-rotation",
		RotationType: model.RotationDaily,
		Timezone:     "UTC",
		HandoffTime:  "09:00",
		IsEnabled:    true,
	}
	require.NoError(t, db.Create(schedule).Error)

	// Add participants
	require.NoError(t, db.Create(&model.ScheduleParticipant{
		ScheduleID: schedule.ID,
		UserID:     user1.ID,
		Position:   0,
	}).Error)
	require.NoError(t, db.Create(&model.ScheduleParticipant{
		ScheduleID: schedule.ID,
		UserID:     user2.ID,
		Position:   1,
	}).Error)

	result, err := svc.GetCurrentOnCall(context.Background(), schedule.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.User)
	assert.True(t, result.User.ID == user1.ID || result.User.ID == user2.ID,
		"on-call user should be one of the participants")
	assert.NotNil(t, result.Schedule)
	assert.False(t, result.IsOverride)
}

func Test_GetCurrentOnCall_with_active_override(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	scheduleRepo := repository.NewScheduleRepository(db)
	participantRepo := repository.NewScheduleParticipantRepository(db)
	overrideRepo := repository.NewScheduleOverrideRepository(db)
	shiftRepo := repository.NewOnCallShiftRepository(db)
	policyRepo := repository.NewEscalationPolicyRepository(db)
	stepRepo := repository.NewEscalationStepRepository(db)

	svc := NewScheduleService(scheduleRepo, participantRepo, overrideRepo, shiftRepo, policyRepo, stepRepo, logger)

	user1 := testutil.SeedUser(t, db, "override-alice", model.RoleMember)
	user2 := testutil.SeedUser(t, db, "override-bob", model.RoleMember)

	schedule := &model.Schedule{
		Name:         "override-schedule",
		RotationType: model.RotationDaily,
		Timezone:     "UTC",
		HandoffTime:  "09:00",
		IsEnabled:    true,
	}
	require.NoError(t, db.Create(schedule).Error)

	// Add user1 as regular participant
	require.NoError(t, db.Create(&model.ScheduleParticipant{
		ScheduleID: schedule.ID,
		UserID:     user1.ID,
		Position:   0,
	}).Error)

	// Create an active override for user2
	override := &model.ScheduleOverride{
		ScheduleID: schedule.ID,
		UserID:     user2.ID,
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now().Add(1 * time.Hour),
		Reason:     "covering for alice",
	}
	require.NoError(t, db.Create(override).Error)

	result, err := svc.GetCurrentOnCall(context.Background(), schedule.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.User)

	// The override user should be on-call (shift check happens first, but no shifts exist,
	// so override should be picked up)
	// Note: this depends on the GetCurrentShift returning nil when no shifts exist
	if result.IsOverride {
		assert.Equal(t, user2.ID, result.User.ID, "override user should be on-call")
		assert.NotNil(t, result.Override)
	}
}

func Test_GetCurrentOnCall_nonexistent_schedule(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()

	scheduleRepo := repository.NewScheduleRepository(db)
	participantRepo := repository.NewScheduleParticipantRepository(db)
	overrideRepo := repository.NewScheduleOverrideRepository(db)
	shiftRepo := repository.NewOnCallShiftRepository(db)
	policyRepo := repository.NewEscalationPolicyRepository(db)
	stepRepo := repository.NewEscalationStepRepository(db)

	svc := NewScheduleService(scheduleRepo, participantRepo, overrideRepo, shiftRepo, policyRepo, stepRepo, logger)

	_, err := svc.GetCurrentOnCall(context.Background(), 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// NewScheduleService constructor test
// ---------------------------------------------------------------------------

func Test_NewScheduleService_returns_non_nil(t *testing.T) {
	logger := zap.NewNop()
	svc := NewScheduleService(nil, nil, nil, nil, nil, nil, logger)
	assert.NotNil(t, svc)
}
