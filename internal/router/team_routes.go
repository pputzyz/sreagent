package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerTeamRoutes registers user, team, and business group routes.
func (h *Handlers) registerTeamRoutes(auth *gin.RouterGroup, adminOnly, manage gin.HandlerFunc) {
	// Users — admin only for management
	users := auth.Group("/users")
	{
		users.GET("", h.User.List)
		users.GET("/:id", h.User.Get)
		users.POST("", adminOnly, middleware.RequirePerm("user.write"), h.User.Create)
		users.POST("/virtual", adminOnly, middleware.RequirePerm("user.write"), h.User.CreateVirtual)
		users.PUT("/:id", adminOnly, middleware.RequirePerm("user.write"), h.User.Update)
		users.PATCH("/:id/active", adminOnly, middleware.RequirePerm("user.write"), h.User.ToggleActive)
		users.PATCH("/:id/password", adminOnly, middleware.RequirePerm("user.write"), h.User.ChangePassword)
		users.DELETE("/:id", adminOnly, middleware.RequirePerm("user.write"), h.User.DeleteUser)
	}

	// Teams
	teams := auth.Group("/teams")
	{
		teams.GET("", h.Team.List)
		teams.GET("/:id", h.Team.Get)
		teams.GET("/:id/members", h.Team.ListMembers)
		teams.POST("", manage, middleware.RequirePerm("team.write"), h.Team.Create)
		teams.PUT("/:id", manage, middleware.RequirePerm("team.write"), h.Team.Update)
		teams.DELETE("/:id", manage, middleware.RequirePerm("team.write"), h.Team.Delete)
		teams.POST("/:id/members", manage, middleware.RequirePerm("team.write"), h.Team.AddMember)
		teams.DELETE("/:id/members/:uid", manage, middleware.RequirePerm("team.write"), h.Team.RemoveMember)
	}

	// Business Groups
	bizGroups := auth.Group("/biz-groups")
	{
		bizGroups.GET("", h.BizGroup.List)
		bizGroups.GET("/tree", h.BizGroup.ListTree)
		bizGroups.GET("/:id", h.BizGroup.Get)
		bizGroups.GET("/:id/members", h.BizGroup.ListMembers)
		bizGroups.POST("", manage, h.BizGroup.Create)
		bizGroups.PUT("/:id", manage, h.BizGroup.Update)
		bizGroups.DELETE("/:id", manage, h.BizGroup.Delete)
		bizGroups.POST("/:id/members", manage, h.BizGroup.AddMember)
		bizGroups.DELETE("/:id/members/:uid", manage, h.BizGroup.RemoveMember)
	}

	// User Contacts (self-service, any authenticated user)
	if h.UserContact != nil {
		contacts := auth.Group("/user/contacts")
		{
			contacts.GET("", h.UserContact.List)
			contacts.POST("", h.UserContact.Create)
			contacts.PUT("/:id", h.UserContact.Update)
			contacts.DELETE("/:id", h.UserContact.Delete)
			contacts.POST("/:id/default", h.UserContact.SetDefault)
			contacts.POST("/:id/verify", h.UserContact.Verify)
		}
	}
}
