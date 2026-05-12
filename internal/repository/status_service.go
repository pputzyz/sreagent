package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type StatusServiceRepository struct {
	db *gorm.DB
}

func NewStatusServiceRepository(db *gorm.DB) *StatusServiceRepository {
	return &StatusServiceRepository{db: db}
}

func (r *StatusServiceRepository) Create(ctx context.Context, svc *model.StatusService) error {
	return r.db.WithContext(ctx).Create(svc).Error
}

func (r *StatusServiceRepository) GetByID(ctx context.Context, id uint) (*model.StatusService, error) {
	var svc model.StatusService
	if err := r.db.WithContext(ctx).First(&svc, id).Error; err != nil {
		return nil, err
	}
	return &svc, nil
}

func (r *StatusServiceRepository) Update(ctx context.Context, svc *model.StatusService) error {
	return r.db.WithContext(ctx).Save(svc).Error
}

func (r *StatusServiceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.StatusService{}, id).Error
}

func (r *StatusServiceRepository) List(ctx context.Context) ([]model.StatusService, error) {
	var services []model.StatusService
	err := r.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&services).Error
	return services, err
}

// ListPublic returns all non-deleted services for the public status page.
func (r *StatusServiceRepository) ListPublic(ctx context.Context) ([]model.StatusService, error) {
	return r.List(ctx)
}
