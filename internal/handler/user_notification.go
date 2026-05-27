package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

type UserNotificationHandler struct {
	svc *service.UserNotificationService
}

func NewUserNotificationHandler(svc *service.UserNotificationService) *UserNotificationHandler {
	return &UserNotificationHandler{svc: svc}
}

// List handles GET /notifications — list user's notifications.
func (h *UserNotificationHandler) List(c *gin.Context) {
	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var isRead *bool
	if v := c.Query("is_read"); v != "" {
		b := v == "true"
		isRead = &b
	}

	items, total, err := h.svc.List(c.Request.Context(), uid, isRead, page, pageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, items, total, page, pageSize)
}

// CountUnread handles GET /notifications/unread-count.
func (h *UserNotificationHandler) CountUnread(c *gin.Context) {
	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	count, err := h.svc.CountUnread(c.Request.Context(), uid)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, gin.H{"count": count})
}

// MarkRead handles PATCH /notifications/:id/read.
func (h *UserNotificationHandler) MarkRead(c *gin.Context) {
	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), uint(id), uid); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// MarkAllRead handles POST /notifications/read-all.
func (h *UserNotificationHandler) MarkAllRead(c *gin.Context) {
	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	if err := h.svc.MarkAllRead(c.Request.Context(), uid); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// Delete handles DELETE /notifications/:id.
func (h *UserNotificationHandler) Delete(c *gin.Context) {
	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id), uid); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
