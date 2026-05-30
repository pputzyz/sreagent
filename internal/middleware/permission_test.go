package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sreagent/sreagent/internal/pkg/rbac"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupPermContext creates a gin test context with the given role and optional team roles.
func setupPermContext(t *testing.T, role string, teamRoles []string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(ContextKeyRole, role)
	c.Set(ContextKeyUserID, uint(1))
	if teamRoles != nil {
		c.Set("user_team_roles", teamRoles)
	}
	return c, w
}

// Test_RequirePerm_admin_passes_all_permissions verifies that an admin role
// is granted access to all permissions.
func Test_RequirePerm_admin_passes_all_permissions(t *testing.T) {
	perms := []string{
		"users.manage", "teams.manage", "rules.manage", "rules.create",
		"rules.edit", "rules.delete", "events.manage", "settings.manage",
		"datasources.manage", "audit.view",
	}

	for _, perm := range perms {
		t.Run(perm, func(t *testing.T) {
			c, _ := setupPermContext(t, "admin", nil)
			handler := RequirePerm(perm)
			handler(c)
			assert.False(t, c.IsAborted(), "admin should pass permission %q", perm)
		})
	}
}

// Test_RequirePerm_member_denied_write_operations verifies that a member role
// is denied write operations like rules.delete and settings.manage.
func Test_RequirePerm_member_denied_write_operations(t *testing.T) {
	deniedPerms := []string{
		"rules.delete", "settings.manage", "users.manage",
		"datasources.manage", "audit.view",
	}

	for _, perm := range deniedPerms {
		t.Run(perm, func(t *testing.T) {
			c, w := setupPermContext(t, "member", nil)
			handler := RequirePerm(perm)
			handler(c)
			assert.True(t, c.IsAborted(), "member should be denied permission %q", perm)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// Test_RequirePerm_member_allowed_read_operations verifies that a member role
// is granted access to read operations.
func Test_RequirePerm_member_allowed_read_operations(t *testing.T) {
	allowedPerms := []string{
		"rules.view", "rules.create", "events.ack",
		"schedules.view", "datasources.view", "dashboards.view",
	}

	for _, perm := range allowedPerms {
		t.Run(perm, func(t *testing.T) {
			c, _ := setupPermContext(t, "member", nil)
			handler := RequirePerm(perm)
			handler(c)
			assert.False(t, c.IsAborted(), "member should pass permission %q", perm)
		})
	}
}

// Test_RequirePerm_viewer_denied_write_operations verifies that a viewer
// cannot perform write operations.
func Test_RequirePerm_viewer_denied_write_operations(t *testing.T) {
	deniedPerms := []string{
		"rules.create", "rules.edit", "rules.delete",
		"events.ack", "settings.manage",
	}

	for _, perm := range deniedPerms {
		t.Run(perm, func(t *testing.T) {
			c, w := setupPermContext(t, "viewer", nil)
			handler := RequirePerm(perm)
			handler(c)
			assert.True(t, c.IsAborted(), "viewer should be denied %q", perm)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// Test_RequirePerm_missing_role_returns_403 verifies that a request without
// a role in context is denied.
func Test_RequirePerm_missing_role_returns_403(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	// No role set

	handler := RequirePerm("rules.view")
	handler(c)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test_RequirePerm_team_role_elevation verifies that a member with a
// team_lead team role can access team_lead-level permissions.
func Test_RequirePerm_team_role_elevation(t *testing.T) {
	// Member globally, but team_lead in a team
	c, _ := setupPermContext(t, "member", []string{"team_lead"})
	handler := RequirePerm("rules.edit")
	handler(c)
	assert.False(t, c.IsAborted(),
		"member with team_lead team role should pass rules.edit")
}

// Test_RequirePerm_team_roles_do_not_elevate_admin verifies that team roles
// cannot elevate a member to admin-level permissions.
func Test_RequirePerm_team_roles_do_not_elevate_admin(t *testing.T) {
	// Member with team_lead team role should still not get admin-only perms
	c, w := setupPermContext(t, "member", []string{"team_lead"})
	handler := RequirePerm("users.manage")
	handler(c)
	assert.True(t, c.IsAborted(),
		"team_lead team role should NOT grant users.manage (admin-only)")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test_OnPermissionDenied_callback_invoked verifies that the OnPermissionDenied
// callback is called when permission is denied.
func Test_OnPermissionDenied_callback_invoked(t *testing.T) {
	var mu sync.Mutex
	var calledUserID uint
	var calledPerm string
	var calledPath string

	originalCallback := OnPermissionDenied
	defer func() { OnPermissionDenied = originalCallback }()

	OnPermissionDenied = func(userID uint, perm string, path string) {
		mu.Lock()
		defer mu.Unlock()
		calledUserID = userID
		calledPerm = perm
		calledPath = path
	}

	c, _ := setupPermContext(t, "viewer", nil)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/rules", nil)

	handler := RequirePerm("rules.delete")
	handler(c)

	assert.True(t, c.IsAborted())
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, uint(1), calledUserID, "callback should receive the user ID")
	assert.Equal(t, "rules.delete", calledPerm, "callback should receive the denied permission")
	assert.Equal(t, "/api/rules", calledPath, "callback should receive the request path")
}

// Test_OnPermissionDenied_not_called_on_grant verifies that the callback
// is NOT invoked when access is granted.
func Test_OnPermissionDenied_not_called_on_grant(t *testing.T) {
	var mu sync.Mutex
	called := false

	originalCallback := OnPermissionDenied
	defer func() { OnPermissionDenied = originalCallback }()

	OnPermissionDenied = func(userID uint, perm string, path string) {
		mu.Lock()
		defer mu.Unlock()
		called = true
	}

	c, _ := setupPermContext(t, "admin", nil)
	handler := RequirePerm("rules.delete")
	handler(c)

	assert.False(t, c.IsAborted())
	mu.Lock()
	defer mu.Unlock()
	assert.False(t, called, "callback should NOT be called when access is granted")
}

// Test_RequirePerm_invalid_role_type_returns_403 verifies that a non-string
// role value in context results in 403.
func Test_RequirePerm_invalid_role_type_returns_403(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(ContextKeyRole, 12345) // int, not string

	handler := RequirePerm("rules.view")
	handler(c)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test_EffectivePerms_member_with_team_lead_elevates verifies that
// team_lead team role elevates a member's permissions for team-scoped endpoints.
func Test_EffectivePerms_member_with_team_lead_elevates(t *testing.T) {
	perms := rbac.EffectivePerms("member", []string{"team_lead"})
	// team_lead has rules.edit, member does not
	assert.True(t, perms["rules.edit"], "should gain rules.edit from team_lead")
	// member already has rules.view
	assert.True(t, perms["rules.view"], "should keep rules.view from member")
}

// Test_EffectivePerms_viewer_no_team_roles verifies that a viewer with
// no team roles only has viewer-level permissions.
func Test_EffectivePerms_viewer_no_team_roles(t *testing.T) {
	perms := rbac.EffectivePerms("viewer", nil)
	assert.True(t, perms["rules.view"])
	assert.False(t, perms["rules.edit"])
}

// Test_EffectivePerms_highest_team_role_selected verifies that when multiple
// team roles are present, the highest one is used for permission elevation.
func Test_EffectivePerms_highest_team_role_selected(t *testing.T) {
	perms := rbac.EffectivePerms("viewer", []string{"member", "team_lead"})
	// team_lead has rules.edit, which viewer doesn't have
	assert.True(t, perms["rules.edit"], "highest team role (team_lead) should elevate")
	// team_lead also has events.manage
	assert.True(t, perms["events.manage"], "team_lead permissions should be merged")
}

// ---------------------------------------------------------------------------
// EnforceMode tests (deny vs warn)
// ---------------------------------------------------------------------------

// Test_RequirePerm_deny_mode_blocks_request verifies that in "deny" mode,
// a permission-denied request is blocked with 403.
func Test_RequirePerm_deny_mode_blocks_request(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("deny")

	c, w := setupPermContext(t, "viewer", nil)
	handler := RequirePerm("rules.delete")
	handler(c)

	assert.True(t, c.IsAborted(), "deny mode should abort the request")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test_RequirePerm_warn_mode_allows_denied_request verifies that in "warn" mode,
// a permission-denied request is ALLOWED through (not blocked).
func Test_RequirePerm_warn_mode_allows_denied_request(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("warn")

	c, w := setupPermContext(t, "viewer", nil)
	handler := RequirePerm("rules.delete")
	handler(c)

	assert.False(t, c.IsAborted(), "warn mode should NOT abort the request")
	assert.Equal(t, http.StatusOK, w.Code, "warn mode should return 200")
}

// Test_RequirePerm_warn_mode_no_role_allows_request verifies that in "warn" mode,
// a request with no role in context is allowed through.
func Test_RequirePerm_warn_mode_no_role_allows_request(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("warn")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	// No role set

	handler := RequirePerm("rules.view")
	handler(c)

	assert.False(t, c.IsAborted(), "warn mode should allow request with no role")
}

// Test_RequirePerm_warn_mode_invalid_role_allows_request verifies that in "warn" mode,
// a request with an invalid role type is allowed through.
func Test_RequirePerm_warn_mode_invalid_role_allows_request(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("warn")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(ContextKeyRole, 12345) // int, not string

	handler := RequirePerm("rules.view")
	handler(c)

	assert.False(t, c.IsAborted(), "warn mode should allow request with invalid role type")
}

// Test_RequirePerm_warn_mode_team_role_elevation_still_works verifies that
// team-role elevation works correctly in warn mode.
func Test_RequirePerm_warn_mode_team_role_elevation_still_works(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("warn")

	// Member globally, team_lead in team → should pass rules.edit without warn
	c, _ := setupPermContext(t, "member", []string{"team_lead"})
	handler := RequirePerm("rules.edit")
	handler(c)

	assert.False(t, c.IsAborted(),
		"team-role elevation should grant access in warn mode (no warn needed)")
}

// Test_SetEnforceMode_valid_values verifies that SetEnforceMode accepts
// "warn" and "deny" and rejects other values.
func Test_SetEnforceMode_valid_values(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)

	SetEnforceMode("warn")
	assert.Equal(t, "warn", getEnforceMode())

	SetEnforceMode("deny")
	assert.Equal(t, "deny", getEnforceMode())

	// Invalid value should not change the mode.
	SetEnforceMode("invalid")
	assert.Equal(t, "deny", getEnforceMode(), "invalid value should not change the mode")
}

// Test_RequirePerm_deny_mode_team_lead_denied_admin_perm verifies that
// team_lead is denied admin-only permissions in deny mode.
func Test_RequirePerm_deny_mode_team_lead_denied_admin_perm(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("deny")

	c, w := setupPermContext(t, "team_lead", nil)
	handler := RequirePerm("users.manage")
	handler(c)

	assert.True(t, c.IsAborted(), "team_lead should be denied users.manage")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test_RequirePerm_warn_mode_callback_still_fires verifies that the
// OnPermissionDenied callback fires even in warn mode.
func Test_RequirePerm_warn_mode_callback_still_fires(t *testing.T) {
	originalMode := getEnforceMode()
	defer SetEnforceMode(originalMode)
	SetEnforceMode("warn")

	var mu sync.Mutex
	var called bool

	originalCallback := OnPermissionDenied
	defer func() { OnPermissionDenied = originalCallback }()

	OnPermissionDenied = func(userID uint, perm string, path string) {
		mu.Lock()
		defer mu.Unlock()
		called = true
	}

	c, _ := setupPermContext(t, "viewer", nil)
	handler := RequirePerm("rules.delete")
	handler(c)

	assert.False(t, c.IsAborted(), "warn mode should not abort")

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, called, "OnPermissionDenied callback should fire even in warn mode")
}
