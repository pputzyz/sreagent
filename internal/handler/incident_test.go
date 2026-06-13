package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
	"github.com/sreagent/sreagent/pkg/types"
)

// setupIncidentTest creates a fully wired IncidentHandler against a real test DB.
func setupIncidentTest(t *testing.T) (*handler.IncidentHandler, *gin.Engine, *gorm.DB) {
	t.Helper()
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	// Ensure incident tables exist
	db.Exec(`CREATE TABLE IF NOT EXISTS incidents (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(512) NOT NULL,
		description TEXT,
		severity VARCHAR(32) NOT NULL DEFAULT 'warning',
		status VARCHAR(32) NOT NULL DEFAULT 'triggered',
		channel_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		assigned_to BIGINT UNSIGNED,
		fingerprint VARCHAR(255),
		snoozed_until DATETIME(3),
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
		deleted_at DATETIME(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	db.Exec(`CREATE TABLE IF NOT EXISTS incident_timelines (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		incident_id BIGINT UNSIGNED NOT NULL,
		user_id BIGINT UNSIGNED,
		action VARCHAR(64) NOT NULL,
		content TEXT,
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	db.Exec(`CREATE TABLE IF NOT EXISTS channels (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(128) NOT NULL,
		team_id BIGINT UNSIGNED,
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
		deleted_at DATETIME(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	db.Exec("DELETE FROM incident_timelines")
	db.Exec("DELETE FROM incidents")
	db.Exec("DELETE FROM channels")

	incRepo := repository.NewIncidentRepository(db)
	svc := service.NewIncidentService(incRepo, nil, testutil.TestLogger())
	h := handler.NewIncidentHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Middleware that simulates authenticated user with role and team IDs
	authMiddleware := func(userID uint, role string, teamIDs []uint) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("role", role)
			if len(teamIDs) > 0 {
				c.Set(middleware.ContextKeyUserTeamIDs, teamIDs)
			}
			c.Next()
		}
	}

	r.GET("/api/v1/incidents/:id", func(c *gin.Context) {
		authMiddleware(1, "member", []uint{100})(c)
		if !c.IsAborted() {
			h.Get(c)
		}
	})
	r.GET("/api/v1/incidents/:id/admin", func(c *gin.Context) {
		authMiddleware(1, "admin", nil)(c)
		if !c.IsAborted() {
			h.Get(c)
		}
	})
	r.GET("/api/v1/incidents/:id/other-team", func(c *gin.Context) {
		authMiddleware(2, "member", []uint{999})(c)
		if !c.IsAborted() {
			h.Get(c)
		}
	})

	return h, r, db
}

func TestIncidentHandler_AuthorizeIncident_IDOR(t *testing.T) {
	_, r, db := setupIncidentTest(t)

	// Create a channel owned by team 100
	require.NoError(t, db.Exec("INSERT INTO channels (id, name, team_id) VALUES (1, 'team-100-channel', 100)").Error)

	// Create an incident in channel 1 (team 100)
	require.NoError(t, db.Exec("INSERT INTO incidents (id, title, channel_id, status, triggered_at) VALUES (1, 'test-incident', 1, 'triggered', NOW())").Error)

	t.Run("admin can access any incident", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/1/admin", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("team member can access own team incident", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("other team member cannot access incident", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/1/other-team", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp types.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 10200, resp.Code) // ErrForbidden
	})

	t.Run("nonexistent incident returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/99999/admin", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
