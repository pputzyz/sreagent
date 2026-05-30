package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/service"
)

// maxSilenceMinutes caps user-supplied silence duration at 30 days.
const maxSilenceMinutes = 30 * 24 * 60 // 43200

// resolveOperator figures out which user id to attribute an action to.
// Priority:
//  1. A valid SREAgent JWT in the Authorization header (auto-identified
//     from the browser's localStorage by the action page JS).
//  2. A name typed into the form — matched against users.username.
//  3. Anonymous (userID=0), operator name recorded in the note only.
//
// Returns (userID, displayName). displayName is the name we should stamp
// into the timeline/comment, falling back to the form input.
func (h *AlertActionHandler) resolveOperator(c *gin.Context, formName string) (uint, string) {
	authz := c.GetHeader("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		claims, err := middleware.ParseToken(strings.TrimPrefix(authz, "Bearer "), h.jwtSecret)
		if err == nil && claims != nil && claims.UserID > 0 {
			name := claims.Username
			// Prefer the display_name from DB when available — username is
			// often a login handle, not a human-readable name.
			if u, uerr := h.userRepo.GetByID(c.Request.Context(), claims.UserID); uerr == nil && u != nil {
				if u.DisplayName != "" {
					name = u.DisplayName
				}
			}
			if name == "" {
				name = formName
			}
			return claims.UserID, name
		}
	}
	if formName != "" {
		if u, err := h.userRepo.GetByUsername(c.Request.Context(), formName); err == nil && u != nil {
			return u.ID, formName
		}
	}
	return 0, formName
}

// AlertActionHandler handles no-auth alert action pages (linked from Lark cards).
type AlertActionHandler struct {
	eventSvc  *service.AlertEventService
	userRepo  service.UserLookupService
	jwtSecret string
	logger    *zap.Logger
}

// NewAlertActionHandler creates a new AlertActionHandler.
func NewAlertActionHandler(
	eventSvc *service.AlertEventService,
	userRepo service.UserLookupService,
	jwtSecret string,
	logger *zap.Logger,
) *AlertActionHandler {
	return &AlertActionHandler{
		eventSvc:  eventSvc,
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// ActionPage serves an HTML page for alert operations (no auth required).
// GET /alert-action/:token
func (h *AlertActionHandler) ActionPage(c *gin.Context) {
	token := c.Param("token")

	eventID, err := service.ParseAlertActionToken(token, h.jwtSecret)
	if err != nil {
		h.logger.Warn("invalid alert action token", zap.Error(err))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusForbidden, renderErrorPage("链接无效或已过期", "该操作链接已过期（24小时有效），请从最新的告警通知中获取链接。"))
		return
	}

	event, err := h.eventSvc.GetByID(c.Request.Context(), eventID)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusNotFound, renderErrorPage("告警不存在", "未找到对应的告警事件，可能已被删除。"))
		return
	}

	// Check if there's a pre-selected action from query param
	preAction := c.Query("action")
	durationStr := c.Query("duration")
	duration := 0
	if durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil && d > 0 {
			duration = d
		}
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderActionPage(event, token, preAction, duration))
}

// ExecuteAction handles the action form submission.
// POST /alert-action/:token
func (h *AlertActionHandler) ExecuteAction(c *gin.Context) {
	token := c.Param("token")

	eventID, err := service.ParseAlertActionToken(token, h.jwtSecret)
	if err != nil {
		h.logger.Warn("invalid alert action token on execute", zap.Error(err))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusForbidden, renderErrorPage("链接无效或已过期", "该操作链接已过期（24小时有效），请从最新的告警通知中获取链接。"))
		return
	}

	// Parse form data (from HTML form POST)
	action := c.PostForm("action")
	operatorName := c.PostForm("operator_name")
	note := c.PostForm("note")
	durationStr := c.PostForm("duration")

	if action == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusBadRequest, renderErrorPage("操作无效", "请选择一个操作。"))
		return
	}

	// Resolve the acting user — prefer an Authorization bearer token
	// (auto-attached by the page's JS when the browser has an SREAgent
	// session), then fall back to a typed-in name, then anonymous.
	userID, operatorDisplay := h.resolveOperator(c, operatorName)
	if operatorDisplay == "" {
		operatorDisplay = operatorName
	}

	actionNote := note
	if operatorDisplay != "" && actionNote == "" {
		actionNote = "操作人: " + operatorDisplay
	} else if operatorDisplay != "" {
		actionNote = "操作人: " + operatorDisplay + " | " + note
	}

	var actionErr error
	var successMsg string

	switch action {
	case "acknowledge":
		actionErr = h.eventSvc.Acknowledge(c.Request.Context(), eventID, userID)
		successMsg = "告警已认领"
		if actionNote != "" {
			if commentErr := h.eventSvc.AddComment(c.Request.Context(), eventID, userID, actionNote); commentErr != nil {
				h.logger.Warn("failed to add action comment", zap.Uint("event_id", eventID), zap.Error(commentErr))
			}
		}
	case "silence":
		duration := 60 // default 1 hour
		if durationStr != "" {
			if d, parseErr := strconv.Atoi(durationStr); parseErr == nil && d > 0 {
				duration = d
			}
		}
		// Sanity cap: 30 days. Guards against typos and oversized values
		// that would effectively disable alerting indefinitely.
		if duration > maxSilenceMinutes {
			duration = maxSilenceMinutes
		}
		reason := actionNote
		if reason == "" {
			reason = "Silenced from Lark card"
		}
		actionErr = h.eventSvc.Silence(c.Request.Context(), eventID, userID, duration, reason)
		successMsg = "告警已静默"
	case "resolve":
		resolution := actionNote
		if resolution == "" {
			resolution = "Resolved from Lark card"
		}
		actionErr = h.eventSvc.Resolve(c.Request.Context(), eventID, userID, resolution)
		successMsg = "告警已解决"
	case "close":
		closeNote := actionNote
		if closeNote == "" {
			closeNote = "Closed from Lark card"
		}
		actionErr = h.eventSvc.Close(c.Request.Context(), eventID, userID, closeNote)
		successMsg = "告警已关闭"
	default:
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusBadRequest, renderErrorPage("操作无效", "不支持的操作类型: "+action))
		return
	}

	if actionErr != nil {
		h.logger.Error("alert action failed",
			zap.Uint("event_id", eventID),
			zap.String("action", action),
			zap.Error(actionErr),
		)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, renderResultPage(false, "操作失败", actionErr.Error()))
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderResultPage(true, successMsg, ""))
}
