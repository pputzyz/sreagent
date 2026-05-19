package rbac

import (
	"testing"
)

func Test_HasPerm_Admin(t *testing.T) {
	if !HasPerm("admin", "users.manage") {
		t.Error("admin should have users.manage")
	}
	if !HasPerm("admin", "rules.delete") {
		t.Error("admin should have rules.delete")
	}
}

func Test_HasPerm_Member(t *testing.T) {
	if !HasPerm("member", "rules.view") {
		t.Error("member should have rules.view")
	}
	if HasPerm("member", "users.manage") {
		t.Error("member should NOT have users.manage")
	}
}

func Test_HasPerm_Viewer(t *testing.T) {
	if !HasPerm("viewer", "rules.view") {
		t.Error("viewer should have rules.view")
	}
	if HasPerm("viewer", "rules.create") {
		t.Error("viewer should NOT have rules.create")
	}
}

func Test_HasPerm_Unknown(t *testing.T) {
	if HasPerm("unknown_role", "rules.view") {
		t.Error("unknown role should not have rules.view")
	}
}

func Test_RoleLevel(t *testing.T) {
	if RoleLevel("admin") <= RoleLevel("team_lead") {
		t.Error("admin should be higher than team_lead")
	}
	if RoleLevel("team_lead") <= RoleLevel("member") {
		t.Error("team_lead should be higher than member")
	}
}

func Test_HighestTeamRole(t *testing.T) {
	r := HighestTeamRole([]string{"member", "team_lead", "viewer"})
	if r != "team_lead" {
		t.Errorf("expected team_lead, got %s", r)
	}
}

func Test_HighestTeamRole_Empty(t *testing.T) {
	r := HighestTeamRole(nil)
	if r != "" {
		t.Errorf("expected empty, got %s", r)
	}
}

func Test_EffectivePerms_MergeTeamRole(t *testing.T) {
	// Global: viewer (read-only), Team: team_lead (can manage rules)
	perms := EffectivePerms("viewer", []string{"team_lead"})
	if !perms["rules.manage"] {
		t.Error("team_lead team role should grant rules.manage even with global viewer")
	}
	if !perms["rules.view"] {
		t.Error("global viewer should retain rules.view")
	}
}

func Test_EffectivePerms_NoTeamRoles(t *testing.T) {
	perms := EffectivePerms("member", nil)
	if !perms["rules.create"] {
		t.Error("member should have rules.create")
	}
	if perms["users.manage"] {
		t.Error("member should not have users.manage")
	}
}

func Test_PermissionsByGlobalRole_AllRolesHavePerms(t *testing.T) {
	for _, role := range []string{"admin", "team_lead", "member", "viewer", "global_viewer", ""} {
		perms := PermissionsByGlobalRole(role)
		if len(perms) == 0 {
			t.Errorf("role %q should have at least one permission", role)
		}
	}
}
