package processors

import (
	"context"
	"fmt"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

func init() {
	pipeline.Register("ai_summary", newAISummary)
}

// aiSummaryProcessor uses AI to analyze the alert and writes the summary
// into event annotations.
type aiSummaryProcessor struct {
	onlyCritical bool
	pipelineSvc  *service.AlertPipeline
}

// SetAIPipeline sets the AlertPipeline service used by the ai_summary processor.
// Called during DI wiring.
var aiPipelineSvc *service.AlertPipeline

func SetAIPipeline(svc *service.AlertPipeline) {
	aiPipelineSvc = svc
}

func newAISummary(config map[string]interface{}) (pipeline.Processor, error) {
	p := &aiSummaryProcessor{
		pipelineSvc: aiPipelineSvc,
	}
	if v, ok := config["only_critical"].(bool); ok {
		p.onlyCritical = v
	}
	return p, nil
}

func (p *aiSummaryProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	if p.pipelineSvc == nil {
		return event, "ai_summary: AI service not configured, skipped", nil
	}

	// Optionally only process critical alerts
	if p.onlyCritical && string(event.Severity) != "critical" {
		return event, "ai_summary: skipped (non-critical severity)", nil
	}

	analysis := p.pipelineSvc.AnalyzeAlert(ctx, event)
	if analysis == nil {
		return event, "ai_summary: no analysis produced", nil
	}

	// Write summary into annotations
	if event.Annotations == nil {
		event.Annotations = make(model.JSONLabels)
	}
	summary := analysis.RootCauseHint
	if summary == "" {
		summary = analysis.Summary
	}
	if summary != "" {
		event.Annotations["ai_summary"] = summary
		return event, fmt.Sprintf("ai_summary: generated (%d chars)", len(summary)), nil
	}
	return event, "ai_summary: analysis produced no summary text", nil
}
