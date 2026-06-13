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

func setupAlertForwarderTest(t *testing.T) (*handler.AlertForwarderHandler, *gin.Engine, func()) {
	t.Helper()
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	// Ensure table exists
	db.Exec(`CREATE TABLE IF NOT EXISTS alert_forwarders (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(128) NOT NULL UNIQUE,
		description VARCHAR(512) NOT NULL DEFAULT '',
		enabled TINYINT(1) NOT NULL DEFAULT 1,
		direction VARCHAR(32) NOT NULL,
		priority INT NOT NULL DEFAULT 0,
		inbound_config JSON DEFAULT NULL,
		outbound_config JSON DEFAULT NULL,
		inbound_severity_mapping JSON DEFAULT NULL,
		outbound_severity_mapping JSON DEFAULT NULL,
		platform_capabilities JSON DEFAULT NULL,
		match_labels JSON DEFAULT NULL,
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	db.Exec("DELETE FROM alert_forwarders")

	forwarderRepo := repository.NewAlertForwarderRepository(db)
	mediaRepo := repository.NewNotifyMediaRepository(db)
	svc := service.NewAlertForwarderService(forwarderRepo, mediaRepo, nil, testutil.TestLogger())
	h := handler.NewAlertForwarderHandler(svc, testutil.TestLogger())

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Routes with simulated auth
	r.POST("/api/v1/alert-forwarders", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("role", "admin")
		h.Create(c)
	})
	r.GET("/api/v1/alert-forwarders", h.List)
	r.GET("/api/v1/alert-forwarders/:id", h.GetByID)
	r.PUT("/api/v1/alert-forwarders/:id", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("role", "admin")
		h.Update(c)
	})
	r.DELETE("/api/v1/alert-forwarders/:id", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("role", "admin")
		h.Delete(c)
	})
	r.POST("/api/v1/alert-forwarders/:id/enable", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		h.Enable(c)
	})
	r.POST("/api/v1/alert-forwarders/:id/disable", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		h.Disable(c)
	})

	cleanup := func() {
		testutil.CleanupDB(t, db)
	}
	return h, r, cleanup
}

func TestAlertForwarderHandler_Create_Outbound(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	body := `{"name":"test-outbound","direction":"outbound","outbound_config":{"target_url":"https://example.com/webhook"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-outbound", data["name"])
	assert.Equal(t, "outbound", data["direction"])
}

func TestAlertForwarderHandler_Create_Inbound_Integrate(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	body := `{"name":"test-integrate","direction":"inbound","inbound_config":{"source_format":"alertmanager","mode":"integrate","auth_type":"none"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-integrate", data["name"])
	// Platform capabilities should be auto-set for integrate mode
	assert.NotNil(t, data["platform_capabilities"])
}

func TestAlertForwarderHandler_Create_InvalidDirection(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	body := `{"name":"test-invalid","direction":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 10001, resp.Code)
}

func TestAlertForwarderHandler_Create_MissingName(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	body := `{"direction":"outbound","outbound_config":{"target_url":"https://example.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAlertForwarderHandler_List_Empty(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alert-forwarders?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestAlertForwarderHandler_EnableDisable(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	// Create
	body := `{"name":"toggle-test","direction":"outbound","outbound_config":{"target_url":"https://example.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	data := createResp.Data.(map[string]interface{})
	id := data["id"].(float64)

	// Disable
	req = httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders/"+formatID(id)+"/disable", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify disabled
	req = httptest.NewRequest(http.MethodGet, "/api/v1/alert-forwarders/"+formatID(id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var getResp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &getResp))
	getData := getResp.Data.(map[string]interface{})
	assert.Equal(t, false, getData["enabled"])

	// Enable
	req = httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders/"+formatID(id)+"/enable", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAlertForwarderHandler_SensitiveFieldsMasked(t *testing.T) {
	_, r, cleanup := setupAlertForwarderTest(t)
	defer cleanup()

	// Create with auth config
	body := `{"name":"sensitive-test","direction":"inbound","inbound_config":{"source_format":"alertmanager","mode":"integrate","auth_type":"bearer","auth_config":{"token":"my-secret-token"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-forwarders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp types.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	data := createResp.Data.(map[string]interface{})

	// Token should be masked in response
	inboundConfig := data["inbound_config"].(map[string]interface{})
	authConfig := inboundConfig["auth_config"].(map[string]interface{})
	assert.Equal(t, "***", authConfig["token"], "token should be masked in response")
}

func formatID(id float64) string {
	b, _ := json.Marshal(int(id))
	return string(b)
}
