package handler

import (
	"testing"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func Test_maskNotifyTarget_BotUser_Masked(t *testing.T) {
	u := &model.User{
		UserType:     model.UserTypeBot,
		NotifyTarget: `{"lark_webhook":"https://open.feishu.cn/open-apis/bot/v2/hook/xxx"}`,
	}
	maskNotifyTarget(u)
	assert.Empty(t, u.NotifyTarget, "bot user's NotifyTarget should be masked for non-admin")
}

func Test_maskNotifyTarget_ChannelUser_Masked(t *testing.T) {
	u := &model.User{
		UserType:     model.UserTypeChannel,
		NotifyTarget: `{"media_id":1}`,
	}
	maskNotifyTarget(u)
	assert.Empty(t, u.NotifyTarget, "channel user's NotifyTarget should be masked for non-admin")
}

func Test_maskNotifyTarget_HumanUser_Unchanged(t *testing.T) {
	u := &model.User{
		UserType:     model.UserTypeHuman,
		NotifyTarget: "some-value",
	}
	maskNotifyTarget(u)
	assert.Equal(t, "some-value", u.NotifyTarget, "human user's NotifyTarget should not be masked")
}

func Test_isCallerAdmin_Admin_ReturnsTrue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.ContextKeyRole, string(model.RoleAdmin))
	assert.True(t, isCallerAdmin(c))
}

func Test_isCallerAdmin_Member_ReturnsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.ContextKeyRole, string(model.RoleMember))
	assert.False(t, isCallerAdmin(c))
}

func Test_isCallerAdmin_TeamLead_ReturnsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.ContextKeyRole, string(model.RoleTeamLead))
	assert.False(t, isCallerAdmin(c))
}

func Test_isCallerAdmin_NoRole_ReturnsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	assert.False(t, isCallerAdmin(c))
}

// Test_ListUsers_NonAdmin_NotifyTargetMasked is a regression test ensuring that
// when a non-admin user lists users, bot/channel users have their NotifyTarget
// (which may contain webhook URLs) stripped from the response.
func Test_ListUsers_NonAdmin_NotifyTargetMasked(t *testing.T) {
	users := []model.User{
		{
			UserType:     model.UserTypeBot,
			NotifyTarget: `{"lark_webhook":"https://open.feishu.cn/open-apis/bot/v2/hook/secret123"}`,
			Username:     "lark-bot",
		},
		{
			UserType:     model.UserTypeHuman,
			NotifyTarget: "",
			Username:     "alice",
		},
		{
			UserType:     model.UserTypeChannel,
			NotifyTarget: `{"media_id":42}`,
			Username:     "alert-channel",
		},
	}

	// Simulate what the List handler does for non-admin callers
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.ContextKeyRole, string(model.RoleMember))

	if !isCallerAdmin(c) {
		for i := range users {
			maskNotifyTarget(&users[i])
		}
	}

	assert.Empty(t, users[0].NotifyTarget, "bot user NotifyTarget must be masked")
	assert.Empty(t, users[1].NotifyTarget, "human user NotifyTarget stays empty")
	assert.Empty(t, users[2].NotifyTarget, "channel user NotifyTarget must be masked")
}

// Test_ListUsers_Admin_NotifyTargetPreserved ensures that admin callers
// can still see bot/channel NotifyTarget values (webhook URLs).
func Test_ListUsers_Admin_NotifyTargetPreserved(t *testing.T) {
	users := []model.User{
		{
			UserType:     model.UserTypeBot,
			NotifyTarget: `{"lark_webhook":"https://open.feishu.cn/open-apis/bot/v2/hook/secret123"}`,
			Username:     "lark-bot",
		},
		{
			UserType:     model.UserTypeChannel,
			NotifyTarget: `{"media_id":42}`,
			Username:     "alert-channel",
		},
	}

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.ContextKeyRole, string(model.RoleAdmin))

	if !isCallerAdmin(c) {
		for i := range users {
			maskNotifyTarget(&users[i])
		}
	}

	assert.Equal(t, `{"lark_webhook":"https://open.feishu.cn/open-apis/bot/v2/hook/secret123"}`, users[0].NotifyTarget,
		"admin should see bot webhook URL")
	assert.Equal(t, `{"media_id":42}`, users[1].NotifyTarget,
		"admin should see channel NotifyTarget")
}
