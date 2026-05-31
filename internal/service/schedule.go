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
	userRepo        *repository.UserRepository
	teamRepo        *repository.TeamRepository
	logger          *zap.Logger
}

func NewScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	participantRepo *repository.ScheduleParticipantRepository,
	overrideRepo *repository.ScheduleOverrideRepository,
	shiftRepo *repository.OnCallShiftRepository,
	policyRepo *repository.EscalationPolicyRepository,
	stepRepo *repository.EscalationStepRepository,
	userRepo *repository.UserRepository,
	teamRepo *repository.TeamRepository,
	logger *zap.Logger,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo:    scheduleRepo,
		participantRepo: participantRepo,
		overrideRepo:    overrideRepo,
		shiftRepo:       shiftRepo,
		policyRepo:      policyRepo,
		stepRepo:        stepRepo,
		userRepo:        userRepo,
		teamRepo:        teamRepo,
		logger:          logger,
	}
}

// ---------------------------------------------------------------------------
// Schedule CRUD
// ---------------------------------------------------------------------------

// CreateSchedule creates a new on-call schedule.
func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *model.Schedule) error {
	if err := validateSchedule(schedule); err != nil {
		return err
	}
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
	if err := validateSchedule(schedule); err != nil {
		return err
	}
	existing, err := s.scheduleRepo.GetByID(ctx, schedule.ID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	existing.Name = schedule.Name
	existing.TeamID = schedule.TeamID
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

// DeleteSchedule deletes a schedule and its participants/overrides/shifts atomically.
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id uint) error {
	if _, err := s.scheduleRepo.GetByID(ctx, id); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "schedule not found")
	}

	if err := s.scheduleRepo.DeleteCascade(ctx, id); err != nil {
		s.logger.Error("failed to delete schedule (cascade)", zap.Error(err), zap.Uint("schedule_id", id))
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

	// Validate that all user IDs exist.
	if s.userRepo != nil {
		for _, uid := range userIDs {
			if _, err := s.userRepo.GetByID(ctx, uid); err != nil {
				return apperr.WithMessage(apperr.ErrNotFound, fmt.Sprintf("user %d not found", uid))
			}
		}
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

	// Validate that the override user exists.
	if s.userRepo != nil {
		if _, err := s.userRepo.GetByID(ctx, override.UserID); err != nil {
			return apperr.WithMessage(apperr.ErrNotFound, fmt.Sprintf("user %d not found", override.UserID))
		}
	}

	if !override.EndTime.After(override.StartTime) {
		return apperr.WithMessage(apperr.ErrBadRequest, "end_time must be after start_time")
	}

	// Check for overlapping overrides in the same schedule.
	overlap, err := s.overrideRepo.HasOverlapOverride(ctx, override.ScheduleID, override.StartTime, override.EndTime)
	if err != nil {
		s.logger.Error("failed to check override overlap", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if overlap {
		return apperr.WithMessage(apperr.ErrBadRequest, "override overlaps with an existing override in this schedule")
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

	// Check for active override first — overrides always take precedence
	// over regular shifts and rotation calculations.
	override, err := s.overrideRepo.GetActiveOverride(ctx, scheduleID, now)
	if err == nil && override != nil {
		return &OnCallResult{
			User:       &override.User,
			Schedule:   schedule,
			IsOverride: true,
			Override:   override,
		}, nil
	}

	// Check OnCallShift records (direct time-slot assignments)
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

	handoffHour, handoffMin, _ := parseHandoffTime(schedule.HandoffTime)
	loc := now.Location()

	refTime := schedule.CreatedAt.In(loc)
	ref := time.Date(refTime.Year(), refTime.Month(), refTime.Day(), handoffHour, handoffMin, 0, 0, loc)
	if ref.After(refTime) {
		switch schedule.RotationType {
		case model.RotationWeekly:
			ref = ref.AddDate(0, 0, -7)
		default:
			ref = ref.AddDate(0, 0, -1)
		}
	}

	periodDays := rotationPeriodDays(schedule)

	if schedule.RotationType == model.RotationWeekly {
		// Align reference to the correct handoff day of the week.
		handoffDay := schedule.HandoffDay % 7
		for ref.Weekday() != time.Weekday(handoffDay) {
			ref = ref.AddDate(0, 0, 1)
		}
		ref = time.Date(ref.Year(), ref.Month(), ref.Day(), handoffHour, handoffMin, 0, 0, loc)
		if ref.After(now) {
			ref = ref.AddDate(0, 0, -7)
		}
	}

	return calendarDayIndex(ref, now, periodDays, numParticipants)
}

// calendarDayIndex computes the rotation index using calendar day arithmetic (H1: DST-safe).
// It counts the number of period boundaries between ref and now in the given timezone.
func calendarDayIndex(ref, now time.Time, periodDays, numParticipants int) int {
	if numParticipants == 0 {
		return 0
	}
	// Count calendar days between ref and now by iterating period boundaries.
	periods := 0
	cursor := ref
	for !cursor.After(now) {
		cursor = cursor.AddDate(0, 0, periodDays)
		if !cursor.After(now) {
			periods++
		}
	}
	return periods % numParticipants
}

// rotationPeriodDays returns the period length in days for the given rotation type.
func rotationPeriodDays(schedule *model.Schedule) int {
	switch schedule.RotationType {
	case model.RotationWeekly:
		return 7
	default: // daily, custom
		return 1
	}
}

// parseHandoffTime parses a "HH:MM" string and returns hour, minute, and error.
// Returns (9, 0, nil) as default if the string is empty or malformed.
func parseHandoffTime(s string) (hour, min int, err error) {
	if len(s) < 5 {
		return 9, 0, nil
	}
	n, err := fmt.Sscanf(s, "%d:%d", &hour, &min)
	if err != nil || n != 2 {
		return 9, 0, fmt.Errorf("invalid handoff_time %q", s)
	}
	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return 9, 0, fmt.Errorf("handoff_time out of range: %s", s)
	}
	return hour, min, nil
}

// ---------------------------------------------------------------------------
// OnCallResolver implementation
// ---------------------------------------------------------------------------

// GetCurrentOnCallForAlert finds the on-call user for all enabled schedules
// whose labels match the alert labels. Returns the first match found.
// Priority: override > shift > rotation (matches GetCurrentOnCall).
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

		// 1. Check override first — overrides always take precedence.
		override, err := s.overrideRepo.GetActiveOverride(ctx, schedule.ID, now)
		if err == nil && override != nil {
			return &override.User, nil
		}

		// 2. Check OnCallShift records for severity-aware dispatch.
		if s.shiftRepo != nil {
			shift, err := s.shiftRepo.GetCurrentShift(ctx, schedule.ID, now)
			if err == nil && shift != nil {
				if matchesSeverityFilter(shift.SeverityFilter, alertSeverity) {
					return &shift.User, nil
				}
				// Shift exists but severity doesn't match — fall through to rotation fallback.
			}
		}

		// 3. Fallback to rotation-based on-call (participants).
		participants, err := s.participantRepo.ListByScheduleID(ctx, schedule.ID)
		if err != nil {
			s.logger.Warn("failed to list participants for schedule",
				zap.Uint("schedule_id", schedule.ID),
				zap.Error(err),
			)
			continue
		}
		if len(participants) == 0 {
			continue
		}

		// Apply schedule-level severity filter.
		if !matchesSeverityFilter(schedule.SeverityFilter, alertSeverity) {
			continue
		}

		loc, locErr := time.LoadLocation(schedule.Timezone)
		if locErr != nil {
			loc = time.UTC
		}
		nowInLoc := now.In(loc)

		index := s.calculateRotationIndex(&schedule, participants, nowInLoc)
		return &participants[index].User, nil
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

	// Check for overlapping shifts in the same schedule.
	overlap, err := s.shiftRepo.HasOverlapShift(ctx, shift.ScheduleID, shift.StartTime, shift.EndTime, 0)
	if err != nil {
		s.logger.Error("failed to check shift overlap", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if overlap {
		return apperr.WithMessage(apperr.ErrBadRequest, "shift overlaps with an existing shift in this schedule")
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

	// Check for overlapping shifts excluding the current one.
	overlap, err := s.shiftRepo.HasOverlapShift(ctx, existing.ScheduleID, shift.StartTime, shift.EndTime, shift.ID)
	if err != nil {
		s.logger.Error("failed to check shift overlap", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if overlap {
		return apperr.WithMessage(apperr.ErrBadRequest, "shift overlaps with an existing shift in this schedule")
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

	// Parse handoff time (M3: validated)
	handoffHour, handoffMin, _ := parseHandoffTime(schedule.HandoffTime)

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

// DeleteEscalationPolicy deletes an escalation policy and its steps atomically.
func (s *ScheduleService) DeleteEscalationPolicy(ctx context.Context, id uint) error {
	if _, err := s.policyRepo.GetByID(ctx, id); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}

	if err := s.policyRepo.DeleteCascade(ctx, id); err != nil {
		s.logger.Error("failed to delete escalation policy (cascade)", zap.Error(err), zap.Uint("policy_id", id))
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

// CreateEscalationStep creates a single escalation step after validation.
func (s *ScheduleService) CreateEscalationStep(ctx context.Context, step *model.EscalationStep) error {
	if _, err := s.policyRepo.GetByID(ctx, step.PolicyID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}

	if err := validateEscalationStep(step); err != nil {
		return err
	}

	// Validate that the referenced target entity exists.
	if err := s.validateEscalationStepTarget(ctx, step.TargetType, step.TargetID); err != nil {
		return err
	}

	if err := s.stepRepo.Create(ctx, step); err != nil {
		s.logger.Error("failed to create escalation step", zap.Error(err), zap.Uint("policy_id", step.PolicyID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ReplaceEscalationSteps replaces all steps for a policy after validating the
// full set. Steps are persisted atomically in a single transaction.
func (s *ScheduleService) ReplaceEscalationSteps(ctx context.Context, policyID uint, steps []model.EscalationStep) error {
	if _, err := s.policyRepo.GetByID(ctx, policyID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "escalation policy not found")
	}

	if err := validateEscalationSteps(steps); err != nil {
		return err
	}

	// Ensure each step references the correct policy and validate target existence.
	for i := range steps {
		steps[i].PolicyID = policyID
		if err := s.validateEscalationStepTarget(ctx, steps[i].TargetType, steps[i].TargetID); err != nil {
			return err
		}
	}

	if err := s.stepRepo.ReplaceByPolicyID(ctx, policyID, steps); err != nil {
		s.logger.Error("failed to replace escalation steps", zap.Error(err), zap.Uint("policy_id", policyID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// validateSchedule validates schedule fields at creation/update time (L1/M3/M4).
func validateSchedule(schedule *model.Schedule) *apperr.AppError {
	if schedule.Name == "" {
		return apperr.WithMessage(apperr.ErrBadRequest, "name is required")
	}
	if _, _, err := parseHandoffTime(schedule.HandoffTime); err != nil {
		return apperr.WithMessage(apperr.ErrBadRequest, err.Error())
	}
	if schedule.HandoffDay < 0 || schedule.HandoffDay > 6 {
		return apperr.WithMessage(apperr.ErrBadRequest, "handoff_day must be between 0 (Sunday) and 6 (Saturday)")
	}
	if schedule.Timezone != "" {
		if _, err := time.LoadLocation(schedule.Timezone); err != nil {
			return apperr.WithMessage(apperr.ErrBadRequest, "invalid timezone: "+schedule.Timezone)
		}
	}
	return nil
}

// validateEscalationStepTarget checks that the referenced target entity exists in the DB.
func (s *ScheduleService) validateEscalationStepTarget(ctx context.Context, targetType string, targetID uint) *apperr.AppError {
	switch targetType {
	case "user":
		if s.userRepo != nil {
			if _, err := s.userRepo.GetByID(ctx, targetID); err != nil {
				return apperr.WithMessage(apperr.ErrNotFound, fmt.Sprintf("user %d not found", targetID))
			}
		}
	case "team":
		if s.teamRepo != nil {
			if _, err := s.teamRepo.GetByID(ctx, targetID); err != nil {
				return apperr.WithMessage(apperr.ErrNotFound, fmt.Sprintf("team %d not found", targetID))
			}
		}
	case "schedule":
		if s.scheduleRepo != nil {
			if _, err := s.scheduleRepo.GetByID(ctx, targetID); err != nil {
				return apperr.WithMessage(apperr.ErrNotFound, fmt.Sprintf("schedule %d not found", targetID))
			}
		}
	}
	return nil
}

// validateEscalationStep validates a single escalation step.
func validateEscalationStep(step *model.EscalationStep) *apperr.AppError {
	if step.DelayMinutes < 0 {
		return apperr.WithMessage(apperr.ErrBadRequest, "delay_minutes must be >= 0")
	}
	if step.TargetType == "" {
		return apperr.WithMessage(apperr.ErrBadRequest, "target_type is required")
	}
	if step.TargetID == 0 {
		return apperr.WithMessage(apperr.ErrBadRequest, "target_id is required")
	}
	validTargets := map[string]bool{"user": true, "team": true, "schedule": true}
	if !validTargets[step.TargetType] {
		return apperr.WithMessage(apperr.ErrBadRequest, "target_type must be one of: user, team, schedule")
	}
	return nil
}

// validateEscalationSteps validates a full ordered set of escalation steps:
//  1. StepOrder values must be sequential starting at 1 (1, 2, 3...) with no gaps or duplicates.
//  2. Each step must have a valid target (target_type + target_id).
//  3. DelayMinutes must be >= 0 for every step.
func validateEscalationSteps(steps []model.EscalationStep) *apperr.AppError {
	if len(steps) == 0 {
		return apperr.WithMessage(apperr.ErrBadRequest, "at least one escalation step is required")
	}

	for i, step := range steps {
		if err := validateEscalationStep(&step); err != nil {
			return err
		}
		expectedOrder := i + 1
		if step.StepOrder != expectedOrder {
			return apperr.WithMessage(apperr.ErrBadRequest,
				fmt.Sprintf("step_order must be sequential: expected %d at position %d, got %d", expectedOrder, i, step.StepOrder))
		}
	}
	return nil
}
