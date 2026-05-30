package service

import (
	"context"
	"errors"
	"sort"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// BizGroupService provides CRUD and tree operations for business groups.
type BizGroupService struct {
	repo   *repository.BizGroupRepository
	logger *zap.Logger
}

// NewBizGroupService creates a new BizGroupService.
func NewBizGroupService(
	repo *repository.BizGroupRepository,
	logger *zap.Logger,
) *BizGroupService {
	return &BizGroupService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new business group.
func (s *BizGroupService) Create(ctx context.Context, group *model.BizGroup) error {
	// Validate parent exists if specified
	if group.ParentID != nil {
		if _, err := s.repo.GetByID(ctx, *group.ParentID); err != nil {
			return apperr.WithMessage(apperr.ErrBizGroupNotFound, "parent group not found")
		}
	}

	if err := s.repo.Create(ctx, group); err != nil {
		s.logger.Error("failed to create biz group", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a business group by its ID.
func (s *BizGroupService) GetByID(ctx context.Context, id uint) (*model.BizGroup, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrBizGroupNotFound
	}
	return group, nil
}

// List returns a paginated list of business groups.
func (s *BizGroupService) List(ctx context.Context, page, pageSize int) ([]model.BizGroup, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list biz groups", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// BizGroupTreeNode represents a node in the business group tree.
type BizGroupTreeNode struct {
	model.BizGroup
	Children []*BizGroupTreeNode `json:"children,omitempty"`
}

// ListTree returns all business groups organized as a tree.
func (s *BizGroupService) ListTree(ctx context.Context) ([]*BizGroupTreeNode, error) {
	groups, err := s.repo.ListTree(ctx)
	if err != nil {
		s.logger.Error("failed to list biz groups for tree", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Build tree structure
	nodeMap := make(map[uint]*BizGroupTreeNode)
	var roots []*BizGroupTreeNode

	// First pass: create all nodes
	for _, g := range groups {
		node := &BizGroupTreeNode{BizGroup: g}
		nodeMap[g.ID] = node
	}

	// Second pass: build parent-child relationships
	for _, g := range groups {
		node := nodeMap[g.ID]
		if g.ParentID != nil {
			if parent, ok := nodeMap[*g.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not found, treat as root
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	return roots, nil
}

// wouldCreateCycle walks up the ancestor chain from newParentID and returns true
// if groupID is found among the ancestors (which would create a circular reference).
func (s *BizGroupService) wouldCreateCycle(ctx context.Context, groupID, newParentID uint) bool {
	visited := make(map[uint]bool)
	current := newParentID
	for current != 0 {
		if current == groupID {
			return true
		}
		if visited[current] {
			// Existing cycle in data (shouldn't happen, but avoid infinite loop)
			break
		}
		visited[current] = true

		parent, err := s.repo.GetByIDLight(ctx, current)
		if err != nil {
			break
		}
		if parent.ParentID == nil {
			break
		}
		current = *parent.ParentID
	}
	return false
}

// Update updates an existing business group.
func (s *BizGroupService) Update(ctx context.Context, group *model.BizGroup) error {
	existing, err := s.repo.GetByID(ctx, group.ID)
	if err != nil {
		return apperr.ErrBizGroupNotFound
	}

	// Validate parent exists if changed
	if group.ParentID != nil {
		// Prevent setting self as parent
		if *group.ParentID == group.ID {
			return apperr.WithMessage(apperr.ErrBadRequest, "cannot set group as its own parent")
		}
		if _, err := s.repo.GetByIDLight(ctx, *group.ParentID); err != nil {
			return apperr.WithMessage(apperr.ErrBizGroupNotFound, "parent group not found")
		}
		// Prevent circular parent references
		if s.wouldCreateCycle(ctx, group.ID, *group.ParentID) {
			return apperr.WithMessage(apperr.ErrBadRequest, "setting this parent would create a circular reference")
		}
	}

	existing.Name = group.Name
	existing.Description = group.Description
	existing.ParentID = group.ParentID
	existing.Labels = group.Labels
	existing.MatchLabels = group.MatchLabels

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update biz group", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a business group by ID.
func (s *BizGroupService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrBizGroupNotFound
	}

	// Prevent deleting a group that has children — re-parent or delete children first.
	children, err := s.repo.ListByParentID(ctx, id)
	if err != nil {
		s.logger.Error("failed to check biz group children", zap.Error(err), zap.Uint("group_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if len(children) > 0 {
		return apperr.WithMessage(apperr.ErrBadRequest, "cannot delete group with children; re-parent or delete children first")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete biz group", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// FindMatchingGroups returns all biz groups whose MatchLabels match the given alert labels.
// Multiple groups can match (e.g., a "ts" group and a parent "trading" group).
// Results are sorted by specificity: more matchers = more specific = first.
func (s *BizGroupService) FindMatchingGroups(ctx context.Context, alertLabels map[string]string) ([]model.BizGroup, error) {
	groups, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	var matches []model.BizGroup
	for _, g := range groups {
		if len(g.MatchLabels) == 0 {
			continue
		}
		if labelmatch.Match(alertLabels, map[string]string(g.MatchLabels)) {
			matches = append(matches, g)
		}
	}
	// Sort: more specific (more matchers) first
	sort.Slice(matches, func(i, j int) bool {
		return len(matches[i].MatchLabels) > len(matches[j].MatchLabels)
	})
	return matches, nil
}

// AddMember adds a user to a business group with the specified role.
func (s *BizGroupService) AddMember(ctx context.Context, groupID, userID uint, role string) error {
	if _, err := s.repo.GetByID(ctx, groupID); err != nil {
		return apperr.ErrBizGroupNotFound
	}

	if role == "" {
		role = "member"
	}

	// Check if user is already a member
	existing, err := s.repo.GetMember(ctx, groupID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to check biz group member", zap.Error(err), zap.Uint("group_id", groupID), zap.Uint("user_id", userID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if existing != nil {
		return apperr.WithMessage(apperr.ErrConflict, "user is already a member of this group")
	}

	if err := s.repo.AddMember(ctx, groupID, userID, role); err != nil {
		s.logger.Error("failed to add biz group member",
			zap.Uint("group_id", groupID),
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("biz group member added",
		zap.Uint("group_id", groupID),
		zap.Uint("user_id", userID),
		zap.String("role", role),
	)
	return nil
}

// RemoveMember removes a user from a business group.
func (s *BizGroupService) RemoveMember(ctx context.Context, groupID, userID uint) error {
	if _, err := s.repo.GetByID(ctx, groupID); err != nil {
		return apperr.ErrBizGroupNotFound
	}

	if _, err := s.repo.GetMember(ctx, groupID, userID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "user is not a member of this group")
	}

	if err := s.repo.RemoveMember(ctx, groupID, userID); err != nil {
		s.logger.Error("failed to remove biz group member",
			zap.Uint("group_id", groupID),
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("biz group member removed",
		zap.Uint("group_id", groupID),
		zap.Uint("user_id", userID),
	)
	return nil
}

// ListMembers returns all members of a business group.
func (s *BizGroupService) ListMembers(ctx context.Context, groupID uint) ([]model.BizGroupMember, error) {
	if _, err := s.repo.GetByID(ctx, groupID); err != nil {
		return nil, apperr.ErrBizGroupNotFound
	}

	members, err := s.repo.ListMembers(ctx, groupID)
	if err != nil {
		s.logger.Error("failed to list biz group members",
			zap.Uint("group_id", groupID),
			zap.Error(err),
		)
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	return members, nil
}
