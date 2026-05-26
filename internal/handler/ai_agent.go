package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
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

	// 异步执行：立即返回任务 ID，前端轮询状态
	uid, _ := c.Get("user_id")
	userID, _ := uid.(uint)
	task, err := h.agentSvc.StartAgent(c.Request.Context(), userID, req.Query)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "Agent 启动失败: "+err.Error()))
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

	// Ownership check: only the task creator or admin can view the task
	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == string(model.RoleAdmin)
	if task.UserID != currentUserID && !isAdmin {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, "无权访问该任务"))
		return
	}

	Success(c, task)
}

// StreamAgentTask godoc
// @Summary SSE 流式推送 Agent 任务更新
// @Description 通过 SSE 实时推送 Agent 任务状态变化，替代轮询
// @Tags AI Agent
// @Produce text/event-stream
// @Param id path string true "任务 ID"
// @Success 200 {string} string "SSE stream"
// @Router /ai/agent/stream/{id} [get]
func (h *AgentHandler) StreamAgentTask(c *gin.Context) {
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

	// Ownership check
	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == string(model.RoleAdmin)
	if task.UserID != currentUserID && !isAdmin {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, "无权访问该任务"))
		return
	}

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.agentSvc.Subscribe(id)
	defer h.agentSvc.Unsubscribe(id, ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case updated, ok := <-ch:
			if !ok {
				return false
			}
			data, err := json.Marshal(updated)
			if err != nil {
				return true // skip bad data, keep stream alive
			}
			fmt.Fprintf(w, "event: task\ndata: %s\n\n", data)
			// 终态关闭流
			if updated.Status == "completed" || updated.Status == "failed" {
				return false
			}
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// ListConversations godoc
// @Summary 列出 AI 会话
// @Tags AI Agent
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} handler.SuccessResponse
// @Router /ai/agent/conversations [get]
func (h *AgentHandler) ListConversations(c *gin.Context) {
	pq := GetPageQuery(c)
	uid, _ := c.Get("user_id")
	userID, _ := uid.(uint)

	convs, total, err := h.agentSvc.ListConversations(c.Request.Context(), userID, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, convs, total, pq.Page, pq.PageSize)
}

// GetConversation godoc
// @Summary 获取 AI 会话详情
// @Tags AI Agent
// @Produce json
// @Param id path int true "会话 ID"
// @Success 200 {object} model.AIConversation
// @Router /ai/agent/conversations/{id} [get]
func (h *AgentHandler) GetConversation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	conv, err := h.agentSvc.GetConversation(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	// Ownership check: only the conversation creator or admin can view it
	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == string(model.RoleAdmin)
	if conv.UserID != currentUserID && !isAdmin {
		Error(c, apperr.ErrForbidden)
		return
	}

	Success(c, conv)
}

// DeleteConversation godoc
// @Summary 删除 AI 会话
// @Tags AI Agent
// @Produce json
// @Param id path int true "会话 ID"
// @Success 200
// @Router /ai/agent/conversations/{id} [delete]
func (h *AgentHandler) DeleteConversation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	// Ownership check: fetch conversation first to verify ownership
	conv, err := h.agentSvc.GetConversation(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == string(model.RoleAdmin)
	if conv.UserID != currentUserID && !isAdmin {
		Error(c, apperr.ErrForbidden)
		return
	}

	if err := h.agentSvc.DeleteConversation(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, nil)
}

// ListToolCalls godoc
// @Summary 列出会话的工具调用记录
// @Tags AI Agent
// @Produce json
// @Param id path int true "会话 ID"
// @Success 200 {array} model.AIToolCall
// @Router /ai/agent/conversations/{id}/tool-calls [get]
func (h *AgentHandler) ListToolCalls(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	// Ownership check: verify the conversation belongs to the current user
	conv, err := h.agentSvc.GetConversation(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == string(model.RoleAdmin)
	if conv.UserID != currentUserID && !isAdmin {
		Error(c, apperr.ErrForbidden)
		return
	}

	calls, err := h.agentSvc.ListToolCalls(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, calls)
}
