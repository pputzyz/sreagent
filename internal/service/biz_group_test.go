package service

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// testBizGroupDB creates an in-memory SQLite with BizGroup tables migrated.
func testBizGroupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open sqlite")
	require.NoError(t, db.AutoMigrate(
		&model.BizGroup{},
		&model.BizGroupMember{},
	))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newTestBizGroupService(db *gorm.DB) *BizGroupService {
	repo := repository.NewBizGroupRepository(db)
	return NewBizGroupService(repo, zap.NewNop())
}

// TestDelete_WithChildren_ReturnsError verifies that deleting a group that has
// child groups returns an error telling the user to re-parent or delete children first.
func TestDelete_WithChildren_ReturnsError(t *testing.T) {
	db := testBizGroupDB(t)
	svc := newTestBizGroupService(db)
	ctx := context.Background()

	// Create a parent group
	parentID := uint(1)
	parent := &model.BizGroup{
		BaseModel: model.BaseModel{ID: parentID},
		Name:      "parent-group",
	}
	require.NoError(t, db.Create(parent).Error)

	// Create a child group under the parent
	child := &model.BizGroup{
		BaseModel: model.BaseModel{ID: 2},
		Name:      "child-group",
		ParentID:  &parentID,
	}
	require.NoError(t, db.Create(child).Error)

	// Attempt to delete the parent — should fail
	err := svc.Delete(ctx, parentID)
	assert.Error(t, err, "deleting a group with children should return an error")
	assert.Contains(t, err.Error(), "children", "error message should mention children")
}

// TestDelete_LeafNode_Success verifies that a group with no children can be
// deleted successfully.
func TestDelete_LeafNode_Success(t *testing.T) {
	db := testBizGroupDB(t)
	svc := newTestBizGroupService(db)
	ctx := context.Background()

	// Create a parent group
	parentID := uint(1)
	parent := &model.BizGroup{
		BaseModel: model.BaseModel{ID: parentID},
		Name:      "parent-group",
	}
	require.NoError(t, db.Create(parent).Error)

	// Create a leaf child group
	leafID := uint(2)
	leaf := &model.BizGroup{
		BaseModel: model.BaseModel{ID: leafID},
		Name:      "leaf-group",
		ParentID:  &parentID,
	}
	require.NoError(t, db.Create(leaf).Error)

	// Delete the leaf — should succeed
	err := svc.Delete(ctx, leafID)
	assert.NoError(t, err, "deleting a leaf group should succeed")

	// Verify it's gone
	_, err = svc.GetByID(ctx, leafID)
	assert.Error(t, err, "deleted group should not be found")
}

// TestDelete_NonExistent_ReturnsNotFound verifies that deleting a non-existent
// group returns the appropriate error.
func TestDelete_NonExistent_ReturnsNotFound(t *testing.T) {
	db := testBizGroupDB(t)
	svc := newTestBizGroupService(db)
	ctx := context.Background()

	err := svc.Delete(ctx, 9999)
	assert.Error(t, err, "deleting a non-existent group should return an error")
}

// TestCreate_WithValidParent_Success verifies that creating a group with a
// valid parent succeeds.
func TestCreate_WithValidParent_Success(t *testing.T) {
	db := testBizGroupDB(t)
	svc := newTestBizGroupService(db)
	ctx := context.Background()

	parentID := uint(1)
	parent := &model.BizGroup{
		BaseModel: model.BaseModel{ID: parentID},
		Name:      "parent",
	}
	require.NoError(t, db.Create(parent).Error)

	child := &model.BizGroup{
		Name:     "child",
		ParentID: &parentID,
	}
	err := svc.Create(ctx, child)
	assert.NoError(t, err, "creating a child with valid parent should succeed")
}

// TestCreate_WithInvalidParent_ReturnsError verifies that creating a group
// with a non-existent parent returns an error.
func TestCreate_WithInvalidParent_ReturnsError(t *testing.T) {
	db := testBizGroupDB(t)
	svc := newTestBizGroupService(db)
	ctx := context.Background()

	nonExistentParentID := uint(9999)
	child := &model.BizGroup{
		Name:     "orphan",
		ParentID: &nonExistentParentID,
	}
	err := svc.Create(ctx, child)
	assert.Error(t, err, "creating a child with non-existent parent should fail")
}
