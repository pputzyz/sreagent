package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type UserPreferenceRepository struct {
	db *gorm.DB
}

func NewUserPreferenceRepository(db *gorm.DB) *UserPreferenceRepository {
	return &UserPreferenceRepository{db: db}
}

// GetByUserID returns the preference for a user. Creates a default record if none exists.
func (r *UserPreferenceRepository) GetByUserID(ctx context.Context, userID uint) (*model.UserPreference, error) {
	var pref model.UserPreference
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		// Return defaults without persisting
		return &model.UserPreference{
			UserID:                 userID,
			Theme:                  "auto",
			Language:               "zh-CN",
			Timezone:               "Asia/Shanghai",
			DefaultTimeRange:       "24h",
			NotificationSeverities: `["critical","warning"]`,
			AIChatMode:             "sidebar",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	// Ensure JSON fields have valid defaults
	if pref.NotificationSeverities == "" || pref.NotificationSeverities == "null" {
		pref.NotificationSeverities = `["critical","warning"]`
	}
	return &pref, nil
}

// Upsert creates or updates a user preference.
func (r *UserPreferenceRepository) Upsert(ctx context.Context, pref *model.UserPreference) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", pref.UserID).
		Assign(pref).
		FirstOrCreate(pref).Error
}
