package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// AnnotationRepository handles annotations persistence.
type AnnotationRepository struct {
	db *gorm.DB
}

// NewAnnotationRepository creates a new AnnotationRepository.
func NewAnnotationRepository(db *gorm.DB) *AnnotationRepository {
	return &AnnotationRepository{db: db}
}

// Create creates a new annotation.
func (r *AnnotationRepository) Create(ctx context.Context, annotation *model.Annotation) error {
	return r.db.WithContext(ctx).Create(annotation).Error
}

// GetByID returns an annotation by its ID.
func (r *AnnotationRepository) GetByID(ctx context.Context, id uint) (*model.Annotation, error) {
	var ann model.Annotation
	err := r.db.WithContext(ctx).First(&ann, id).Error
	if err != nil {
		return nil, err
	}
	return &ann, nil
}

// Update updates an existing annotation.
func (r *AnnotationRepository) Update(ctx context.Context, annotation *model.Annotation) error {
	return r.db.WithContext(ctx).Save(annotation).Error
}

// Delete soft-deletes an annotation by ID.
func (r *AnnotationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Annotation{}, id).Error
}

// ListByDashboard returns annotations for a specific dashboard within a time range.
// If from/to are zero values, no time filter is applied.
func (r *AnnotationRepository) ListByDashboard(ctx context.Context, dashboardID uint, from, to time.Time) ([]model.Annotation, error) {
	var list []model.Annotation
	query := r.db.WithContext(ctx).Where("dashboard_id = ?", dashboardID)
	if !from.IsZero() {
		query = query.Where("time >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("time <= ?", to)
	}
	err := query.Order("time ASC").Find(&list).Error
	return list, err
}

// BatchCreate inserts multiple annotations in a single transaction.
func (r *AnnotationRepository) BatchCreate(ctx context.Context, annotations []model.Annotation) error {
	if len(annotations) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&annotations).Error
}
