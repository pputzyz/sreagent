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

	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
	"github.com/sreagent/sreagent/pkg/types"
)

// ---------------------------------------------------------------------------
// Validation tests (no service dependency)
// ---------------------------------------------------------------------------

// Test_AlertRule_Create_MissingSeverity — POST /alert-rules with missing severity
// should return 400 because severity has binding:"required".
func Test_AlertRule_Create_MissingSeverity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewAlertRuleHandler(nil)

	body := map[string]interface{}{
		"name":       "test-rule",
		"expression": "up == 0",
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/alert-rules", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseAlertRuleResponse(t, w)
	assert.NotEqual(t, 0, resp.Code, "error code should be non-zero")
	assert.Contains(t, resp.Message, "Severity", "error message should mention missing Severity field")
}

// Test_AlertRule_BatchEnable_EmptyIDs — POST /alert-rules/batch/enable with empty ids
// should return 400 because ids has binding:"required,min=1".
func Test_AlertRule_BatchEnable_EmptyIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewAlertRuleHandler(nil)

	body := map[string]interface{}{
		"ids": []uint{},
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/alert-rules/batch/enable", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchEnable(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseAlertRuleResponse(t, w)
	assert.NotEqual(t, 0, resp.Code, "error code should be non-zero")
}

// Test_AlertRule_BatchEnable_MissingBody — POST /alert-rules/batch/enable with no body
// should return 400.
func Test_AlertRule_BatchEnable_MissingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewAlertRuleHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/alert-rules/batch/enable", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchEnable(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_AlertRule_Create_InvalidJSON — POST /alert-rules with malformed JSON
// should return 400.
func Test_AlertRule_Create_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewAlertRuleHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/alert-rules", bytes.NewReader([]byte("{invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_AlertRule_Get_InvalidID — GET /alert-rules/abc with non-numeric ID
// should return 400 via GetIDParam.
func Test_AlertRule_Get_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewAlertRuleHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/alert-rules/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseAlertRuleResponse(t, w)
	assert.NotEqual(t, 0, resp.Code)
}

// Test_AlertRule_ToggleStatus_MissingStatus — POST /alert-rules/1/toggle with no status
// should return 400.
func Test_AlertRule_ToggleStatus_MissingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewAlertRuleHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/alert-rules/1/toggle", bytes.NewReader([]byte("{}")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.ToggleStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// DB integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

// Test_AlertRule_LabelValidationPreview_Empty — GET /alert-rules/label-validation-preview
// Test with no rules — should return {total:0, passing:0, failing:0}.
func Test_AlertRule_LabelValidationPreview_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	logger := testutil.TestLogger()
	ruleRepo := repository.NewAlertRuleRepository(db)
	historyRepo := repository.NewAlertRuleHistoryRepository(db)
	dsRepo := repository.NewDataSourceRepository(db)
	svc := service.NewAlertRuleService(ruleRepo, historyRepo, dsRepo, logger)

	h := handler.NewAlertRuleHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/alert-rules/label-validation-preview", nil)

	h.LabelValidationPreview(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	// The data should be a LabelValidationResult with zero counts
	data, err := json.Marshal(resp.Data)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	total, _ := result["total"].(float64)
	passing, _ := result["passing"].(float64)
	failing, _ := result["failing"].(float64)
	assert.Equal(t, float64(0), total, "total should be 0 with no rules")
	assert.Equal(t, float64(0), passing, "passing should be 0 with no rules")
	assert.Equal(t, float64(0), failing, "failing should be 0 with no rules")
}

// parseAlertRuleResponse decodes the response body into a types.Response struct.
func parseAlertRuleResponse(t *testing.T, w *httptest.ResponseRecorder) types.Response {
	t.Helper()
	var resp types.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response body: %s", w.Body.String())
	return resp
}
