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

type DashboardService struct {
	repo         *repository.DashboardRepository
	bizGroupRepo *repository.DashboardBizGroupRepository
	dsRepo       *repository.DataSourceRepository
	logger       *zap.Logger
}

func NewDashboardService(repo *repository.DashboardRepository, logger *zap.Logger) *DashboardService {
	return &DashboardService{repo: repo, logger: logger}
}

// SetBizGroupRepository injects the dashboard-biz-group binding repository.
func (s *DashboardService) SetBizGroupRepository(repo *repository.DashboardBizGroupRepository) {
	s.bizGroupRepo = repo
}

// SetDataSourceRepository injects the datasource repository for panel validation.
func (s *DashboardService) SetDataSourceRepository(repo *repository.DataSourceRepository) {
	s.dsRepo = repo
}

func (s *DashboardService) Create(ctx context.Context, d *model.Dashboard) error {
	if err := s.validateConfigDatasources(ctx, d.Config); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error("failed to create dashboard", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *DashboardService) GetByID(ctx context.Context, id uint) (*model.Dashboard, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return d, nil
}

func (s *DashboardService) List(ctx context.Context, search string, page, pageSize int) ([]model.Dashboard, int64, error) {
	return s.repo.List(ctx, search, page, pageSize)
}

func (s *DashboardService) Update(ctx context.Context, d *model.Dashboard) error {
	existing, err := s.repo.GetByID(ctx, d.ID)
	if err != nil {
		return apperr.ErrNotFound
	}

	if err := s.validateConfigDatasources(ctx, d.Config); err != nil {
		return err
	}

	existing.Name = d.Name
	existing.Description = d.Description
	existing.Tags = d.Tags
	existing.Config = d.Config
	existing.IsPublic = d.IsPublic
	existing.UpdatedBy = d.UpdatedBy

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update dashboard", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// validateConfigDatasources parses the dashboard config JSON and validates that
// all datasource_id references in panels point to existing data sources.
func (s *DashboardService) validateConfigDatasources(ctx context.Context, config string) error {
	if s.dsRepo == nil || config == "" {
		return nil
	}
	var cfg struct {
		Panels []struct {
			DatasourceID *uint `json:"datasource_id"`
		} `json:"panels"`
	}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid dashboard config JSON")
	}
	seen := make(map[uint]bool)
	for _, p := range cfg.Panels {
		if p.DatasourceID != nil && *p.DatasourceID > 0 {
			seen[*p.DatasourceID] = true
		}
	}
	for dsID := range seen {
		if _, err := s.dsRepo.GetByID(ctx, dsID); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("datasource_id %d does not exist", dsID))
		}
	}
	return nil
}

func (s *DashboardService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	// Clean up biz group bindings when deleting a dashboard.
	if s.bizGroupRepo != nil {
		if err := s.bizGroupRepo.DeleteByDashboard(ctx, id); err != nil {
			s.logger.Error("failed to delete dashboard biz group bindings", zap.Error(err))
		}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete dashboard", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// BindToBizGroup binds a dashboard to a business group with the given permission flag.
func (s *DashboardService) BindToBizGroup(ctx context.Context, dashboardID, bizGroupID uint, permFlag string) error {
	if s.bizGroupRepo == nil {
		return apperr.WithMessage(apperr.ErrInternal, "biz group binding not configured")
	}
	if permFlag == "" {
		permFlag = "ro"
	}
	if permFlag != "ro" && permFlag != "rw" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "perm_flag must be 'ro' or 'rw'")
	}
	// Verify dashboard exists.
	if _, err := s.repo.GetByID(ctx, dashboardID); err != nil {
		return apperr.ErrNotFound
	}
	// Check if binding already exists.
	existing, err := s.bizGroupRepo.GetBinding(ctx, dashboardID, bizGroupID)
	if err == nil && existing != nil {
		// Update perm flag if different.
		if existing.PermFlag != permFlag {
			return s.bizGroupRepo.UpdatePermFlag(ctx, dashboardID, bizGroupID, permFlag)
		}
		return nil // already bound with same perm
	}
	if err := s.bizGroupRepo.BindDashboardToGroup(ctx, dashboardID, bizGroupID, permFlag); err != nil {
		s.logger.Error("failed to bind dashboard to biz group", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// UnbindFromBizGroup removes the binding between a dashboard and a business group.
func (s *DashboardService) UnbindFromBizGroup(ctx context.Context, dashboardID, bizGroupID uint) error {
	if s.bizGroupRepo == nil {
		return apperr.WithMessage(apperr.ErrInternal, "biz group binding not configured")
	}
	if err := s.bizGroupRepo.UnbindDashboardFromGroup(ctx, dashboardID, bizGroupID); err != nil {
		s.logger.Error("failed to unbind dashboard from biz group", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ListBizGroups returns all biz group bindings for a dashboard.
func (s *DashboardService) ListBizGroups(ctx context.Context, dashboardID uint) ([]model.DashboardBizGroup, error) {
	if s.bizGroupRepo == nil {
		return nil, nil
	}
	bindings, err := s.bizGroupRepo.ListGroupsByDashboard(ctx, dashboardID)
	if err != nil {
		s.logger.Error("failed to list dashboard biz groups", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return bindings, nil
}

// ListDashboardsByGroup returns all dashboards accessible to a business group.
func (s *DashboardService) ListDashboardsByGroup(ctx context.Context, bizGroupID uint) ([]uint, error) {
	if s.bizGroupRepo == nil {
		return nil, nil
	}
	ids, err := s.bizGroupRepo.ListDashboardsByGroup(ctx, bizGroupID)
	if err != nil {
		s.logger.Error("failed to list dashboards by biz group", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return ids, nil
}
