package handler

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type ScheduleHandler struct {
	svc      *service.ScheduleService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

func NewScheduleHandler(svc *service.ScheduleService, logger ...*zap.Logger) *ScheduleHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &ScheduleHandler{svc: svc, log: l}
}

// SetAuditService injects the audit log service.
func (h *ScheduleHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// CreateScheduleRequest is the request body for creating a schedule.
type CreateScheduleRequest struct {
	Name               string             `json:"name" binding:"required"`
	TeamID             *uint              `json:"team_id"`
	Description        string             `json:"description"`
	RotationType       model.RotationType `json:"rotation_type" binding:"required"`
	Timezone           string             `json:"timezone"`
	HandoffTime        string             `json:"handoff_time"`
	HandoffDay         int                `json:"handoff_day"`
	RotationPeriodDays int                `json:"rotation_period_days"`
	IsEnabled          *bool              `json:"is_enabled"`
}

// UpdateScheduleRequest is the request body for updating a schedule.
type UpdateScheduleRequest struct {
	Name               string             `json:"name" binding:"required"`
	TeamID             *uint              `json:"team_id"`
	Description        string             `json:"description"`
	RotationType       model.RotationType `json:"rotation_type" binding:"required"`
	Timezone           string             `json:"timezone"`
	HandoffTime        string             `json:"handoff_time"`
	HandoffDay         int                `json:"handoff_day"`
	RotationPeriodDays int                `json:"rotation_period_days"`
	IsEnabled          *bool              `json:"is_enabled"`
}

// SetParticipantsRequest is the request body for setting schedule participants.
type SetParticipantsRequest struct {
	UserIDs []uint `json:"user_ids" binding:"required"`
}

// CreateOverrideRequest is the request body for creating a schedule override.
type CreateOverrideRequest struct {
	UserID    uint      `json:"user_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	Reason    string    `json:"reason"`
}

// CreateShiftRequest is the request body for creating an on-call shift.
type CreateShiftRequest struct {
	UserID         uint      `json:"user_id" binding:"required"`
	StartTime      time.Time `json:"start_time" binding:"required"`
	EndTime        time.Time `json:"end_time" binding:"required"`
	SeverityFilter string    `json:"severity_filter"`
	Note           string    `json:"note"`
}

// UpdateShiftRequest is the request body for updating an on-call shift.
type UpdateShiftRequest struct {
	UserID         uint      `json:"user_id" binding:"required"`
	StartTime      time.Time `json:"start_time" binding:"required"`
	EndTime        time.Time `json:"end_time" binding:"required"`
	SeverityFilter string    `json:"severity_filter"`
	Note           string    `json:"note"`
}

// GenerateShiftsRequest is the request body for auto-generating rotation shifts.
type GenerateShiftsRequest struct {
	Weeks int `json:"weeks" binding:"required,min=1,max=52"`
}

// CreateEscalationPolicyRequest is the request body for creating an escalation policy.
type CreateEscalationPolicyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"` // P1-09
	TeamID      uint                   `json:"team_id"`     // 0 = global policy (no team)
	IsEnabled   *bool                  `json:"is_enabled"`
	Steps       []model.EscalationStep `json:"steps"`
}

// UpdateEscalationPolicyRequest is the request body for updating an escalation policy.
type UpdateEscalationPolicyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"` // P1-09
	TeamID      uint                   `json:"team_id"`     // 0 = global policy (no team)
	IsEnabled   *bool                  `json:"is_enabled"`
	Steps       []model.EscalationStep `json:"steps"`
}

// ---------------------------------------------------------------------------
// Schedule CRUD Handlers
// ---------------------------------------------------------------------------

// CreateSchedule creates a new on-call schedule.
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	handoffTime := req.HandoffTime
	if handoffTime == "" {
		handoffTime = "09:00"
	}

	h.log.Info("schedule create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("rotation_type", string(req.RotationType)),
		zap.String("request_id", c.GetString("request_id")))

	schedule := &model.Schedule{
		Name:               req.Name,
		TeamID:             req.TeamID,
		Description:        req.Description,
		RotationType:       req.RotationType,
		Timezone:           timezone,
		HandoffTime:        handoffTime,
		HandoffDay:         req.HandoffDay,
		RotationPeriodDays: req.RotationPeriodDays,
		IsEnabled:          isEnabled,
	}

	if err := h.svc.CreateSchedule(c.Request.Context(), schedule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceSchedule, ResourceID: &schedule.ID, ResourceName: schedule.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, schedule)
}

