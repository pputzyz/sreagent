package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

type TodoItemHandler struct {
	svc *service.TodoItemService
}

func NewTodoItemHandler(svc *service.TodoItemService) *TodoItemHandler {
	return &TodoItemHandler{svc: svc}
}

// List handles GET /todos — list user's todo items.
func (h *TodoItemHandler) List(c *gin.Context) {
	uid := GetCurrentUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	items, total, err := h.svc.List(c.Request.Context(), uid, status, page, pageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, items, total, page, pageSize)
}

// Create handles POST /todos.
func (h *TodoItemHandler) Create(c *gin.Context) {
	uid := GetCurrentUserID(c)
	var req service.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	item, err := h.svc.Create(c.Request.Context(), uid, &req)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, item)
}

// Update handles PUT /todos/:id.
func (h *TodoItemHandler) Update(c *gin.Context) {
	uid := GetCurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	var req service.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	item, err := h.svc.Update(c.Request.Context(), uint(id), uid, &req)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, item)
}

// Complete handles PATCH /todos/:id/complete.
func (h *TodoItemHandler) Complete(c *gin.Context) {
	uid := GetCurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	if err := h.svc.Complete(c.Request.Context(), uint(id), uid); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// Delete handles DELETE /todos/:id.
func (h *TodoItemHandler) Delete(c *gin.Context) {
	uid := GetCurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id), uid); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// CountPending handles GET /todos/pending-count.
func (h *TodoItemHandler) CountPending(c *gin.Context) {
	uid := GetCurrentUserID(c)
	count, err := h.svc.CountPending(c.Request.Context(), uid)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, gin.H{"count": count})
}
