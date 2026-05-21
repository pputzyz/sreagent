package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// InspectionRepository handles CRUD for inspection tasks and runs.
type InspectionRepository struct {
	db *gorm.DB
}

// NewInspectionRepository creates a new InspectionRepository.
func NewInspectionRepository(db *gorm.DB) *InspectionRepository {
	return &InspectionRepository{db: db}
}

// ── Task CRUD ──

func (r *InspectionRepository) CreateTask(ctx context.Context, task *model.InspectionTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *InspectionRepository) GetTask(ctx context.Context, id uint) (*model.InspectionTask, error) {
	var task model.InspectionTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	return &task, err
}

func (r *InspectionRepository) UpdateTask(ctx context.Context, task *model.InspectionTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *InspectionRepository) DeleteTask(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.InspectionTask{}, id).Error
}

func (r *InspectionRepository) ListTasks(ctx context.Context, enabled *bool) ([]model.InspectionTask, error) {
	var list []model.InspectionTask
	q := r.db.WithContext(ctx)
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

// ── Run CRUD ──

func (r *InspectionRepository) CreateRun(ctx context.Context, run *model.InspectionRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *InspectionRepository) GetRun(ctx context.Context, id uint) (*model.InspectionRun, error) {
	var run model.InspectionRun
	err := r.db.WithContext(ctx).First(&run, id).Error
	return &run, err
}

func (r *InspectionRepository) UpdateRun(ctx context.Context, run *model.InspectionRun) error {
	return r.db.WithContext(ctx).Save(run).Error
}

func (r *InspectionRepository) ListRuns(ctx context.Context, taskID *uint, page, pageSize int) ([]model.InspectionRun, int64, error) {
	var list []model.InspectionRun
	var total int64
	q := r.db.WithContext(ctx)
	if taskID != nil {
		q = q.Where("task_id = ?", *taskID)
	}
	if err := q.Model(&model.InspectionRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListEnabledTasks returns all enabled tasks (for scheduler registration).
func (r *InspectionRepository) ListEnabledTasks(ctx context.Context) ([]model.InspectionTask, error) {
	var list []model.InspectionTask
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&list).Error
	return list, err
}
