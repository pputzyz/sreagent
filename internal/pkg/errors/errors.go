package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a structured application error.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus maps error codes to HTTP status codes.
func (e *AppError) HTTPStatus() int {
	switch {
	case e.Code >= 10000 && e.Code < 10100:
		return http.StatusBadRequest
	case e.Code >= 10100 && e.Code < 10200:
		return http.StatusUnauthorized
	case e.Code >= 10200 && e.Code < 10300:
		return http.StatusForbidden
	case e.Code >= 10300 && e.Code < 10400:
		return http.StatusNotFound
	case e.Code >= 10400 && e.Code < 10500:
		return http.StatusConflict
	case e.Code >= 10500 && e.Code < 10600:
		return http.StatusTooManyRequests
	case e.Code >= 40000 && e.Code < 40100:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// Predefined error codes
var (
	// 10000-10099: Validation errors
	ErrBadRequest   = &AppError{Code: 10000, Message: "bad request"}
	ErrInvalidParam = &AppError{Code: 10001, Message: "invalid parameter"}
	ErrMissingParam = &AppError{Code: 10000, Message: "missing required parameter"}
	ErrBusiness     = &AppError{Code: 10002, Message: "business error"}

	// 10100-10199: Authentication errors
	ErrUnauthorized = &AppError{Code: 10100, Message: "unauthorized"}

	// 40001: Canonical unauthorized (CLAUDE.md spec)
	ErrUnauth = &AppError{Code: 40001, Message: "unauthorized"}
	ErrInvalidToken = &AppError{Code: 10101, Message: "invalid or expired token"}
	ErrInvalidCreds = &AppError{Code: 10102, Message: "invalid credentials"}

	// 10200-10299: Authorization errors
	ErrForbidden    = &AppError{Code: 10200, Message: "forbidden"}
	ErrNoPermission = &AppError{Code: 10201, Message: "no permission"}

	// 10300-10399: Not found errors
	ErrNotFound        = &AppError{Code: 10300, Message: "resource not found"}
	ErrUserNotFound    = &AppError{Code: 10301, Message: "user not found"}
	ErrRuleNotFound    = &AppError{Code: 10302, Message: "alert rule not found"}
	ErrEventNotFound   = &AppError{Code: 10303, Message: "alert event not found"}
	ErrDSNotFound      = &AppError{Code: 10304, Message: "datasource not found"}
	ErrChannelNotFound = &AppError{Code: 10305, Message: "notification channel not found"}
	ErrPolicyNotFound  = &AppError{Code: 10306, Message: "notification policy not found"}
	ErrTeamNotFound    = &AppError{Code: 10307, Message: "team not found"}

	// Notification system v2 errors
	ErrNotifyRuleNotFound    = &AppError{Code: 10308, Message: "notify rule not found"}
	ErrNotifyMediaNotFound   = &AppError{Code: 10309, Message: "notify media not found"}
	ErrTemplateNotFound      = &AppError{Code: 10310, Message: "message template not found"}
	ErrSubscribeRuleNotFound = &AppError{Code: 10311, Message: "subscribe rule not found"}
	ErrBizGroupNotFound      = &AppError{Code: 10312, Message: "business group not found"}
	ErrBuiltinDelete         = &AppError{Code: 10313, Message: "cannot delete built-in resource"}
	ErrTemplateRender        = &AppError{Code: 10314, Message: "template rendering failed"}

	// v2 Channel / Incident errors
	ErrCollabChannelNotFound = &AppError{Code: 10315, Message: "collaboration channel not found"}
	ErrIncidentNotFound      = &AppError{Code: 10316, Message: "incident not found"}

	// 10400-10499: Conflict errors
	ErrConflict          = &AppError{Code: 10400, Message: "resource already exists"}
	ErrDuplicateName     = &AppError{Code: 10401, Message: "name already taken"}
	ErrInvalidTransition = &AppError{Code: 10402, Message: "invalid state transition"}
	ErrVersionConflict   = &AppError{Code: 10403, Message: "version conflict, resource was modified by another request"}

	// 10500-10599: Rate limit errors (HTTP 429)
	ErrRateLimitExceeded = &AppError{Code: 10500, Message: "rate limit exceeded"}

	// 50000+: Internal errors
	ErrInternal    = &AppError{Code: 50000, Message: "internal server error"}
	ErrDatabase    = &AppError{Code: 50001, Message: "database error"}
	ErrRedis       = &AppError{Code: 50002, Message: "redis error"}
	ErrExternalAPI = &AppError{Code: 50003, Message: "external api error"}
)

// Wrap wraps an existing error with an AppError.
func Wrap(base *AppError, err error) *AppError {
	return &AppError{
		Code:    base.Code,
		Message: base.Message,
		Err:     err,
	}
}

// WithMessage creates a new AppError with a custom message.
func WithMessage(base *AppError, msg string) *AppError {
	return &AppError{
		Code:    base.Code,
		Message: msg,
		Err:     base.Err,
	}
}
