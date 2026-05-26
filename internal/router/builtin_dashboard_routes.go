package router

import (
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registerBuiltinDashboardRoutes(auth *gin.RouterGroup, adminOnly, manage gin.HandlerFunc) {
	bd := auth.Group("/builtin-dashboards")
	{
		bd.GET("", h.BuiltinDashboard.List)
		bd.GET("/categories", h.BuiltinDashboard.Categories)
		bd.GET("/components", h.BuiltinDashboard.Components)
		bd.GET("/:id", h.BuiltinDashboard.Get)
		bd.GET("/ident/:ident", h.BuiltinDashboard.GetByIdent)
		bd.POST("/:ident/import", h.BuiltinDashboard.Import)
		bd.POST("", adminOnly, h.BuiltinDashboard.Create)
	}
}
