package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// IncidentAggregator bridges AlertEvent lifecycle to Incident management.
// When alerts fire/resolve, it automatically creates/updates/resolves incidents.
type IncidentAggregator struct {
	incidentSvc *IncidentService
	eventRepo   *repository.AlertEventRepository
	incidentRepo *repository.IncidentRepository
	logger      *zap.Logger
}

func NewIncidentAggregator(
	incidentSvc *IncidentService,
	eventRepo *repository.AlertEventRepository,
	incidentRepo *repository.IncidentRepository,
	logger *zap.Logger,
) *IncidentAggregator {
	return &IncidentAggregator{
		incidentSvc:  incidentSvc,
		eventRepo:    eventRepo,
		incidentRepo: incidentRepo,
		logger:       logger,
	}
}

// OnEventFired is called when a new alert event fires.
// It checks if there's an open incident with the same fingerprint.
// If yes, increments the event count. If no, creates a new incident.
func (a *IncidentAggregator) OnEventFired(ctx context.Context, event *model.AlertEvent) {
	if a.incidentSvc == nil {
		return
	}

	// Find open incident with matching fingerprint
	existing, err := a.incidentRepo.FindOpenByFingerprint(ctx, event.Fingerprint)
	if err != nil {
		// No existing incident — create a new one
		inc := &model.Incident{
			Title:       event.AlertName,
			Description: event.Annotations["summary"],
			Severity:    toIncidentSeverity(event.Severity),
			Status:      model.IncidentStatusTriggered,
			ChannelID:   1, // default channel
			Labels:      event.Labels,
			TriggeredAt: event.FiredAt,
			AlertCount:  1,
			EventCount:  1,
		}
		if err := a.incidentSvc.Create(ctx, inc); err != nil {
			a.logger.Error("failed to auto-create incident from alert",
				zap.Uint("event_id", event.ID),
				zap.String("fingerprint", event.Fingerprint),
				zap.Error(err),
			)
			return
		}
		a.logger.Info("auto-created incident from alert",
			zap.Uint("incident_id", inc.ID),
			zap.Uint("event_id", event.ID),
			zap.String("alert_name", event.AlertName),
		)
		return
	}

	// Existing incident — increment counters
	existing.EventCount++
	if event.FiredAt.After(existing.TriggeredAt) {
		// Update severity if the new event is more severe
		newSev := toIncidentSeverity(event.Severity)
		if severityWeight(newSev) > severityWeight(existing.Severity) {
			existing.Severity = newSev
		}
	}
	if err := a.incidentRepo.Update(ctx, existing); err != nil {
		a.logger.Error("failed to update incident counters",
			zap.Uint("incident_id", existing.ID),
			zap.Error(err),
		)
	}
}

// OnEventResolved is called when an alert event resolves.
// It checks if all events for this fingerprint are resolved, and if so,
// auto-resolves the associated incident.
func (a *IncidentAggregator) OnEventResolved(ctx context.Context, event *model.AlertEvent) {
	if a.incidentSvc == nil {
		return
	}

	// Find open incident with matching fingerprint
	incident, err := a.incidentRepo.FindOpenByFingerprint(ctx, event.Fingerprint)
	if err != nil {
		// No open incident — nothing to do
		return
	}

	// Check if all events for this fingerprint are resolved
	firingCount, err := a.eventRepo.CountByFingerprintAndStatus(ctx, event.Fingerprint, model.EventStatusFiring)
	if err != nil {
		a.logger.Error("failed to count firing events",
			zap.String("fingerprint", event.Fingerprint),
			zap.Error(err),
		)
		return
	}

	if firingCount == 0 {
		// All events resolved — auto-resolve the incident
		now := time.Now()
		if err := a.incidentRepo.UpdateStatus(ctx, incident.ID, model.IncidentStatusClosed, map[string]interface{}{
			"resolved_at": now,
			"closed_at":   now,
		}); err != nil {
			a.logger.Error("failed to auto-resolve incident",
				zap.Uint("incident_id", incident.ID),
				zap.Error(err),
			)
			return
		}
		a.logger.Info("auto-resolved incident (all alerts recovered)",
			zap.Uint("incident_id", incident.ID),
			zap.String("fingerprint", event.Fingerprint),
		)
	}
}

// toIncidentSeverity converts alert severity to incident severity.
func toIncidentSeverity(sev model.AlertSeverity) model.IncidentSeverity {
	switch sev {
	case model.SeverityCritical:
		return model.IncidentSeverityCritical
	case model.SeverityWarning:
		return model.IncidentSeverityWarning
	default:
		return model.IncidentSeverityInfo
	}
}

// severityWeight returns a numeric weight for severity comparison.
func severityWeight(sev model.IncidentSeverity) int {
	switch sev {
	case model.IncidentSeverityCritical:
		return 3
	case model.IncidentSeverityWarning:
		return 2
	case model.IncidentSeverityInfo:
		return 1
	default:
		return 0
	}
}
