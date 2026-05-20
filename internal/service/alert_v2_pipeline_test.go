package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

func Test_buildAlertKey_CrossDatasource_NoCollision(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(42)
	ds1 := uint(1)
	ds2 := uint(2)
	labels := model.JSONLabels{"job": "api", "instance": "host1:9090"}

	event1 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &ds1,
		Labels:       labels,
	}
	event2 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &ds2,
		Labels:       labels,
	}

	key1 := p.buildAlertKey(event1)
	key2 := p.buildAlertKey(event2)

	assert.NotEqual(t, key1, key2, "same rule+labels but different datasource should produce different keys")
	assert.Len(t, key1, 32, "md5 hex should be 32 chars")
	assert.Len(t, key2, 32)
}

func Test_buildAlertKey_SameInput_Stable(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(1)
	dsID := uint(1)
	labels := model.JSONLabels{"env": "prod", "job": "web"}

	event := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       labels,
	}

	key1 := p.buildAlertKey(event)
	key2 := p.buildAlertKey(event)
	assert.Equal(t, key1, key2, "same input should produce stable key")
}

func Test_buildAlertKey_LabelOrder_Stable(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(1)
	dsID := uint(1)

	event1 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"b": "2", "a": "1"},
	}
	event2 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"a": "1", "b": "2"},
	}

	key1 := p.buildAlertKey(event1)
	key2 := p.buildAlertKey(event2)
	assert.Equal(t, key1, key2, "label order should not affect key")
}

func Test_buildAlertKey_DifferentLabels_DifferentKey(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(1)
	dsID := uint(1)

	event1 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"job": "api"},
	}
	event2 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"job": "web"},
	}

	key1 := p.buildAlertKey(event1)
	key2 := p.buildAlertKey(event2)
	assert.NotEqual(t, key1, key2)
}

func Test_buildAlertKey_NilIDs(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	event := &model.AlertEvent{
		Labels: model.JSONLabels{"job": "api"},
	}

	key := p.buildAlertKey(event)
	assert.Len(t, key, 32, "should handle nil ruleID and datasourceID")
}
