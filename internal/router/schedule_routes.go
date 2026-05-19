package router

import (
	"github.com/gin-gonic/gin"
)

// registerScheduleRoutes registers schedule and escalation policy routes.
func (h *Handlers) registerScheduleRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	// Schedules
	schedules := auth.Group("/schedules")
	{
		schedules.GET("", h.Schedule.ListSchedules)
		schedules.GET("/:id", h.Schedule.GetSchedule)
		schedules.GET("/:id/oncall", h.Schedule.GetCurrentOnCall)
		schedules.GET("/:id/participants", h.Schedule.GetParticipants)
		schedules.GET("/:id/shifts", h.Schedule.ListShifts)
		schedules.POST("", manage, h.Schedule.CreateSchedule)
		schedules.PUT("/:id", manage, h.Schedule.UpdateSchedule)
		schedules.DELETE("/:id", manage, h.Schedule.DeleteSchedule)
		schedules.PUT("/:id/participants", manage, h.Schedule.SetParticipants)
		schedules.GET("/:id/overrides", h.Schedule.ListOverrides)
		schedules.POST("/:id/overrides", manage, h.Schedule.CreateOverride)
		schedules.DELETE("/:id/overrides/:oid", manage, h.Schedule.DeleteOverride)
		schedules.POST("/:id/shifts", manage, h.Schedule.CreateShift)
		schedules.PUT("/:id/shifts/:shiftId", manage, h.Schedule.UpdateShift)
		schedules.DELETE("/:id/shifts/:shiftId", manage, h.Schedule.DeleteShift)
		schedules.POST("/:id/generate-shifts", manage, h.Schedule.GenerateShifts)
		schedules.GET("/:id/ical", h.Schedule.ExportICal)
	}

	// Escalation Policies
	escalation := auth.Group("/escalation-policies")
	{
		escalation.GET("", h.Schedule.ListEscalationPolicies)
		escalation.GET("/:id", h.Schedule.GetEscalationPolicy)
		escalation.POST("", manage, h.Schedule.CreateEscalationPolicy)
		escalation.PUT("/:id", manage, h.Schedule.UpdateEscalationPolicy)
		escalation.DELETE("/:id", manage, h.Schedule.DeleteEscalationPolicy)
	}
}
