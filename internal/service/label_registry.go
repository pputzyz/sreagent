package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/repository"
)

type LabelRegistryService struct {
	repo   *repository.LabelRegistryRepository
	dsRepo *repository.DataSourceRepository
	logger *zap.Logger
}

func NewLabelRegistryService(
	repo *repository.LabelRegistryRepository,
	dsRepo *repository.DataSourceRepository,
	logger *zap.Logger,
) *LabelRegistryService {
	return &LabelRegistryService{repo: repo, dsRepo: dsRepo, logger: logger}
}

// SyncDatasource scrapes all label key/values from one Prometheus-compatible datasource.
func (s *LabelRegistryService) SyncDatasource(ctx context.Context, ds *model.DataSource) error {
	labels, err := datasource.FetchAllLabels(ctx, string(ds.Type), ds.Endpoint, ds.AuthType, ds.AuthConfig)
	if err != nil {
		return err
	}
	if len(labels) == 0 {
		return nil
	}

	now := time.Now()
	entries := make([]*model.LabelRegistry, 0, len(labels)*10)
	for key, values := range labels {
		for _, val := range values {
			if len(val) > 2048 {
				val = val[:2048]
			}
			entries = append(entries, &model.LabelRegistry{
				DatasourceID: ds.ID,
				LabelKey:     key,
				LabelValue:   val,
				Source:       "sync",
				LastSeenAt:   now,
				HitCount:     1,
			})
		}
	}
	return s.repo.UpsertBatch(entries)
}

// RecordFromLabels passively records labels from an alert event (works for all datasource types).
func (s *LabelRegistryService) RecordFromLabels(datasourceID uint, labels map[string]string) {
	if len(labels) == 0 {
		return
	}
	now := time.Now()
	entries := make([]*model.LabelRegistry, 0, len(labels))
	for k, v := range labels {
		if k == "" || v == "" {
			continue
		}
		if len(v) > 2048 {
			v = v[:2048]
		}
		entries = append(entries, &model.LabelRegistry{
			DatasourceID: datasourceID,
			LabelKey:     k,
			LabelValue:   v,
			Source:       "event",
			LastSeenAt:   now,
			HitCount:     1,
		})
	}
	if err := s.repo.UpsertBatch(entries); err != nil {
		s.logger.Warn("label registry passive record failed", zap.Error(err))
	}
}

// SyncAll scrapes all enabled Prom/VM datasources.
func (s *LabelRegistryService) SyncAll(ctx context.Context) {
	dsList, err := s.dsRepo.ListEnabled(ctx)
	if err != nil {
		s.logger.Error("label registry sync: list datasources failed", zap.Error(err))
		return
	}
	for _, ds := range dsList {
		dsType := string(ds.Type)
		if dsType != "prometheus" && dsType != "victoriametrics" {
			continue
		}
		dsCopy := ds
		if err := s.SyncDatasource(ctx, &dsCopy); err != nil {
			s.logger.Warn("label registry sync failed",
				zap.String("ds", ds.Name), zap.Error(err))
		} else {
			s.logger.Info("label registry synced", zap.String("ds", ds.Name))
		}
	}
}

// StartSyncWorker runs SyncAll on a ticker until ctx is cancelled.
func (s *LabelRegistryService) StartSyncWorker(ctx context.Context, interval time.Duration) {
	// Run immediately on startup
	s.SyncAll(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SyncAll(ctx)
		}
	}
}

// GetValues returns autocomplete values for a label key.
func (s *LabelRegistryService) GetValues(key string, datasourceIDs []uint) ([]string, error) {
	return s.repo.GetValues(key, datasourceIDs)
}

// GetKeys returns known label keys (for key autocomplete).
func (s *LabelRegistryService) GetKeys(datasourceIDs []uint) ([]string, error) {
	return s.repo.GetKeys(datasourceIDs)
}

// GetKeysByDatasource returns label keys for a specific datasource.
func (s *LabelRegistryService) GetKeysByDatasource(ctx context.Context, datasourceID uint) ([]string, error) {
	return s.repo.GetKeysByDatasource(ctx, datasourceID)
}

// GetValuesByDatasource returns label values for a specific key in a specific datasource.
func (s *LabelRegistryService) GetValuesByDatasource(ctx context.Context, datasourceID uint, key string) ([]string, error) {
	return s.repo.GetValuesByDatasource(ctx, datasourceID, key)
}
