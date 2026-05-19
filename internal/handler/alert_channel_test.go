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

// setupAlertChannelTest creates a fully wired AlertChannelHandler against
// a real test database (requires SREAGENT_TEST_DSN).
func setupAlertChannelTest(t *testing.T) (*handler.AlertChannelHandler, *gin.Engine, func()) {
	t.Helper()
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	logger := testutil.TestLogger()
	channelRepo := repository.NewAlertChannelRepository(db)
	mediaRepo := repository.NewNotifyMediaRepository(db)
	svc := service.NewAlertChannelService(channelRepo, mediaRepo, logger)
	h := handler.NewAlertChannelHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/alert-channels", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		h.Create(c)
	})
	r.GET("/api/v1/alert-channels", h.List)
	r.GET("/api/v1/alert-channels/:id", h.Get)
	r.PUT("/api/v1/alert-channels/:id", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		h.Update(c)
	})
	r.DELETE("/api/v1/alert-channels/:id", h.Delete)
	r.POST("/api/v1/alert-channels/:id/test", h.Test)

	cleanup := func() {
		testutil.CleanupDB(t, db)
	}
	return h, r, cleanup
}

func TestAlertChannelHandler_List(t *testing.T) {
	_, r, cleanup := setupAlertChannelTest(t)
	defer cleanup()

	t.Run("empty list returns zero total", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/alert-channels?page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Code)

		pageData, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		list, ok := pageData["list"].([]interface{})
		require.True(t, ok)
		assert.Empty(t, list)
		assert.Equal(t, float64(0), pageData["total"])
	})
}

func TestAlertChannelHandler_Create_InvalidBody(t *testing.T) {
	_, r, cleanup := setupAlertChannelTest(t)
	defer cleanup()

	t.Run("missing required fields returns 10001", func(t *testing.T) {
		body := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 10001, resp.Code)
	})

	t.Run("malformed JSON returns 10001", func(t *testing.T) {
		body := bytes.NewBufferString(`{bad json`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 10001, resp.Code)
	})

	t.Run("valid body creates channel and returns 0", func(t *testing.T) {
		isEnabled := true
		reqBody := handler.CreateAlertChannelRequest{
			Name:        "test-channel",
			Description: "a test channel",
			MediaID:     1,
			IsEnabled:   &isEnabled,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Code)

		data, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-channel", data["name"])
	})
}

func TestAlertChannelHandler_Get_NotFound(t *testing.T) {
	_, r, cleanup := setupAlertChannelTest(t)
	defer cleanup()

	t.Run("non-existent ID returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/alert-channels/99999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, 0, resp.Code)
	})

	t.Run("invalid ID param returns invalid param error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/alert-channels/not-a-number", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 10001, resp.Code)
	})
}

func TestAlertChannelHandler_Test(t *testing.T) {
	_, r, cleanup := setupAlertChannelTest(t)
	defer cleanup()

	t.Run("non-existent channel returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-channels/99999/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, 0, resp.Code)
	})
}
