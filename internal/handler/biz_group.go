package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// BizGroupHandler handles HTTP requests for business groups.
type BizGroupHandler struct {
	svc *service.BizGroupService
	log *zap.Logger
}

// NewBizGroupHandler creates a new BizGroupHandler.
func NewBizGroupHandler(svc *service.BizGroupService, logger ...*zap.Logger) *BizGroupHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &BizGroupHandler{svc: svc, log: l}
}

// CreateBizGroupRequest is the request body for creating a business group.
type CreateBizGroupRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	ParentID    *uint            `json:"parent_id"`
	Labels      model.JSONLabels `json:"labels"`
}

// UpdateBizGroupRequest is the request body for updating a business group.
type UpdateBizGroupRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	ParentID    *uint            `json:"parent_id"`
	Labels      model.JSONLabels `json:"labels"`
}

// AddBizGroupMemberRequest is the request body for adding a member.
type AddBizGroupMemberRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role"` // admin, member
}

// Create creates a new business group.
func (h *BizGroupHandler) Create(c *gin.Context) {
	var req CreateBizGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("biz group create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	group := &model.BizGroup{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		Labels:      req.Labels,
	}

	if err := h.svc.Create(c.Request.Context(), group); err != nil {
		Error(c, err)
		return
	}

	Success(c, group)
}

// Get returns a single business group by ID.
func (h *BizGroupHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	group, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, group)
}

// List returns a paginated list of business groups.
func (h *BizGroupHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// ListTree returns all business groups as a tree structure.
func (h *BizGroupHandler) ListTree(c *gin.Context) {
	tree, err := h.svc.ListTree(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, tree)
}

// Update updates a business group.
func (h *BizGroupHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateBizGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("biz group update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("group_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	group := &model.BizGroup{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		Labels:      req.Labels,
	}
	group.ID = id

	if err := h.svc.Update(c.Request.Context(), group); err != nil {
		Error(c, err)
		return
	}

	Success(c, group)
}

// Delete deletes a business group.
func (h *BizGroupHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("biz group delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("group_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// AddMember adds a user to a business group.
func (h *BizGroupHandler) AddMember(c *gin.Context) {
	groupID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req AddBizGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.AddMember(c.Request.Context(), groupID, req.UserID, req.Role); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// RemoveMember removes a user from a business group.
func (h *BizGroupHandler) RemoveMember(c *gin.Context) {
	groupID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID, err := GetIDParam(c, "uid")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.RemoveMember(c.Request.Context(), groupID, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// ListMembers returns all members of a business group.
func (h *BizGroupHandler) ListMembers(c *gin.Context) {
	groupID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	members, err := h.svc.ListMembers(c.Request.Context(), groupID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, members)
}
