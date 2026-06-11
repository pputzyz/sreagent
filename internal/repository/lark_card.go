package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// LarkCardRepository manages lark_card_entities and lark_card_messages.
type LarkCardRepository struct {
	db *gorm.DB
}

// NewLarkCardRepository creates a new LarkCardRepository.
func NewLarkCardRepository(db *gorm.DB) *LarkCardRepository {
	return &LarkCardRepository{db: db}
}

// CreateEntity inserts a new card entity record.
func (r *LarkCardRepository) CreateEntity(ctx context.Context, entity *model.LarkCardEntity) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// GetEntityByEventID returns the active card entity for a given alert event.
func (r *LarkCardRepository) GetEntityByEventID(ctx context.Context, eventID uint) (*model.LarkCardEntity, error) {
	var entity model.LarkCardEntity
	err := r.db.WithContext(ctx).
		Where("event_id = ? AND card_status = ?", eventID, "active").
		Order("id DESC").
		First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// GetEntityByID returns a card entity by its primary key.
func (r *LarkCardRepository) GetEntityByID(ctx context.Context, id uint) (*model.LarkCardEntity, error) {
	var entity model.LarkCardEntity
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// IncrementSequence atomically increments the sequence counter and returns the new value.
// This avoids read-modify-write races: UPDATE ... SET sequence = sequence + 1.
func (r *LarkCardRepository) IncrementSequence(ctx context.Context, id uint) (int64, error) {
	err := r.db.WithContext(ctx).
		Model(&model.LarkCardEntity{}).
		Where("id = ?", id).
		Update("sequence", gorm.Expr("sequence + 1")).Error
	if err != nil {
		return 0, err
	}
	var entity model.LarkCardEntity
	if err := r.db.WithContext(ctx).Select("sequence").First(&entity, id).Error; err != nil {
		return 0, err
	}
	return entity.Sequence, nil
}

// UpdateCardID sets the CardKit card_id on an entity (used after CreateCardEntity API call).
func (r *LarkCardRepository) UpdateCardID(ctx context.Context, id uint, cardID string) error {
	return r.db.WithContext(ctx).
		Model(&model.LarkCardEntity{}).
		Where("id = ?", id).
		Update("card_id", cardID).Error
}

// UpdateStatus updates the card_status field.
func (r *LarkCardRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.LarkCardEntity{}).
		Where("id = ?", id).
		Update("card_status", status).Error
}

// ExpireOldCards marks all active cards past their expires_at as expired.
// Returns the number of cards affected.
func (r *LarkCardRepository) ExpireOldCards(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.LarkCardEntity{}).
		Where("card_status = ? AND expires_at < ?", "active", time.Now()).
		Update("card_status", "expired")
	return result.RowsAffected, result.Error
}

// CreateMessage inserts a card-to-chat delivery record.
func (r *LarkCardRepository) CreateMessage(ctx context.Context, msg *model.LarkCardMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// GetMessagesByEntityID returns all chat deliveries for a card entity.
func (r *LarkCardRepository) GetMessagesByEntityID(ctx context.Context, entityID uint) ([]model.LarkCardMessage, error) {
	var msgs []model.LarkCardMessage
	err := r.db.WithContext(ctx).
		Where("card_entity_id = ?", entityID).
		Find(&msgs).Error
	return msgs, err
}
