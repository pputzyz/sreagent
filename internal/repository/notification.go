package repository

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// NotifyChannelRepository handles notify_channels persistence.
type NotifyChannelRepository struct {
	db *gorm.DB
}

func NewNotifyChannelRepository(db *gorm.DB) *NotifyChannelRepository {
	return &NotifyChannelRepository{db: db}
}

func (r *NotifyChannelRepository) GetByID(ctx context.Context, id uint) (*model.NotifyChannel, error) {
	var channel model.NotifyChannel
	err := r.db.WithContext(ctx).First(&channel, id).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// ListByLabels returns all enabled channels whose labels are a subset match
// of the provided labels (channel labels must all be present in the given labels).
func (r *NotifyChannelRepository) ListByLabels(ctx context.Context, labels map[string]string) ([]model.NotifyChannel, error) {
	var allChannels []model.NotifyChannel
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Find(&allChannels).Error
	if err != nil {
		return nil, err
	}

	var matched []model.NotifyChannel
	for _, ch := range allChannels {
		if labelmatch.Match(labels, ch.Labels) {
			matched = append(matched, ch)
		}
	}
	return matched, nil
}

// NotifyRecordRepository handles notify_records persistence.
type NotifyRecordRepository struct {
	db *gorm.DB
}

func NewNotifyRecordRepository(db *gorm.DB) *NotifyRecordRepository {
	return &NotifyRecordRepository{db: db}
}

func (r *NotifyRecordRepository) Create(ctx context.Context, record *model.NotifyRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *NotifyRecordRepository) ListByEventID(ctx context.Context, eventID uint) ([]model.NotifyRecord, error) {
	var list []model.NotifyRecord
	err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

// GetLastSentRecord returns the most recent successfully sent notification record
// for a given channel and policy combination.
func (r *NotifyRecordRepository) GetLastSentRecord(ctx context.Context, channelID, policyID uint) (*model.NotifyRecord, error) {
	var record model.NotifyRecord
	err := r.db.WithContext(ctx).
		Where("channel_id = ? AND policy_id = ? AND status = ?", channelID, policyID, "sent").
		Order("created_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// severityMatches checks if the given severity is contained in the comma-separated severities string.
func severityMatches(severities, severity string) bool {
	// Parse the comma-separated list
	var list []string
	// Handle as simple comma-separated string
	for _, s := range splitCSV(severities) {
		list = append(list, s)
	}
	for _, s := range list {
		if s == severity {
			return true
		}
	}
	return false
}

// splitCSV splits a comma-separated string and trims spaces.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	// Try JSON array first
	var jsonArr []string
	if err := json.Unmarshal([]byte(s), &jsonArr); err == nil {
		return jsonArr
	}
	// Fall back to comma-separated
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			val := trimSpace(s[start:i])
			if val != "" {
				result = append(result, val)
			}
			start = i + 1
		}
	}
	val := trimSpace(s[start:])
	if val != "" {
		result = append(result, val)
	}
	return result
}

// trimSpace trims leading and trailing whitespace from a string.
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
