package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// PostMortemRepository handles CRUD for incident post-mortems (故障复盘).
type PostMortemRepository struct {
	db *gorm.DB
}

func NewPostMortemRepository(db *gorm.DB) *PostMortemRepository {
	return &PostMortemRepository{db: db}
}

func (r *PostMortemRepository) Create(ctx context.Context, pm *model.PostMortem) error {
	return r.db.WithContext(ctx).Create(pm).Error
}

func (r *PostMortemRepository) GetByIncidentID(ctx context.Context, incidentID uint) (*model.PostMortem, error) {
	var pm model.PostMortem
	err := r.db.WithContext(ctx).
		Preload("Incident").
		Preload("Author").
		Where("incident_id = ?", incidentID).
		First(&pm).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (r *PostMortemRepository) GetByID(ctx context.Context, id uint) (*model.PostMortem, error) {
	var pm model.PostMortem
	err := r.db.WithContext(ctx).
		Preload("Incident").
		Preload("Author").
		First(&pm, id).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (r *PostMortemRepository) Update(ctx context.Context, pm *model.PostMortem) error {
	return r.db.WithContext(ctx).Save(pm).Error
}

func (r *PostMortemRepository) List(ctx context.Context, channelID uint, status string, page, pageSize int) ([]model.PostMortem, int64, error) {
	var list []model.PostMortem
	var total int64

	q := r.db.WithContext(ctx).Model(&model.PostMortem{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	// Filter by channel via incident join
	if channelID > 0 {
		q = q.Joins("JOIN incidents ON incidents.id = post_mortems.incident_id AND incidents.deleted_at IS NULL").
			Where("incidents.channel_id = ?", channelID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := q.Preload("Incident").Preload("Author").
		Offset(offset).Limit(pageSize).Order("post_mortems.id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *PostMortemRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.PostMortem{}, id).Error
}
