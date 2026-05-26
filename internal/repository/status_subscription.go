package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sreagent/sreagent/internal/model"
)

type StatusSubscriptionRepository struct {
	db *gorm.DB
}

func NewStatusSubscriptionRepository(db *gorm.DB) *StatusSubscriptionRepository {
	return &StatusSubscriptionRepository{db: db}
}

func (r *StatusSubscriptionRepository) Subscribe(ctx context.Context, email string) error {
	sub := model.StatusSubscription{Email: email, IsActive: true}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}},
			DoUpdates: clause.AssignmentColumns([]string{"is_active", "updated_at"}),
		}).
		Create(&sub).Error
}

func (r *StatusSubscriptionRepository) Unsubscribe(ctx context.Context, email string) error {
	return r.db.WithContext(ctx).
		Model(&model.StatusSubscription{}).
		Where("email = ?", email).
		Update("is_active", false).Error
}

func (r *StatusSubscriptionRepository) List(ctx context.Context) ([]model.StatusSubscription, error) {
	var subs []model.StatusSubscription
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("created_at DESC").Find(&subs).Error
	return subs, err
}
