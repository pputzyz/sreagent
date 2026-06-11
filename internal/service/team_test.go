package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

// mockTeamRepo implements TeamRepository for unit-testing TeamService.
type mockTeamRepo struct {
	teams      map[uint]*model.Team
	references map[string]int64
	deleted    []uint
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{teams: make(map[uint]*model.Team)}
}

func (m *mockTeamRepo) Create(_ context.Context, team *model.Team) error { return nil }
func (m *mockTeamRepo) GetByID(_ context.Context, id uint) (*model.Team, error) {
	if t, ok := m.teams[id]; ok {
		return t, nil
	}
	return nil, assertNotFoundErr
}
func (m *mockTeamRepo) GetByName(_ context.Context, name string) (*model.Team, error) {
	return nil, assertNotFoundErr
}
func (m *mockTeamRepo) List(_ context.Context, page, pageSize int) ([]model.Team, int64, error) {
	return nil, 0, nil
}
func (m *mockTeamRepo) Update(_ context.Context, team *model.Team) error { return nil }
func (m *mockTeamRepo) Delete(_ context.Context, id uint) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockTeamRepo) AddMember(_ context.Context, member *model.TeamMember) error { return nil }
func (m *mockTeamRepo) RemoveMember(_ context.Context, teamID, userID uint) error   { return nil }
func (m *mockTeamRepo) ListMembers(_ context.Context, teamID uint) ([]model.TeamMember, error) {
	return nil, nil
}
func (m *mockTeamRepo) GetMember(_ context.Context, teamID, userID uint) (*model.TeamMember, error) {
	return nil, assertNotFoundErr
}
func (m *mockTeamRepo) UpdateMember(_ context.Context, member *model.TeamMember) error { return nil }
func (m *mockTeamRepo) ListByUser(_ context.Context, userID uint) ([]model.TeamMember, error) {
	return nil, nil
}
func (m *mockTeamRepo) CountReferences(_ context.Context, teamID uint) (map[string]int64, error) {
	return m.references, nil
}

// assertNotFoundErr mimics gorm.ErrRecordNotFound for the mock.
var assertNotFoundErr = errNotFoundSentinel{}

type errNotFoundSentinel struct{}

func (errNotFoundSentinel) Error() string { return "record not found" }

// Test_DeleteTeam_ReferencedByRule_Blocked: a team still referenced by alert
// rules / escalation steps must NOT be deletable — deleting it would leave
// dangling team_id pointers and escalation targets resolving to nobody.
func Test_DeleteTeam_ReferencedByRule_Blocked(t *testing.T) {
	repo := newMockTeamRepo()
	repo.teams[1] = &model.Team{BaseModel: model.BaseModel{ID: 1}, Name: "sre"}
	repo.references = map[string]int64{"alert rules": 2, "escalation steps": 1}

	svc := NewTeamService(repo, zap.NewNop())
	err := svc.Delete(context.Background(), 1)

	assert.Error(t, err, "delete must be blocked while references exist")
	assert.Contains(t, err.Error(), "referenced", "error should explain the block")
	assert.Empty(t, repo.deleted, "underlying Delete must NOT be called")
}

// Test_DeleteTeam_NoReferences_Succeeds: an unreferenced team deletes normally.
func Test_DeleteTeam_NoReferences_Succeeds(t *testing.T) {
	repo := newMockTeamRepo()
	repo.teams[2] = &model.Team{BaseModel: model.BaseModel{ID: 2}, Name: "empty-team"}
	repo.references = nil

	svc := NewTeamService(repo, zap.NewNop())
	err := svc.Delete(context.Background(), 2)

	assert.NoError(t, err)
	assert.Equal(t, []uint{2}, repo.deleted, "underlying Delete must be called once")
}
