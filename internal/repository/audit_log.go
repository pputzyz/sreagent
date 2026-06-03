package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// AuditLogRepository handles persistence for audit logs.
type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create inserts a new audit log record.
func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// AuditLogFilter holds optional filter criteria for listing audit logs.
type AuditLogFilter struct {
	UserID       *uint
	Username     string // P1-26: filter by username
	Action       string
	ResourceType string
	Keyword      string    // P1-26: free-text search across resource_name, action
	StartTime    *time.Time
	EndTime      *time.Time
}

// List returns a paginated list of audit logs with optional filters.
func (r *AuditLogRepository) List(ctx context.Context, f AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AuditLog{})

	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.Username != "" {
		// P1-26: Filter by username (join with users table)
		q = q.Joins("LEFT JOIN users ON users.id = audit_logs.user_id").
			Where("users.username = ?", f.Username)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}
	if f.Keyword != "" {
		// P1-26: Free-text search across resource_name and action
		like := "%" + f.Keyword + "%"
		q = q.Where("(resource_name LIKE ? OR action LIKE ?)", like, like)
	}
	if f.StartTime != nil {
		q = q.Where("created_at >= ?", *f.StartTime)
	}
	if f.EndTime != nil {
		q = q.Where("created_at <= ?", *f.EndTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.AuditLog
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
