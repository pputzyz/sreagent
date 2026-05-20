package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// AgentHandler handles AI Agent API endpoints.
type AgentHandler struct {
	agentSvc *service.AgentService
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(agentSvc *service.AgentService) *AgentHandler {
	return &AgentHandler{agentSvc: agentSvc}
}

// RunAgent godoc
// @Summary 执行 Agent 任务
// @Description 提交一个自然语言查询，Agent 会自主规划和执行多步骤任务
// @Tags AI Agent
// @Accept json
// @Produce json
// @Param body body object true "查询参数"
// @Success 200 {object} service.AgentTask
// @Router /ai/agent/run [post]
func (h *AgentHandler) RunAgent(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required,max=2000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	task, err := h.agentSvc.RunAgent(c.Request.Context(), req.Query)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "Agent 执行失败: "+err.Error()))
		return
	}

	Success(c, task)
}

// GetAgentTask godoc
// @Summary 获取 Agent 任务详情
// @Description 根据任务 ID 获取 Agent 任务的执行状态和结果
// @Tags AI Agent
// @Produce json
// @Param id path string true "任务 ID"
// @Success 200 {object} service.AgentTask
// @Router /ai/agent/tasks/{id} [get]
func (h *AgentHandler) GetAgentTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "任务 ID 不能为空"))
		return
	}

	task, ok := h.agentSvc.GetTask(id)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrNotFound, "任务不存在"))
		return
	}

	Success(c, task)
}
