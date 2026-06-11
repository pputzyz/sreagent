package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type StatusSubscriptionHandler struct {
	svc    *service.StatusSubscriptionService
	logger *zap.Logger
}

func NewStatusSubscriptionHandler(svc *service.StatusSubscriptionService, logger *zap.Logger) *StatusSubscriptionHandler {
	return &StatusSubscriptionHandler{svc: svc, logger: logger}
}

type subscribeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *StatusSubscriptionHandler) Subscribe(c *gin.Context) {
	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid email"))
		return
	}
	if err := h.svc.Subscribe(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("failed to subscribe", zap.String("email", req.Email), zap.Error(err))
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}
	Success(c, gin.H{"message": "subscribed successfully"})
}

func (h *StatusSubscriptionHandler) Unsubscribe(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "email is required"))
		return
	}
	if err := h.svc.Unsubscribe(c.Request.Context(), email); err != nil {
		h.logger.Error("failed to unsubscribe", zap.String("email", email), zap.Error(err))
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}
	Success(c, gin.H{"message": "unsubscribed successfully"})
}

func (h *StatusSubscriptionHandler) List(c *gin.Context) {
	subs, err := h.svc.List(c.Request.Context())
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}
	Success(c, subs)
}
