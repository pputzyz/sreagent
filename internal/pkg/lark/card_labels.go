package lark

// webhookCardLabels holds the localizable static labels for the group-broadcast
// webhook alert card. Group webhooks fan out to many recipients, so the language is
// a per-channel (per NotifyMedia) configuration rather than per-recipient.
type webhookCardLabels struct {
	StatusFiring string
	StatusResol  string
	StatusAcked  string
	FieldStatus  string
	FieldLevel   string
	FieldFiredAt string
	LabelsTitle  string
	NoLabels     string
	DescTitle    string
	NoDesc       string
	AITitle      string
	AISummary    string
	AICauses     string
	AIImpact     string
	AISteps      string
	BtnDetail    string
}

// cardLabelsFor returns the label set for the given language. Anything other than
// "en" falls back to Simplified Chinese (the platform default), so an empty/unknown
// value preserves existing behavior.
func cardLabelsFor(lang string) webhookCardLabels {
	if lang == "en" {
		return webhookCardLabels{
			StatusFiring: "Firing",
			StatusResol:  "Resolved",
			StatusAcked:  "Acknowledged",
			FieldStatus:  "Status",
			FieldLevel:   "Severity",
			FieldFiredAt: "Fired at",
			LabelsTitle:  "Labels",
			NoLabels:     "_no extra labels_",
			DescTitle:    "Description",
			NoDesc:       "_no description_",
			AITitle:      "AI Analysis",
			AISummary:    "Summary",
			AICauses:     "Probable causes",
			AIImpact:     "Impact",
			AISteps:      "Recommended steps",
			BtnDetail:    "📊 View details",
		}
	}
	return webhookCardLabels{
		StatusFiring: "告警中",
		StatusResol:  "已恢复",
		StatusAcked:  "已确认",
		FieldStatus:  "状态",
		FieldLevel:   "级别",
		FieldFiredAt: "触发时间",
		LabelsTitle:  "标签",
		NoLabels:     "_无额外标签_",
		DescTitle:    "描述",
		NoDesc:       "_无描述_",
		AITitle:      "AI 分析",
		AISummary:    "摘要",
		AICauses:     "可能原因",
		AIImpact:     "影响范围",
		AISteps:      "建议操作",
		BtnDetail:    "📊 查看详情",
	}
}
