package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type AutoActionRepository struct {
	db *gorm.DB
}

func NewAutoActionRepository(db *gorm.DB) *AutoActionRepository {
	return &AutoActionRepository{db: db}
}

func (r *AutoActionRepository) Create(ctx context.Context, action *model.AutoAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

func (r *AutoActionRepository) GetByID(ctx context.Context, id uint) (*model.AutoAction, error) {
	var action model.AutoAction
	err := r.db.WithContext(ctx).First(&action, id).Error
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (r *AutoActionRepository) List(ctx context.Context, level string, enabled *bool, page, pageSize int) ([]model.AutoAction, int64, error) {
	var list []model.AutoAction
	var total int64

	q := r.db.WithContext(ctx)
	if level != "" {
		q = q.Where("level = ?", level)
	}
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}

	if err := q.Model(&model.AutoAction{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *AutoActionRepository) Update(ctx context.Context, action *model.AutoAction) error {
	return r.db.WithContext(ctx).Save(action).Error
}

func (r *AutoActionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AutoAction{}, id).Error
}

// FindMatching finds enabled auto actions whose trigger_labels match the given labels.
func (r *AutoActionRepository) FindMatching(ctx context.Context, labels map[string]string, severity string) ([]model.AutoAction, error) {
	var actions []model.AutoAction
	q := r.db.WithContext(ctx).Where("enabled = ?", true)
	if severity != "" {
		q = q.Where("trigger_severity = ? OR trigger_severity IS NULL", severity)
	}
	if err := q.Find(&actions).Error; err != nil {
		return nil, err
	}

	var matched []model.AutoAction
	for _, a := range actions {
		if len(a.TriggerLabels) == 0 {
			matched = append(matched, a)
			continue
		}
		allMatch := true
		for k, v := range a.TriggerLabels {
			if lv, ok := labels[k]; !ok || lv != v {
				allMatch = false
				break
			}
		}
		if allMatch {
			matched = append(matched, a)
		}
	}

	return matched, nil
}

// UpdateConfidence atomically increments success or failure count.
func (r *AutoActionRepository) UpdateConfidence(ctx context.Context, id uint, success bool) error {
	if success {
		return r.db.WithContext(ctx).Model(&model.AutoAction{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"success_count": gorm.Expr("success_count + 1"),
				"confidence":    gorm.Expr("LEAST(100, confidence + 5)"),
				"last_run_at":   time.Now(),
			}).Error
	}
	return r.db.WithContext(ctx).Model(&model.AutoAction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failure_count": gorm.Expr("failure_count + 1"),
			"confidence":    gorm.Expr("GREATEST(0, confidence - 10)"),
			"last_run_at":   time.Now(),
		}).Error
}

// --- Action Logs ---

func (r *AutoActionRepository) CreateLog(ctx context.Context, log *model.AutoActionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AutoActionRepository) UpdateLog(ctx context.Context, log *model.AutoActionLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *AutoActionRepository) ListLogs(ctx context.Context, actionID uint, page, pageSize int) ([]model.AutoActionLog, int64, error) {
	var list []model.AutoActionLog
	var total int64

	q := r.db.WithContext(ctx).Where("action_id = ?", actionID)
	if err := q.Model(&model.AutoActionLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
