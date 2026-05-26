package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

type DashboardV2Handler struct {
	svc *service.DashboardService
}

func NewDashboardV2Handler(svc *service.DashboardService) *DashboardV2Handler {
	return &DashboardV2Handler{svc: svc}
}

type CreateDashboardRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Tags        model.JSONLabels `json:"tags"`
	Config      string           `json:"config"`
	IsPublic    bool             `json:"is_public"`
}

// Create creates a new dashboard.
func (h *DashboardV2Handler) Create(c *gin.Context) {
	var req CreateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	d := &model.Dashboard{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		Config:      req.Config,
		IsPublic:    req.IsPublic,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := h.svc.Create(c.Request.Context(), d); err != nil {
		Error(c, err)
		return
	}

	Success(c, d)
}

// Get returns a single dashboard by ID.
func (h *DashboardV2Handler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	d, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, d)
}

// List returns a paginated list of dashboards.
func (h *DashboardV2Handler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	search := c.Query("search")

	list, total, err := h.svc.List(c.Request.Context(), search, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a dashboard.
func (h *DashboardV2Handler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	d := &model.Dashboard{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		Config:      req.Config,
		IsPublic:    req.IsPublic,
		UpdatedBy:   userID,
	}
	d.ID = id

	if err := h.svc.Update(c.Request.Context(), d); err != nil {
		Error(c, err)
		return
	}

	Success(c, d)
}

// Delete deletes a dashboard.
func (h *DashboardV2Handler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// BindBizGroupRequest is the request body for binding a dashboard to a biz group.
type BindBizGroupRequest struct {
	BizGroupID uint   `json:"biz_group_id" binding:"required"`
	PermFlag   string `json:"perm_flag"` // ro or rw, defaults to ro
}

// BindBizGroup binds a dashboard to a business group.
func (h *DashboardV2Handler) BindBizGroup(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req BindBizGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.BindToBizGroup(c.Request.Context(), id, req.BizGroupID, req.PermFlag); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// UnbindBizGroup unbinds a dashboard from a business group.
func (h *DashboardV2Handler) UnbindBizGroup(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	gid, err := GetIDParam(c, "gid")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.UnbindFromBizGroup(c.Request.Context(), id, gid); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// ListBizGroups returns all biz group bindings for a dashboard.
func (h *DashboardV2Handler) ListBizGroups(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	bindings, err := h.svc.ListBizGroups(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, bindings)
}
