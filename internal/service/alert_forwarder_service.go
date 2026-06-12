package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// AlertForwarderService provides CRUD and event processing for alert forwarders.
type AlertForwarderService struct {
	forwarderRepo *repository.AlertForwarderRepository
	mediaRepo     *repository.NotifyMediaRepository
	mediaSvc      *NotifyMediaService
	// Platform capability dependencies
	eventRepo     *repository.AlertEventRepository
	notifySvc     *NotificationService
	muteSvc       *MuteRuleService
	inhibitorSvc  *InhibitionRuleService
	pipelineEngine interface{} // *ppipeline.Engine - avoid import cycle
	logger        *zap.Logger
}

// NewAlertForwarderService creates a new AlertForwarderService.
func NewAlertForwarderService(
	forwarderRepo *repository.AlertForwarderRepository,
	mediaRepo *repository.NotifyMediaRepository,
	mediaSvc *NotifyMediaService,
	logger *zap.Logger,
) *AlertForwarderService {
	return &AlertForwarderService{
		forwarderRepo: forwarderRepo,
		mediaRepo:     mediaRepo,
		mediaSvc:      mediaSvc,
		logger:        logger,
	}
}

// SetEventRepository injects the event repository for platform capabilities.
func (s *AlertForwarderService) SetEventRepository(repo *repository.AlertEventRepository) {
	s.eventRepo = repo
}

// SetNotificationService injects the notification service for routing.
func (s *AlertForwarderService) SetNotificationService(svc *NotificationService) {
	s.notifySvc = svc
}

// SetMuteRuleService injects the mute rule service.
func (s *AlertForwarderService) SetMuteRuleService(svc *MuteRuleService) {
	s.muteSvc = svc
}

// SetInhibitionRuleService injects the inhibition rule service.
func (s *AlertForwarderService) SetInhibitionRuleService(svc *InhibitionRuleService) {
	s.inhibitorSvc = svc
}

