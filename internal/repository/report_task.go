package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ReportTaskRepository handles CRUD for report tasks and runs.
type ReportTaskRepository struct {
	db *gorm.DB
}

// NewReportTaskRepository creates a new ReportTaskRepository.
func NewReportTaskRepository(db *gorm.DB) *ReportTaskRepository {
	return &ReportTaskRepository{db: db}
}

// ── Task CRUD ──

func (r *ReportTaskRepository) CreateTask(ctx context.Context, task *model.ReportTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *ReportTaskRepository) GetTask(ctx context.Context, id uint) (*model.ReportTask, error) {
	var task model.ReportTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	return &task, err
}

func (r *ReportTaskRepository) UpdateTask(ctx context.Context, task *model.ReportTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *ReportTaskRepository) DeleteTask(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ReportTask{}, id).Error
}

func (r *ReportTaskRepository) ListTasks(ctx context.Context, enabled *bool) ([]model.ReportTask, error) {
	var list []model.ReportTask
	q := r.db.WithContext(ctx)
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

// ── Run CRUD ──

func (r *ReportTaskRepository) CreateRun(ctx context.Context, run *model.ReportRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *ReportTaskRepository) GetRun(ctx context.Context, id uint) (*model.ReportRun, error) {
	var run model.ReportRun
	err := r.db.WithContext(ctx).First(&run, id).Error
	return &run, err
}

func (r *ReportTaskRepository) UpdateRun(ctx context.Context, run *model.ReportRun) error {
	return r.db.WithContext(ctx).Save(run).Error
}

func (r *ReportTaskRepository) ListRuns(ctx context.Context, taskID *uint, page, pageSize int) ([]model.ReportRun, int64, error) {
	var list []model.ReportRun
	var total int64
	q := r.db.WithContext(ctx)
	if taskID != nil {
		q = q.Where("task_id = ?", *taskID)
	}
	if err := q.Model(&model.ReportRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListEnabledTasks returns all enabled tasks (for scheduler registration).
func (r *ReportTaskRepository) ListEnabledTasks(ctx context.Context) ([]model.ReportTask, error) {
	var list []model.ReportTask
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&list).Error
	return list, err
}
