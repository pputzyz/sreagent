package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// RuleGeneratorService provides AI-powered rule generation from natural language.
type RuleGeneratorService struct {
	aiSvc       *AIService
	labelRegSvc *LabelRegistryService
	dsSvc       DataSourceQuerier
	ruleSvc     AlertRuleOperator
	presetRepo  *repository.PresetRuleRepository
	dsRepo      *repository.DataSourceRepository
	cache       *RuleGenCache
	logger      *zap.Logger
}

// NewRuleGeneratorService creates a new RuleGeneratorService.
func NewRuleGeneratorService(
	aiSvc *AIService,
	labelRegSvc *LabelRegistryService,
	dsSvc DataSourceQuerier,
	ruleSvc AlertRuleOperator,
	presetRepo *repository.PresetRuleRepository,
	dsRepo *repository.DataSourceRepository,
	logger *zap.Logger,
) *RuleGeneratorService {
	return &RuleGeneratorService{
		aiSvc:       aiSvc,
		labelRegSvc: labelRegSvc,
		dsSvc:       dsSvc,
		ruleSvc:     ruleSvc,
		presetRepo:  presetRepo,
		dsRepo:      dsRepo,
		cache:       NewRuleGenCache(10 * time.Minute),
		logger:      logger,
	}
}

// RuleGenerateRequest is the input for rule generation.
type RuleGenerateRequest struct {
	Description  string          `json:"description" binding:"required"`
	DatasourceID *uint           `json:"datasource_id"`
	RuleType     string          `json:"rule_type"` // "alert" or "inhibition"
	SaveAsDraft  bool            `json:"save_as_draft"`
	Context      GenerateContext `json:"context"`
}

// GenerateContext holds optional context for rule generation.
type GenerateContext struct {
	ExistingRules   bool  `json:"existing_rules"`
	IncludeLabels   bool  `json:"include_labels"`
	IncludeRouting  bool  `json:"include_routing"`
	TargetChannelID *uint `json:"target_channel_id"`
}

