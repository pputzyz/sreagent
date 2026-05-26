package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type BuiltinDashboardHandler struct {
	svc *service.BuiltinDashboardService
}

func NewBuiltinDashboardHandler(svc *service.BuiltinDashboardService) *BuiltinDashboardHandler {
	return &BuiltinDashboardHandler{svc: svc}
}

// List returns builtin dashboards with optional filters.
// GET /builtin-dashboards?category=&component=&query=&page=1&page_size=20
func (h *BuiltinDashboardHandler) List(c *gin.Context) {
	category := c.Query("category")
	component := c.Query("component")
	query := c.Query("query")
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), category, component, query, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a builtin dashboard by ID.
// GET /builtin-dashboards/:id
func (h *BuiltinDashboardHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	d, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, d)
}

// GetByIdent returns a builtin dashboard by slug identifier.
// GET /builtin-dashboards/ident/:ident
func (h *BuiltinDashboardHandler) GetByIdent(c *gin.Context) {
	ident := c.Param("ident")
	if ident == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "ident is required"))
		return
	}

	d, err := h.svc.GetByIdent(c.Request.Context(), ident)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, d)
}

// Import copies a builtin dashboard into the user's dashboard collection.
// POST /builtin-dashboards/:ident/import
func (h *BuiltinDashboardHandler) Import(c *gin.Context) {
	ident := c.Param("ident")
	if ident == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "ident is required"))
		return
	}

	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}

	dash, err := h.svc.Import(c.Request.Context(), ident, userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, dash)
}

// Create adds a new builtin dashboard (admin only).
// POST /builtin-dashboards
func (h *BuiltinDashboardHandler) Create(c *gin.Context) {
	var req CreateBuiltinDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	username := GetCurrentUsername(c)

	d := &model.BuiltinDashboard{
		Name:      req.Name,
		Ident:     req.Ident,
		Category:  req.Category,
		Component: req.Component,
		Tags:      req.Tags,
		Config:    req.Config,
		Version:   req.Version,
		CreateBy:  username,
	}
	if d.Version == 0 {
		d.Version = 1
	}

	if err := h.svc.Create(c.Request.Context(), d); err != nil {
		Error(c, err)
		return
	}
	Success(c, d)
}

// Categories returns distinct categories.
// GET /builtin-dashboards/categories
func (h *BuiltinDashboardHandler) Categories(c *gin.Context) {
	categories, err := h.svc.GetCategories(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, categories)
}

// Components returns distinct components.
// GET /builtin-dashboards/components
func (h *BuiltinDashboardHandler) Components(c *gin.Context) {
	components, err := h.svc.GetComponents(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, components)
}

// --- Request types ---

type CreateBuiltinDashboardRequest struct {
	Name      string `json:"name" binding:"required"`
	Ident     string `json:"ident" binding:"required"`
	Category  string `json:"category"`
	Component string `json:"component"`
	Tags      string `json:"tags"`
	Config    string `json:"config"`
	Version   int    `json:"version"`
}
