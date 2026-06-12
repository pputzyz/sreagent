package repository

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// forwarderCache caches the result of ListEnabled to avoid repeated DB queries.
type forwarderCache struct {
	mu         sync.RWMutex
	forwarders []model.AlertForwarder
	cachedAt   time.Time
	ttl        time.Duration
}

func (c *forwarderCache) Get() ([]model.AlertForwarder, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.forwarders == nil || time.Since(c.cachedAt) > c.ttl {
		return nil, false
	}
	out := make([]model.AlertForwarder, len(c.forwarders))
	copy(out, c.forwarders)
	return out, true
}

func (c *forwarderCache) Set(forwarders []model.AlertForwarder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forwarders = forwarders
	c.cachedAt = time.Now()
}

func (c *forwarderCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forwarders = nil
	c.cachedAt = time.Time{}
}

// AlertForwarderRepository handles alert_forwarders persistence.
type AlertForwarderRepository struct {
	db    *gorm.DB
	cache forwarderCache
}

// NewAlertForwarderRepository creates a new AlertForwarderRepository.
func NewAlertForwarderRepository(db *gorm.DB) *AlertForwarderRepository {
	return &AlertForwarderRepository{
		db: db,
		cache: forwarderCache{
			ttl: 30 * time.Second,
		},
	}
}

// Create creates a new alert forwarder.
func (r *AlertForwarderRepository) Create(ctx context.Context, forwarder *model.AlertForwarder) error {
	if err := r.db.WithContext(ctx).Create(forwarder).Error; err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// GetByID returns an alert forwarder by its ID.
func (r *AlertForwarderRepository) GetByID(ctx context.Context, id uint) (*model.AlertForwarder, error) {
	var forwarder model.AlertForwarder
	err := r.db.WithContext(ctx).First(&forwarder, id).Error
	if err != nil {
		return nil, err
	}
	return &forwarder, nil
}

// GetByName returns an alert forwarder by its name.
func (r *AlertForwarderRepository) GetByName(ctx context.Context, name string) (*model.AlertForwarder, error) {
	var forwarder model.AlertForwarder
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&forwarder).Error
	if err != nil {
		return nil, err
	}
	return &forwarder, nil
}

// List returns a paginated list of alert forwarders.
func (r *AlertForwarderRepository) List(ctx context.Context, page, pageSize int, direction string, enabled *bool) ([]model.AlertForwarder, int64, error) {
	var list []model.AlertForwarder
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AlertForwarder{})

	if direction != "" {
		query = query.Where("direction = ?", direction)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("priority DESC, id ASC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Update updates an existing alert forwarder.
func (r *AlertForwarderRepository) Update(ctx context.Context, forwarder *model.AlertForwarder) error {
	if err := r.db.WithContext(ctx).Save(forwarder).Error; err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// Delete deletes an alert forwarder by ID.
func (r *AlertForwarderRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.AlertForwarder{}, id).Error; err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// ListEnabled returns all enabled alert forwarders. Results are cached in memory
// for 30 seconds to avoid repeated DB queries during alert processing.
func (r *AlertForwarderRepository) ListEnabled(ctx context.Context) ([]model.AlertForwarder, error) {
	if cached, ok := r.cache.Get(); ok {
		return cached, nil
	}
	var list []model.AlertForwarder
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("priority DESC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	r.cache.Set(list)
	return list, nil
}

// ListEnabledByDirection returns all enabled alert forwarders for a specific direction.
func (r *AlertForwarderRepository) ListEnabledByDirection(ctx context.Context, direction model.ForwarderDirection) ([]model.AlertForwarder, error) {
	allEnabled, err := r.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []model.AlertForwarder
	for _, f := range allEnabled {
		if f.Direction == direction || f.Direction == model.ForwarderDirectionBidirectional {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

// BatchUpdateEnabled sets enabled for all forwarders whose IDs are in ids.
func (r *AlertForwarderRepository) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.AlertForwarder{}).
			Where("id IN ?", ids).
			Update("enabled", enabled).Error
	})
	if err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// BatchDelete deletes all forwarders whose IDs are in ids.
func (r *AlertForwarderRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Where("id IN ?", ids).Delete(&model.AlertForwarder{}).Error
	})
	if err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// CountByDirection counts forwarders by direction.
func (r *AlertForwarderRepository) CountByDirection(ctx context.Context) (map[string]int64, error) {
	type result struct {
		Direction string
		Count     int64
	}
	var results []result
	err := r.db.WithContext(ctx).Model(&model.AlertForwarder{}).
		Select("direction, count(*) as count").
		Group("direction").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Direction] = r.Count
	}
	return counts, nil
}