// RuleGenerateResult is the AI-generated rule.
type RuleGenerateResult struct {
	Type        string `json:"type"` // "alert" or "inhibition"
	// For alert rules
	Expression  string            `json:"expression,omitempty"`
	ForDuration string            `json:"for_duration,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	// For inhibition rules
	SourceLabels []string `json:"source_labels,omitempty"`
	SourceValue  string   `json:"source_value,omitempty"`
	TargetLabels []string `json:"target_labels,omitempty"`
	EqualLabels  []string `json:"equal_labels,omitempty"`
	// Common
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	Confidence         float64             `json:"confidence"`
	Warnings           []string            `json:"warnings"`
	SuggestedChannel   *ChannelSuggestion  `json:"suggested_channel,omitempty"`
}

// ChannelSuggestion is an AI-suggested notification channel.
type ChannelSuggestion struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Generate creates a rule from natural language description.
func (s *RuleGeneratorService) Generate(ctx context.Context, req *RuleGenerateRequest) (*RuleGenerateResult, error) {
	// 0. Check cache
	if cached := s.cache.Get(req.Description, req.DatasourceID, req.RuleType); cached != nil {
		return cached, nil
	}

	// 1. Check AI is enabled and rule_gen module is enabled
	if err := s.checkAIEnabled(ctx); err != nil {
		return nil, err
	}

	// 2. Build context for the LLM
	labelContext, err := s.buildLabelContext(ctx, req.DatasourceID)
	if err != nil {
		s.logger.Warn("failed to build label context, proceeding without it", zap.Error(err))
		labelContext = "暂无可用标签信息"
	}

	existingRules, err := s.buildExistingRulesContext(ctx)
	if err != nil {
		s.logger.Warn("failed to build existing rules context", zap.Error(err))
		existingRules = "暂无已有规则信息"
	}

	presetMatches, err := s.buildPresetMatches(ctx, req.Description)
	if err != nil {
		s.logger.Warn("failed to build preset matches", zap.Error(err))
		presetMatches = "暂无匹配的预置规则"
	}

	// 3. Build system prompt with few-shot examples
	labelKeys, _ := s.labelRegSvc.GetKeys(ctx, nil)
	systemPrompt := s.buildSystemPrompt(labelContext, existingRules, presetMatches) + "\n\n" + fewShotAlertRule(labelKeys)

	// 4. Build user prompt
	userPrompt := req.Description
	if req.DatasourceID != nil {
		ds, err := s.dsRepo.GetByID(ctx, *req.DatasourceID)
		if err == nil {
			userPrompt += fmt.Sprintf("\n\n数据源: %s (类型: %s)", ds.Name, ds.Type)
		}
	}
	if req.RuleType != "" {
		userPrompt += fmt.Sprintf("\n规则类型: %s", req.RuleType)
	}

	// 5. Call LLM
	var result RuleGenerateResult
	if err := s.aiSvc.callLLMJSON(ctx, s.mustLoadConfig(ctx), systemPrompt, userPrompt, &result); err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, fmt.Sprintf("AI rule generation failed: %v", err))
	}

	// 6. Post-process and validate
	s.postProcessResult(&result)

	// 7. Cache result
	s.cache.Set(req.Description, req.DatasourceID, req.RuleType, &result)

	return &result, nil
}

// GenerateInhibition generates an inhibition rule from natural language.
func (s *RuleGeneratorService) GenerateInhibition(ctx context.Context, description string, datasourceID *uint) (*RuleGenerateResult, error) {
	if err := s.checkAIEnabled(ctx); err != nil {
		return nil, err
	}

	systemPrompt := `你是 SRE 告警抑制规则生成助手。根据用户的自然语言描述，生成标准的抑制规则。

输出格式要求（严格 JSON）：
{
  "type": "inhibition",
  "name": "抑制规则名称",
  "description": "抑制规则说明",
  "source_labels": ["alertname"],
  "source_value": "SourceAlertName",
  "target_labels": ["label1", "label2"],
  "equal_labels": ["label1", "label2"],
  "confidence": 0.9,
  "warnings": []
}

注意：
- source_labels: 源告警需要匹配的标签名列表
- source_value: 源告警的 alertname 值
- target_labels: 被抑制告警需要匹配的标签名列表
- equal_labels: 源和目标告警需要相等的标签名列表（空表示总是抑制）
- 如果信息不足，在 warnings 中列出需要确认的事项

` + fewShotInhibition()

	var result RuleGenerateResult
	if err := s.aiSvc.callLLMJSON(ctx, s.mustLoadConfig(ctx), systemPrompt, description, &result); err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, fmt.Sprintf("AI inhibition rule generation failed: %v", err))
	}

	result.Type = "inhibition"
	s.postProcessResult(&result)

	return &result, nil
}

// MuteRuleGenerateResult is the AI-generated mute rule.
type MuteRuleGenerateResult struct {
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	MatchLabels   map[string]string `json:"match_labels"`
	Severities    []string `json:"severities"`
	StartTime     string   `json:"start_time,omitempty"`
	EndTime       string   `json:"end_time,omitempty"`
	PeriodicStart string   `json:"periodic_start,omitempty"`
	PeriodicEnd   string   `json:"periodic_end,omitempty"`
	DaysOfWeek    []string `json:"days_of_week"`
	Timezone      string   `json:"timezone"`
	RuleIDs       []uint   `json:"rule_ids,omitempty"`
	Confidence    float64  `json:"confidence"`
	Warnings      []string `json:"warnings"`
}

// GenerateMute generates a mute rule from natural language.
func (s *RuleGeneratorService) GenerateMute(ctx context.Context, description string, timezone string) (*MuteRuleGenerateResult, error) {
	if err := s.checkAIEnabled(ctx); err != nil {
		return nil, err
	}

	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	systemPrompt := fmt.Sprintf(`你是 SRE 告警静默规则生成助手。根据用户的自然语言描述，生成标准的静默规则。

输出格式要求（严格 JSON）：
{
  "type": "mute",
  "name": "静默规则名称",
  "description": "静默规则说明",
  "match_labels": {"label_key": "label_value"},
  "severities": ["critical", "warning"],
  "start_time": "2026-01-01T02:00:00+08:00",
  "end_time": "2026-01-01T06:00:00+08:00",
  "periodic_start": "02:00",
  "periodic_end": "06:00",
  "days_of_week": ["1","2","3","4","5"],
  "timezone": "%s",
  "rule_ids": [],
  "confidence": 0.9,
  "warnings": []
}

重要规则：
- match_labels: 必须是 key-value 对，用于匹配告警标签。常见标签：service, env, severity, category, biz_project, tenant
- severities: 要静默的严重等级数组，可选 critical/warning/info，空数组表示全部
- 一次性静默：填写 start_time + end_time（ISO 8601 格式），periodic_start/end 留空
- 周期性静默：填写 periodic_start + periodic_end（HH:MM 格式）+ days_of_week（1=周一到7=周日），start_time/end_time 留空
- days_of_week: 空数组表示每天
- rule_ids: 如果用户指定了特定规则，填规则 ID 数组；否则留空表示按标签匹配
- 如果信息不足，在 warnings 中列出需要确认的事项
- confidence: 0.0-1.0，信息越完整越高

` + fewShotMute() + `

例子：
用户: "凌晨 2 点到 6 点静默 staging 环境的告警"
输出: {"type":"mute","name":"staging-凌晨维护静默","description":"staging 环境每日凌晨维护窗口","match_labels":{"env":"staging"},"severities":[],"periodic_start":"02:00","periodic_end":"06:00","days_of_week":[],"timezone":"%s","confidence":0.95,"warnings":[]}`, timezone, timezone)

	var result MuteRuleGenerateResult
	if err := s.aiSvc.callLLMJSON(ctx, s.mustLoadConfig(ctx), systemPrompt, description, &result); err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, fmt.Sprintf("AI mute rule generation failed: %v", err))
	}

	result.Type = "mute"
	if result.Timezone == "" {
		result.Timezone = timezone
	}

	return &result, nil
}

// GenerateWithDraftResult wraps a RuleGenerateResult with optional draft info.
type GenerateWithDraftResult struct {
	*RuleGenerateResult
	RuleID uint `json:"rule_id,omitempty"`
	Draft  bool `json:"draft,omitempty"`
}

// SaveDraft persists an AI-generated alert rule as a draft (status=draft, enabled=false).
// Only alert-type rules can be saved as draft; inhibition rules are returned as-is.
func (s *RuleGeneratorService) SaveDraft(ctx context.Context, result *RuleGenerateResult, datasourceID *uint, userID uint) (*GenerateWithDraftResult, error) {
	res := &GenerateWithDraftResult{RuleGenerateResult: result}

	if result.Type != "alert" {
		return res, nil
	}

	labels := make(model.JSONLabels)
	for k, v := range result.Labels {
		labels[k] = v
	}
	annotations := make(model.JSONLabels)
	for k, v := range result.Annotations {
		annotations[k] = v
	}

	rule := &model.AlertRule{
		Name:           result.Name,
		Description:    result.Description,
		DataSourceID:   datasourceID,
		Expression:     result.Expression,
		ForDuration:    result.ForDuration,
		Severity:       model.AlertSeverity(result.Severity),
		Labels:         labels,
		Annotations:    annotations,
		Status:         model.RuleStatusDraft,
		CreatedBy:      userID,
		UpdatedBy:      userID,
	}

	if err := s.ruleSvc.Create(ctx, rule, "ai_draft"); err != nil {
		return nil, apperr.WithMessage(apperr.ErrDatabase, fmt.Sprintf("failed to save draft rule: %v", err))
	}

	res.RuleID = rule.ID
	res.Draft = true
	return res, nil
}

// checkAIEnabled verifies AI and rule_gen module are enabled.
func (s *RuleGeneratorService) checkAIEnabled(ctx context.Context) error {
	cfg, err := s.aiSvc.loadConfig(ctx)
	if err != nil {
		return apperr.WithMessage(apperr.ErrExternalAPI, "failed to load AI config: "+err.Error())
	}
	if !cfg.Enabled {
		return apperr.WithMessage(apperr.ErrExternalAPI, "AI 未启用，请在系统设置中配置 AI")
	}

	modules, err := s.aiSvc.GetAIModules(ctx)
	if err != nil {
		return apperr.WithMessage(apperr.ErrExternalAPI, "failed to load AI modules: "+err.Error())
	}
	if !modules.RuleGen.Enabled {
		return apperr.WithMessage(apperr.ErrExternalAPI, "AI 规则生成功能未启用，请在 AI 模块设置中开启")
	}

	return nil
}

// mustLoadConfig loads AI config or returns a zero config.
func (s *RuleGeneratorService) mustLoadConfig(ctx context.Context) AIConfig {
	cfg, err := s.aiSvc.loadConfig(ctx)
	if err != nil {
		s.logger.Error("failed to load AI config", zap.Error(err))
		return AIConfig{}
	}
	return cfg
}

// buildLabelContext builds a context string from the label registry.
// A 5-second timeout is enforced; on timeout or error, cached keys are used as fallback.
func (s *RuleGeneratorService) buildLabelContext(ctx context.Context, datasourceID *uint) (string, error) {
	var dsIDs []uint
	if datasourceID != nil {
		dsIDs = []uint{*datasourceID}
	}

	// 5-second timeout for label registry queries
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type keysResult struct {
		keys []string
		err  error
	}
	ch := make(chan keysResult, 1)
	go func() {
		k, e := s.labelRegSvc.GetKeys(ctx, dsIDs)
		ch <- keysResult{keys: k, err: e}
	}()

	var keys []string
	select {
	case <-timeoutCtx.Done():
		s.logger.Warn("label context timeout, using cached fallback", zap.Error(timeoutCtx.Err()))
		keys = s.labelRegSvc.GetKeysFallback(dsIDs)
		if len(keys) == 0 {
			return s.buildFallbackLabelContext(datasourceID), nil
		}
	case res := <-ch:
		if res.err != nil {
			s.logger.Warn("label context query failed, using cached fallback", zap.Error(res.err))
			keys = s.labelRegSvc.GetKeysFallback(dsIDs)
			if len(keys) == 0 {
				return s.buildFallbackLabelContext(datasourceID), nil
			}
		} else {
			keys = res.keys
			// Cache the successful result for future fallback
			s.labelRegSvc.SetKeys(dsIDs, keys)
		}
	}

	var sb strings.Builder
	sb.WriteString("可用标签键:\n")
	limit := 50
	if len(keys) < limit {
		limit = len(keys)
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(fmt.Sprintf("  - %s\n", keys[i]))
		// Also get top values for this key
		vals, err := s.labelRegSvc.GetValues(ctx, keys[i], dsIDs)
		if err == nil && len(vals) > 0 {
			valLimit := 5
			if len(vals) < valLimit {
				valLimit = len(vals)
			}
			sb.WriteString(fmt.Sprintf("    常用值: %s\n", strings.Join(vals[:valLimit], ", ")))
		}
	}

	return sb.String(), nil
}

// buildFallbackLabelContext returns a basic label context when the label registry
// is unavailable (timeout or error).
func (s *RuleGeneratorService) buildFallbackLabelContext(datasourceID *uint) string {
	return "Common labels: job, instance, severity, alertname, cluster, namespace, pod, container"
}

// buildExistingRulesContext builds a context string from existing rules.
func (s *RuleGeneratorService) buildExistingRulesContext(ctx context.Context) (string, error) {
	rules, _, err := s.ruleSvc.List(ctx, "", "", "", "", "", nil, 1, 100)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("已有告警规则（避免重复）:\n")
	limit := 30
	if len(rules) < limit {
		limit = len(rules)
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", rules[i].Name, rules[i].Expression))
	}

	return sb.String(), nil
}

// buildPresetMatches builds a context string from matching preset rules.
func (s *RuleGeneratorService) buildPresetMatches(ctx context.Context, description string) (string, error) {
	// Extract keywords from description for search
	keywords := extractKeywords(description)
	if len(keywords) == 0 {
		return "暂无匹配的预置规则", nil
	}

	var allMatches []string
	for _, kw := range keywords {
		presets, _, err := s.presetRepo.List(ctx, "", kw, 1, 5)
		if err != nil {
			continue
		}
		for _, p := range presets {
			allMatches = append(allMatches, fmt.Sprintf("  - %s: %s (severity: %s)", p.Name, p.Expression, p.Severity))
		}
	}

	if len(allMatches) == 0 {
		return "暂无匹配的预置规则", nil
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, m := range allMatches {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}

	return "参考预置规则:\n" + strings.Join(unique, "\n"), nil
}

// buildSystemPrompt builds the complete system prompt for rule generation.
func (s *RuleGeneratorService) buildSystemPrompt(labelContext, existingRules, presetMatches string) string {
	return fmt.Sprintf(`你是 SRE 告警规则生成助手。根据用户的自然语言描述，生成标准的告警规则或抑制规则。

%s

%s

%s

输出格式要求（严格 JSON）：
对于告警规则：
{
  "type": "alert",
  "name": "AlertName",
  "expression": "PromQL表达式",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"service": "xxx", "env": "prod", "component": "xxx"},
  "annotations": {"summary": "中文摘要", "description": "详细描述"},
  "confidence": 0.9,
  "description": "规则说明"
}

