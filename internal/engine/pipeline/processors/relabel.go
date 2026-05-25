package processors

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
)

func init() {
	pipeline.Register("relabel", newRelabel)
}

// relabelProcessor applies Prometheus-style relabeling to event labels.
type relabelProcessor struct {
	// SourceLabels are the label names to concatenate for regex matching.
	SourceLabels []string `json:"source_labels"`
	// Separator is used to join source label values (default: ";").
	Separator string `json:"separator"`
	// Regex is the regular expression to match against the joined source labels.
	Regex string `json:"regex"`
	// TargetLabel is the label to write the result into.
	TargetLabel string `json:"target_label"`
	// Replacement is the replacement string (supports regex capture groups).
	Replacement string `json:"replacement"`
	// Action defines what to do: replace, keep, drop, labelmap, hashmod.
	Action string `json:"action"`
	// Modulus for hashmod action.
	Modulus uint64 `json:"modulus"`
}

func newRelabel(config map[string]interface{}) (pipeline.Processor, error) {
	p := &relabelProcessor{
		Separator:   ";",
		Replacement: "$1",
		Action:      "replace",
	}
	if v, ok := config["source_labels"].([]interface{}); ok {
		for _, s := range v {
			if sv, ok := s.(string); ok {
				p.SourceLabels = append(p.SourceLabels, sv)
			}
		}
	}
	if v, ok := config["separator"].(string); ok {
		p.Separator = v
	}
	if v, ok := config["regex"].(string); ok {
		p.Regex = v
	}
	if v, ok := config["target_label"].(string); ok {
		p.TargetLabel = v
	}
	if v, ok := config["replacement"].(string); ok {
		p.Replacement = v
	}
	if v, ok := config["action"].(string); ok {
		p.Action = v
	}
	if v, ok := config["modulus"].(float64); ok {
		p.Modulus = uint64(v)
	}
	return p, nil
}

func (p *relabelProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	if event.Labels == nil {
		event.Labels = make(model.JSONLabels)
	}

	// Build the source value by joining source label values
	var parts []string
	for _, sl := range p.SourceLabels {
		parts = append(parts, event.Labels[sl])
	}
	sourceVal := strings.Join(parts, p.Separator)

	switch p.Action {
	case "replace":
		if p.Regex == "" || p.TargetLabel == "" {
			return event, "relabel: skipped (missing regex or target_label)", nil
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return event, "", fmt.Errorf("relabel: invalid regex %q: %w", p.Regex, err)
		}
		if re.MatchString(sourceVal) {
			result := re.ReplaceAllString(sourceVal, p.Replacement)
			event.Labels[p.TargetLabel] = result
			return event, fmt.Sprintf("relabel: set %s=%s", p.TargetLabel, result), nil
		}
		return event, "relabel: no match", nil

	case "keep":
		if p.Regex == "" {
			return event, "relabel: keep skipped (missing regex)", nil
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return event, "", fmt.Errorf("relabel: invalid regex %q: %w", p.Regex, err)
		}
		if !re.MatchString(sourceVal) {
			return nil, "relabel: dropped by keep (no match)", nil
		}
		return event, "relabel: kept", nil

	case "drop":
		if p.Regex == "" {
			return event, "relabel: drop skipped (missing regex)", nil
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return event, "", fmt.Errorf("relabel: invalid regex %q: %w", p.Regex, err)
		}
		if re.MatchString(sourceVal) {
			return nil, "relabel: dropped by drop (match)", nil
		}
		return event, "relabel: not dropped", nil

	case "labelmap":
		if p.Regex == "" {
			return event, "relabel: labelmap skipped (missing regex)", nil
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return event, "", fmt.Errorf("relabel: invalid regex %q: %w", p.Regex, err)
		}
		for k, v := range event.Labels {
			if re.MatchString(k) {
				newKey := re.ReplaceAllString(k, p.Replacement)
				if newKey != k {
					event.Labels[newKey] = v
					delete(event.Labels, k)
				}
			}
		}
		return event, "relabel: labelmap applied", nil

	case "hashmod":
		if p.Regex == "" || p.TargetLabel == "" || p.Modulus == 0 {
			return event, "relabel: hashmod skipped (missing config)", nil
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return event, "", fmt.Errorf("relabel: invalid regex %q: %w", p.Regex, err)
		}
		if re.MatchString(sourceVal) {
			h := sha256.Sum256([]byte(sourceVal))
			mod := uint64(h[0])<<24 | uint64(h[1])<<16 | uint64(h[2])<<8 | uint64(h[3])
			event.Labels[p.TargetLabel] = fmt.Sprintf("%d", mod%p.Modulus)
			return event, fmt.Sprintf("relabel: hashmod %s=%s", p.TargetLabel, event.Labels[p.TargetLabel]), nil
		}
		return event, "relabel: hashmod no match", nil

	default:
		return event, fmt.Sprintf("relabel: unknown action %q, skipped", p.Action), nil
	}
}
