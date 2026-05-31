package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockTeamIDQuerier implements the TeamIDQuerier interface for tests.
type mockTeamIDQuerier struct {
	teamIDs []uint
	err     error
}

func (m *mockTeamIDQuerier) ListUserTeamIDs(userID uint) ([]uint, error) {
	return m.teamIDs, m.err
}

func setupTeamScopeContext(t *testing.T, userID interface{}, role string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if userID != nil {
		c.Set(ContextKeyUserID, userID)
	}
	if role != "" {
		c.Set(ContextKeyRole, role)
	}
	return c, w
}

// TestTeamScoped_AdminBypass verifies that admin users get their team IDs loaded
// (for audit/display), but service-layer filtering is skipped for admins.
func TestTeamScoped_AdminBypass(t *testing.T) {
	originalQuerier := TeamIDQuerier
	defer func() { TeamIDQuerier = originalQuerier }()

	TeamIDQuerier = &mockTeamIDQuerier{teamIDs: []uint{10, 20, 30}}

	c, _ := setupTeamScopeContext(t, uint(1), "admin")

	handler := TeamScoped()
	handler(c)

	// Admin still gets team IDs loaded into context
	ids := GetUserTeamIDs(c)
	assert.NotNil(t, ids, "admin should have team IDs loaded")
	assert.Equal(t, []uint{10, 20, 30}, ids)
}

// TestTeamScoped_MemberFiltered verifies that a regular member user gets their
// team IDs loaded into the gin context for downstream filtering.
func TestTeamScoped_MemberFiltered(t *testing.T) {
	originalQuerier := TeamIDQuerier
	defer func() { TeamIDQuerier = originalQuerier }()

	TeamIDQuerier = &mockTeamIDQuerier{teamIDs: []uint{5, 15}}

	c, _ := setupTeamScopeContext(t, uint(42), "member")

	handler := TeamScoped()
	handler(c)

	ids := GetUserTeamIDs(c)
	assert.NotNil(t, ids, "member should have team IDs loaded")
	assert.Equal(t, []uint{5, 15}, ids)
}

// TestTeamScoped_DBDegraded verifies that when the DB query fails, the middleware
// sets a "team_scope_degraded" flag so downstream handlers can detect degraded state.
func TestTeamScoped_DBDegraded(t *testing.T) {
	originalQuerier := TeamIDQuerier
	defer func() { TeamIDQuerier = originalQuerier }()

	TeamIDQuerier = &mockTeamIDQuerier{err: errors.New("database connection refused")}

	c, _ := setupTeamScopeContext(t, uint(42), "member")

	handler := TeamScoped()
	handler(c)

	// Should set degraded flag
	degraded, exists := c.Get("team_scope_degraded")
	assert.True(t, exists, "team_scope_degraded flag should be set on DB error")
	assert.Equal(t, true, degraded)

	// Team IDs should NOT be set
	ids := GetUserTeamIDs(c)
	assert.Nil(t, ids, "team IDs should not be set when degraded")
}

// TestTeamScoped_NilQuerier verifies that when TeamIDQuerier is nil,
// the middleware passes through without loading any team IDs.
func TestTeamScoped_NilQuerier(t *testing.T) {
	originalQuerier := TeamIDQuerier
	defer func() { TeamIDQuerier = originalQuerier }()

	TeamIDQuerier = nil

	c, _ := setupTeamScopeContext(t, uint(42), "member")

	handler := TeamScoped()
	handler(c)

	ids := GetUserTeamIDs(c)
	assert.Nil(t, ids, "nil querier should not load team IDs")
}

// TestTeamScoped_NoUserID verifies that when no user ID is set in context,
// the middleware passes through gracefully.
func TestTeamScoped_NoUserID(t *testing.T) {
	originalQuerier := TeamIDQuerier
	defer func() { TeamIDQuerier = originalQuerier }()

	TeamIDQuerier = &mockTeamIDQuerier{teamIDs: []uint{1}}

	c, _ := setupTeamScopeContext(t, nil, "")

	handler := TeamScoped()
	handler(c)

	ids := GetUserTeamIDs(c)
	assert.Nil(t, ids, "no user ID should result in no team IDs")
}

// TestGetUserTeamIDs_EmptyContext verifies that GetUserTeamIDs returns nil
// when the context has no team IDs set.
func TestGetUserTeamIDs_EmptyContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	ids := GetUserTeamIDs(c)
	assert.Nil(t, ids, "empty context should return nil")
}

// TestGetUserTeamIDs_NonSliceValue verifies that GetUserTeamIDs returns nil
// when the context value is not a []uint.
func TestGetUserTeamIDs_NonSliceValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(ContextKeyUserTeamIDs, "not-a-slice")

	ids := GetUserTeamIDs(c)
	assert.Nil(t, ids, "non-slice value should return nil")
}
