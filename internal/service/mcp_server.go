package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// MCPServerService provides business logic for MCP servers.
type MCPServerService struct {
	repo   *repository.MCPServerRepository
	client *MCPClient
	logger *zap.Logger
}

// NewMCPServerService creates a new MCPServerService.
func NewMCPServerService(
	repo *repository.MCPServerRepository,
	logger *zap.Logger,
) *MCPServerService {
	return &MCPServerService{
		repo:   repo,
		client: NewMCPClient(),
		logger: logger,
	}
}

// Create creates a new MCP server.
func (s *MCPServerService) Create(ctx context.Context, srv *model.MCPServer) error {
	if err := srv.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}
	if err := s.repo.Create(ctx, srv); err != nil {
		s.logger.Error("failed to create MCP server", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an MCP server by its ID.
func (s *MCPServerService) GetByID(ctx context.Context, id uint) (*model.MCPServer, error) {
	srv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get MCP server", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrNotFound, err)
	}
	return srv, nil
}

// Update updates an existing MCP server.
func (s *MCPServerService) Update(ctx context.Context, existing *model.MCPServer, input *model.MCPServer) error {
	if err := input.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}

	input.ID = existing.ID
	input.CreatedAt = existing.CreatedAt

	if err := s.repo.Update(ctx, input); err != nil {
		s.logger.Error("failed to update MCP server", zap.Uint("id", existing.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes an MCP server by ID.
func (s *MCPServerService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete MCP server", zap.Uint("id", id), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// List returns a paginated list of MCP servers.
func (s *MCPServerService) List(ctx context.Context, page, pageSize int) ([]model.MCPServer, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list MCP servers", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// TestConnection attempts to connect to the MCP server's SSE endpoint.
func (s *MCPServerService) TestConnection(ctx context.Context, srv *model.MCPServer) error {
	return s.client.TestConnection(ctx, srv.URL, srv.GetHeadersMap())
}

// ListTools connects to the MCP server and enumerates available tools.
func (s *MCPServerService) ListTools(ctx context.Context, srv *model.MCPServer) ([]MCPTool, error) {
	tools, err := s.client.ListTools(ctx, srv.URL, srv.GetHeadersMap())
	if err != nil {
		s.logger.Error("failed to list MCP tools", zap.Uint("server_id", srv.ID), zap.Error(err))
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	return tools, nil
}
