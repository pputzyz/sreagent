package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// OnCallResult represents who is currently on-call for a schedule.
type OnCallResult struct {
	User       *model.User             `json:"user"`
	Schedule   *model.Schedule         `json:"schedule"`
	IsOverride bool                    `json:"is_override"`
	Override   *model.ScheduleOverride `json:"override,omitempty"`
}

type ScheduleService struct {
	scheduleRepo    *repository.ScheduleRepository
	participantRepo *repository.ScheduleParticipantRepository
	overrideRepo    *repository.ScheduleOverrideRepository
	shiftRepo       *repository.OnCallShiftRepository
	policyRepo      *repository.EscalationPolicyRepository
	stepRepo        *repository.EscalationStepRepository
	logger          *zap.Logger
}

func NewScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	participantRepo *repository.ScheduleParticipantRepository,
	overrideRepo *repository.ScheduleOverrideRepository,
	shiftRepo *repository.OnCallShiftRepository,
	policyRepo *repository.EscalationPolicyRepository,
	stepRepo *repository.EscalationStepRepository,
	logger *zap.Logger,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo:    scheduleRepo,
		participantRepo: participantRepo,
		overrideRepo:    overrideRepo,
		shiftRepo:       shiftRepo,
		policyRepo:      policyRepo,
		stepRepo:        stepRepo,
		logger:          logger,
	}
}

// ---------------------------------------------------------------------------
// Schedule CRUD
// ---------------------------------------------------------------------------

