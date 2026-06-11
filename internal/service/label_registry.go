package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/crypto"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/repository"
)

type labelKeyCacheEntry struct {
	keys      []string
	expiresAt time.Time
}

type LabelRegistryService struct {
	repo   *repository.LabelRegistryRepository
	dsRepo *repository.DataSourceRepository
	logger *zap.Logger

	keyCache   sync.RWMutex
	keyEntries map[string]*labelKeyCacheEntry // key: "all" or comma-separated dsIDs
	keyTTL     time.Duration
}

func NewLabelRegistryService(
	repo *repository.LabelRegistryRepository,
	dsRepo *repository.DataSourceRepository,
	logger *zap.Logger,
) *LabelRegistryService {
	return &LabelRegistryService{
		repo:       repo,
		dsRepo:     dsRepo,
		logger:     logger,
		keyEntries: make(map[string]*labelKeyCacheEntry),
		keyTTL:     10 * time.Minute,
	}
}

// SyncDatasource scrapes all label key/values from one Prometheus-compatible datasource.
func (s *LabelRegistryService) SyncDatasource(ctx context.Context, ds *model.DataSource) error {
	// Decrypt AuthConfig if encrypted so the datasource client receives plaintext credentials.
	authConfig := ds.AuthConfig
	if crypto.IsEncrypted(authConfig) {
		decrypted, err := crypto.DecryptString(authConfig)
		if err == nil {
			authConfig = decrypted
		}
	}
	labels, err := datasource.FetchAllLabels(ctx, string(ds.Type), ds.Endpoint, ds.AuthType, authConfig)
	if err != nil {
		return err
	}
	if len(labels) == 0 {
		return nil
	}

	// Delete stale entries for this datasource before upserting fresh ones.
	// This ensures labels removed upstream are cleaned up.
	if err := s.repo.DeleteByDatasource(ctx, ds.ID); err != nil {
		s.logger.Warn("label registry: failed to delete stale entries",
			zap.Uint("ds_id", ds.ID), zap.Error(err))
		// Continue with upsert even if delete fails — stale entries are better than no sync
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
	return s.repo.UpsertBatch(ctx, entries)
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
	if err := s.repo.UpsertBatch(context.Background(), entries); err != nil {
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
func (s *LabelRegistryService) GetValues(ctx context.Context, key string, datasourceIDs []uint) ([]string, error) {
	return s.repo.GetValues(ctx, key, datasourceIDs)
}

// GetKeys returns known label keys (for key autocomplete).
func (s *LabelRegistryService) GetKeys(ctx context.Context, datasourceIDs []uint) ([]string, error) {
	return s.repo.GetKeys(ctx, datasourceIDs)
}

// keyCacheKey builds a deterministic cache key from datasource IDs.
func keyCacheKey(datasourceIDs []uint) string {
	if len(datasourceIDs) == 0 {
		return "all"
	}
	// Use a simple concatenation since dsIDs are small integers
	b := make([]byte, 0, len(datasourceIDs)*4)
	for _, id := range datasourceIDs {
		b = append(b, byte(id), byte(id>>8), byte(id>>16), byte(id>>24))
	}
	return string(b)
}

// SetKeys caches label keys for the given datasource IDs.
func (s *LabelRegistryService) SetKeys(datasourceIDs []uint, keys []string) {
	s.keyCache.Lock()
	defer s.keyCache.Unlock()
	s.keyEntries[keyCacheKey(datasourceIDs)] = &labelKeyCacheEntry{
		keys:      keys,
		expiresAt: time.Now().Add(s.keyTTL),
	}
}

// GetKeysFallback returns cached label keys if available, otherwise queries
// the repo and caches the result. Returns empty slice on error (never nil).
func (s *LabelRegistryService) GetKeysFallback(datasourceIDs []uint) []string {
	ck := keyCacheKey(datasourceIDs)

	// Try cache first
	s.keyCache.RLock()
	if entry, ok := s.keyEntries[ck]; ok && time.Now().Before(entry.expiresAt) {
		s.keyCache.RUnlock()
		return entry.keys
	}
	s.keyCache.RUnlock()

	// Cache miss — query repo
	keys, err := s.repo.GetKeys(context.Background(), datasourceIDs)
	if err != nil {
		s.logger.Warn("GetKeysFallback: repo query failed", zap.Error(err))
		return []string{}
	}

	// Store in cache
	s.keyCache.Lock()
	s.keyEntries[ck] = &labelKeyCacheEntry{
		keys:      keys,
		expiresAt: time.Now().Add(s.keyTTL),
	}
	s.keyCache.Unlock()

	return keys
}
