package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type BuiltinDashboardService struct {
	repo      *repository.BuiltinDashboardRepository
	dashRepo  *repository.DashboardRepository
	logger    *zap.Logger
}

func NewBuiltinDashboardService(
	repo *repository.BuiltinDashboardRepository,
	dashRepo *repository.DashboardRepository,
	logger *zap.Logger,
) *BuiltinDashboardService {
	return &BuiltinDashboardService{repo: repo, dashRepo: dashRepo, logger: logger}
}

// List returns builtin dashboards with optional filters.
func (s *BuiltinDashboardService) List(ctx context.Context, category, component, query string, page, pageSize int) ([]model.BuiltinDashboard, int64, error) {
	return s.repo.List(ctx, category, component, query, page, pageSize)
}

// GetByID returns a builtin dashboard by ID.
func (s *BuiltinDashboardService) GetByID(ctx context.Context, id uint) (*model.BuiltinDashboard, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return d, nil
}

// GetByIdent returns a builtin dashboard by slug identifier.
func (s *BuiltinDashboardService) GetByIdent(ctx context.Context, ident string) (*model.BuiltinDashboard, error) {
	d, err := s.repo.GetByIdent(ctx, ident)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return d, nil
}

// GetCategories returns distinct categories.
func (s *BuiltinDashboardService) GetCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetCategories(ctx)
}

// GetComponents returns distinct components.
func (s *BuiltinDashboardService) GetComponents(ctx context.Context) ([]string, error) {
	return s.repo.GetComponents(ctx)
}

// Create adds a new builtin dashboard (admin only).
func (s *BuiltinDashboardService) Create(ctx context.Context, d *model.BuiltinDashboard) error {
	if d.Ident == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "ident is required")
	}
	if d.Name == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "name is required")
	}
	d.BuiltIn = true
	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error("failed to create builtin dashboard", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Import copies a builtin dashboard config into the user's dashboard collection.
func (s *BuiltinDashboardService) Import(ctx context.Context, ident string, userID uint) (*model.Dashboard, error) {
	builtin, err := s.repo.GetByIdent(ctx, ident)
	if err != nil {
		return nil, apperr.ErrNotFound
	}

	dash := &model.Dashboard{
		Name:      builtin.Name,
		Config:    builtin.Config,
		CreatedBy: userID,
		UpdatedBy: userID,
		IsPublic:  false,
	}

	if err := s.dashRepo.Create(ctx, dash); err != nil {
		s.logger.Error("failed to import builtin dashboard", zap.String("ident", ident), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("imported builtin dashboard",
		zap.String("ident", ident),
		zap.Uint("user_id", userID),
		zap.Uint("dashboard_id", dash.ID),
	)
	return dash, nil
}

// SeedDefaults seeds the built-in dashboards if the table is empty.
func (s *BuiltinDashboardService) SeedDefaults(ctx context.Context) error {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	dashboards := model.SeedBuiltinDashboards()
	for i := range dashboards {
		if err := s.repo.Create(ctx, &dashboards[i]); err != nil {
			s.logger.Error("failed to seed builtin dashboard",
				zap.String("ident", dashboards[i].Ident),
				zap.Error(err),
			)
			continue
		}
	}

	s.logger.Info("seeded built-in dashboards", zap.Int("count", len(dashboards)))
	return nil
}