// CreateSchedule creates a new on-call schedule.
func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *model.Schedule) error {
	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		s.logger.Error("failed to create schedule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetScheduleByID retrieves a schedule by its ID.
func (s *ScheduleService) GetScheduleByID(ctx context.Context, id uint) (*model.Schedule, error) {
	schedule, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}
	return schedule, nil
}

// ListSchedules returns a paginated list of schedules, optionally filtered by team.
func (s *ScheduleService) ListSchedules(ctx context.Context, teamID uint, page, pageSize int) ([]model.Schedule, int64, error) {
	list, total, err := s.scheduleRepo.List(ctx, teamID, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list schedules", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// UpdateSchedule updates an existing schedule.
func (s *ScheduleService) UpdateSchedule(ctx context.Context, schedule *model.Schedule) error {
	existing, err := s.scheduleRepo.GetByID(ctx, schedule.ID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	existing.Name = schedule.Name
	existing.Description = schedule.Description
	existing.RotationType = schedule.RotationType
	existing.Timezone = schedule.Timezone
	existing.HandoffTime = schedule.HandoffTime
	existing.HandoffDay = schedule.HandoffDay
	existing.IsEnabled = schedule.IsEnabled

	if err := s.scheduleRepo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update schedule", zap.Error(err), zap.Uint("schedule_id", schedule.ID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// DeleteSchedule deletes a schedule and its participants/overrides.
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id uint) error {
	if _, err := s.scheduleRepo.GetByID(ctx, id); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	// Clean up participants
	_ = s.participantRepo.DeleteByScheduleID(ctx, id)

	if err := s.scheduleRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete schedule", zap.Error(err), zap.Uint("schedule_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Participant Management
// ---------------------------------------------------------------------------

// SetParticipants replaces all participants for a schedule with the given list.
func (s *ScheduleService) SetParticipants(ctx context.Context, scheduleID uint, userIDs []uint) error {
	if _, err := s.scheduleRepo.GetByID(ctx, scheduleID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	// Delete existing participants
	if err := s.participantRepo.DeleteByScheduleID(ctx, scheduleID); err != nil {
		s.logger.Error("failed to delete existing participants", zap.Error(err), zap.Uint("schedule_id", scheduleID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Create new participants in order
	for i, userID := range userIDs {
		p := &model.ScheduleParticipant{
			ScheduleID: scheduleID,
			UserID:     userID,
			Position:   i,
		}
		if err := s.participantRepo.Create(ctx, p); err != nil {
			s.logger.Error("failed to create participant",
				zap.Error(err),
				zap.Uint("schedule_id", scheduleID),
				zap.Uint("user_id", userID),
			)
			return apperr.Wrap(apperr.ErrDatabase, err)
		}
	}

	s.logger.Info("schedule participants updated",
		zap.Uint("schedule_id", scheduleID),
		zap.Int("count", len(userIDs)),
	)
	return nil
}

// ListParticipants returns all participants for a schedule ordered by position.
func (s *ScheduleService) ListParticipants(ctx context.Context, scheduleID uint) ([]model.ScheduleParticipant, error) {
	participants, err := s.participantRepo.ListByScheduleID(ctx, scheduleID)
	if err != nil {
		s.logger.Error("failed to list participants", zap.Error(err), zap.Uint("schedule_id", scheduleID))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return participants, nil
}

// ---------------------------------------------------------------------------
// Override Management
// ---------------------------------------------------------------------------

// CreateOverride creates a schedule override.
func (s *ScheduleService) CreateOverride(ctx context.Context, override *model.ScheduleOverride) error {
	if _, err := s.scheduleRepo.GetByID(ctx, override.ScheduleID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	if !override.EndTime.After(override.StartTime) {
		return apperr.WithMessage(apperr.ErrBadRequest, "end_time must be after start_time")
	}

	if err := s.overrideRepo.Create(ctx, override); err != nil {
		s.logger.Error("failed to create schedule override", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("schedule override created",
		zap.Uint("schedule_id", override.ScheduleID),
		zap.Uint("user_id", override.UserID),
		zap.Time("start", override.StartTime),
		zap.Time("end", override.EndTime),
	)
	return nil
}

// DeleteOverride deletes a schedule override.
func (s *ScheduleService) DeleteOverride(ctx context.Context, id uint) error {
	if err := s.overrideRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete schedule override", zap.Error(err), zap.Uint("override_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ListOverrides returns all overrides for a schedule.
func (s *ScheduleService) ListOverrides(ctx context.Context, scheduleID uint) ([]model.ScheduleOverride, error) {
	overrides, err := s.overrideRepo.ListByScheduleID(ctx, scheduleID)
	if err != nil {
		s.logger.Error("failed to list overrides", zap.Error(err), zap.Uint("schedule_id", scheduleID))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return overrides, nil
}

// ---------------------------------------------------------------------------
// On-Call Calculation
// ---------------------------------------------------------------------------

// GetCurrentOnCall calculates who is currently on-call for the given schedule.
// It first checks OnCallShift records, then active overrides, then falls back to rotation logic.
func (s *ScheduleService) GetCurrentOnCall(ctx context.Context, scheduleID uint) (*OnCallResult, error) {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	if !schedule.IsEnabled {
		return nil, apperr.WithMessage(apperr.ErrBadRequest, "schedule is disabled")
	}

	now := time.Now()

	// Load timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		s.logger.Warn("invalid timezone, falling back to UTC",
			zap.String("timezone", schedule.Timezone),
			zap.Error(err),
		)
		loc = time.UTC
	}
	now = now.In(loc)

	// Check OnCallShift records first (direct time-slot assignments)
	if s.shiftRepo != nil {
		shift, err := s.shiftRepo.GetCurrentShift(ctx, scheduleID, now)
		if err != nil {
			s.logger.Warn("failed to query current shift, falling through",
				zap.Uint("schedule_id", scheduleID),
				zap.Error(err),
			)
		} else if shift != nil {
			return &OnCallResult{
				User:       &shift.User,
				Schedule:   schedule,
				IsOverride: false,
			}, nil
		}
	}

	// Check for active override next
	override, err := s.overrideRepo.GetActiveOverride(ctx, scheduleID, now)
	if err == nil && override != nil {
		return &OnCallResult{
			User:       &override.User,
			Schedule:   schedule,
			IsOverride: true,
			Override:   override,
		}, nil
	}

	// Fall back to rotation calculation
	participants, err := s.participantRepo.ListByScheduleID(ctx, scheduleID)
	if err != nil {
		s.logger.Error("failed to list participants for on-call calculation", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	if len(participants) == 0 {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "no participants configured for this schedule")
	}

	// Calculate rotation index based on rotation type
	index := s.calculateRotationIndex(schedule, participants, now)

	onCallParticipant := participants[index]
	return &OnCallResult{
		User:       &onCallParticipant.User,
		Schedule:   schedule,
		IsOverride: false,
	}, nil
}

// calculateRotationIndex determines which participant is on-call based on the
// rotation type, handoff settings, and the current time.
func (s *ScheduleService) calculateRotationIndex(
	schedule *model.Schedule,
	participants []model.ScheduleParticipant,
	now time.Time,
) int {
	numParticipants := len(participants)
	if numParticipants == 0 {
		return 0
	}

	// Parse handoff time (HH:MM)
	handoffHour, handoffMin := 9, 0
	if len(schedule.HandoffTime) >= 5 {
		fmt.Sscanf(schedule.HandoffTime, "%d:%d", &handoffHour, &handoffMin)
	}

	// Reference point: the schedule's creation time (beginning of rotation)
	refTime := schedule.CreatedAt
	loc := now.Location()
	refTime = refTime.In(loc)

	// Align reference to the first handoff boundary
	ref := time.Date(refTime.Year(), refTime.Month(), refTime.Day(), handoffHour, handoffMin, 0, 0, loc)
	if ref.After(refTime) {
		// Move back one period so the ref is at or before creation
		switch schedule.RotationType {
		case model.RotationWeekly:
			ref = ref.AddDate(0, 0, -7)
		default: // daily or custom
			ref = ref.AddDate(0, 0, -1)
		}
	}

	switch schedule.RotationType {
	case model.RotationDaily:
		// Calculate number of complete handoff periods since reference
		elapsed := now.Sub(ref)
		periods := int(elapsed.Hours() / 24)
		return periods % numParticipants

	case model.RotationWeekly:
		// Adjust reference to the correct handoff day of the week
		for ref.Weekday() != time.Weekday(schedule.HandoffDay) {
			ref = ref.AddDate(0, 0, 1)
		}
		ref = time.Date(ref.Year(), ref.Month(), ref.Day(), handoffHour, handoffMin, 0, 0, loc)
		if ref.After(now) {
			ref = ref.AddDate(0, 0, -7)
		}
		elapsed := now.Sub(ref)
		periods := int(elapsed.Hours() / (24 * 7))
		return periods % numParticipants

	case model.RotationCustom:
		// Custom: fall back to daily rotation as a sensible default
		elapsed := now.Sub(ref)
		periods := int(elapsed.Hours() / 24)
		return periods % numParticipants

	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// OnCallResolver implementation
// ---------------------------------------------------------------------------

// GetCurrentOnCallForAlert finds the on-call user for all enabled schedules
// whose labels match the alert labels. Returns the first match found.
// It checks severity filter on OnCallShift records when applicable.
func (s *ScheduleService) GetCurrentOnCallForAlert(ctx context.Context, alertLabels map[string]string) (*model.User, error) {
	// List all schedules (unpaged - use a large page size)
	schedules, _, err := s.scheduleRepo.List(ctx, 0, 1, 1000)
	if err != nil {
		return nil, err
	}

	alertSeverity := alertLabels["severity"]
	now := time.Now()

	for _, schedule := range schedules {
		if !schedule.IsEnabled {
			continue
		}

		// Check OnCallShift records directly for severity-aware dispatch.
		if s.shiftRepo != nil {
			shift, err := s.shiftRepo.GetCurrentShift(ctx, schedule.ID, now)
			if err == nil && shift != nil {
				// Verify severity filter if the shift specifies one.
				if matchesSeverityFilter(shift.SeverityFilter, alertSeverity) {
					return &shift.User, nil
				}
				// Shift exists but severity doesn't match - skip this schedule.
				continue
			}
		}

		// Fall back to existing on-call logic (override → rotation).
		result, err := s.GetCurrentOnCall(ctx, schedule.ID)
		if err != nil {
			s.logger.Warn("failed to get on-call for schedule",
				zap.Uint("schedule_id", schedule.ID),
				zap.Error(err),
			)
			continue
		}
		if result != nil && result.User != nil {
			// Apply schedule-level severity filter.
			if matchesSeverityFilter(schedule.SeverityFilter, alertSeverity) {
				return result.User, nil
			}
		}
	}

	return nil, nil
}

// matchesSeverityFilter checks whether an alert's severity matches the filter string.
// An empty filter matches all severities. The filter is a comma-separated list of severity values.
func matchesSeverityFilter(filter, alertSeverity string) bool {
	if filter == "" {
		return true
	}
	if alertSeverity == "" {
		return true
	}
	for _, s := range splitComma(filter) {
		if s == alertSeverity {
			return true
		}
	}
	return false
}

// splitComma splits a comma-separated string into trimmed, non-empty tokens.
func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			token := strings.TrimSpace(s[start:i])
			if token != "" {
				out = append(out, token)
			}
			start = i + 1
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// OnCallShift Management
// ---------------------------------------------------------------------------

// CreateShift creates a new on-call shift.
func (s *ScheduleService) CreateShift(ctx context.Context, shift *model.OnCallShift) error {
	if _, err := s.scheduleRepo.GetByID(ctx, shift.ScheduleID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}
	if !shift.EndTime.After(shift.StartTime) {
		return apperr.WithMessage(apperr.ErrBadRequest, "end_time must be after start_time")
	}
	if err := s.shiftRepo.Create(ctx, shift); err != nil {
		s.logger.Error("failed to create shift", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// UpdateShift updates an existing on-call shift.
func (s *ScheduleService) UpdateShift(ctx context.Context, shift *model.OnCallShift) error {
	existing, err := s.shiftRepo.GetByID(ctx, shift.ID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "shift not found")
	}
	if !shift.EndTime.After(shift.StartTime) {
		return apperr.WithMessage(apperr.ErrBadRequest, "end_time must be after start_time")
	}
	existing.UserID = shift.UserID
	existing.StartTime = shift.StartTime
	existing.EndTime = shift.EndTime
	existing.SeverityFilter = shift.SeverityFilter
	existing.Source = shift.Source
	existing.Note = shift.Note
	if err := s.shiftRepo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update shift", zap.Error(err), zap.Uint("shift_id", shift.ID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// DeleteShift deletes an on-call shift by ID.
func (s *ScheduleService) DeleteShift(ctx context.Context, shiftID uint) error {
	if err := s.shiftRepo.Delete(ctx, shiftID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.WithMessage(apperr.ErrNotFound, "shift not found")
		}
		s.logger.Error("failed to delete shift", zap.Error(err), zap.Uint("shift_id", shiftID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ListShifts returns shifts for a schedule within the given time range.
func (s *ScheduleService) ListShifts(ctx context.Context, scheduleID uint, start, end time.Time) ([]model.OnCallShift, error) {
	if _, err := s.scheduleRepo.GetByID(ctx, scheduleID); err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}
	shifts, err := s.shiftRepo.ListBySchedule(ctx, scheduleID, start, end)
	if err != nil {
		s.logger.Error("failed to list shifts", zap.Error(err), zap.Uint("schedule_id", scheduleID))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return shifts, nil
}

// GenerateRotationShifts auto-generates OnCallShift records from the schedule's
// rotation configuration for the given number of weeks ahead.
// Existing auto-generated shifts in that range are replaced.
func (s *ScheduleService) GenerateRotationShifts(ctx context.Context, scheduleID uint, weeks int) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	participants, err := s.participantRepo.ListByScheduleID(ctx, scheduleID)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if len(participants) == 0 {
		return apperr.WithMessage(apperr.ErrBadRequest, "no participants configured for this schedule")
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// Parse handoff time
	handoffHour, handoffMin := 9, 0
	if len(schedule.HandoffTime) >= 5 {
		fmt.Sscanf(schedule.HandoffTime, "%d:%d", &handoffHour, &handoffMin)
	}

	now := time.Now().In(loc)
	// Align start to today's handoff boundary
	genStart := time.Date(now.Year(), now.Month(), now.Day(), handoffHour, handoffMin, 0, 0, loc)
	if genStart.After(now) {
		genStart = genStart.AddDate(0, 0, -1)
	}
	genEnd := genStart.AddDate(0, 0, weeks*7)

	// Determine period duration in days
	periodDays := 1
	if schedule.RotationType == model.RotationWeekly {
		periodDays = 7
	}

	// Remove existing auto-generated shifts in range
	if err := s.shiftRepo.DeleteByScheduleAndTimeRange(ctx, scheduleID, genStart, genEnd); err != nil {
		s.logger.Error("failed to clean up existing shifts", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Build new shifts
	var shifts []model.OnCallShift
	cursor := genStart
	idx := s.calculateRotationIndex(schedule, participants, cursor)
	for cursor.Before(genEnd) {
		nextCursor := cursor.AddDate(0, 0, periodDays)
		shifts = append(shifts, model.OnCallShift{
			ScheduleID:     scheduleID,
			UserID:         participants[idx%len(participants)].UserID,
			StartTime:      cursor,
			EndTime:        nextCursor,
			SeverityFilter: schedule.SeverityFilter,
			Source:         "rotation",
		})
		cursor = nextCursor
		idx++
	}

	if err := s.shiftRepo.BulkCreate(ctx, shifts); err != nil {
		s.logger.Error("failed to bulk create shifts", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("rotation shifts generated",
		zap.Uint("schedule_id", scheduleID),
		zap.Int("count", len(shifts)),
		zap.Int("weeks", weeks),
	)
	return nil
}

// ---------------------------------------------------------------------------
// Escalation Policy CRUD
// ---------------------------------------------------------------------------

// CreateEscalationPolicy creates a new escalation policy.
func (s *ScheduleService) CreateEscalationPolicy(ctx context.Context, policy *model.EscalationPolicy) error {
	if err := s.policyRepo.Create(ctx, policy); err != nil {
		s.logger.Error("failed to create escalation policy", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetEscalationPolicyByID retrieves an escalation policy by ID.
func (s *ScheduleService) GetEscalationPolicyByID(ctx context.Context, id uint) (*model.EscalationPolicy, error) {
	policy, err := s.policyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}
	return policy, nil
}

// ListEscalationPolicies returns escalation policies, optionally filtered by team.
func (s *ScheduleService) ListEscalationPolicies(ctx context.Context, teamID uint) ([]model.EscalationPolicy, error) {
	list, err := s.policyRepo.ListByTeamID(ctx, teamID)
	if err != nil {
		s.logger.Error("failed to list escalation policies", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

// UpdateEscalationPolicy updates an existing escalation policy.
func (s *ScheduleService) UpdateEscalationPolicy(ctx context.Context, policy *model.EscalationPolicy) error {
	existing, err := s.policyRepo.GetByID(ctx, policy.ID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}

	existing.Name = policy.Name
	existing.TeamID = policy.TeamID
	existing.IsEnabled = policy.IsEnabled

	if err := s.policyRepo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update escalation policy", zap.Error(err), zap.Uint("policy_id", policy.ID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// DeleteEscalationPolicy deletes an escalation policy and its steps.
func (s *ScheduleService) DeleteEscalationPolicy(ctx context.Context, id uint) error {
	if _, err := s.policyRepo.GetByID(ctx, id); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}

	// Delete associated steps first
	steps, err := s.stepRepo.ListByPolicyID(ctx, id)
	if err != nil {
		s.logger.Error("failed to list escalation steps for deletion", zap.Error(err), zap.Uint("policy_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	for _, step := range steps {
		if err := s.stepRepo.Delete(ctx, step.ID); err != nil {
			s.logger.Error("failed to delete escalation step", zap.Error(err), zap.Uint("step_id", step.ID), zap.Uint("policy_id", id))
		}
	}

	if err := s.policyRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete escalation policy", zap.Error(err), zap.Uint("policy_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Escalation Step CRUD
// ---------------------------------------------------------------------------

// CreateEscalationStep creates a new step in an escalation policy.
func (s *ScheduleService) CreateEscalationStep(ctx context.Context, step *model.EscalationStep) error {
	// Verify the policy exists
	if _, err := s.policyRepo.GetByID(ctx, step.PolicyID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}

	if err := s.stepRepo.Create(ctx, step); err != nil {
		s.logger.Error("failed to create escalation step", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ListEscalationSteps returns all steps for a given escalation policy.
func (s *ScheduleService) ListEscalationSteps(ctx context.Context, policyID uint) ([]model.EscalationStep, error) {
	steps, err := s.stepRepo.ListByPolicyID(ctx, policyID)
	if err != nil {
		s.logger.Error("failed to list escalation steps", zap.Error(err), zap.Uint("policy_id", policyID))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return steps, nil
}

// UpdateEscalationStep updates an escalation step.
func (s *ScheduleService) UpdateEscalationStep(ctx context.Context, step *model.EscalationStep) error {
	existing, err := s.stepRepo.ListByPolicyID(ctx, step.PolicyID)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	found := false
	for _, e := range existing {
		if e.ID == step.ID {
			found = true
			break
		}
	}
	if !found {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation step not found")
	}

	if err := s.stepRepo.Update(ctx, step); err != nil {
		s.logger.Error("failed to update escalation step", zap.Error(err), zap.Uint("step_id", step.ID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// DeleteEscalationStep deletes an escalation step.
func (s *ScheduleService) DeleteEscalationStep(ctx context.Context, id uint) error {
	if err := s.stepRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.WithMessage(apperr.ErrNotFound, "escalation step not found")
		}
		s.logger.Error("failed to delete escalation step", zap.Error(err), zap.Uint("step_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}
