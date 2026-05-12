package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type PetRepository struct {
	db *gorm.DB
}

func NewPetRepository(db *gorm.DB) *PetRepository {
	return &PetRepository{db: db}
}

// GetByUserID returns the pet for a given user, or nil if not found.
func (r *PetRepository) GetByUserID(ctx context.Context, userID uint) (*model.Pet, error) {
	var pet model.Pet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

// Create inserts a new pet record.
func (r *PetRepository) Create(ctx context.Context, pet *model.Pet) error {
	return r.db.WithContext(ctx).Create(pet).Error
}

// Update saves changes to an existing pet.
func (r *PetRepository) Update(ctx context.Context, pet *model.Pet) error {
	return r.db.WithContext(ctx).Save(pet).Error
}

// CreateInteraction inserts a new interaction record.
func (r *PetRepository) CreateInteraction(ctx context.Context, interaction *model.PetInteraction) error {
	return r.db.WithContext(ctx).Create(interaction).Error
}

// FeedAtomic atomically reduces hunger and adds exp.
func (r *PetRepository) FeedAtomic(ctx context.Context, userID uint, hungerDelta, expDelta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Pet{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"hunger": gorm.Expr("GREATEST(hunger - ?, 0)", hungerDelta),
			"exp":    gorm.Expr("exp + ?", expDelta),
		}).Error
}

// PlayAtomic atomically increases mood and adds exp.
func (r *PetRepository) PlayAtomic(ctx context.Context, userID uint, moodDelta, expDelta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Pet{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"mood": gorm.Expr("LEAST(mood + ?, 100)", moodDelta),
			"exp":  gorm.Expr("exp + ?", expDelta),
		}).Error
}

// AddExpAtomic atomically adds exp.
func (r *PetRepository) AddExpAtomic(ctx context.Context, userID uint, expDelta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Pet{}).
		Where("user_id = ?", userID).
		Update("exp", gorm.Expr("exp + ?", expDelta)).Error
}

// ListInteractions returns recent interactions for a pet, ordered by created_at DESC.
func (r *PetRepository) ListInteractions(ctx context.Context, petID uint, limit int) ([]model.PetInteraction, error) {
	var interactions []model.PetInteraction
	err := r.db.WithContext(ctx).
		Where("pet_id = ?", petID).
		Order("created_at DESC").
		Limit(limit).
		Find(&interactions).Error
	return interactions, err
}
