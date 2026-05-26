package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// TaskRecordRepository handles CRUD for task execution records and host records.
type TaskRecordRepository struct {
	db *gorm.DB
}

// NewTaskRecordRepository creates a new TaskRecordRepository.
func NewTaskRecordRepository(db *gorm.DB) *TaskRecordRepository {
	return &TaskRecordRepository{db: db}
}

// ── TaskRecord CRUD ──

// CreateRecord inserts a new task execution record.
func (r *TaskRecordRepository) CreateRecord(ctx context.Context, rec *model.TaskRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

// GetRecordByID retrieves a task record by ID.
func (r *TaskRecordRepository) GetRecordByID(ctx context.Context, id uint) (*model.TaskRecord, error) {
	var rec model.TaskRecord
	err := r.db.WithContext(ctx).First(&rec, id).Error
	return &rec, err
}

// UpdateRecord saves changes to a task record.
func (r *TaskRecordRepository) UpdateRecord(ctx context.Context, rec *model.TaskRecord) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

// ListRecords returns a paginated list of task records with optional filters.
func (r *TaskRecordRepository) ListRecords(ctx context.Context, tplID *uint, eventID *uint, status *int, page, pageSize int) ([]model.TaskRecord, int64, error) {
	var list []model.TaskRecord
	var total int64

	q := r.db.WithContext(ctx).Model(&model.TaskRecord{})

	if tplID != nil {
		q = q.Where("tpl_id = ?", *tplID)
	}
	if eventID != nil {
		q = q.Where("event_id = ?", *eventID)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
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

// ── TaskHostRecord CRUD ──

// CreateHostRecord inserts a new host execution record.
func (r *TaskRecordRepository) CreateHostRecord(ctx context.Context, rec *model.TaskHostRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

// UpdateHostRecord saves changes to a host record.
func (r *TaskRecordRepository) UpdateHostRecord(ctx context.Context, rec *model.TaskHostRecord) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

// ListHostRecords returns all host records for a given task.
func (r *TaskRecordRepository) ListHostRecords(ctx context.Context, taskID uint) ([]model.TaskHostRecord, error) {
	var list []model.TaskHostRecord
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("id ASC").Find(&list).Error
	return list, err
}

// GetHostRecordByID retrieves a host record by ID.
func (r *TaskRecordRepository) GetHostRecordByID(ctx context.Context, id uint) (*model.TaskHostRecord, error) {
	var rec model.TaskHostRecord
	err := r.db.WithContext(ctx).First(&rec, id).Error
	return &rec, err
}
