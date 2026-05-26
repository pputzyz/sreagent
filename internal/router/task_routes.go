package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerTaskRoutes registers task template and task execution API routes.
func (h *Handlers) registerTaskRoutes(auth *gin.RouterGroup, manage, operate gin.HandlerFunc) {
	// Task Templates (CRUD — manage permission)
	if h.TaskTpl != nil {
		tpl := auth.Group("/task-tpls")
		{
			tpl.GET("", h.TaskTpl.List)
			tpl.GET("/:id", h.TaskTpl.Get)
			tpl.POST("", manage, middleware.RequirePerm("task.write"), h.TaskTpl.Create)
			tpl.PUT("/:id", manage, middleware.RequirePerm("task.write"), h.TaskTpl.Update)
			tpl.DELETE("/:id", manage, middleware.RequirePerm("task.write"), h.TaskTpl.Delete)
		}
	}

	// Task Execution (operate permission)
	if h.Task != nil {
		tasks := auth.Group("/tasks")
		{
			tasks.GET("", h.Task.ListRecords)
			tasks.GET("/:id", h.Task.GetRecord)
			tasks.GET("/:id/hosts", h.Task.ListHostRecords)
			tasks.POST("", operate, middleware.RequirePerm("task.execute"), h.Task.Execute)
			tasks.POST("/direct", operate, middleware.RequirePerm("task.execute"), h.Task.ExecuteDirect)
		}
		// Host record detail by host record ID
		auth.GET("/tasks/hosts/:id", h.Task.GetHostRecord)
	}
}
