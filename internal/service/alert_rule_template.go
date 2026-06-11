package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type AlertRuleTemplateService struct {
	repo   *repository.AlertRuleTemplateRepository
	logger *zap.Logger
}

func NewAlertRuleTemplateService(repo *repository.AlertRuleTemplateRepository, logger *zap.Logger) *AlertRuleTemplateService {
	return &AlertRuleTemplateService{repo: repo, logger: logger}
}

func (s *AlertRuleTemplateService) List(ctx context.Context, category, search string, page, pageSize int) ([]model.AlertRuleTemplate, int64, error) {
	return s.repo.List(ctx, category, search, page, pageSize)
}

func (s *AlertRuleTemplateService) GetByID(ctx context.Context, id uint) (*model.AlertRuleTemplate, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AlertRuleTemplateService) Create(ctx context.Context, tpl *model.AlertRuleTemplate) error {
	if tpl.Name == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "name is required")
	}
	if tpl.Expression == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "expression is required")
	}
	return s.repo.Create(ctx, tpl)
}

func (s *AlertRuleTemplateService) Update(ctx context.Context, tpl *model.AlertRuleTemplate) error {
	existing, err := s.repo.GetByID(ctx, tpl.ID)
	if err != nil {
		return apperr.ErrNotFound
	}
	if existing.IsBuiltin {
		return apperr.WithMessage(apperr.ErrBuiltinDelete, "built-in templates cannot be modified")
	}
	// Merge updates into existing template (preserve created_at, etc.)
	existing.Name = tpl.Name
	existing.Description = tpl.Description
	existing.Category = tpl.Category
	existing.DatasourceType = tpl.DatasourceType
	existing.Expression = tpl.Expression
	existing.ForDuration = tpl.ForDuration
	existing.Severity = tpl.Severity
	existing.Labels = tpl.Labels
	existing.Annotations = tpl.Annotations
	existing.GroupName = tpl.GroupName
	existing.EvalInterval = tpl.EvalInterval
	existing.NoDataEnabled = tpl.NoDataEnabled
	existing.NoDataDuration = tpl.NoDataDuration
	existing.AckSlaMinutes = tpl.AckSlaMinutes
	return s.repo.Update(ctx, existing)
}

func (s *AlertRuleTemplateService) Delete(ctx context.Context, id uint) error {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if tpl.IsBuiltin {
		return apperr.WithMessage(apperr.ErrBuiltinDelete, "built-in templates cannot be deleted")
	}
	return s.repo.Delete(ctx, id)
}

func (s *AlertRuleTemplateService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.ListCategories(ctx)
}

// ApplyTemplate creates an AlertRule from a template. The caller fills in DataSourceID.
func (s *AlertRuleTemplateService) ApplyTemplate(ctx context.Context, templateID uint, overrides *model.AlertRule) (*model.AlertRule, error) {
	tpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}

	rule := &model.AlertRule{
		Name:           tpl.Name,
		Description:    tpl.Description,
		DatasourceType: tpl.DatasourceType,
		Expression:     tpl.Expression,
		ForDuration:    tpl.ForDuration,
		Severity:       tpl.Severity,
		Labels:         tpl.Labels,
		Annotations:    tpl.Annotations,
		GroupName:      tpl.GroupName,
		Category:       tpl.Category,
		EvalInterval:   tpl.EvalInterval,
		NoDataEnabled:  tpl.NoDataEnabled,
		NoDataDuration: tpl.NoDataDuration,
		AckSlaMinutes:  tpl.AckSlaMinutes,
	}

	if overrides != nil {
		if overrides.Name != "" {
			rule.Name = overrides.Name
		}
		if overrides.DataSourceID != nil {
			rule.DataSourceID = overrides.DataSourceID
		}
		if overrides.DatasourceType != "" {
			rule.DatasourceType = overrides.DatasourceType
		}
		if overrides.Expression != "" {
			rule.Expression = overrides.Expression
		}
		if overrides.Severity != "" {
			rule.Severity = overrides.Severity
		}
		if overrides.Labels != nil {
			rule.Labels = overrides.Labels
		}
		if overrides.Annotations != nil {
			rule.Annotations = overrides.Annotations
		}
	}

	_ = s.repo.IncrementUsage(ctx, templateID)
	return rule, nil
}
