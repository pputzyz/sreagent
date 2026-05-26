package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// TaskTplRepository handles CRUD for task templates.
type TaskTplRepository struct {
	db *gorm.DB
}

// NewTaskTplRepository creates a new TaskTplRepository.
func NewTaskTplRepository(db *gorm.DB) *TaskTplRepository {
	return &TaskTplRepository{db: db}
}

// Create inserts a new task template.
func (r *TaskTplRepository) Create(ctx context.Context, tpl *model.TaskTpl) error {
	return r.db.WithContext(ctx).Create(tpl).Error
}

// GetByID retrieves a task template by ID.
func (r *TaskTplRepository) GetByID(ctx context.Context, id uint) (*model.TaskTpl, error) {
	var tpl model.TaskTpl
	err := r.db.WithContext(ctx).First(&tpl, id).Error
	return &tpl, err
}

// Update saves changes to a task template.
func (r *TaskTplRepository) Update(ctx context.Context, tpl *model.TaskTpl) error {
	return r.db.WithContext(ctx).Save(tpl).Error
}

// Delete soft-deletes a task template by ID.
func (r *TaskTplRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TaskTpl{}, id).Error
}

// List returns a paginated list of task templates with optional keyword search.
func (r *TaskTplRepository) List(ctx context.Context, keyword string, page, pageSize int) ([]model.TaskTpl, int64, error) {
	var list []model.TaskTpl
	var total int64

	q := r.db.WithContext(ctx).Model(&model.TaskTpl{})

	if keyword != "" {
		words := strings.Fields(keyword)
		for _, w := range words {
			arg := "%" + w + "%"
			q = q.Where("name LIKE ? OR tags LIKE ? OR note LIKE ?", arg, arg, arg)
		}
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// GetByName retrieves a task template by name (for uniqueness check).
func (r *TaskTplRepository) GetByName(ctx context.Context, name string) (*model.TaskTpl, error) {
	var tpl model.TaskTpl
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tpl).Error
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}
