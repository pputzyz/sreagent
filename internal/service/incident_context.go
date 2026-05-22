package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// IncidentContext aggregates all relevant context for an incident to feed into AI analysis.
type IncidentContext struct {
	Incident          *model.Incident         `json:"incident"`
	RelatedAlerts     []model.AlertEvent      `json:"related_alerts"`
	OnCallPerson      string                  `json:"on_call_person"`
	TeamName          string                  `json:"team_name"`
	Runbook           string                  `json:"runbook"`
	Labels            map[string]string       `json:"labels"`
	KnowledgeResults  []*model.KnowledgeDocument `json:"knowledge_results"`
}

// IncidentContextService builds the full context for an incident.
type IncidentContextService struct {
	incidentRepo *repository.IncidentRepository
	eventRepo    *repository.AlertEventRepository
	kbSvc        *KnowledgeBaseService
	scheduleSvc  *ScheduleService
	bizGroupSvc  *BizGroupService
	logger       *zap.Logger
}

func NewIncidentContextService(
	incidentRepo *repository.IncidentRepository,
	eventRepo *repository.AlertEventRepository,
	kbSvc *KnowledgeBaseService,
	scheduleSvc *ScheduleService,
	bizGroupSvc *BizGroupService,
	logger *zap.Logger,
) *IncidentContextService {
	return &IncidentContextService{
		incidentRepo: incidentRepo,
		eventRepo:    eventRepo,
		kbSvc:        kbSvc,
		scheduleSvc:  scheduleSvc,
		bizGroupSvc:  bizGroupSvc,
		logger:       logger,
	}
}

// BuildContext builds the full incident context for AI analysis.
func (s *IncidentContextService) BuildContext(ctx context.Context, incidentID uint) (*IncidentContext, error) {
	inc, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("incident not found: %w", err)
	}

	ictx := &IncidentContext{
		Incident: inc,
		Labels:   make(map[string]string),
	}

	// Copy labels
	if inc.Labels != nil {
		for k, v := range inc.Labels {
			ictx.Labels[k] = v
		}
	}

	// Related alerts — search by matching labels (simplified: use alert_name from incident title)
	if inc.Title != "" {
		alerts, _, _ := s.eventRepo.List(ctx, "", "", 1, 50)
		for _, a := range alerts {
			if strings.Contains(strings.ToLower(a.AlertName), strings.ToLower(inc.Title)) {
				ictx.RelatedAlerts = append(ictx.RelatedAlerts, a)
			}
		}
	}

	// On-call person (best effort)
	if s.scheduleSvc != nil {
		// Try to find on-call for the team
		if teamName, ok := ictx.Labels["owner"]; ok {
			ictx.TeamName = teamName
		}
	}

	// Knowledge base search (RAG)
	if s.kbSvc != nil && inc.Title != "" {
		docs, err := s.kbSvc.Search(ctx, inc.Title, "", 5)
		if err == nil {
			ictx.KnowledgeResults = docs
		}
	}

	// Runbook from labels
	if rb, ok := ictx.Labels["runbook_url"]; ok && rb != "" {
		ictx.Runbook = rb
	}

	return ictx, nil
}

// FormatForPrompt formats the incident context into a structured prompt section.
func (ictx *IncidentContext) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("## 事故上下文\n\n")

	// Incident info
	if ictx.Incident != nil {
			fmt.Fprintf(&sb, "**事故:** %s (ID: %d)\n", ictx.Incident.Title, ictx.Incident.ID)
		fmt.Fprintf(&sb, "**状态:** %s | **严重等级:** %s\n", ictx.Incident.Status, ictx.Incident.Severity)
		if ictx.Incident.Description != "" {
			fmt.Fprintf(&sb, "**描述:** %s\n", ictx.Incident.Description)
		}
		sb.WriteString("\n")
	}

	// Labels
	if len(ictx.Labels) > 0 {
		sb.WriteString("**标签:**\n")
		for k, v := range ictx.Labels {
			fmt.Fprintf(&sb, "- %s: %s\n", k, v)
		}
		sb.WriteString("\n")
	}

	// Related alerts
	if len(ictx.RelatedAlerts) > 0 {
		fmt.Fprintf(&sb, "**相关告警 (%d 条):**\n", len(ictx.RelatedAlerts))
		for i, a := range ictx.RelatedAlerts {
			if i >= 10 {
				fmt.Fprintf(&sb, "... 还有 %d 条\n", len(ictx.RelatedAlerts)-10)
				break
			}
			fmt.Fprintf(&sb, "- [%s] %s (%s) — %s\n", a.Severity, a.AlertName, a.Status, a.FiredAt.Format("01-02 15:04"))
		}
		sb.WriteString("\n")
	}

	// Team / On-call
	if ictx.TeamName != "" {
		fmt.Fprintf(&sb, "**负责团队:** %s\n", ictx.TeamName)
	}
	if ictx.OnCallPerson != "" {
		fmt.Fprintf(&sb, "**当前值班人:** %s\n", ictx.OnCallPerson)
	}

	// Knowledge results
	if len(ictx.KnowledgeResults) > 0 {
		sb.WriteString("**相关知识库文档:**\n")
		for i, doc := range ictx.KnowledgeResults {
			if i >= 3 {
				break
			}
			fmt.Fprintf(&sb, "- [%s] %s\n", doc.Source, doc.Title)
			if doc.Summary != "" {
				fmt.Fprintf(&sb, "  摘要: %s\n", doc.Summary)
			}
		}
		sb.WriteString("\n")
	}

	// Runbook
	if ictx.Runbook != "" {
		fmt.Fprintf(&sb, "**处理手册:** %s\n\n", ictx.Runbook)
	}

	return sb.String()
}
