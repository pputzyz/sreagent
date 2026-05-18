package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sreagent/sreagent/internal/model"
)

type LabelRegistryRepository struct {
	db *gorm.DB
}

func NewLabelRegistryRepository(db *gorm.DB) *LabelRegistryRepository {
	return &LabelRegistryRepository{db: db}
}

// UpsertBatch inserts or updates label registry entries in a single batch.
// On conflict (datasource_id, label_key, label_value) it increments hit_count and updates last_seen_at.
func (r *LabelRegistryRepository) UpsertBatch(entries []*model.LabelRegistry) error {
	if len(entries) == 0 {
		return nil
	}
	// Process in chunks of 500 to avoid huge INSERT statements
	chunkSize := 500
	for i := 0; i < len(entries); i += chunkSize {
		end := i + chunkSize
		if end > len(entries) {
			end = len(entries)
		}
		chunk := entries[i:end]
		err := r.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "datasource_id"},
				{Name: "label_key"},
				{Name: "label_value"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"last_seen_at": time.Now(),
				"hit_count":    gorm.Expr("hit_count + 1"),
			}),
		}).Create(&chunk).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// GetValues returns label values for a given key, optionally filtered by datasource IDs.
// Results are ordered by hit_count descending (most common first), limited to 100.
func (r *LabelRegistryRepository) GetValues(key string, datasourceIDs []uint) ([]string, error) {
	query := r.db.Model(&model.LabelRegistry{}).
		Where("label_key = ?", key).
		Order("hit_count DESC").
		Limit(100)
	if len(datasourceIDs) > 0 {
		query = query.Where("datasource_id IN ?", datasourceIDs)
	}
	var entries []model.LabelRegistry
	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}
	vals := make([]string, 0, len(entries))
	seen := make(map[string]bool)
	for _, e := range entries {
		if !seen[e.LabelValue] {
			seen[e.LabelValue] = true
			vals = append(vals, e.LabelValue)
		}
	}
	return vals, nil
}

// GetKeys returns distinct label keys, optionally filtered by datasource IDs.
// Ordered by total hit_count desc, limited to 100.
func (r *LabelRegistryRepository) GetKeys(datasourceIDs []uint) ([]string, error) {
	query := r.db.Model(&model.LabelRegistry{}).
		Select("label_key, SUM(hit_count) AS total").
		Group("label_key").
		Order("total DESC").
		Limit(100)
	if len(datasourceIDs) > 0 {
		query = query.Where("datasource_id IN ?", datasourceIDs)
	}
	type row struct {
		LabelKey string
		Total    int64
	}
	var rows []row
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(rows))
	for i, r := range rows {
		keys[i] = r.LabelKey
	}
	return keys, nil
}

// DeleteByDatasource removes all entries for a given datasource (used when DS is deleted).
func (r *LabelRegistryRepository) DeleteByDatasource(datasourceID uint) error {
	return r.db.Where("datasource_id = ?", datasourceID).Delete(&model.LabelRegistry{}).Error
}

// GetKeysByDatasource returns distinct label keys for a specific datasource.
// Ordered by total hit_count desc, limited to 100.
func (r *LabelRegistryRepository) GetKeysByDatasource(ctx context.Context, datasourceID uint) ([]string, error) {
	type row struct {
		LabelKey string
		Total    int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Model(&model.LabelRegistry{}).
		Select("label_key, SUM(hit_count) AS total").
		Where("datasource_id = ?", datasourceID).
		Group("label_key").
		Order("total DESC").
		Limit(100).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(rows))
	for i, r := range rows {
		keys[i] = r.LabelKey
	}
	return keys, nil
}

// GetValuesByDatasource returns label values for a given key in a specific datasource.
// Results are ordered by hit_count descending, limited to 100.
func (r *LabelRegistryRepository) GetValuesByDatasource(ctx context.Context, datasourceID uint, key string) ([]string, error) {
	var entries []model.LabelRegistry
	if err := r.db.WithContext(ctx).Model(&model.LabelRegistry{}).
		Where("datasource_id = ? AND label_key = ?", datasourceID, key).
		Order("hit_count DESC").
		Limit(100).
		Find(&entries).Error; err != nil {
		return nil, err
	}
	vals := make([]string, 0, len(entries))
	seen := make(map[string]bool)
	for _, e := range entries {
		if !seen[e.LabelValue] {
			seen[e.LabelValue] = true
			vals = append(vals, e.LabelValue)
		}
	}
	return vals, nil
}
