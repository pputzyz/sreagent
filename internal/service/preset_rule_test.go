package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/testutil"
)

func Test_autoMatchDatasource_empty_cluster(t *testing.T) {
	svc := &PresetRuleService{
		dsRepo: nil, // not used for empty cluster check
		logger: testutil.TestLogger(),
	}

	_, err := svc.autoMatchDatasource(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty cluster")
}

func Test_autoMatchDatasource_match(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	dsRepo := repository.NewDataSourceRepository(db)

	// Seed a datasource with cluster label
	ds := &model.DataSource{
		Name:      "prom-prod",
		Type:      model.DSTypePrometheus,
		Endpoint:  "http://localhost:9090",
		Labels:    model.JSONLabels{"cluster": "prod-cn"},
		IsEnabled: true,
	}
	require.NoError(t, dsRepo.Create(context.Background(), ds))

	svc := &PresetRuleService{
		dsRepo: dsRepo,
		logger: testutil.TestLogger(),
	}

	id, err := svc.autoMatchDatasource(context.Background(), "prod-cn")
	require.NoError(t, err)
	assert.Equal(t, ds.ID, id)
}

func Test_autoMatchDatasource_no_match(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	dsRepo := repository.NewDataSourceRepository(db)

	// Seed a datasource with a different cluster label
	ds := &model.DataSource{
		Name:      "prom-staging",
		Type:      model.DSTypePrometheus,
		Endpoint:  "http://localhost:9090",
		Labels:    model.JSONLabels{"cluster": "staging"},
		IsEnabled: true,
	}
	require.NoError(t, dsRepo.Create(context.Background(), ds))

	svc := &PresetRuleService{
		dsRepo: dsRepo,
		logger: testutil.TestLogger(),
	}

	_, err := svc.autoMatchDatasource(context.Background(), "prod-cn")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no datasource")
}

func Test_autoMatchDatasource_disabled_ds_ignored(t *testing.T) {
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	dsRepo := repository.NewDataSourceRepository(db)

	// Seed a DISABLED datasource with matching cluster (GORM default:true overrides false)
	ds := &model.DataSource{
		Name:      "prom-prod-disabled",
		Type:      model.DSTypePrometheus,
		Endpoint:  "http://localhost:9090",
		Labels:    model.JSONLabels{"cluster": "prod-cn"},
		IsEnabled: true,
	}
	require.NoError(t, dsRepo.Create(context.Background(), ds))
	require.NoError(t, db.Model(&model.DataSource{}).Where("id = ?", ds.ID).Update("is_enabled", false).Error)

	svc := &PresetRuleService{
		dsRepo: dsRepo,
		logger: testutil.TestLogger(),
	}

	_, err := svc.autoMatchDatasource(context.Background(), "prod-cn")
	assert.Error(t, err, "disabled datasource should not match")
}
