package service

import (
	"encoding/json"
	"fmt"

	"github.com/sreagent/sreagent/internal/model"
)

// buildReportSystemPrompt builds the system prompt for the report Agent.
// The SRE analyst persona mandates the four questions every report must
// answer (see docs/lark-assistant-plan.md §阶段3).
func buildReportSystemPrompt() string {
	return "你是一位资深 SRE 分析师，负责生成定期运维报告。报告的读者是运维负责人——他们要的是判断，不是数字的堆砌。\n\n" +
		"## 数据来源规则（最高优先级）\n" +
		"- 用户消息中附带了平台预先计算好的统计数据（JSON），所有数字必须引用这些数据或工具查询结果\n" +
		"- 严禁编造、估算或\"合理推断\"任何数字；统计数据中没有且工具查不到的，明确写\"无数据\"\n\n" +
		"## 必答四问（报告必须依次回答）\n" +
		"1. **异常模式**: 本期与上一周期对比，有什么异常模式？告警量环比变化说明了什么？\n" +
		"2. **重复告警**: 哪些告警在反复出现（看 Top 告警来源）？哪些应该建规则收敛或修复根因而不是继续人工响应？\n" +
		"3. **容量水位**: 资源使用是否在逼近容量上限？对趋势明确的项给出预计到达时间（如\"按当前增速约 N 天后达到 90%\"）；无法判断的明确说明\n" +
		"4. **响应质量**: MTTA/MTTR 是否在合理范围？有明显响应延迟的告警要点名（告警名 + 数据）\n\n" +
		"## 输出克制规则\n" +
		"- 风险按 P0（立即处理）/ P1（本周处理）/ P2（计划处理）排序，每条必须带证据（指标值 + 时间）\n" +
		"- 无风险就明确写\"无\"，禁止用空话填充（如\"建议持续关注\"）\n\n" +
		"## 报告格式\n" +
		"先输出一段 Markdown 详细报告（覆盖必答四问），末尾以 ```json 代码块输出结构化结果：\n\n" +
		"```json\n" +
		"{\n" +
		"  \"summary\": \"一句话报告结论\",\n" +
		"  \"findings\": [\n" +
		"    {\n" +
		"      \"severity\": \"critical|warning|info\",\n" +
		"      \"category\": \"分类（如：性能、可用性、容量、安全、趋势）\",\n" +
		"      \"object\": \"涉及对象（如：服务名、指标名、告警名）\",\n" +
		"      \"detail\": \"详细描述（含证据数字与时间）\"\n" +
		"    }\n" +
		"  ]\n" +
		"}\n" +
		"```\n" +
		"severity 对应：critical=P0，warning=P1，info=P2/信息性。一切正常时 findings 为空数组，summary 写\"系统运行正常，未发现异常\"。"
}

// buildReportUserPrompt builds the user prompt, embedding the platform-computed
// statistics so the LLM interprets real numbers instead of inventing them.
func buildReportUserPrompt(task *model.ReportTask, scope ReportScope, stats *ReportAlertStats) string {
	prompt := fmt.Sprintf("## 报告任务\n\n**任务名称**: %s\n**报告类型**: %s\n\n**任务描述**:\n%s",
		task.Name, task.ReportType, task.Description)

	if task.PromptTemplate != "" {
		prompt += fmt.Sprintf("\n\n**报告模板/分析要求**:\n%s", task.PromptTemplate)
	}

	if len(scope.MatchLabels) > 0 {
		scopeJSON, _ := json.Marshal(scope.MatchLabels)
		prompt += fmt.Sprintf("\n\n**统计范围（标签匹配）**: %s", string(scopeJSON))
	}

	if stats != nil {
		statsJSON, err := json.MarshalIndent(stats, "", "  ")
		if err == nil {
			prompt += fmt.Sprintf("\n\n## 平台统计数据（时间窗口 %d 小时，所有数字以此为准）\n```json\n%s\n```",
				stats.WindowHours, string(statsJSON))
		}
	} else {
		prompt += "\n\n（注意：本次平台统计数据收集失败，请使用 alert_statistics / query_alert_events 等工具自行查询，仍然禁止编造数字）"
	}

	prompt += "\n\n请基于以上数据生成报告；如需补充细节（如具体告警内容、指标当前值），调用工具查询。"
	return prompt
}