// GetSchedule returns a schedule by ID.
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	schedule, err := h.svc.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, schedule)
}

// ListSchedules returns a paginated list of schedules.
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	pq := GetPageQuery(c)

	var teamID uint
	if tidStr := c.Query("team_id"); tidStr != "" {
		if tid, err := strconv.ParseUint(tidStr, 10, 64); err == nil {
			teamID = uint(tid)
		}
	}

	list, total, err := h.svc.ListSchedules(c.Request.Context(), teamID, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// UpdateSchedule updates a schedule.
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("schedule update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("schedule_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	schedule := &model.Schedule{
		Name:               req.Name,
		TeamID:             req.TeamID,
		Description:        req.Description,
		RotationType:       req.RotationType,
		Timezone:           req.Timezone,
		HandoffTime:        req.HandoffTime,
		HandoffDay:         req.HandoffDay,
		RotationPeriodDays: req.RotationPeriodDays,
		IsEnabled:          isEnabled,
	}
	schedule.ID = id

	if err := h.svc.UpdateSchedule(c.Request.Context(), schedule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceSchedule, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, schedule)
}

// DeleteSchedule deletes a schedule.
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("schedule delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("schedule_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.DeleteSchedule(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: model.AuditResourceSchedule, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// ---------------------------------------------------------------------------
// On-Call
// ---------------------------------------------------------------------------

// GetCurrentOnCall returns the user currently on-call for the given schedule.
func (h *ScheduleHandler) GetCurrentOnCall(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	result, err := h.svc.GetCurrentOnCall(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// ---------------------------------------------------------------------------
// Participants
// ---------------------------------------------------------------------------

// SetParticipants sets the participant list for a schedule.
func (h *ScheduleHandler) SetParticipants(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req SetParticipantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.SetParticipants(c.Request.Context(), id, req.UserIDs); err != nil {
		Error(c, err)
		return
	}

	// Return the updated participant list
	participants, err := h.svc.ListParticipants(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, participants)
}

// GetParticipants returns the participant list for a schedule.
func (h *ScheduleHandler) GetParticipants(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	participants, err := h.svc.ListParticipants(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, participants)
}

// ---------------------------------------------------------------------------
// Overrides
// ---------------------------------------------------------------------------

// CreateOverride creates a schedule override.
func (h *ScheduleHandler) CreateOverride(c *gin.Context) {
	scheduleID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("schedule override create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("schedule_id", scheduleID),
		zap.Uint("override_user_id", req.UserID),
		zap.String("request_id", c.GetString("request_id")))

	override := &model.ScheduleOverride{
		ScheduleID: scheduleID,
		UserID:     req.UserID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Reason:     req.Reason,
	}

	if err := h.svc.CreateOverride(c.Request.Context(), override); err != nil {
		Error(c, err)
		return
	}

	Success(c, override)
}

// ListOverrides returns all overrides for a schedule.
func (h *ScheduleHandler) ListOverrides(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	overrides, err := h.svc.ListOverrides(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, overrides)
}

// DeleteOverride deletes a schedule override.
func (h *ScheduleHandler) DeleteOverride(c *gin.Context) {
	oid, err := GetIDParam(c, "oid")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.DeleteOverride(c.Request.Context(), oid); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// ---------------------------------------------------------------------------
// Escalation Policy CRUD Handlers
// ---------------------------------------------------------------------------

// CreateEscalationPolicy creates a new escalation policy.
func (h *ScheduleHandler) CreateEscalationPolicy(c *gin.Context) {
	var req CreateEscalationPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("escalation policy create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.Uint("team_id", req.TeamID),
		zap.String("request_id", c.GetString("request_id")))

	policy := &model.EscalationPolicy{
		Name:        req.Name,
		Description: req.Description, // P1-09
		TeamID:      req.TeamID,
		IsEnabled:   isEnabled,
	}

	if err := h.svc.CreateEscalationPolicy(c.Request.Context(), policy); err != nil {
		Error(c, err)
		return
	}

	// Replace escalation steps if provided.
	if len(req.Steps) > 0 {
		if err := h.svc.ReplaceEscalationSteps(c.Request.Context(), policy.ID, req.Steps); err != nil {
			Error(c, err)
			return
		}
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceEscalationPolicy, ResourceID: &policy.ID, ResourceName: policy.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, policy)
}

// GetEscalationPolicy returns an escalation policy by ID.
func (h *ScheduleHandler) GetEscalationPolicy(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	policy, err := h.svc.GetEscalationPolicyByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	// Also fetch the steps
	steps, _ := h.svc.ListEscalationSteps(c.Request.Context(), id)

	Success(c, gin.H{
		"policy": policy,
		"steps":  steps,
	})
}

// ListEscalationPolicies returns escalation policies, optionally filtered by team.
// B11-9: team_id=0 returns only global policies; omit team_id to get all policies.
func (h *ScheduleHandler) ListEscalationPolicies(c *gin.Context) {
	tidStr := c.Query("team_id")
	if tidStr == "" {
		// No team_id filter — return all policies
		list, err := h.svc.ListAllEscalationPolicies(c.Request.Context())
		if err != nil {
			Error(c, err)
			return
		}
		Success(c, list)
		return
	}

	tid, err := strconv.ParseUint(tidStr, 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid team_id"))
		return
	}

	list, err := h.svc.ListEscalationPolicies(c.Request.Context(), uint(tid))
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, list)
}

// UpdateEscalationPolicy updates an escalation policy.
func (h *ScheduleHandler) UpdateEscalationPolicy(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateEscalationPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("escalation policy update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("policy_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	policy := &model.EscalationPolicy{
		Name:        req.Name,
		Description: req.Description, // P1-09
		TeamID:      req.TeamID,
		IsEnabled:   isEnabled,
	}
	policy.ID = id

	if err := h.svc.UpdateEscalationPolicy(c.Request.Context(), policy); err != nil {
		Error(c, err)
		return
	}

	// Replace escalation steps if provided.
	if len(req.Steps) > 0 {
		if err := h.svc.ReplaceEscalationSteps(c.Request.Context(), id, req.Steps); err != nil {
			Error(c, err)
			return
		}
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceEscalationPolicy, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, policy)
}

// DeleteEscalationPolicy deletes an escalation policy.
func (h *ScheduleHandler) DeleteEscalationPolicy(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("escalation policy delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("policy_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.DeleteEscalationPolicy(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: model.AuditResourceEscalationPolicy, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// ---------------------------------------------------------------------------
// OnCallShift Handlers
// ---------------------------------------------------------------------------

// ListShifts returns shifts for a schedule in the given time window.
// GET /schedules/:id/shifts?start=<RFC3339>&end=<RFC3339>
func (h *ScheduleHandler) ListShifts(c *gin.Context) {
	scheduleID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	if startStr != "" {
		if start, err = time.Parse(time.RFC3339, startStr); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid start time, use RFC3339 format"))
			return
		}
	} else {
		start = time.Now().AddDate(0, 0, -7)
	}
	if endStr != "" {
		if end, err = time.Parse(time.RFC3339, endStr); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid end time, use RFC3339 format"))
			return
		}
	} else {
		end = time.Now().AddDate(0, 0, 30)
	}

	shifts, err := h.svc.ListShifts(c.Request.Context(), scheduleID, start, end)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, shifts)
}

// CreateShift creates a new on-call shift for a schedule.
// POST /schedules/:id/shifts
func (h *ScheduleHandler) CreateShift(c *gin.Context) {
	scheduleID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("schedule shift create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("schedule_id", scheduleID),
		zap.Uint("shift_user_id", req.UserID),
		zap.String("request_id", c.GetString("request_id")))

	shift := &model.OnCallShift{
		ScheduleID:     scheduleID,
		UserID:         req.UserID,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		SeverityFilter: req.SeverityFilter,
		Source:         "manual",
		Note:           req.Note,
	}

	if err := h.svc.CreateShift(c.Request.Context(), shift); err != nil {
		Error(c, err)
		return
	}

	Success(c, shift)
}

// UpdateShift updates an existing on-call shift.
// PUT /schedules/:id/shifts/:shiftId
func (h *ScheduleHandler) UpdateShift(c *gin.Context) {
	_, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	shiftID, err := GetIDParam(c, "shiftId")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("schedule shift update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("shift_id", shiftID),
		zap.String("request_id", c.GetString("request_id")))

	shift := &model.OnCallShift{
		UserID:         req.UserID,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		SeverityFilter: req.SeverityFilter,
		Note:           req.Note,
	}
	shift.ID = shiftID

	if err := h.svc.UpdateShift(c.Request.Context(), shift); err != nil {
		Error(c, err)
		return
	}

	Success(c, shift)
}

// DeleteShift deletes an on-call shift.
// DELETE /schedules/:id/shifts/:shiftId
func (h *ScheduleHandler) DeleteShift(c *gin.Context) {
	_, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	shiftID, err := GetIDParam(c, "shiftId")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("schedule shift delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("shift_id", shiftID),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.DeleteShift(c.Request.Context(), shiftID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// GenerateShifts auto-generates rotation shifts for a schedule.
// POST /schedules/:id/generate-shifts
func (h *ScheduleHandler) GenerateShifts(c *gin.Context) {
	scheduleID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req GenerateShiftsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.GenerateRotationShifts(c.Request.Context(), scheduleID, req.Weeks); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"message": "shifts generated", "weeks": req.Weeks})
}

// ExportICal generates an RFC 5545 iCalendar feed for a schedule's upcoming shifts.
// GET /api/v1/schedules/:id/ical
// Returns text/calendar content for import into Google Calendar, Outlook, Apple Calendar, etc.
func (h *ScheduleHandler) ExportICal(c *gin.Context) {
	scheduleID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	// Fetch schedule metadata
	schedule, err := h.svc.GetScheduleByID(c.Request.Context(), scheduleID)
	if err != nil {
		Error(c, err)
		return
	}

	// Fetch upcoming shifts (next 90 days)
	now := time.Now()
	end := now.Add(90 * 24 * time.Hour)
	shifts, err := h.svc.ListShifts(c.Request.Context(), scheduleID, now.Add(-30*24*time.Hour), end)
	if err != nil {
		Error(c, err)
		return
	}

	// Build RFC 5545 VCALENDAR
	var buf bytes.Buffer
	buf.WriteString("BEGIN:VCALENDAR\r\n")
	buf.WriteString("VERSION:2.0\r\n")
	buf.WriteString("PRODID:-//SREAgent//OnCall Schedule//EN\r\n")
	buf.WriteString("CALSCALE:GREGORIAN\r\n")
	buf.WriteString("METHOD:PUBLISH\r\n")
	fmt.Fprintf(&buf, "X-WR-CALNAME:%s\r\n", icalEscape(schedule.Name))
	fmt.Fprintf(&buf, "X-WR-TIMEZONE:%s\r\n", schedule.Timezone)

	for _, shift := range shifts {
		userName := shift.User.DisplayName
		if userName == "" {
			userName = shift.User.Username
		}

		uid := fmt.Sprintf("shift-%d-%d@sreagent", shift.ID, scheduleID)
		dtStart := shift.StartTime.UTC().Format("20060102T150405Z")
		dtEnd := shift.EndTime.UTC().Format("20060102T150405Z")
		dtStamp := now.UTC().Format("20060102T150405Z")

		summary := fmt.Sprintf("On-call: %s (%s)", userName, schedule.Name)
		description := fmt.Sprintf("Schedule: %s\\nOn-call: %s", schedule.Name, userName)
		if shift.Note != "" {
			description += "\\nNote: " + icalEscape(shift.Note)
		}

		buf.WriteString("BEGIN:VEVENT\r\n")
		fmt.Fprintf(&buf, "UID:%s\r\n", uid)
		fmt.Fprintf(&buf, "DTSTAMP:%s\r\n", dtStamp)
		fmt.Fprintf(&buf, "DTSTART:%s\r\n", dtStart)
		fmt.Fprintf(&buf, "DTEND:%s\r\n", dtEnd)
		fmt.Fprintf(&buf, "SUMMARY:%s\r\n", icalEscape(summary))
		fmt.Fprintf(&buf, "DESCRIPTION:%s\r\n", description)
		buf.WriteString("END:VEVENT\r\n")
	}

	buf.WriteString("END:VCALENDAR\r\n")

	filename := fmt.Sprintf("oncall-%d.ics", scheduleID)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(200, "text/calendar; charset=utf-8", buf.Bytes())
}

// icalEscape escapes special characters in iCalendar text values.
func icalEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
