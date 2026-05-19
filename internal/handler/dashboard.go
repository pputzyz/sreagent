package handler

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

type DashboardHandler struct {
	statsSvc *service.DashboardStatsService
}

func NewDashboardHandler(statsSvc *service.DashboardStatsService) *DashboardHandler {
	return &DashboardHandler{statsSvc: statsSvc}
}

// GetStats returns aggregated dashboard statistics.
func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.statsSvc.GetStats()
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, stats)
}

// GetMTTRStats returns MTTA and MTTR over a configurable window.
// GET /api/v1/dashboard/mttr-stats?hours=24
func (h *DashboardHandler) GetMTTRStats(c *gin.Context) {
	hours := 24
	if v := c.Query("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hours = n
		}
	}

	stats, err := h.statsSvc.GetMTTRStats(hours)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, stats)
}

// GetMTTRTrend returns day-by-day MTTA/MTTR means.
// GET /api/v1/dashboard/mttr-trend?days=30
func (h *DashboardHandler) GetMTTRTrend(c *gin.Context) {
	days := parseDays(c, 30, 365)

	result, err := h.statsSvc.GetMTTRTrend(days)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// GetAlertTrend returns daily fired/resolved counts for trend charts.
// GET /api/v1/dashboard/alert-trend?days=30
func (h *DashboardHandler) GetAlertTrend(c *gin.Context) {
	days := parseDays(c, 30, 365)

	result, err := h.statsSvc.GetAlertTrend(days)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// GetTopRules returns the most frequently firing alert rules.
// GET /api/v1/dashboard/top-rules?days=30&limit=10
func (h *DashboardHandler) GetTopRules(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			days = n
		}
	}
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 50 {
			limit = n
		}
	}

	result, err := h.statsSvc.GetTopRules(days, limit)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// GetSeverityHistory returns daily alert counts broken down by severity.
// GET /api/v1/dashboard/severity-history?days=30
func (h *DashboardHandler) GetSeverityHistory(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			days = n
		}
	}

	result, err := h.statsSvc.GetSeverityHistory(days)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// ExportReport streams a CSV report covering daily alert counts and MTTA/MTTR.
// GET /api/v1/dashboard/export?start_date=2006-01-02&end_date=2006-01-02
func (h *DashboardHandler) ExportReport(c *gin.Context) {
	const dateFmt = "2006-01-02"
	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, 0, -29)

	if v := c.Query("start_date"); v != "" {
		if t, err := time.Parse(dateFmt, v); err == nil {
			startDate = t
		}
	}
	if v := c.Query("end_date"); v != "" {
		if t, err := time.Parse(dateFmt, v); err == nil {
			endDate = t
		}
	}
	if endDate.Before(startDate) {
		endDate = startDate
	}
	if endDate.Sub(startDate) > 366*24*time.Hour {
		startDate = endDate.AddDate(0, 0, -365)
	}

	data, err := h.statsSvc.ExportReport(startDate, endDate)
	if err != nil {
		Error(c, err)
		return
	}

	// Stream CSV
	fname := service.GetExportFilename(startDate, endDate)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+fname)

	w := csv.NewWriter(c.Writer)

	// Section 1: daily summary
	_ = w.Write([]string{"# Daily Alert Summary"})
	_ = w.Write([]string{
		"Date", "Total", "Critical", "Warning", "Info",
		"Resolved", "Avg MTTA (min)", "Avg MTTR (min)",
	})
	fmtF := func(f float64) string {
		if f < 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f", f)
	}
	for _, d := range data.Dates {
		s := data.DayMap[d]
		total := s.Critical + s.Warning + s.Info
		_ = w.Write([]string{
			d,
			strconv.FormatInt(total, 10),
			strconv.FormatInt(s.Critical, 10),
			strconv.FormatInt(s.Warning, 10),
			strconv.FormatInt(s.Info, 10),
			strconv.FormatInt(s.Resolved, 10),
			fmtF(s.AvgMTTA),
			fmtF(s.AvgMTTR),
		})
	}

	// Section 2: top rules
	_ = w.Write([]string{})
	_ = w.Write([]string{"# Top Alert Rules"})
	_ = w.Write([]string{"Rule Name", "Total", "Critical", "Warning", "Info"})
	for _, r := range data.TopRules {
		_ = w.Write([]string{
			r.AlertName,
			strconv.FormatInt(r.Cnt, 10),
			strconv.FormatInt(r.Critical, 10),
			strconv.FormatInt(r.Warning, 10),
			strconv.FormatInt(r.Info, 10),
		})
	}
	w.Flush()
}

// ChannelStats returns incident statistics grouped by channel.
// GET /api/v1/dashboard/channel-stats?days=30
func (h *DashboardHandler) ChannelStats(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 {
			days = d
		}
	}

	rows, err := h.statsSvc.ChannelStats(days)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rows)
}

// TeamStats returns incident statistics grouped by team.
// GET /api/v1/dashboard/team-stats?days=30
func (h *DashboardHandler) TeamStats(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 {
			days = d
		}
	}

	rows, err := h.statsSvc.TeamStats(days)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rows)
}

// IncidentTrend returns daily incident counts for the last N days.
// GET /api/v1/dashboard/incident-trend?days=30
func (h *DashboardHandler) IncidentTrend(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 {
			days = d
		}
	}

	result, err := h.statsSvc.IncidentTrend(days)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// IncidentStats returns overall v2 incident statistics.
// GET /api/v1/dashboard/incident-stats
func (h *DashboardHandler) IncidentStats(c *gin.Context) {
	stats, err := h.statsSvc.IncidentStats()
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, stats)
}

// ──────────────────────── Helpers ─────────────────────────

// parseDays extracts a "days" query param clamped to [1, maxDays].
func parseDays(c *gin.Context, defaultDays, maxDays int) int {
	if v := c.Query("days"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= maxDays {
			return n
		}
	}
	return defaultDays
}
