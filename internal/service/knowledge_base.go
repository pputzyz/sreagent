package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// KnowledgeBaseService manages the knowledge base (SOP, incident cases, runbooks).
type KnowledgeBaseService struct {
	repo   *repository.KnowledgeRepository
	aiSvc  *AIService
	logger *zap.Logger
}

func NewKnowledgeBaseService(repo *repository.KnowledgeRepository, aiSvc *AIService, logger *zap.Logger) *KnowledgeBaseService {
	return &KnowledgeBaseService{repo: repo, aiSvc: aiSvc, logger: logger}
}

// Add creates a new knowledge document.
func (s *KnowledgeBaseService) Add(ctx context.Context, doc *model.KnowledgeDocument) error {
	if doc.Status == "" {
		doc.Status = "active"
	}
	return s.repo.Create(ctx, doc)
}

// GetByID returns a knowledge document by ID and increments view count.
func (s *KnowledgeBaseService) GetByID(ctx context.Context, id uint) (*model.KnowledgeDocument, error) {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.IncrementViewCount(ctx, id); err != nil {
		s.logger.Debug("failed to increment view count", zap.Uint("id", id), zap.Error(err))
	}
	return doc, nil
}

// Update updates a knowledge document.
func (s *KnowledgeBaseService) Update(ctx context.Context, doc *model.KnowledgeDocument) error {
	return s.repo.Update(ctx, doc)
}

// Delete soft-deletes a knowledge document.
func (s *KnowledgeBaseService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// Search performs FULLTEXT search on the knowledge base.
func (s *KnowledgeBaseService) Search(ctx context.Context, query string, source string, topK int) ([]*model.KnowledgeDocument, error) {
	return s.repo.Search(ctx, query, source, topK)
}

// List returns paginated knowledge documents.
func (s *KnowledgeBaseService) List(ctx context.Context, source string, page, pageSize int) ([]model.KnowledgeDocument, int64, error) {
	return s.repo.List(ctx, source, page, pageSize)
}

// IncreaseHelpful increments the helpful count for a document.
func (s *KnowledgeBaseService) IncreaseHelpful(ctx context.Context, id uint) error {
	return s.repo.IncrementHelpfulCount(ctx, id)
}

// SummarizeWithLLM generates an AI summary for a document (async).
func (s *KnowledgeBaseService) SummarizeWithLLM(ctx context.Context, id uint) error {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	if len(doc.Content) < 200 {
		return nil // too short to summarize
	}

	cfg, err := s.aiSvc.loadConfig(ctx)
	if err != nil || !cfg.Enabled {
		return nil // AI not configured, skip
	}

	systemPrompt := "你是知识库摘要助手。请为以下文档生成一段简洁的中文摘要（不超过 200 字）。只输出摘要文本。"
	summary, err := s.aiSvc.callLLMWithSystem(ctx, cfg, systemPrompt, doc.Content)
	if err != nil {
		s.logger.Warn("knowledge summarize failed", zap.Uint("doc_id", id), zap.Error(err))
		return nil // don't fail the whole flow
	}

	doc.Summary = summary
	return s.repo.Update(ctx, doc)
}

// IngestFromAlertEvent creates a knowledge document from a resolved alert event.
// TODO: Implement alert-to-knowledge ingestion (P2.4)
func (s *KnowledgeBaseService) IngestFromAlertEvent(ctx context.Context, event interface{}) error {
	return fmt.Errorf("IngestFromAlertEvent not yet implemented")
}
