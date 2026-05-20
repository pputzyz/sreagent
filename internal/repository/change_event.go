package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type ChangeEventRepository struct {
	db *gorm.DB
}

func NewChangeEventRepository(db *gorm.DB) *ChangeEventRepository {
	return &ChangeEventRepository{db: db}
}

func (r *ChangeEventRepository) Create(ctx context.Context, event *model.ChangeEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *ChangeEventRepository) GetByID(ctx context.Context, id uint) (*model.ChangeEvent, error) {
	var event model.ChangeEvent
	err := r.db.WithContext(ctx).First(&event, id).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *ChangeEventRepository) List(ctx context.Context, service, environment, source string, page, pageSize int) ([]model.ChangeEvent, int64, error) {
	var list []model.ChangeEvent
	var total int64

	q := r.db.WithContext(ctx)
	if service != "" {
		q = q.Where("service = ?", service)
	}
	if environment != "" {
		q = q.Where("environment = ?", environment)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}

	if err := q.Model(&model.ChangeEvent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("timestamp DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// FindByTimeWindow finds change events in a time window for a given service.
func (r *ChangeEventRepository) FindByTimeWindow(ctx context.Context, svc string, start, end time.Time) ([]model.ChangeEvent, error) {
	var events []model.ChangeEvent
	q := r.db.WithContext(ctx).Where("timestamp BETWEEN ? AND ?", start, end)
	if svc != "" {
		q = q.Where("service = ?", svc)
	}
	err := q.Order("timestamp DESC").Find(&events).Error
	return events, err
}

func (r *ChangeEventRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ChangeEvent{}, id).Error
}
