package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

type TeamHandler struct {
	svc *service.TeamService
}

func NewTeamHandler(svc *service.TeamService) *TeamHandler {
	return &TeamHandler{svc: svc}
}

// CreateTeamRequest is the request body for creating a team.
type CreateTeamRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Labels      model.JSONLabels `json:"labels"`
}

// UpdateTeamRequest is the request body for updating a team.
type UpdateTeamRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Labels      model.JSONLabels `json:"labels"`
}

// AddMemberRequest is the request body for adding a member to a team.
type AddMemberRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role"` // lead, member
}

// Create creates a new team.
func (h *TeamHandler) Create(c *gin.Context) {
	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	team := &model.Team{
		Name:        req.Name,
		Description: req.Description,
		Labels:      req.Labels,
	}

	if err := h.svc.Create(c.Request.Context(), team); err != nil {
		Error(c, err)
		return
	}

	Success(c, team)
}

// Get returns a team by ID.
func (h *TeamHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	team, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, team)
}

// List returns a paginated list of teams.
func (h *TeamHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a team.
func (h *TeamHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	team := &model.Team{
		Name:        req.Name,
		Description: req.Description,
		Labels:      req.Labels,
	}
	team.ID = id

	if err := h.svc.Update(c.Request.Context(), team); err != nil {
		Error(c, err)
		return
	}

	Success(c, team)
}

// Delete deletes a team.
func (h *TeamHandler) Delete(c *gin.Context) {
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

// AddMember adds a user to a team.
func (h *TeamHandler) AddMember(c *gin.Context) {
	teamID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.AddMember(c.Request.Context(), teamID, req.UserID, req.Role); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// RemoveMember removes a user from a team.
func (h *TeamHandler) RemoveMember(c *gin.Context) {
	teamID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID, err := GetIDParam(c, "uid")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.RemoveMember(c.Request.Context(), teamID, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// ListMembers returns all members of a team.
func (h *TeamHandler) ListMembers(c *gin.Context) {
	teamID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	members, err := h.svc.ListMembers(c.Request.Context(), teamID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, members)
}
