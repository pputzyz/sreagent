package service

import (
	"context"
	"fmt"
	"strings"
	"time"

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
// Extracts alert name, severity, labels, annotations, and resolution from the event
// and stores it as an auto-generated incident_case document.
func (s *KnowledgeBaseService) IngestFromAlertEvent(ctx context.Context, event *model.AlertEvent) error {
	if event == nil {
		return fmt.Errorf("IngestFromAlertEvent: event is nil")
	}
	// Only ingest resolved/closed events — firing events are not yet actionable knowledge.
	if event.Status != model.EventStatusResolved && event.Status != model.EventStatusClosed {
		return nil
	}

	// Build label summary
	labelParts := make([]string, 0, len(event.Labels))
	for k, v := range event.Labels {
		labelParts = append(labelParts, fmt.Sprintf("%s=%s", k, v))
	}
	labelStr := ""
	if len(labelParts) > 0 {
		labelStr = "Labels: " + strings.Join(labelParts, ", ")
	}

	// Build annotation summary
	annotationParts := make([]string, 0, len(event.Annotations))
	for k, v := range event.Annotations {
		annotationParts = append(annotationParts, fmt.Sprintf("%s=%s", k, v))
	}
	annotationStr := ""
	if len(annotationParts) > 0 {
		annotationStr = "Annotations: " + strings.Join(annotationParts, ", ")
	}

	// Auto-generate title
	title := fmt.Sprintf("[%s] %s — Resolved", strings.ToUpper(string(event.Severity)), event.AlertName)

	// Build content from event metadata
	var sb strings.Builder
	fmt.Fprintf(&sb, "Alert: %s\n", event.AlertName)
	fmt.Fprintf(&sb, "Severity: %s\n", event.Severity)
	fmt.Fprintf(&sb, "Status: %s\n", event.Status)
	fmt.Fprintf(&sb, "Fired At: %s\n", event.FiredAt.Format(time.RFC3339))
	if event.ResolvedAt != nil {
		fmt.Fprintf(&sb, "Resolved At: %s\n", event.ResolvedAt.Format(time.RFC3339))
		duration := event.ResolvedAt.Sub(event.FiredAt)
		fmt.Fprintf(&sb, "Duration: %s\n", duration.Round(time.Second))
	}
	if event.FireCount > 1 {
		fmt.Fprintf(&sb, "Fire Count: %d\n", event.FireCount)
	}
	if labelStr != "" {
		sb.WriteString(labelStr + "\n")
	}
	if annotationStr != "" {
		sb.WriteString(annotationStr + "\n")
	}
	if event.Resolution != "" {
		fmt.Fprintf(&sb, "\nResolution: %s\n", event.Resolution)
	}
	fmt.Fprintf(&sb, "\nSource: %s\n", event.Source)

	// Build tags from labels for searchability
	tags := make(model.JSONLabels)
	if event.Severity != "" {
		tags["severity"] = string(event.Severity)
	}
	if event.Source != "" {
		tags["source"] = event.Source
	}
	if v, ok := event.Labels["job"]; ok {
		tags["job"] = v
	}
	if v, ok := event.Labels["instance"]; ok {
		tags["instance"] = v
	}

	doc := &model.KnowledgeDocument{
		Source:    model.KBSourceIncidentCase,
		Title:     title,
		Content:   sb.String(),
		Tags:      tags,
		SourceRef: fmt.Sprintf("alert_event:%d", event.ID),
		Status:    "active",
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		s.logger.Error("failed to ingest knowledge from alert event",
			zap.Uint("event_id", event.ID),
			zap.String("alert_name", event.AlertName),
			zap.Error(err),
		)
		return fmt.Errorf("ingest knowledge document: %w", err)
	}

	s.logger.Info("knowledge document ingested from alert event",
		zap.Uint("event_id", event.ID),
		zap.String("alert_name", event.AlertName),
		zap.Uint("doc_id", doc.ID),
	)
	return nil
}
