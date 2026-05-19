package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/handler"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/pkg/types"
)

// setupGinContext creates a gin.Context with a recording response writer
// for unit-testing handler helper functions.
func setupGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// parseResponse decodes the response body into a types.Response struct.
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) types.Response {
	t.Helper()
	var resp types.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response body: %s", w.Body.String())
	return resp
}

// Test_Error_app_error_returns_correct_status verifies that passing an AppError
// to handler.Error produces the correct HTTP status code and error response body.
func Test_Error_app_error_returns_correct_status(t *testing.T) {
	tests := []struct {
		name       string
		appErr     *apperr.AppError
		wantStatus int
		wantCode   int
	}{
		{
			name:       "validation error returns 400",
			appErr:     &apperr.AppError{Code: 10001, Message: "invalid parameter"},
			wantStatus: http.StatusBadRequest,
			wantCode:   10001,
		},
		{
			name:       "business error returns 400",
			appErr:     &apperr.AppError{Code: 10002, Message: "missing required parameter"},
			wantStatus: http.StatusBadRequest,
			wantCode:   10002,
		},
		{
			name:       "not found error returns 404",
			appErr:     apperr.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   10300,
		},
		{
			name:       "forbidden error returns 403",
			appErr:     apperr.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantCode:   10200,
		},
		{
			name:       "unauthorized error returns 401",
			appErr:     apperr.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantCode:   10100,
		},
		{
			name:       "database error returns 500",
			appErr:     apperr.ErrDatabase,
			wantStatus: http.StatusInternalServerError,
			wantCode:   50001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupGinContext()
			handler.Error(c, tt.appErr)

			assert.Equal(t, tt.wantStatus, w.Code)

			resp := parseResponse(t, w)
			assert.Equal(t, tt.wantCode, resp.Code)
			assert.Equal(t, tt.appErr.Message, resp.Message)
		})
	}
}

// Test_Error_generic_error_returns_500 verifies that passing a non-AppError
// to handler.Error results in a 500 Internal Server Error response.
func Test_Error_generic_error_returns_500(t *testing.T) {
	c, w := setupGinContext()

	handler.Error(c, errors.New("something unexpected happened"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 50000, resp.Code)
	assert.Equal(t, "internal server error", resp.Message)
}

// Test_Error_record_not_found_returns_404 verifies that gorm.ErrRecordNotFound
// is mapped to a 404 Not Found response.
func Test_Error_record_not_found_returns_404(t *testing.T) {
	c, w := setupGinContext()

	handler.Error(c, gorm.ErrRecordNotFound)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 10300, resp.Code)
	assert.Equal(t, "resource not found", resp.Message)
}
