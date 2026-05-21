package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// KnowledgeHandler implements the handler pattern used by the project.

// KnowledgeHandler handles knowledge base CRUD + search.
type KnowledgeHandler struct {
	svc *service.KnowledgeBaseService
}

func NewKnowledgeHandler(svc *service.KnowledgeBaseService) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc}
}

// List godoc
// @Summary 列出知识库文档
// @Tags Knowledge
// @Produce json
// @Param source query string false "来源过滤: sop / incident_case / runbook / template_example / wiki"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} handler.SuccessResponse
// @Router /knowledge [get]
func (h *KnowledgeHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	source := c.Query("source")

	docs, total, err := h.svc.List(c.Request.Context(), source, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, docs, total, pq.Page, pq.PageSize)
}

// Get godoc
// @Summary 获取知识库文档详情
// @Tags Knowledge
// @Produce json
// @Param id path int true "文档 ID"
// @Success 200 {object} model.KnowledgeDocument
// @Router /knowledge/{id} [get]
func (h *KnowledgeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	doc, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	Success(c, doc)
}

// Create godoc
// @Summary 创建知识库文档
// @Tags Knowledge
// @Accept json
// @Produce json
// @Param body body model.KnowledgeDocument true "文档内容"
// @Success 200 {object} model.KnowledgeDocument
// @Router /knowledge [post]
func (h *KnowledgeHandler) Create(c *gin.Context) {
	var doc model.KnowledgeDocument
	if err := c.ShouldBindJSON(&doc); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	uid, ok := h.GetCurrentUserID(c)
	if ok {
		doc.OwnerID = &uid
	}

	if err := h.svc.Add(c.Request.Context(), &doc); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, doc)
}

// Update godoc
// @Summary 更新知识库文档
// @Tags Knowledge
// @Accept json
// @Produce json
// @Param id path int true "文档 ID"
// @Param body body model.KnowledgeDocument true "更新内容"
// @Success 200 {object} model.KnowledgeDocument
// @Router /knowledge/{id} [put]
func (h *KnowledgeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	doc, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	if err := c.ShouldBindJSON(doc); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.Update(c.Request.Context(), doc); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, doc)
}

// Delete godoc
// @Summary 删除知识库文档
// @Tags Knowledge
// @Produce json
// @Param id path int true "文档 ID"
// @Success 200
// @Router /knowledge/{id} [delete]
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, nil)
}

// Search godoc
// @Summary 搜索知识库
// @Tags Knowledge
// @Accept json
// @Produce json
// @Param body body object true "搜索参数"
// @Success 200 {array} model.KnowledgeDocument
// @Router /knowledge/search [post]
func (h *KnowledgeHandler) Search(c *gin.Context) {
	var req struct {
		Query  string `json:"query"`
		Source string `json:"source"`
		TopK   int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if req.TopK <= 0 {
		req.TopK = 10
	}

	docs, err := h.svc.Search(c.Request.Context(), req.Query, req.Source, req.TopK)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, docs)
}

// Helpful godoc
// @Summary 标记文档有用
// @Tags Knowledge
// @Produce json
// @Param id path int true "文档 ID"
// @Success 200
// @Router /knowledge/{id}/helpful [post]
func (h *KnowledgeHandler) Helpful(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.svc.IncreaseHelpful(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, nil)
}

// GetCurrentUserID extracts the current user ID from the gin context.
func (h *KnowledgeHandler) GetCurrentUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok
}