// Create creates a new alert forwarder.
func (s *AlertForwarderService) Create(ctx context.Context, forwarder *model.AlertForwarder) error {
	// Validate direction
	if !forwarder.Direction.IsValid() {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid direction: must be inbound, outbound, or bidirectional")
	}

	// Validate inbound config for inbound/bidirectional
	if forwarder.Direction == model.ForwarderDirectionInbound || forwarder.Direction == model.ForwarderDirectionBidirectional {
		if forwarder.InboundConfig == nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, "inbound_config is required for inbound/bidirectional forwarders")
		}
		if !forwarder.InboundConfig.SourceFormat.IsValid() {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid source_format")
		}
		if !forwarder.InboundConfig.AuthType.IsValid() {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid auth_type")
		}
	}

	// Validate outbound config for outbound/bidirectional
	if forwarder.Direction == model.ForwarderDirectionOutbound || forwarder.Direction == model.ForwarderDirectionBidirectional {
		if forwarder.OutboundConfig == nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, "outbound_config is required for outbound/bidirectional forwarders")
		}
		if forwarder.OutboundConfig.TargetMediaID == nil && forwarder.OutboundConfig.TargetURL == "" {
			return apperr.WithMessage(apperr.ErrInvalidParam, "either target_media_id or target_url is required")
		}
	}

	// Validate severity mapping
	if forwarder.SeverityMapping != nil && forwarder.SeverityMapping.Enabled {
		if !forwarder.SeverityMapping.Direction.IsValid() {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid severity_mapping direction")
		}
	}

	// Set defaults
	if forwarder.OutboundConfig != nil {
		if forwarder.OutboundConfig.Method == "" {
			forwarder.OutboundConfig.Method = "POST"
		}
		if forwarder.OutboundConfig.Timeout == 0 {
			forwarder.OutboundConfig.Timeout = 30000
		}
		if forwarder.OutboundConfig.RetryTimes == 0 {
			forwarder.OutboundConfig.RetryTimes = 3
		}
		if forwarder.OutboundConfig.RetryInterval == 0 {
			forwarder.OutboundConfig.RetryInterval = 100
		}
	}

	// Set default platform capabilities if not provided
	if forwarder.PlatformCapabilities == nil {
		forwarder.PlatformCapabilities = &model.PlatformCapabilitiesConfig{
			EnableEscalation:   false,
			EnableMute:         false,
			EnableInhibition:   false,
			EnableNotification: true,
			EnableAIAnalysis:   false,
		}
	}

	if err := s.forwarderRepo.Create(ctx, forwarder); err != nil {
		s.logger.Error("failed to create alert forwarder", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an alert forwarder by its ID.
func (s *AlertForwarderService) GetByID(ctx context.Context, id uint) (*model.AlertForwarder, error) {
	forwarder, err := s.forwarderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return forwarder, nil
}

// List returns a paginated list of alert forwarders.
func (s *AlertForwarderService) List(ctx context.Context, page, pageSize int, direction string, enabled *bool) ([]model.AlertForwarder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.forwarderRepo.List(ctx, page, pageSize, direction, enabled)
}

// Update updates an existing alert forwarder.
func (s *AlertForwarderService) Update(ctx context.Context, forwarder *model.AlertForwarder) error {
	// Verify forwarder exists
	existing, err := s.forwarderRepo.GetByID(ctx, forwarder.ID)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Validate direction
	if !forwarder.Direction.IsValid() {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid direction")
	}

	// Validate severity mapping
	if forwarder.SeverityMapping != nil && forwarder.SeverityMapping.Enabled {
		if !forwarder.SeverityMapping.Direction.IsValid() {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid severity_mapping direction")
		}
	}

	// Preserve creation time
	forwarder.CreatedAt = existing.CreatedAt

	if err := s.forwarderRepo.Update(ctx, forwarder); err != nil {
		s.logger.Error("failed to update alert forwarder", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes an alert forwarder by ID.
func (s *AlertForwarderService) Delete(ctx context.Context, id uint) error {
	if err := s.forwarderRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete alert forwarder", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Enable enables an alert forwarder.
func (s *AlertForwarderService) Enable(ctx context.Context, id uint) error {
	forwarder, err := s.forwarderRepo.GetByID(ctx, id)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	forwarder.Enabled = true
	if err := s.forwarderRepo.Update(ctx, forwarder); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Disable disables an alert forwarder.
func (s *AlertForwarderService) Disable(ctx context.Context, id uint) error {
	forwarder, err := s.forwarderRepo.GetByID(ctx, id)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	forwarder.Enabled = false
	if err := s.forwarderRepo.Update(ctx, forwarder); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// BatchEnable enables multiple alert forwarders.
func (s *AlertForwarderService) BatchEnable(ctx context.Context, ids []uint) error {
	return s.forwarderRepo.BatchUpdateEnabled(ctx, ids, true)
}

// BatchDisable disables multiple alert forwarders.
func (s *AlertForwarderService) BatchDisable(ctx context.Context, ids []uint) error {
	return s.forwarderRepo.BatchUpdateEnabled(ctx, ids, false)
}

// BatchDelete deletes multiple alert forwarders.
func (s *AlertForwarderService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.forwarderRepo.BatchDelete(ctx, ids)
}

// GetStats returns statistics about alert forwarders.
func (s *AlertForwarderService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	counts, err := s.forwarderRepo.CountByDirection(ctx)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	enabledForwarders, err := s.forwarderRepo.ListEnabled(ctx)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	return map[string]interface{}{
		"by_direction":       counts,
		"enabled_count":      len(enabledForwarders),
	}, nil
}

// TestForwarder tests a forwarder configuration by sending a test alert.
func (s *AlertForwarderService) TestForwarder(ctx context.Context, id uint) (map[string]interface{}, error) {
	forwarder, err := s.forwarderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	result := map[string]interface{}{
		"forwarder_id":   forwarder.ID,
		"forwarder_name": forwarder.Name,
		"direction":      forwarder.Direction,
		"enabled":        forwarder.Enabled,
	}

	// Test outbound config if applicable
	if forwarder.Direction == model.ForwarderDirectionOutbound || forwarder.Direction == model.ForwarderDirectionBidirectional {
		if forwarder.OutboundConfig != nil && forwarder.OutboundConfig.TargetMediaID != nil {
			media, err := s.mediaRepo.GetByID(ctx, *forwarder.OutboundConfig.TargetMediaID)
			if err != nil {
				result["outbound_error"] = fmt.Sprintf("failed to load target media: %v", err)
			} else {
				result["outbound_media_name"] = media.Name
				result["outbound_media_type"] = media.Type
			}
		}
	}

	// Test inbound config if applicable
	if forwarder.Direction == model.ForwarderDirectionInbound || forwarder.Direction == model.ForwarderDirectionBidirectional {
		if forwarder.InboundConfig != nil {
			result["inbound_source_format"] = forwarder.InboundConfig.SourceFormat
			result["inbound_auth_type"] = forwarder.InboundConfig.AuthType
			result["inbound_path"] = fmt.Sprintf("/api/v1/forwarders/%d/inbound", forwarder.ID)
		}
	}

	// Severity mapping
	if forwarder.SeverityMapping != nil && forwarder.SeverityMapping.Enabled {
		result["severity_mapping_enabled"] = true
		result["severity_mapping_direction"] = forwarder.SeverityMapping.Direction
		mappingJSON, _ := json.Marshal(forwarder.SeverityMapping.Mapping)
		result["severity_mapping"] = string(mappingJSON)
	}

	// Platform capabilities
	if forwarder.PlatformCapabilities != nil {
		result["platform_escalation"] = forwarder.PlatformCapabilities.EnableEscalation
		result["platform_mute"] = forwarder.PlatformCapabilities.EnableMute
		result["platform_inhibition"] = forwarder.PlatformCapabilities.EnableInhibition
		result["platform_notification"] = forwarder.PlatformCapabilities.EnableNotification
		result["platform_ai_analysis"] = forwarder.PlatformCapabilities.EnableAIAnalysis
	}

	return result, nil
}
