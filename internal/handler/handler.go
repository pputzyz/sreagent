package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/i18n"
	"github.com/sreagent/sreagent/pkg/types"
)

// localeOf resolves the request locale from an explicit X-Lang header (set by the
// SPA to mirror its language toggle) or the standard Accept-Language header.
func localeOf(c *gin.Context) string {
	return i18n.Negotiate(c.GetHeader("X-Lang"), c.GetHeader("Accept-Language"))
}

// Success returns a successful JSON response.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, types.Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// SuccessPage returns a successful paginated JSON response.
func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, types.Response{
		Code:    0,
		Message: "ok",
		Data: types.PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// Error returns an error JSON response.
func Error(c *gin.Context, err error) {
	locale := localeOf(c)
	if appErr, ok := err.(*apperr.AppError); ok {
		c.JSON(appErr.HTTPStatus(), types.Response{
			Code:    appErr.Code,
			Message: i18n.LocalizeMessage(locale, appErr.Message),
		})
		return
	}
	// Log unexpected errors for debugging — previously silently swallowed.
	if l, exists := c.Get("logger"); exists {
		if zapLogger, ok := l.(*zap.Logger); ok {
			zapLogger.Error("unhandled error in handler", zap.Error(err))
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, types.Response{
			Code:    apperr.ErrNotFound.Code,
			Message: i18n.LocalizeMessage(locale, "resource not found"),
		})
		return
	}
	c.JSON(http.StatusInternalServerError, types.Response{
		Code:    apperr.ErrInternal.Code,
		Message: i18n.LocalizeMessage(locale, "internal server error"),
	})
}

// GetPageQuery extracts pagination parameters from the request.
func GetPageQuery(c *gin.Context) types.PageQuery {
	pq := types.DefaultPageQuery()
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		pq.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && pageSize > 0 && pageSize <= 100 {
		pq.PageSize = pageSize
	}
	return pq
}

// queryMulti reads a repeated query parameter, tolerating both the bare form
// (?status=a&status=b) and the bracketed form (?status[]=a&status[]=b) that some
// HTTP clients (e.g. axios default) emit for arrays. Returns nil when absent.
func queryMulti(c *gin.Context, key string) []string {
	vals := c.QueryArray(key)
	if bracketed := c.QueryArray(key + "[]"); len(bracketed) > 0 {
		vals = append(vals, bracketed...)
	}
	return vals
}

// GetIDParam extracts an ID parameter from the URL path.
func GetIDParam(c *gin.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, apperr.ErrInvalidParam
	}
	return uint(id), nil
}

// GetCurrentUserID gets the authenticated user's ID from context.
// Returns 0 if not found (should not happen on authenticated routes).
func GetCurrentUserID(c *gin.Context) uint {
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uint); ok {
			return uid
		}
	}
	return 0
}

// GetCurrentUserIDOK is like GetCurrentUserID but also returns a bool
// indicating whether the user ID was found. Use this in handlers where
// a missing user ID should result in a 401 response.
func GetCurrentUserIDOK(c *gin.Context) (uint, bool) {
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uint); ok && uid > 0 {
			return uid, true
		}
	}
	return 0, false
}

// GetCurrentUsername gets the authenticated user's username from context.
func GetCurrentUsername(c *gin.Context) string {
	if v, exists := c.Get("username"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
