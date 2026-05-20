package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sreagent/sreagent/internal/model"
)

// Test_SupportsQuery_prometheus_returns_true verifies that Prometheus
// datasources support direct querying.
func Test_SupportsQuery_prometheus_returns_true(t *testing.T) {
	ds := &model.DataSource{Type: model.DSTypePrometheus}
	assert.True(t, ds.SupportsQuery(), "prometheus should support query")
}

// Test_SupportsQuery_victoriametrics_returns_true verifies that VictoriaMetrics
// datasources support direct querying.
func Test_SupportsQuery_victoriametrics_returns_true(t *testing.T) {
	ds := &model.DataSource{Type: model.DSTypeVictoriaMetrics}
	assert.True(t, ds.SupportsQuery(), "victoriametrics should support query")
}

// Test_SupportsQuery_victorialogs_returns_true verifies that VictoriaLogs
// datasources support direct querying (log queries).
func Test_SupportsQuery_victorialogs_returns_true(t *testing.T) {
	ds := &model.DataSource{Type: model.DSTypeVictoriaLogs}
	assert.True(t, ds.SupportsQuery(), "victorialogs should support query")
}

// Test_SupportsQuery_zabbix_returns_false verifies that Zabbix datasources
// do NOT support direct querying (alert ingestion only).
func Test_SupportsQuery_zabbix_returns_false(t *testing.T) {
	ds := &model.DataSource{Type: model.DSTypeZabbix}
	assert.False(t, ds.SupportsQuery(), "zabbix should NOT support query")
}

// Test_DataSourceType_constants verifies that datasource type constants
// have the expected string values.
func Test_DataSourceType_constants(t *testing.T) {
	assert.Equal(t, "prometheus", string(model.DSTypePrometheus))
	assert.Equal(t, "victoriametrics", string(model.DSTypeVictoriaMetrics))
	assert.Equal(t, "zabbix", string(model.DSTypeZabbix))
	assert.Equal(t, "victorialogs", string(model.DSTypeVictoriaLogs))
}

// Test_DataSourceStatus_constants verifies datasource status constants.
func Test_DataSourceStatus_constants(t *testing.T) {
	assert.Equal(t, "healthy", string(model.DSStatusHealthy))
	assert.Equal(t, "unhealthy", string(model.DSStatusUnhealthy))
	assert.Equal(t, "unknown", string(model.DSStatusUnknown))
}

// Test_DataSource_AfterFind_populates_SupportsQueryField verifies that
// the GORM AfterFind hook correctly sets the computed SupportsQueryField.
func Test_DataSource_AfterFind_populates_SupportsQueryField(t *testing.T) {
	// Simulate what AfterFind does for a prometheus datasource
	promDS := &model.DataSource{Type: model.DSTypePrometheus}
	_ = promDS.AfterFind(nil)
	assert.True(t, promDS.SupportsQueryField)

	// Simulate for a zabbix datasource
	zabbixDS := &model.DataSource{Type: model.DSTypeZabbix}
	_ = zabbixDS.AfterFind(nil)
	assert.False(t, zabbixDS.SupportsQueryField)
}

// Test_DataSource_MultiType_Routing verifies that different datasource types
// are correctly identified for routing decisions in multi-datasource setups.
func Test_DataSource_MultiType_Routing(t *testing.T) {
	datasources := []model.DataSource{
		{Name: "prom-1", Type: model.DSTypePrometheus},
		{Name: "vm-1", Type: model.DSTypeVictoriaMetrics},
		{Name: "zabbix-1", Type: model.DSTypeZabbix},
		{Name: "vlogs-1", Type: model.DSTypeVictoriaLogs},
	}

	queryable := make([]string, 0)
	nonQueryable := make([]string, 0)

	for _, ds := range datasources {
		if ds.SupportsQuery() {
			queryable = append(queryable, ds.Name)
		} else {
			nonQueryable = append(nonQueryable, ds.Name)
		}
	}

	assert.Len(t, queryable, 3, "prometheus, victoriametrics, victorialogs should be queryable")
	assert.Contains(t, queryable, "prom-1")
	assert.Contains(t, queryable, "vm-1")
	assert.Contains(t, queryable, "vlogs-1")

	assert.Len(t, nonQueryable, 1, "only zabbix should be non-queryable")
	assert.Contains(t, nonQueryable, "zabbix-1")
}
