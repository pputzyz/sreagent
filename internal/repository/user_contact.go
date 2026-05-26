package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// UserContactRepository handles user contact persistence.
type UserContactRepository struct {
	db *gorm.DB
}

// NewUserContactRepository creates a new UserContactRepository.
func NewUserContactRepository(db *gorm.DB) *UserContactRepository {
	return &UserContactRepository{db: db}
}

// Create creates a new user contact.
func (r *UserContactRepository) Create(ctx context.Context, contact *model.UserContact) error {
	return r.db.WithContext(ctx).Create(contact).Error
}

// GetByID returns a user contact by its ID.
func (r *UserContactRepository) GetByID(ctx context.Context, id uint) (*model.UserContact, error) {
	var contact model.UserContact
	err := r.db.WithContext(ctx).First(&contact, id).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// ListByUserID returns all contacts for a user.
func (r *UserContactRepository) ListByUserID(ctx context.Context, userID uint) ([]model.UserContact, error) {
	var contacts []model.UserContact
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, type ASC, id ASC").
		Find(&contacts).Error
	return contacts, err
}

// Update updates a user contact.
func (r *UserContactRepository) Update(ctx context.Context, contact *model.UserContact) error {
	return r.db.WithContext(ctx).Save(contact).Error
}

// Delete soft-deletes a user contact.
func (r *UserContactRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.UserContact{}, id).Error
}

// CountByUserAndType returns the number of contacts of a given type for a user.
func (r *UserContactRepository) CountByUserAndType(ctx context.Context, userID uint, contactType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserContact{}).
		Where("user_id = ? AND type = ?", userID, contactType).
		Count(&count).Error
	return count, err
}

// GetByUserAndValue returns a contact by user, type, and value (for uniqueness check).
func (r *UserContactRepository) GetByUserAndValue(ctx context.Context, userID uint, contactType, value string) (*model.UserContact, error) {
	var contact model.UserContact
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND value = ?", userID, contactType, value).
		First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// ClearDefault clears the is_default flag for all contacts of a given type for a user.
func (r *UserContactRepository) ClearDefault(ctx context.Context, userID uint, contactType string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserContact{}).
		Where("user_id = ? AND type = ? AND is_default = ?", userID, contactType, true).
		Update("is_default", false).Error
}
