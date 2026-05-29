package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// AuditLogFilter holds optional filter criteria for listing audit logs.
// Defined in service layer so handlers don't need to import repository.
type AuditLogFilter = repository.AuditLogFilter

// maxAsyncAuditLogs caps concurrent async audit-log writes.
const maxAsyncAuditLogs = 50

// AuditLogService records and queries operational audit logs.
type AuditLogService struct {
	repo        *repository.AuditLogRepository
	logger      *zap.Logger
	dispatchSem chan struct{}
}

func NewAuditLogService(repo *repository.AuditLogRepository, logger *zap.Logger) *AuditLogService {
	return &AuditLogService{
		repo:        repo,
		logger:      logger,
		dispatchSem: make(chan struct{}, maxAsyncAuditLogs),
	}
}

// Record persists an audit log entry asynchronously so it never blocks the request path.
// The entry's UserID/Username/Action/ResourceType must be set by the caller.
func (s *AuditLogService) Record(entry *model.AuditLog) {
	select {
	case s.dispatchSem <- struct{}{}:
		go func() {
			defer func() { <-s.dispatchSem }()
			if err := s.repo.Create(context.Background(), entry); err != nil {
				s.logger.Warn("failed to write audit log",
					zap.String("action", entry.Action),
					zap.String("resource", entry.ResourceType),
					zap.Error(err),
				)
			}
		}()
	default:
		s.logger.Warn("dropping async audit log, too many in flight")
	}
}

// List returns a paginated list of audit log entries.
func (s *AuditLogService) List(ctx context.Context, f repository.AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error) {
	return s.repo.List(ctx, f, page, pageSize)
}
