package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ChannelRepository handles CRUD for collaboration channels (协作空间).
type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelV2Repository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) Create(ctx context.Context, ch *model.Channel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *ChannelRepository) GetByID(ctx context.Context, id uint) (*model.Channel, error) {
	var ch model.Channel
	err := r.db.WithContext(ctx).Preload("Team").First(&ch, id).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *ChannelRepository) GetByName(ctx context.Context, name string) (*model.Channel, error) {
	var ch model.Channel
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&ch).Error
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// List returns paginated channels with optional filters.
// query: fuzzy search on name/description.
// status: filter by channel status (active/disabled).
func (r *ChannelRepository) List(ctx context.Context, query, status string, page, pageSize int) ([]model.Channel, int64, error) {
	var list []model.Channel
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Channel{})
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Preload("Team").Offset(offset).Limit(pageSize).Order("sort_order ASC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *ChannelRepository) Update(ctx context.Context, ch *model.Channel) error {
	return r.db.WithContext(ctx).Save(ch).Error
}

func (r *ChannelRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Channel{}, id).Error
}

// ListActive returns all active channels (for internal use / dropdown).
func (r *ChannelRepository) ListActive(ctx context.Context) ([]model.Channel, error) {
	var list []model.Channel
	err := r.db.WithContext(ctx).Where("status = ?", model.ChannelStatusActive).
		Order("sort_order ASC, id DESC").Find(&list).Error
	return list, err
}

// --- ChannelStar ---

func (r *ChannelRepository) Star(ctx context.Context, userID, channelID uint) error {
	star := model.ChannelStar{UserID: userID, ChannelID: channelID}
	return r.db.WithContext(ctx).Create(&star).Error
}

func (r *ChannelRepository) Unstar(ctx context.Context, userID, channelID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		Delete(&model.ChannelStar{}).Error
}

func (r *ChannelRepository) ListStarred(ctx context.Context, userID uint) ([]uint, error) {
	var stars []model.ChannelStar
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&stars).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(stars))
	for i, s := range stars {
		ids[i] = s.ChannelID
	}
	return ids, nil
}

// IncrementActiveIncidentCount atomically increments the counter.
func (r *ChannelRepository) IncrementActiveIncidentCount(ctx context.Context, channelID uint, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Channel{}).
		Where("id = ?", channelID).
		UpdateColumn("active_incident_count", gorm.Expr("GREATEST(active_incident_count + ?, 0)", delta)).
		Error
}
