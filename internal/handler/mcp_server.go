package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// MCPServerHandler handles HTTP requests for MCP servers.
type MCPServerHandler struct {
	svc      *service.MCPServerService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewMCPServerHandler creates a new MCPServerHandler.
func NewMCPServerHandler(svc *service.MCPServerService, logger *zap.Logger) *MCPServerHandler {
	return &MCPServerHandler{svc: svc, log: logger}
}

// SetAuditService injects the audit log service.
func (h *MCPServerHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request types ---

// CreateMCPServerRequest is the request body for creating an MCP server.
type CreateMCPServerRequest struct {
	Name        string            `json:"name" binding:"required,max=128"`
	URL         string            `json:"url" binding:"required,max=512"`
	Headers     map[string]string `json:"headers"`
	Description string            `json:"description" binding:"max=1024"`
	Enabled     *bool             `json:"enabled"`
}

// UpdateMCPServerRequest is the request body for updating an MCP server.
type UpdateMCPServerRequest struct {
	Name        string            `json:"name" binding:"required,max=128"`
	URL         string            `json:"url" binding:"required,max=512"`
	Headers     map[string]string `json:"headers"`
	Description string            `json:"description" binding:"max=1024"`
	Enabled     *bool             `json:"enabled"`
}

// --- Handler methods ---

// List returns a paginated list of MCP servers.
// GET /mcp-servers?page=1&page_size=20
func (h *MCPServerHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a single MCP server by ID.
// GET /mcp-servers/:id
func (h *MCPServerHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	srv, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, srv)
}

// Create creates a new MCP server.
// POST /mcp-servers
func (h *MCPServerHandler) Create(c *gin.Context) {
	var req CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	srv := &model.MCPServer{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		Enabled:     enabled,
	}
	srv.SetHeadersMap(req.Headers)

	if err := h.svc.Create(c.Request.Context(), srv); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "mcp_server",
			ResourceID:   &srv.ID,
			IP:           c.ClientIP(),
		})
	}

	Success(c, srv)
}

// Update updates an existing MCP server.
// PUT /mcp-servers/:id
func (h *MCPServerHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	input := &model.MCPServer{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		Enabled:     enabled,
	}
	input.SetHeadersMap(req.Headers)

	if err := h.svc.Update(c.Request.Context(), existing, input); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionUpdate,
			ResourceType: "mcp_server",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Delete deletes an MCP server.
// DELETE /mcp-servers/:id
func (h *MCPServerHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionDelete,
			ResourceType: "mcp_server",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// TestConnection tests connectivity to an MCP server.
// POST /mcp-servers/:id/test
func (h *MCPServerHandler) TestConnection(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	srv, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.TestConnection(c.Request.Context(), srv); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, err.Error()))
		return
	}

	Success(c, gin.H{"message": "connection successful"})
}

// ListTools lists available tools from an MCP server.
// GET /mcp-servers/:id/tools
func (h *MCPServerHandler) ListTools(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	srv, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	tools, err := h.svc.ListTools(c.Request.Context(), srv)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, tools)
}
