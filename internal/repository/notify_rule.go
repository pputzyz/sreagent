package repository

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// notifyRuleCache caches the result of ListEnabled to avoid repeated DB queries.
type notifyRuleCache struct {
	mu       sync.RWMutex
	rules    []model.NotifyRule
	cachedAt time.Time
	ttl      time.Duration
}

func (c *notifyRuleCache) Get() ([]model.NotifyRule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.rules == nil || time.Since(c.cachedAt) > c.ttl {
		return nil, false
	}
	// Return a copy to prevent callers from mutating cached data
	out := make([]model.NotifyRule, len(c.rules))
	copy(out, c.rules)
	return out, true
}

func (c *notifyRuleCache) Set(rules []model.NotifyRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = rules
	c.cachedAt = time.Now()
}

func (c *notifyRuleCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = nil
	c.cachedAt = time.Time{}
}

// NotifyRuleRepository handles notify_rules persistence.
type NotifyRuleRepository struct {
	db    *gorm.DB
	cache notifyRuleCache
}

// NewNotifyRuleRepository creates a new NotifyRuleRepository.
func NewNotifyRuleRepository(db *gorm.DB) *NotifyRuleRepository {
	return &NotifyRuleRepository{
		db: db,
		cache: notifyRuleCache{
			ttl: 30 * time.Second,
		},
	}
}

// Create creates a new notify rule.
func (r *NotifyRuleRepository) Create(ctx context.Context, rule *model.NotifyRule) error {
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// GetByID returns a notify rule by its ID.
func (r *NotifyRuleRepository) GetByID(ctx context.Context, id uint) (*model.NotifyRule, error) {
	var rule model.NotifyRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List returns a paginated list of notify rules.
func (r *NotifyRuleRepository) List(ctx context.Context, page, pageSize int) ([]model.NotifyRule, int64, error) {
	var list []model.NotifyRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.NotifyRule{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Update updates an existing notify rule.
func (r *NotifyRuleRepository) Update(ctx context.Context, rule *model.NotifyRule) error {
	if err := r.db.WithContext(ctx).Save(rule).Error; err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// Delete soft-deletes a notify rule by ID.
func (r *NotifyRuleRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.NotifyRule{}, id).Error; err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// ListEnabled returns all enabled notify rules. Results are cached in memory
// for 30 seconds to avoid repeated DB queries during alert processing.
func (r *NotifyRuleRepository) ListEnabled(ctx context.Context) ([]model.NotifyRule, error) {
	if cached, ok := r.cache.Get(); ok {
		return cached, nil
	}
	var list []model.NotifyRule
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	r.cache.Set(list)
	return list, nil
}

// BatchUpdateEnabled sets is_enabled for all rules whose IDs are in ids.
func (r *NotifyRuleRepository) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.NotifyRule{}).
			Where("id IN ?", ids).
			Update("is_enabled", enabled).Error
	})
	if err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// BatchDelete soft-deletes all rules whose IDs are in ids.
func (r *NotifyRuleRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Where("id IN ?", ids).Delete(&model.NotifyRule{}).Error
	})
	if err != nil {
		return err
	}
	r.cache.Invalidate()
	return nil
}

// FindMatchingRules returns all enabled notify rules whose match_labels are a subset
// of the given event labels, and whose severity filter matches (or is empty for all).
// dataSourceID filters rules by datasource (nil pattern = wildcard, matches any).
func (r *NotifyRuleRepository) FindMatchingRules(ctx context.Context, labels map[string]string, severity string, dataSourceID *uint) ([]model.NotifyRule, error) {
	allRules, err := r.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	var matched []model.NotifyRule
	for _, rule := range allRules {
		// Check label + datasource matching
		if !labelmatch.MatchWithSourceID(labels, dataSourceID, map[string]string(rule.MatchLabels), rule.DataSourceID) {
			continue
		}

		// Check severity filter
		if rule.Severities != "" {
			if !severityMatches(rule.Severities, severity) {
				continue
			}
		}

		matched = append(matched, rule)
	}

	return matched, nil
}
