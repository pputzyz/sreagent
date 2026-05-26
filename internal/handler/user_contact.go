package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// UserContactHandler handles HTTP requests for user contacts.
type UserContactHandler struct {
	svc *service.UserContactService
	log *zap.Logger
}

// NewUserContactHandler creates a new UserContactHandler.
func NewUserContactHandler(svc *service.UserContactService, logger *zap.Logger) *UserContactHandler {
	return &UserContactHandler{svc: svc, log: logger}
}

// ContactRequest is the request body for creating or updating a contact.
type ContactRequest struct {
	Type  string `json:"type" binding:"required"`
	Value string `json:"value" binding:"required"`
	Name  string `json:"name"`
}

// List returns all contacts for the current user.
func (h *UserContactHandler) List(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}

	contacts, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, contacts)
}

// Create creates a new contact for the current user.
func (h *UserContactHandler) Create(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}

	var req ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	contact := &model.UserContact{
		UserID: userID,
		Type:   req.Type,
		Value:  req.Value,
		Name:   req.Name,
	}

	if err := h.svc.Create(c.Request.Context(), contact); err != nil {
		Error(c, err)
		return
	}

	Success(c, contact)
}

// Update updates a contact. Only the owner can update.
func (h *UserContactHandler) Update(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}

	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	// Verify ownership.
	existing, err := h.svc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		Error(c, err)
		return
	}

	var req ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	existing.Type = req.Type
	existing.Value = req.Value
	existing.Name = req.Name

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		Error(c, err)
		return
	}

	Success(c, existing)
}

// Delete deletes a contact. Only the owner can delete.
func (h *UserContactHandler) Delete(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}

	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// SetDefault sets a contact as the default for its type.
func (h *UserContactHandler) SetDefault(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}

	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.SetDefault(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Verify sends a verification message to the contact (placeholder).
func (h *UserContactHandler) Verify(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}

	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	// Verify ownership.
	if _, err := h.svc.GetByID(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	// Placeholder: verification logic will be implemented when notification channels are integrated.
	h.log.Info("contact verification requested",
		zap.Uint("user_id", userID),
		zap.Uint("contact_id", id),
	)

	Success(c, gin.H{"message": "verification sent"})
}
