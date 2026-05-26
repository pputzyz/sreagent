package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// MCPServerRepository handles MCP server persistence.
type MCPServerRepository struct {
	db *gorm.DB
}

// NewMCPServerRepository creates a new MCPServerRepository.
func NewMCPServerRepository(db *gorm.DB) *MCPServerRepository {
	return &MCPServerRepository{db: db}
}

// Create creates a new MCP server.
func (r *MCPServerRepository) Create(ctx context.Context, s *model.MCPServer) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetByID returns an MCP server by its ID.
func (r *MCPServerRepository) GetByID(ctx context.Context, id uint) (*model.MCPServer, error) {
	var s model.MCPServer
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// Update updates an existing MCP server.
func (r *MCPServerRepository) Update(ctx context.Context, s *model.MCPServer) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// Delete soft-deletes an MCP server by ID.
func (r *MCPServerRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.MCPServer{}, id).Error
}

// List returns all MCP servers with pagination.
func (r *MCPServerRepository) List(ctx context.Context, page, pageSize int) ([]model.MCPServer, int64, error) {
	var list []model.MCPServer
	var total int64

	query := r.db.WithContext(ctx).Model(&model.MCPServer{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// ListEnabled returns all enabled MCP servers.
func (r *MCPServerRepository) ListEnabled(ctx context.Context) ([]model.MCPServer, error) {
	var list []model.MCPServer
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