对于抑制规则：
{
  "type": "inhibition",
  "name": "抑制规则名称",
  "source_labels": ["alertname"],
  "source_value": "SourceAlertName",
  "target_labels": ["label1", "label2"],
  "equal_labels": ["label1", "label2"],
  "confidence": 0.9,
  "description": "抑制规则说明"
}

注意：
- severity 必须是 critical/warning/info 之一
- PromQL 必须使用真实存在的指标名
- labels 必须包含 service, env, component
- for_duration 使用 Go duration 格式（如 1m, 5m, 10m）
- 如果信息不足，在 warnings 中列出需要确认的事项
- 回复中只包含 JSON，不要添加其他文本`, labelContext, existingRules, presetMatches)
}

// postProcessResult normalizes the generated result.
func (s *RuleGeneratorService) postProcessResult(result *RuleGenerateResult) {
	// Normalize severity
	switch result.Severity {
	case "critical", "warning", "info":
		// valid
	default:
		if result.Severity != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("severity '%s' 不标准，已自动修正为 warning", result.Severity))
		}
		result.Severity = "warning"
	}

	// Default for_duration
	if result.Type == "alert" && result.ForDuration == "" {
		result.ForDuration = "0s"
	}

	// Ensure labels map
	if result.Type == "alert" && result.Labels == nil {
		result.Labels = make(map[string]string)
	}

	// Ensure annotations map
	if result.Type == "alert" && result.Annotations == nil {
		result.Annotations = make(map[string]string)
	}

	// Clamp confidence
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}
}

// extractMetricNames extracts PromQL metric names from an expression.
var metricRegexp = regexp.MustCompile(`[a-zA-Z_:][a-zA-Z0-9_:]*`)

func extractMetricNames(expr string) []string {
	// Simple heuristic: find identifiers that look like metric names
	// Exclude PromQL keywords and functions
	keywords := map[string]bool{
		"sum": true, "avg": true, "min": true, "max": true, "count": true,
		"rate": true, "irate": true, "increase": true, "delta": true,
		"by": true, "without": true, "on": true, "ignoring": true,
		"group_left": true, "group_right": true, "bool": true,
		"topk": true, "bottomk": true, "sort": true, "sort_desc": true,
		"abs": true, "ceil": true, "floor": true, "round": true,
		"clamp_min": true, "clamp_max": true, "label_replace": true,
		"label_join": true, "absent": true, "absent_over_time": true,
		"vector": true, "scalar": true, "time": true, "timestamp": true,
		"histogram_quantile": true,
	}

	matches := metricRegexp.FindAllString(expr, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		if keywords[m] {
			continue
		}
		// Skip if it looks like a label value (inside quotes) - rough heuristic
		if seen[m] {
			continue
		}
		seen[m] = true
		// Only include strings that contain at least one underscore or look like metrics
		if strings.Contains(m, "_") || (len(m) > 2 && strings.ToLower(m) == m) {
			result = append(result, m)
		}
	}
	return result
}

// stopWords is a set of common Chinese and English stop words used by extractKeywords
// to filter out noise from natural language descriptions.
var stopWords = map[string]bool{
	"的": true, "了": true, "在": true, "是": true, "我": true,
	"有": true, "和": true, "就": true, "不": true, "人": true,
	"都": true, "一": true, "一个": true, "上": true, "也": true,
	"很": true, "到": true, "说": true, "要": true, "去": true,
	"你": true, "会": true, "着": true, "没有": true, "看": true,
	"好": true, "自己": true, "这": true, "他": true, "她": true,
	"它": true, "们": true, "那": true, "什么": true,
	"怎么": true, "如何": true, "请": true, "帮": true, "生成": true,
	"创建": true, "添加": true, "规则": true, "告警": true,
	"when": true, "the": true, "a": true, "an": true, "is": true,
	"are": true, "was": true, "were": true, "be": true, "been": true,
	"being": true, "have": true, "has": true, "had": true, "do": true,
	"does": true, "did": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "shall": true, "can": true,
	"need": true, "must": true, "it": true, "its": true, "this": true,
	"that": true, "these": true, "those": true, "i": true, "me": true,
	"my": true, "we": true, "our": true, "you": true, "your": true,
	"he": true, "him": true, "his": true, "she": true, "her": true,
	"they": true, "them": true, "their": true, "what": true, "which": true,
	"who": true, "whom": true, "where": true, "why": true,
	"how": true, "all": true, "each": true, "every": true, "both": true,
	"few": true, "more": true, "most": true, "other": true, "some": true,
	"such": true, "no": true, "nor": true, "not": true, "only": true,
	"own": true, "same": true, "so": true, "than": true, "too": true,
	"very": true, "just": true, "because": true, "as": true, "until": true,
	"while": true, "of": true, "at": true, "by": true, "for": true,
	"with": true, "about": true, "against": true, "between": true,
	"through": true, "during": true, "before": true, "after": true,
	"above": true, "below": true, "to": true, "from": true, "up": true,
	"down": true, "in": true, "out": true, "on": true, "off": true,
	"over": true, "under": true, "again": true, "further": true,
	"then": true, "once": true, "here": true, "there": true,
}

// extractKeywords extracts search keywords from a natural language description.
func extractKeywords(desc string) []string {
	words := strings.Fields(desc)
	var keywords []string
	seen := make(map[string]bool)
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,!?;:\"'()[]{}"))
		if len(w) < 2 || stopWords[w] {
			continue
		}
		if !seen[w] {
			seen[w] = true
			keywords = append(keywords, w)
		}
	}

	// Limit to 5 keywords
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}
	return keywords
}
