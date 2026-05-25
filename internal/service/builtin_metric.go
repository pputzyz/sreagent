package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

type BuiltinMetricService struct {
	repo   *repository.BuiltinMetricRepository
	logger *zap.Logger
}

func NewBuiltinMetricService(repo *repository.BuiltinMetricRepository, logger *zap.Logger) *BuiltinMetricService {
	return &BuiltinMetricService{repo: repo, logger: logger}
}

func (s *BuiltinMetricService) Create(ctx context.Context, m *model.BuiltinMetric) error {
	m.FE2DB()
	if err := m.Verify(); err != nil {
		return err
	}
	m.ID = 0
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return s.repo.Create(ctx, m)
}

func (s *BuiltinMetricService) GetByID(ctx context.Context, id uint) (*model.BuiltinMetric, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BuiltinMetricService) List(ctx context.Context, collector, typ, query, unit, lang string, page, pageSize int) ([]model.BuiltinMetric, int64, error) {
	return s.repo.List(ctx, collector, typ, query, unit, lang, page, pageSize)
}

func (s *BuiltinMetricService) Update(ctx context.Context, m *model.BuiltinMetric) error {
	m.FE2DB()
	if err := m.Verify(); err != nil {
		return err
	}
	m.UpdatedAt = time.Now()
	return s.repo.Update(ctx, m)
}

func (s *BuiltinMetricService) Delete(ctx context.Context, ids []uint) error {
	return s.repo.Delete(ctx, ids)
}

func (s *BuiltinMetricService) GetTypes(ctx context.Context, collector, query, lang string) ([]string, error) {
	return s.repo.GetTypes(ctx, collector, query, lang)
}

func (s *BuiltinMetricService) GetCollectors(ctx context.Context, typ, query, lang string) ([]string, error) {
	return s.repo.GetCollectors(ctx, typ, query, lang)
}

// MetricFilter service

type MetricFilterService struct {
	repo   *repository.MetricFilterRepository
	logger *zap.Logger
}

func NewMetricFilterService(repo *repository.MetricFilterRepository, logger *zap.Logger) *MetricFilterService {
	return &MetricFilterService{repo: repo, logger: logger}
}

func (s *MetricFilterService) Create(ctx context.Context, f *model.MetricFilter) error {
	f.FE2DB()
	f.ID = 0
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	return s.repo.Create(ctx, f)
}

func (s *MetricFilterService) List(ctx context.Context, createdBy string) ([]model.MetricFilter, error) {
	return s.repo.List(ctx, createdBy)
}

func (s *MetricFilterService) Update(ctx context.Context, f *model.MetricFilter) error {
	f.FE2DB()
	f.UpdatedAt = time.Now()
	return s.repo.Update(ctx, f)
}

func (s *MetricFilterService) Delete(ctx context.Context, ids []uint) error {
	return s.repo.Delete(ctx, ids)
}
