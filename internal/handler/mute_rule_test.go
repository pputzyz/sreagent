package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
	"github.com/sreagent/sreagent/pkg/types"
)

// ---------------------------------------------------------------------------
// Validation tests (no service dependency)
// ---------------------------------------------------------------------------

// Test_MuteRule_PreviewOne_InvalidID — GET /mute-rules/abc/preview
// should return 400 because "abc" is not a valid uint ID.
func Test_MuteRule_PreviewOne_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// PreviewOne first calls GetIDParam, which fails before touching any service.
	h := handler.NewMuteRuleHandler(nil, nil, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mute-rules/abc/preview", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.PreviewOne(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMuteRuleResponse(t, w)
	assert.NotEqual(t, 0, resp.Code, "error code should be non-zero")
}

// Test_MuteRule_Get_InvalidID — GET /mute-rules/abc with non-numeric ID
// should return 400 via GetIDParam.
func Test_MuteRule_Get_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewMuteRuleHandler(nil, nil, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mute-rules/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMuteRuleResponse(t, w)
	assert.NotEqual(t, 0, resp.Code)
}

// Test_MuteRule_Delete_InvalidID — DELETE /mute-rules/xyz with non-numeric ID
// should return 400.
func Test_MuteRule_Delete_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewMuteRuleHandler(nil, nil, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/mute-rules/xyz", nil)
	c.Params = gin.Params{{Key: "id", Value: "xyz"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_MuteRule_BatchEnable_EmptyIDs — POST /mute-rules/batch/enable with empty ids
// should return 400.
func Test_MuteRule_BatchEnable_EmptyIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewMuteRuleHandler(nil, nil, zap.NewNop())

	body := []byte(`{"ids": []}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/mute-rules/batch/enable", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchEnable(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// DB integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

// Test_MuteRule_PreviewOne_NotFound — GET /mute-rules/99999/preview
// should return 404 or appropriate error for non-existent rule.
func Test_MuteRule_PreviewOne_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	logger := testutil.TestLogger()
	muteRepo := repository.NewMuteRuleRepository(db)
	eventRepo := repository.NewAlertEventRepository(db)

	muteSvc := service.NewMuteRuleService(muteRepo, logger)
	eventSvc := service.NewAlertEventService(eventRepo, nil, nil, nil, nil, nil, nil, logger)

	h := handler.NewMuteRuleHandler(muteSvc, eventSvc, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mute-rules/99999/preview", nil)
	c.Params = gin.Params{{Key: "id", Value: "99999"}}

	h.PreviewOne(c)

	// Non-existent ID should result in an error (404 or 500)
	assert.NotEqual(t, http.StatusOK, w.Code, "non-existent rule should not return 200")

	var resp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code, "error code should be non-zero")
}

// parseMuteRuleResponse decodes the response body into a types.Response struct.
func parseMuteRuleResponse(t *testing.T, w *httptest.ResponseRecorder) types.Response {
	t.Helper()
	var resp types.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response body: %s", w.Body.String())
	return resp
}
