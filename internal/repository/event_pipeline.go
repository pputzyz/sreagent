package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// EventPipelineRepository handles database operations for event pipelines.
type EventPipelineRepository struct {
	db *gorm.DB
}

// NewEventPipelineRepository creates a new EventPipelineRepository.
func NewEventPipelineRepository(db *gorm.DB) *EventPipelineRepository {
	return &EventPipelineRepository{db: db}
}

// Create inserts a new event pipeline.
func (r *EventPipelineRepository) Create(ctx context.Context, pipeline *model.EventPipeline) error {
	return r.db.WithContext(ctx).Create(pipeline).Error
}

// GetByID returns an event pipeline by ID.
func (r *EventPipelineRepository) GetByID(ctx context.Context, id uint) (*model.EventPipeline, error) {
	var pipeline model.EventPipeline
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&pipeline).Error; err != nil {
		return nil, err
	}
	pipeline.DB2FE()
	return &pipeline, nil
}

// List returns a paginated list of event pipelines with optional filters.
func (r *EventPipelineRepository) List(ctx context.Context, page, pageSize int, disabled *bool, query string) ([]model.EventPipeline, int64, error) {
	var pipelines []model.EventPipeline
	var total int64

	q := r.db.WithContext(ctx).Model(&model.EventPipeline{})
	if disabled != nil {
		q = q.Where("disabled = ?", *disabled)
	}
	if query != "" {
		q = q.Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&pipelines).Error; err != nil {
		return nil, 0, err
	}

	for i := range pipelines {
		pipelines[i].DB2FE()
	}
	return pipelines, total, nil
}

// Update modifies an existing event pipeline.
func (r *EventPipelineRepository) Update(ctx context.Context, pipeline *model.EventPipeline) error {
	return r.db.WithContext(ctx).Model(pipeline).Select("*").Updates(pipeline).Error
}

// Delete soft-deletes an event pipeline.
func (r *EventPipelineRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.EventPipeline{}, id).Error
}

// ListAllEnabled returns all non-disabled pipelines (for engine cache).
func (r *EventPipelineRepository) ListAllEnabled(ctx context.Context) ([]model.EventPipeline, error) {
	var pipelines []model.EventPipeline
	if err := r.db.WithContext(ctx).Where("disabled = false").Find(&pipelines).Error; err != nil {
		return nil, err
	}
	for i := range pipelines {
		pipelines[i].DB2FE()
	}
	return pipelines, nil
}

// EventPipelineExecutionRepository handles database operations for pipeline executions.
type EventPipelineExecutionRepository struct {
	db *gorm.DB
}

// NewEventPipelineExecutionRepository creates a new EventPipelineExecutionRepository.
func NewEventPipelineExecutionRepository(db *gorm.DB) *EventPipelineExecutionRepository {
	return &EventPipelineExecutionRepository{db: db}
}

// Create inserts a new execution record.
func (r *EventPipelineExecutionRepository) Create(ctx context.Context, exec *model.EventPipelineExecution) error {
	return r.db.WithContext(ctx).Create(exec).Error
}

// GetByID returns an execution record by ID.
func (r *EventPipelineExecutionRepository) GetByID(ctx context.Context, id string) (*model.EventPipelineExecution, error) {
	var exec model.EventPipelineExecution
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// ListByPipelineID returns paginated executions for a specific pipeline.
func (r *EventPipelineExecutionRepository) ListByPipelineID(ctx context.Context, pipelineID uint, page, pageSize int) ([]model.EventPipelineExecution, int64, error) {
	var execs []model.EventPipelineExecution
	var total int64

	q := r.db.WithContext(ctx).Model(&model.EventPipelineExecution{}).Where("pipeline_id = ?", pipelineID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&execs).Error; err != nil {
		return nil, 0, err
	}
	return execs, total, nil
}

// Update updates an execution record.
func (r *EventPipelineExecutionRepository) Update(ctx context.Context, exec *model.EventPipelineExecution) error {
	return r.db.WithContext(ctx).Save(exec).Error
}

// CleanOlderThan deletes execution records older than the specified number of days.
func (r *EventPipelineExecutionRepository) CleanOlderThan(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&model.EventPipelineExecution{})
	return result.RowsAffected, result.Error
}

// GetLatestByEventID returns the most recent execution for a given event.
func (r *EventPipelineExecutionRepository) GetLatestByEventID(ctx context.Context, eventID uint) (*model.EventPipelineExecution, error) {
	var exec model.EventPipelineExecution
	if err := r.db.WithContext(ctx).Where("event_id = ?", eventID).Order("id DESC").First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}
