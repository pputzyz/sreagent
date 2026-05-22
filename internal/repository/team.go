package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// TeamRepository handles teams and team_members persistence.
type TeamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, team *model.Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *TeamRepository) GetByID(ctx context.Context, id uint) (*model.Team, error) {
	var team model.Team
	err := r.db.WithContext(ctx).Preload("Members").First(&team, id).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// GetByName finds a team by name.
func (r *TeamRepository) GetByName(ctx context.Context, name string) (*model.Team, error) {
	var team model.Team
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&team).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// GetMember returns a specific team member.
func (r *TeamRepository) GetMember(ctx context.Context, teamID, userID uint) (*model.TeamMember, error) {
	var member model.TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *TeamRepository) List(ctx context.Context, page, pageSize int) ([]model.Team, int64, error) {
	var list []model.Team
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Team{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *TeamRepository) Update(ctx context.Context, team *model.Team) error {
	return r.db.WithContext(ctx).Save(team).Error
}

func (r *TeamRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove all team members first
		if err := tx.WithContext(ctx).Where("team_id = ?", id).Delete(&model.TeamMember{}).Error; err != nil {
			return err
		}
		// Delete the team
		return tx.WithContext(ctx).Delete(&model.Team{}, id).Error
	})
}

// AddMember adds a user to a team.
func (r *TeamRepository) AddMember(ctx context.Context, member *model.TeamMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// UpdateMember updates an existing team member's role.
func (r *TeamRepository) UpdateMember(ctx context.Context, member *model.TeamMember) error {
	return r.db.WithContext(ctx).Save(member).Error
}

// RemoveMember removes a user from a team.
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&model.TeamMember{}).Error
}

// ListMembers returns all members of a team with their user info.
func (r *TeamRepository) ListMembers(ctx context.Context, teamID uint) ([]model.TeamMember, error) {
	var members []model.TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Find(&members).Error
	return members, err
}

// GetByLabels finds teams whose labels are a subset match of the provided labels.
// NOTE: Teams table is small (<100 rows typical), so full-scan + in-memory filter is acceptable.
// A LIMIT guard prevents unbounded scans if the table unexpectedly grows.
func (r *TeamRepository) GetByLabels(ctx context.Context, labels map[string]string) ([]model.Team, error) {
	const maxScanRows = 1000
	var allTeams []model.Team
	err := r.db.WithContext(ctx).Limit(maxScanRows).Find(&allTeams).Error
	if err != nil {
		return nil, err
	}

	var matched []model.Team
	for _, team := range allTeams {
		if labelmatch.Match(labels, map[string]string(team.Labels)) {
			matched = append(matched, team)
		}
	}
	return matched, nil
}

// ListByUser returns all team memberships for a given user.
func (r *TeamRepository) ListByUser(ctx context.Context, userID uint) ([]model.TeamMember, error) {
	var members []model.TeamMember
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&members).Error
	return members, err
}
