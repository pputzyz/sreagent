package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// recordingRuleAllowedFields is the allowlist for UpdateFields.
// Only these columns may be updated via the partial-update endpoint.
var recordingRuleAllowedFields = map[string]bool{
	"name":          true,
	"prom_ql":       true,
	"datasource_ids": true,
	"cron_pattern":  true,
	"disabled":      true,
	"write_back":    true,
	"append_tags":   true,
	"note":          true,
	"query_configs": true,
	"updated_by":    true,
	"updated_at":    true,
}

type RecordingRuleService struct {
	repo   *repository.RecordingRuleRepository
	logger *zap.Logger
}

func NewRecordingRuleService(repo *repository.RecordingRuleRepository, logger *zap.Logger) *RecordingRuleService {
	return &RecordingRuleService{repo: repo, logger: logger}
}

func (s *RecordingRuleService) Create(ctx context.Context, rule *model.RecordingRule) error {
	rule.FE2DB()
	if err := rule.Verify(); err != nil {
		return err
	}
	rule.ID = 0
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	return s.repo.Create(ctx, rule)
}

func (s *RecordingRuleService) GetByID(ctx context.Context, id uint) (*model.RecordingRule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RecordingRuleService) ListByGroupID(ctx context.Context, groupID uint) ([]model.RecordingRule, error) {
	return s.repo.ListByGroupID(ctx, groupID)
}

func (s *RecordingRuleService) ListByGroupIDs(ctx context.Context, groupIDs []uint) ([]model.RecordingRule, error) {
	return s.repo.ListByGroupIDs(ctx, groupIDs)
}

func (s *RecordingRuleService) ListWithFilter(ctx context.Context, groupID uint, query string, disabled *int, page, pageSize int) ([]model.RecordingRule, int64, error) {
	return s.repo.ListWithFilter(ctx, groupID, query, disabled, page, pageSize)
}

func (s *RecordingRuleService) Update(ctx context.Context, existing *model.RecordingRule, input *model.RecordingRule) error {
	input.FE2DB()
	if err := input.Verify(); err != nil {
		return err
	}
	// Preserve immutable fields
	input.ID = existing.ID
	input.GroupID = existing.GroupID
	input.CreatedBy = existing.CreatedBy
	input.CreatedAt = existing.CreatedAt
	input.UpdatedAt = time.Now()
	return s.repo.Update(ctx, input)
}

func (s *RecordingRuleService) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	// Validate that all requested fields are in the allowlist.
	for key := range fields {
		if !recordingRuleAllowedFields[key] {
			return fmt.Errorf("field %q is not allowed for partial update", key)
		}
	}
	fields["updated_at"] = time.Now()
	return s.repo.UpdateFields(ctx, id, fields)
}

func (s *RecordingRuleService) Delete(ctx context.Context, id uint, groupID uint) error {
	return s.repo.Delete(ctx, id, groupID)
}

func (s *RecordingRuleService) DeleteByIDs(ctx context.Context, ids []uint, groupID uint) error {
	return s.repo.DeleteByIDs(ctx, ids, groupID)
}

func (s *RecordingRuleService) BatchCreate(ctx context.Context, rules []model.RecordingRule) map[string]string {
	results := make(map[string]string, len(rules))
	for _, rule := range rules {
		r := rule // capture
		r.FE2DB()
		if err := r.Verify(); err != nil {
			results[r.Name] = err.Error()
			continue
		}
		r.ID = 0
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
		if err := s.repo.Create(ctx, &r); err != nil {
			results[r.Name] = err.Error()
		} else {
			results[r.Name] = ""
		}
	}
	return results
}
