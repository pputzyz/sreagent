package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestInspectionExecutor() *InspectionExecutor {
	return &InspectionExecutor{
		taskRepo: nil,
		runRepo:  nil,
		agentSvc: nil,
		logger:   zap.NewNop(),
	}
}

func Test_parseReport_json_block(t *testing.T) {
	exec := newTestInspectionExecutor()

	output := `## 巡检结果

以下是本次巡检的发现：

` + "```json" + `
{
  "summary": "发现 2 个问题需要关注",
  "findings": [
    {
      "severity": "warning",
      "category": "performance",
      "object": "api-server-01",
      "detail": "CPU 使用率持续超过 80%"
    },
    {
      "severity": "critical",
      "category": "availability",
      "object": "redis-cluster",
      "detail": "主从同步延迟超过 10s"
    }
  ]
}
` + "```" + `

建议尽快处理以上问题。`

	report := exec.parseReport(output)

	assert.Equal(t, "发现 2 个问题需要关注", report.Summary)
	require.Len(t, report.Findings, 2)

	assert.Equal(t, "warning", report.Findings[0].Severity)
	assert.Equal(t, "performance", report.Findings[0].Category)
	assert.Equal(t, "api-server-01", report.Findings[0].Object)
	assert.Equal(t, "CPU 使用率持续超过 80%", report.Findings[0].Detail)

	assert.Equal(t, "critical", report.Findings[1].Severity)
	assert.Equal(t, "availability", report.Findings[1].Category)
	assert.Equal(t, "redis-cluster", report.Findings[1].Object)
	assert.Equal(t, "主从同步延迟超过 10s", report.Findings[1].Detail)
}

func Test_parseReport_fallback_plaintext(t *testing.T) {
	exec := newTestInspectionExecutor()

	output := "本次巡检未发现异常，系统运行正常。所有服务健康检查通过。"

	report := exec.parseReport(output)

	assert.Equal(t, output, report.Summary)
	assert.Nil(t, report.Findings)
}

func Test_parseReport_fallback_empty_output(t *testing.T) {
	exec := newTestInspectionExecutor()

	report := exec.parseReport("")

	// When output is empty, parseReport replaces the default summary with
	// the trimmed (empty) output — no JSON block found, so it falls back.
	assert.Equal(t, "", report.Summary)
	assert.Nil(t, report.Findings)
}

func Test_parseReport_malformed_json_block(t *testing.T) {
	exec := newTestInspectionExecutor()

	output := "Some text\n```json\n{broken json\n```\nMore text"

	report := exec.parseReport(output)

	// Malformed JSON should fall back to plain text summary.
	assert.Nil(t, report.Findings)
	// Summary is the truncated plain text.
	assert.NotEmpty(t, report.Summary)
}

func Test_parseReport_json_block_no_findings(t *testing.T) {
	exec := newTestInspectionExecutor()

	output := "```json\n{\"summary\": \"一切正常\", \"findings\": []}\n```"

	report := exec.parseReport(output)

	assert.Equal(t, "一切正常", report.Summary)
	assert.Empty(t, report.Findings)
}
