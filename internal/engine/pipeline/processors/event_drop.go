package processors

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
)

func init() {
	pipeline.Register("event_drop", newEventDrop)
}

// eventDropProcessor drops events when a Go template evaluates to "true".
type eventDropProcessor struct {
	Condition string `json:"condition"` // Go template expression
}

func newEventDrop(config map[string]interface{}) (pipeline.Processor, error) {
	p := &eventDropProcessor{}
	if v, ok := config["condition"].(string); ok {
		p.Condition = v
	}
	if p.Condition == "" {
		return nil, fmt.Errorf("event_drop: condition is required")
	}
	return p, nil
}

func (p *eventDropProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	// Build template data from event
	data := map[string]interface{}{
		"AlertName":   event.AlertName,
		"Severity":    string(event.Severity),
		"Status":      string(event.Status),
		"Labels":      map[string]string(event.Labels),
		"Annotations": map[string]string(event.Annotations),
		"Source":      event.Source,
	}

	tmpl, err := template.New("condition").Parse(p.Condition)
	if err != nil {
		return event, "", fmt.Errorf("event_drop: invalid template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return event, "", fmt.Errorf("event_drop: template execution failed: %w", err)
	}

	result := buf.String()
	if result == "true" {
		return nil, "event_drop: condition matched, event dropped", nil
	}
	return event, fmt.Sprintf("event_drop: condition evaluated to %q, event kept", result), nil
}
