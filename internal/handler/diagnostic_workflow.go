package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// DiagnosticWorkflowHandler handles diagnostic workflow API endpoints.
type DiagnosticWorkflowHandler struct {
	svc *service.DiagnosticWorkflowService
}

func NewDiagnosticWorkflowHandler(svc *service.DiagnosticWorkflowService) *DiagnosticWorkflowHandler {
	return &DiagnosticWorkflowHandler{svc: svc}
}

// --- Workflow CRUD ---

func (h *DiagnosticWorkflowHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	category := c.Query("category")
	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		v := e == "true" || e == "1"
		enabled = &v
	}

	wfs, total, err := h.svc.List(c.Request.Context(), category, enabled, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, gin.H{"list": wfs, "total": total})
}

func (h *DiagnosticWorkflowHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	wf, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	// Load steps
	steps, _ := h.svc.ListSteps(c.Request.Context(), uint(id))
	Success(c, gin.H{"workflow": wf, "steps": steps})
}

func (h *DiagnosticWorkflowHandler) Create(c *gin.Context) {
	var req struct {
		Workflow model.DiagnosticWorkflow       `json:"workflow"`
		Steps    []model.DiagnosticWorkflowStep  `json:"steps"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.Create(c.Request.Context(), &req.Workflow); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	if len(req.Steps) > 0 {
		if err := h.svc.ReplaceSteps(c.Request.Context(), req.Workflow.ID, req.Steps); err != nil {
			Error(c, apperr.Wrap(apperr.ErrDatabase, err))
			return
		}
	}

	Success(c, req.Workflow)
}

func (h *DiagnosticWorkflowHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	wf, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	if err := c.ShouldBindJSON(wf); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.Update(c.Request.Context(), wf); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, wf)
}

func (h *DiagnosticWorkflowHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, nil)
}

// --- Steps ---

func (h *DiagnosticWorkflowHandler) ReplaceSteps(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	var steps []model.DiagnosticWorkflowStep
	if err := c.ShouldBindJSON(&steps); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.ReplaceSteps(c.Request.Context(), uint(id), steps); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, nil)
}

// --- Run ---

func (h *DiagnosticWorkflowHandler) StartRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	var req struct {
		IncidentID *uint `json:"incident_id"`
	}
	_ = c.ShouldBindJSON(&req)

	uid, _ := c.Get("user_id")
	userID, _ := uid.(uint)

	run, err := h.svc.StartRun(c.Request.Context(), uint(id), req.IncidentID, &userID)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, err.Error()))
		return
	}

	Success(c, run)
}

func (h *DiagnosticWorkflowHandler) GetRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	run, err := h.svc.GetRun(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	steps, _ := h.svc.ListRunSteps(c.Request.Context(), uint(id))
	Success(c, gin.H{"run": run, "steps": steps})
}

func (h *DiagnosticWorkflowHandler) ListRuns(c *gin.Context) {
	pq := GetPageQuery(c)
	status := c.Query("status")

	var workflowID *uint
	if wid := c.Query("workflow_id"); wid != "" {
		if v, err := strconv.ParseUint(wid, 10, 64); err == nil {
			uv := uint(v)
			workflowID = &uv
		}
	}

	var incidentID *uint
	if iid := c.Query("incident_id"); iid != "" {
		if v, err := strconv.ParseUint(iid, 10, 64); err == nil {
			uv := uint(v)
			incidentID = &uv
		}
	}

	runs, total, err := h.svc.ListRuns(c.Request.Context(), workflowID, incidentID, status, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, gin.H{"list": runs, "total": total})
}

// MatchWorkflows finds workflows matching given labels.
func (h *DiagnosticWorkflowHandler) MatchWorkflows(c *gin.Context) {
	var req struct {
		Labels   map[string]string `json:"labels"`
		Severity string            `json:"severity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	wfs, err := h.svc.FindMatching(c.Request.Context(), req.Labels, req.Severity)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, wfs)
}
