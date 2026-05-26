package processors

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
)

func init() {
	pipeline.Register("logic.if", newLogicIf)
}

// logicIfProcessor conditionally executes one of two child processor chains.
// Config example:
//
//	{
//	  "condition": "labels.severity == 'critical'",
//	  "then": [{"type": "callback", "config": {"url": "..."}}],
//	  "else": [{"type": "event_drop", "config": {"condition": "true"}}]
//	}
//
// Supported condition forms:
//   - labels.<key> == '<value>'
//   - labels.<key> != '<value>'
//   - labels.<key> =~ '<regex>'
//   - severity == '<value>'
type logicIfProcessor struct {
	condition     string
	thenChain     []pipeline.Processor
	elseChain     []pipeline.Processor
	conditionExpr *conditionExpr
}

// conditionExpr is a parsed condition expression.
type conditionExpr struct {
	subject string // "labels.<key>" or "severity"
	op      string // "==", "!=", "=~"
	value   string // expected value or regex pattern
}

func newLogicIf(config map[string]interface{}) (pipeline.Processor, error) {
	p := &logicIfProcessor{}

	if v, ok := config["condition"].(string); ok {
		p.condition = v
	}
	if p.condition == "" {
		return nil, fmt.Errorf("logic.if: condition is required")
	}

	expr, err := parseCondition(p.condition)
	if err != nil {
		return nil, fmt.Errorf("logic.if: %w", err)
	}
	p.conditionExpr = expr

	// Parse "then" child processors
	if thenRaw, ok := config["then"].([]interface{}); ok {
		procs, err := buildChildProcessors(thenRaw)
		if err != nil {
			return nil, fmt.Errorf("logic.if: failed to build 'then' processors: %w", err)
		}
		p.thenChain = procs
	}

	// Parse "else" child processors (optional)
	if elseRaw, ok := config["else"].([]interface{}); ok {
		procs, err := buildChildProcessors(elseRaw)
		if err != nil {
			return nil, fmt.Errorf("logic.if: failed to build 'else' processors: %w", err)
		}
		p.elseChain = procs
	}

	return p, nil
}

func (p *logicIfProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	matched := evaluateCondition(p.conditionExpr, event)

	var chain []pipeline.Processor
	var branch string
	if matched {
		chain = p.thenChain
		branch = "then"
	} else {
		chain = p.elseChain
		branch = "else"
	}

	if len(chain) == 0 {
		return event, fmt.Sprintf("logic.if: condition=%v, branch=%s (no processors)", matched, branch), nil
	}

	// Execute the child processor chain
	current := event
	for _, proc := range chain {
		result, _, err := proc.Process(ctx, current)
		if err != nil {
			return current, "", fmt.Errorf("logic.if: child processor failed in %s branch: %w", branch, err)
		}
		if result == nil {
			// Child dropped the event
			return nil, fmt.Sprintf("logic.if: event dropped by child in %s branch", branch), nil
		}
		current = result
	}

	return current, fmt.Sprintf("logic.if: condition=%v, branch=%s (%d processors executed)", matched, branch, len(chain)), nil
}

// parseCondition parses a condition string like "labels.severity == 'critical'" into a conditionExpr.
func parseCondition(s string) (*conditionExpr, error) {
	s = strings.TrimSpace(s)

	// Try operators in order of longest first to avoid ambiguity
	for _, op := range []string{"=~", "!=", "=="} {
		idx := strings.Index(s, op)
		if idx < 0 {
			continue
		}

		subject := strings.TrimSpace(s[:idx])
		value := strings.TrimSpace(s[idx+len(op):])

		// Strip surrounding quotes from value
		value = strings.Trim(value, "'\"")

		if subject == "" || value == "" {
			return nil, fmt.Errorf("invalid condition %q: empty subject or value", s)
		}

		return &conditionExpr{
			subject: subject,
			op:      op,
			value:   value,
		}, nil
	}

	return nil, fmt.Errorf("unsupported condition format %q: expected 'subject op value' where op is ==, !=, or =~", s)
}

// evaluateCondition evaluates a parsed condition against an alert event.
func evaluateCondition(expr *conditionExpr, event *model.AlertEvent) bool {
	actual := resolveEventField(expr.subject, event)

	switch expr.op {
	case "==":
		return actual == expr.value
	case "!=":
		return actual != expr.value
	case "=~":
		re, err := regexp.Compile(expr.value)
		if err != nil {
			return false
		}
		return re.MatchString(actual)
	default:
		return false
	}
}

// buildChildProcessors creates a chain of processors from a list of config maps.
// Each entry must have "type" and optionally "config".
func buildChildProcessors(items []interface{}) ([]pipeline.Processor, error) {
	var procs []pipeline.Processor
	for i, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("child processor [%d]: expected object, got %T", i, item)
		}

		typ, _ := itemMap["type"].(string)
		if typ == "" {
			return nil, fmt.Errorf("child processor [%d]: 'type' is required", i)
		}

		cfg, _ := itemMap["config"].(map[string]interface{})
		if cfg == nil {
			cfg = map[string]interface{}{}
		}

		proc, err := pipeline.Get(typ, cfg)
		if err != nil {
			return nil, fmt.Errorf("child processor [%d] type=%q: %w", i, typ, err)
		}
		procs = append(procs, proc)
	}
	return procs, nil
}
