package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
)

// TeamRepository defines the interface for team data access.
type TeamRepository interface {
	Create(ctx context.Context, team *model.Team) error
	GetByID(ctx context.Context, id uint) (*model.Team, error)
	GetByName(ctx context.Context, name string) (*model.Team, error)
	List(ctx context.Context, page, pageSize int) ([]model.Team, int64, error)
	Update(ctx context.Context, team *model.Team) error
	Delete(ctx context.Context, id uint) error
	AddMember(ctx context.Context, member *model.TeamMember) error
	RemoveMember(ctx context.Context, teamID, userID uint) error
	ListMembers(ctx context.Context, teamID uint) ([]model.TeamMember, error)
	GetMember(ctx context.Context, teamID, userID uint) (*model.TeamMember, error)
	UpdateMember(ctx context.Context, member *model.TeamMember) error
	ListByUser(ctx context.Context, userID uint) ([]model.TeamMember, error)
}

// TeamService provides team management operations.
type TeamService struct {
	repo   TeamRepository
	logger *zap.Logger
}

func NewTeamService(repo TeamRepository, logger *zap.Logger) *TeamService {
	return &TeamService{repo: repo, logger: logger}
}

// Create creates a new team.
func (s *TeamService) Create(ctx context.Context, team *model.Team) error {
	// Check if team name already exists
	existing, err := s.repo.GetByName(ctx, team.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to check team name", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if existing != nil {
		return apperr.WithMessage(apperr.ErrDuplicateName, fmt.Sprintf("team '%s' already exists", team.Name))
	}

	if err := s.repo.Create(ctx, team); err != nil {
		s.logger.Error("failed to create team", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

// GetByID retrieves a team by its ID.
func (s *TeamService) GetByID(ctx context.Context, id uint) (*model.Team, error) {
	team, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "team not found")
	}
	return team, nil
}

// List returns a paginated list of teams.
func (s *TeamService) List(ctx context.Context, page, pageSize int) ([]model.Team, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list teams", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates a team's information.
func (s *TeamService) Update(ctx context.Context, team *model.Team) error {
	existing, err := s.repo.GetByID(ctx, team.ID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "team not found")
	}

	existing.Name = team.Name
	existing.Description = team.Description
	existing.Labels = team.Labels

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update team", zap.Error(err), zap.Uint("team_id", team.ID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

// Delete deletes a team by its ID.
func (s *TeamService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "team not found")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete team", zap.Error(err), zap.Uint("team_id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

// AddMember adds a user to a team with the given role.
func (s *TeamService) AddMember(ctx context.Context, teamID, userID uint, role string) error {
	// Validate role
	validRoles := map[string]bool{"team_lead": true, "member": true}
	if role != "" && !validRoles[role] {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid team role: must be one of team_lead, member")
	}

	// Check team exists
	if _, err := s.repo.GetByID(ctx, teamID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "team not found")
	}

	// Check if user is already a member — idempotent: return success if so
	existing, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to check team member", zap.Error(err), zap.Uint("team_id", teamID), zap.Uint("user_id", userID))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if existing != nil {
		// Already a member — idempotent no-op, update role if changed
		if existing.Role != role && role != "" {
			existing.Role = role
			if err := s.repo.UpdateMember(ctx, existing); err != nil {
				s.logger.Error("failed to update team member role", zap.Error(err), zap.Uint("team_id", teamID), zap.Uint("user_id", userID))
				return apperr.Wrap(apperr.ErrDatabase, err)
			}
		}
		return nil
	}

	if role == "" {
		role = "member"
	}

	member := &model.TeamMember{
		TeamID: teamID,
		UserID: userID,
		Role:   role,
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		s.logger.Error("failed to add team member",
			zap.Error(err),
			zap.Uint("team_id", teamID),
			zap.Uint("user_id", userID),
		)
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("team member added",
		zap.Uint("team_id", teamID),
		zap.Uint("user_id", userID),
		zap.String("role", role),
	)
	return nil
}

// RemoveMember removes a user from a team.
func (s *TeamService) RemoveMember(ctx context.Context, teamID, userID uint) error {
	// Check team exists
	if _, err := s.repo.GetByID(ctx, teamID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "team not found")
	}

	// Check if user is a member
	if _, err := s.repo.GetMember(ctx, teamID, userID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "user is not a member of this team")
	}

	if err := s.repo.RemoveMember(ctx, teamID, userID); err != nil {
		s.logger.Error("failed to remove team member",
			zap.Error(err),
			zap.Uint("team_id", teamID),
			zap.Uint("user_id", userID),
		)
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("team member removed",
		zap.Uint("team_id", teamID),
		zap.Uint("user_id", userID),
	)
	return nil
}

// ListMembers returns all members of a team.
func (s *TeamService) ListMembers(ctx context.Context, teamID uint) ([]model.TeamMember, error) {
	// Check team exists
	if _, err := s.repo.GetByID(ctx, teamID); err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "team not found")
	}

	members, err := s.repo.ListMembers(ctx, teamID)
	if err != nil {
		s.logger.Error("failed to list team members", zap.Error(err), zap.Uint("team_id", teamID))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	return members, nil
}

// ListByUser returns all team memberships for a given user.
func (s *TeamService) ListByUser(ctx context.Context, userID uint) ([]model.TeamMember, error) {
	return s.repo.ListByUser(ctx, userID)
}
