package processors

import (
	"context"
	"fmt"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
)

func init() {
	pipeline.Register("logic.switch", newLogicSwitch)
}

// logicSwitchProcessor routes events to different child processor chains based on a field value.
// Config example:
//
//	{
//	  "field": "labels.severity",
//	  "cases": {
//	    "critical": [{"type": "callback", "config": {"url": "https://pager.example.com"}}],
//	    "warning":  [{"type": "ai_summary", "config": {}}],
//	    "default":  [{"type": "event_drop", "config": {"condition": "true"}}]
//	  }
//	}
//
// The "field" supports:
//   - labels.<key>  — reads from event labels
//   - annotations.<key> — reads from event annotations
//   - severity — reads event severity
//   - status — reads event status
//
// The "cases" maps field values to child processor chains. The special key "default"
// is used when no other case matches.
type logicSwitchProcessor struct {
	field string
	cases map[string][]pipeline.Processor
}

func newLogicSwitch(config map[string]interface{}) (pipeline.Processor, error) {
	p := &logicSwitchProcessor{
		cases: make(map[string][]pipeline.Processor),
	}

	if v, ok := config["field"].(string); ok {
		p.field = v
	}
	if p.field == "" {
		return nil, fmt.Errorf("logic.switch: field is required")
	}

	casesRaw, ok := config["cases"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("logic.switch: 'cases' must be an object mapping values to processor chains")
	}

	for key, chainRaw := range casesRaw {
		chainSlice, ok := chainRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("logic.switch: cases[%q] must be an array of processors", key)
		}

		procs, err := buildChildProcessors(chainSlice)
		if err != nil {
			return nil, fmt.Errorf("logic.switch: cases[%q]: %w", key, err)
		}
		p.cases[key] = procs
	}

	return p, nil
}

func (p *logicSwitchProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	actual := resolveEventField(p.field, event)

	// Look for exact match first
	chain, found := p.cases[actual]

	// Fall back to "default" if no match
	caseKey := actual
	if !found {
		chain, found = p.cases["default"]
		caseKey = "default"
	}

	if !found || len(chain) == 0 {
		return event, fmt.Sprintf("logic.switch: field=%s=%q, no matching case or default", p.field, actual), nil
	}

	// Execute the matched child processor chain
	current := event
	for _, proc := range chain {
		result, _, err := proc.Process(ctx, current)
		if err != nil {
			return current, "", fmt.Errorf("logic.switch: child processor failed in case=%q: %w", caseKey, err)
		}
		if result == nil {
			return nil, fmt.Sprintf("logic.switch: event dropped by child in case=%q", caseKey), nil
		}
		current = result
	}

	return current, fmt.Sprintf("logic.switch: field=%s=%q, matched case=%q (%d processors)", p.field, actual, caseKey, len(chain)), nil
}
