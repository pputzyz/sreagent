package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// PostMortemHandler manages incident post-mortems.
type PostMortemHandler struct {
	svc *service.PostMortemService
	ai  *service.AIService
}

func NewPostMortemHandler(svc *service.PostMortemService, ai *service.AIService) *PostMortemHandler {
	return &PostMortemHandler{svc: svc, ai: ai}
}

// Get returns (or creates) the post-mortem for an incident.
// GET /api/v1/incidents/:id/post-mortem
func (h *PostMortemHandler) Get(c *gin.Context) {
	incidentID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	userID := GetCurrentUserID(c)
	pm, err := h.svc.GetOrCreate(c.Request.Context(), incidentID, userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pm)
}

// Update saves post-mortem content changes.
// PUT /api/v1/incidents/:id/post-mortem
func (h *PostMortemHandler) Update(c *gin.Context) {
	incidentID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	pm, err := h.svc.Update(c.Request.Context(), incidentID, req.Title, req.Content, req.Status)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pm)
}

// Publish marks a post-mortem as published.
// POST /api/v1/incidents/:id/post-mortem/publish
func (h *PostMortemHandler) Publish(c *gin.Context) {
	incidentID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	pm, err := h.svc.Publish(c.Request.Context(), incidentID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pm)
}

// List returns post-mortems with optional channel/status filter.
// GET /api/v1/post-mortems?channel_id=&status=&page=&page_size=
func (h *PostMortemHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	var channelID uint
	if v := c.Query("channel_id"); v != "" {
		var id uint64
		if _, err := parseUintFromString(v, &id); err == nil {
			channelID = uint(id)
		}
	}
	status := c.Query("status")

	list, total, err := h.svc.List(c.Request.Context(), channelID, status, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// AIGenerate generates a post-mortem draft using AI.
// POST /api/v1/incidents/:id/post-mortem/ai-generate
func (h *PostMortemHandler) AIGenerate(c *gin.Context) {
	if h.ai == nil {
		Error(c, apperr.WithMessage(apperr.ErrMissingParam, "AI service not configured"))
		return
	}

	incidentID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	userID := GetCurrentUserID(c)

	// Get or create post-mortem first
	pm, err := h.svc.GetOrCreate(c.Request.Context(), incidentID, userID)
	if err != nil {
		Error(c, err)
		return
	}

	// Build context for AI from the post-mortem and incident data
	var incidentTitle, incidentDesc string
	if pm.Incident != nil {
		incidentTitle = pm.Incident.Title
		incidentDesc = pm.Incident.Description
	}

	contextText := "故障标题: " + incidentTitle + "\n故障描述: " + incidentDesc + "\n当前复盘草稿:\n" + pm.Content

	analysis, err := h.ai.AnalyzeAlertWithContext(c.Request.Context(), contextText)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "AI generation failed: "+err.Error()))
		return
	}

	// Build AI-generated post-mortem content
	aiContent := buildPostMortemFromAnalysis(incidentTitle, analysis)

	// Save as draft
	updated, err := h.svc.Update(c.Request.Context(), incidentID, "", aiContent, "draft")
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, updated)
}

// AISummary returns an AI summary without saving (for preview).
// POST /api/v1/incidents/:id/post-mortem/ai-summary
func (h *PostMortemHandler) AISummary(c *gin.Context) {
	if h.ai == nil {
		Error(c, apperr.WithMessage(apperr.ErrMissingParam, "AI service not configured"))
		return
	}

	incidentID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	userID := GetCurrentUserID(c)

	pm, err := h.svc.GetOrCreate(c.Request.Context(), incidentID, userID)
	if err != nil {
		Error(c, err)
		return
	}

	var incidentTitle, incidentDesc string
	if pm.Incident != nil {
		incidentTitle = pm.Incident.Title
		incidentDesc = pm.Incident.Description
	}

	contextText := "故障标题: " + incidentTitle + "\n故障描述: " + incidentDesc
	analysis, err := h.ai.AnalyzeAlertWithContext(c.Request.Context(), contextText)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "AI analysis failed: "+err.Error()))
		return
	}

	Success(c, analysis)
}

func buildPostMortemFromAnalysis(title string, a *service.AlertAnalysis) string {
	causes := ""
	for _, c := range a.ProbableCauses {
		causes += "- " + c + "\n"
	}
	steps := ""
	for i, s := range a.RecommendedSteps {
		steps += formatStep(i+1, s)
	}

	return `## 故障概述

**故障标题：** ` + title + `

` + a.Summary + `

---

## 故障影响

` + a.Impact + `

---

## 根因分析

` + a.RootCauseHint + `

### 可能原因

` + causes + `

---

## 解决建议

` + steps + `

---

## 预防措施

（基于以上分析，请补充具体预防措施）

---

*本复盘初稿由 AI 自动生成，请人工审核后发布。*
`
}

func formatStep(n int, s string) string {
	return fmt.Sprintf("%d. %s\n", n, s)
}
