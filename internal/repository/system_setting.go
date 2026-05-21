package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sreagent/sreagent/internal/model"
)

// SystemSettingRepository provides CRUD access to the system_settings table.
type SystemSettingRepository struct {
	db *gorm.DB
}

// NewSystemSettingRepository creates a new SystemSettingRepository.
func NewSystemSettingRepository(db *gorm.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// Get returns the value for the given group+key.
// Returns ("", nil) when the row does not exist.
func (r *SystemSettingRepository) Get(ctx context.Context, group, key string) (string, error) {
	var s model.SystemSetting
	err := r.db.WithContext(ctx).
		Where("`group` = ? AND `key` = ?", group, key).
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return s.Value, nil
}

// Set upserts the value for the given group+key.
// updated_at is managed by MySQL's ON UPDATE CURRENT_TIMESTAMP.
func (r *SystemSettingRepository) Set(ctx context.Context, group, key, value string) error {
	s := model.SystemSetting{
		Group: group,
		Key:   key,
		Value: value,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).
		Create(&s).Error
}

// ListByGroup returns all settings for a given group as a map[key]value.
func (r *SystemSettingRepository) ListByGroup(ctx context.Context, group string) (map[string]string, error) {
	var rows []model.SystemSetting
	if err := r.db.WithContext(ctx).
		Where("`group` = ?", group).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

// SetGroup upserts all key-value pairs for a given group in a single transaction.
// updated_at is managed by MySQL's ON UPDATE CURRENT_TIMESTAMP.
func (r *SystemSettingRepository) SetGroup(ctx context.Context, group string, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	rows := make([]model.SystemSetting, 0, len(kv))
	for k, v := range kv {
		rows = append(rows, model.SystemSetting{Group: group, Key: k, Value: v})
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).
		Create(&rows).Error
}

// Delete removes a single setting by group+key.
func (r *SystemSettingRepository) Delete(ctx context.Context, group, key string) error {
	return r.db.WithContext(ctx).
		Where("`group` = ? AND `key` = ?", group, key).
		Delete(&model.SystemSetting{}).Error
}
