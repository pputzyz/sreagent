package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type TeamNotifyChannelRepository struct {
	db *gorm.DB
}

func NewTeamNotifyChannelRepository(db *gorm.DB) *TeamNotifyChannelRepository {
	return &TeamNotifyChannelRepository{db: db}
}

func (r *TeamNotifyChannelRepository) Create(ctx context.Context, ch *model.TeamNotifyChannel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *TeamNotifyChannelRepository) GetByID(ctx context.Context, id uint) (*model.TeamNotifyChannel, error) {
	var ch model.TeamNotifyChannel
	if err := r.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *TeamNotifyChannelRepository) ListByTeam(ctx context.Context, teamID uint) ([]model.TeamNotifyChannel, error) {
	var channels []model.TeamNotifyChannel
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Order("is_default DESC, id ASC").Find(&channels).Error
	return channels, err
}

func (r *TeamNotifyChannelRepository) Update(ctx context.Context, ch *model.TeamNotifyChannel) error {
	return r.db.WithContext(ctx).Save(ch).Error
}

func (r *TeamNotifyChannelRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TeamNotifyChannel{}, id).Error
}

// ClearDefault clears is_default for all channels of a team.
func (r *TeamNotifyChannelRepository) ClearDefault(ctx context.Context, teamID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.TeamNotifyChannel{}).
		Where("team_id = ?", teamID).
		Update("is_default", false).Error
}
