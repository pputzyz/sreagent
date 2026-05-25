package pipeline

import (
	"context"
	"fmt"

	"github.com/sreagent/sreagent/internal/model"
)

// Processor is the interface that all pipeline processors must implement.
// Process receives an AlertEvent and returns:
//   - the (possibly modified) event
//   - a human-readable message describing what was done
//   - an error if processing failed
//
// If the returned event is nil, the event should be dropped (no further processing).
type Processor interface {
	Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error)
}

// NewProcessorFn is a factory function that creates a Processor from config.
type NewProcessorFn func(config map[string]interface{}) (Processor, error)

var registry = map[string]NewProcessorFn{}

// Register registers a processor factory for the given type name.
func Register(typ string, fn NewProcessorFn) {
	registry[typ] = fn
}

// Get creates a Processor instance for the given type and config.
func Get(typ string, config map[string]interface{}) (Processor, error) {
	fn, ok := registry[typ]
	if !ok {
		return nil, fmt.Errorf("unknown processor type: %s", typ)
	}
	return fn(config)
}

// AvailableTypes returns the list of registered processor type names.
func AvailableTypes() []string {
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
