package processors

import (
	"strings"

	"github.com/sreagent/sreagent/internal/model"
)

// resolveEventField extracts a field value from an alert event.
// Supported field paths:
//   - "severity"  → event severity
//   - "status"    → event status
//   - "labels.<key>" → event labels[key]
//   - "annotations.<key>" → event annotations[key]
//   - anything else → event labels[field] (direct label key fallback)
func resolveEventField(field string, event *model.AlertEvent) string {
	switch {
	case field == "severity":
		return string(event.Severity)
	case field == "status":
		return string(event.Status)
	case strings.HasPrefix(field, "labels."):
		key := strings.TrimPrefix(field, "labels.")
		if event.Labels == nil {
			return ""
		}
		return event.Labels[key]
	case strings.HasPrefix(field, "annotations."):
		key := strings.TrimPrefix(field, "annotations.")
		if event.Annotations == nil {
			return ""
		}
		return event.Annotations[key]
	default:
		if event.Labels != nil {
			return event.Labels[field]
		}
		return ""
	}
}
