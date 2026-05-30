package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// BizGroupRepository handles biz_groups and biz_group_members persistence.
type BizGroupRepository struct {
	db *gorm.DB
}

// NewBizGroupRepository creates a new BizGroupRepository.
func NewBizGroupRepository(db *gorm.DB) *BizGroupRepository {
	return &BizGroupRepository{db: db}
}

// Create creates a new business group.
func (r *BizGroupRepository) Create(ctx context.Context, group *model.BizGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

// GetByID returns a business group by its ID, with members preloaded.
func (r *BizGroupRepository) GetByID(ctx context.Context, id uint) (*model.BizGroup, error) {
	var group model.BizGroup
	err := r.db.WithContext(ctx).Preload("Members").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByIDLight returns a business group by its ID without preloading members.
// Used for lightweight lookups such as ancestor chain walking.
func (r *BizGroupRepository) GetByIDLight(ctx context.Context, id uint) (*model.BizGroup, error) {
	var group model.BizGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// List returns a paginated list of business groups.
func (r *BizGroupRepository) List(ctx context.Context, page, pageSize int) ([]model.BizGroup, int64, error) {
	var list []model.BizGroup
	var total int64

	query := r.db.WithContext(ctx).Model(&model.BizGroup{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id ASC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// ListAll returns all business groups without pagination.
func (r *BizGroupRepository) ListAll(ctx context.Context) ([]model.BizGroup, error) {
	var groups []model.BizGroup
	err := r.db.WithContext(ctx).Order("id ASC").Find(&groups).Error
	return groups, err
}

// ListTree returns all business groups organized as a tree (by parent_id).
// Returns all groups - the caller can build the tree structure.
func (r *BizGroupRepository) ListTree(ctx context.Context) ([]model.BizGroup, error) {
	var list []model.BizGroup
	err := r.db.WithContext(ctx).
		Order("parent_id IS NULL DESC, parent_id ASC, id ASC").
		Find(&list).Error
	return list, err
}

// ListByParentID returns all direct children of a parent group.
func (r *BizGroupRepository) ListByParentID(ctx context.Context, parentID uint) ([]model.BizGroup, error) {
	var list []model.BizGroup
	err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

// ListRoots returns all root-level groups (parent_id IS NULL).
func (r *BizGroupRepository) ListRoots(ctx context.Context) ([]model.BizGroup, error) {
	var list []model.BizGroup
	err := r.db.WithContext(ctx).
		Where("parent_id IS NULL").
		Order("id ASC").
		Find(&list).Error
	return list, err
}

// Update updates an existing business group.
func (r *BizGroupRepository) Update(ctx context.Context, group *model.BizGroup) error {
	return r.db.WithContext(ctx).Save(group).Error
}

// Delete soft-deletes a business group and removes all member associations.
func (r *BizGroupRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove all group members first
		if err := tx.WithContext(ctx).Where("biz_group_id = ?", id).Delete(&model.BizGroupMember{}).Error; err != nil {
			return err
		}
		// Delete the group
		return tx.WithContext(ctx).Delete(&model.BizGroup{}, id).Error
	})
}

// AddMember adds a user to a business group with the specified role.
func (r *BizGroupRepository) AddMember(ctx context.Context, groupID, userID uint, role string) error {
	member := &model.BizGroupMember{
		BizGroupID: groupID,
		UserID:     userID,
		Role:       role,
	}
	// Use FirstOrCreate to avoid duplicate key errors, then update role if needed
	result := r.db.WithContext(ctx).
		Where("biz_group_id = ? AND user_id = ?", groupID, userID).
		Assign(model.BizGroupMember{Role: role}).
		FirstOrCreate(member)
	return result.Error
}

// RemoveMember removes a user from a business group.
func (r *BizGroupRepository) RemoveMember(ctx context.Context, groupID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("biz_group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.BizGroupMember{}).Error
}

// ListMembers returns all members of a business group.
func (r *BizGroupRepository) ListMembers(ctx context.Context, groupID uint) ([]model.BizGroupMember, error) {
	var members []model.BizGroupMember
	err := r.db.WithContext(ctx).
		Where("biz_group_id = ?", groupID).
		Find(&members).Error
	return members, err
}

// GetMember returns a specific member of a business group.
func (r *BizGroupRepository) GetMember(ctx context.Context, groupID, userID uint) (*model.BizGroupMember, error) {
	var member model.BizGroupMember
	err := r.db.WithContext(ctx).
		Where("biz_group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}
