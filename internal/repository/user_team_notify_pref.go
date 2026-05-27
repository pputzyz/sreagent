package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type UserTeamNotifyPrefRepository struct {
	db *gorm.DB
}

func NewUserTeamNotifyPrefRepository(db *gorm.DB) *UserTeamNotifyPrefRepository {
	return &UserTeamNotifyPrefRepository{db: db}
}

func (r *UserTeamNotifyPrefRepository) Create(ctx context.Context, pref *model.UserTeamNotifyPref) error {
	return r.db.WithContext(ctx).Create(pref).Error
}

func (r *UserTeamNotifyPrefRepository) GetByUserTeamMedia(ctx context.Context, userID, teamID, mediaID uint) (*model.UserTeamNotifyPref, error) {
	var pref model.UserTeamNotifyPref
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND team_id = ? AND media_id = ?", userID, teamID, mediaID).
		First(&pref).Error
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *UserTeamNotifyPrefRepository) ListByUser(ctx context.Context, userID uint) ([]model.UserTeamNotifyPref, error) {
	var prefs []model.UserTeamNotifyPref
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&prefs).Error
	return prefs, err
}

func (r *UserTeamNotifyPrefRepository) DeleteByUser(ctx context.Context, id, userID uint) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.UserTeamNotifyPref{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *UserTeamNotifyPrefRepository) Update(ctx context.Context, pref *model.UserTeamNotifyPref) error {
	return r.db.WithContext(ctx).Save(pref).Error
}

func (r *UserTeamNotifyPrefRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.UserTeamNotifyPref{}, id).Error
}
