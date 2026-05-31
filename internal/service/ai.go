package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
)

// AlertAnalysis represents the structured output of an LLM alert analysis.
type AlertAnalysis struct {
	Summary          string   `json:"summary"`           // 1-2 sentence summary
	Severity         string   `json:"severity"`          // LLM's assessment: critical/warning/info
	ProbableCauses   []string `json:"probable_causes"`   // ranked list of probable causes
	Impact           string   `json:"impact"`            // impact assessment
	RecommendedSteps []string `json:"recommended_steps"` // recommended SOP steps
	RootCauseHint    string   `json:"root_cause_hint"`   // root cause hypothesis
}

// AIService provides AI/LLM integration for alert analysis.
// Configuration is loaded from the DB via SystemSettingService on every call,
// so changes made in the Web UI take effect immediately without a restart.
type AIService struct {
	settingSvc   *SystemSettingService
	toolRegistry *AIToolRegistry
	client       *http.Client
	logger       *zap.Logger
}

// SetToolRegistry 注入工具注册表（延迟注入，避免 DI 循环依赖）
func (s *AIService) SetToolRegistry(registry *AIToolRegistry) {
	s.toolRegistry = registry
}

// ListTools 返回所有已注册的 AI 工具元数据（供 API 暴露）
func (s *AIService) ListTools() []*AITool {
	if s.toolRegistry == nil {
		return nil
	}
	return s.toolRegistry.List()
}

// NewAIService creates a new AIService backed by DB-stored configuration.
func NewAIService(settingSvc *SystemSettingService, logger *zap.Logger) *AIService {
	return &AIService{
		settingSvc: settingSvc,
		client:     safehttp.NewInternalClient(30 * time.Second),
		logger:     logger,
	}
}

// loadConfig fetches the current AI config from the DB (default provider).
func (s *AIService) loadConfig(ctx context.Context) (AIConfig, error) {
	return s.settingSvc.GetAIConfig(ctx)
}

// loadProviderConfig fetches the config for a specific provider by key.
// If providerKey is empty, the default provider is used.
func (s *AIService) loadProviderConfig(ctx context.Context, providerKey string) (AIProviderConfig, error) {
	return s.settingSvc.GetProviderConfig(ctx, providerKey)
}

// truncateResp truncates an API response body for safe inclusion in error messages (M2).
func truncateResp(body []byte, maxLen int) string {
	s := string(body)
	if len(s) > maxLen {
		return s[:maxLen] + "...(truncated)"
	}
	return s
}

// providerToAIConfig converts an AIProviderConfig to the legacy AIConfig struct.
func providerToAIConfig(p AIProviderConfig) AIConfig {
	return AIConfig{
		Provider:        p.Provider,
		APIKey:          p.APIKey,
		BaseURL:         p.BaseURL,
		Model:           p.Model,
		Enabled:         p.Enabled,
		Temperature:     p.Temperature,
		MaxTokens:       p.MaxTokens,
		TopP:            p.TopP,
		SystemPrompt:    p.SystemPrompt,
		RetryMax:        p.RetryMax,
		ContextMaxChars: p.ContextMaxChars,
	}
}

// GetAIModules returns the AI module configuration.
func (s *AIService) GetAIModules(ctx context.Context) (*AIModuleConfig, error) {
	return s.settingSvc.GetAIModules(ctx)
}

// UpdateAIModules persists the AI module configuration.
func (s *AIService) UpdateAIModules(ctx context.Context, cfg *AIModuleConfig) error {
	return s.settingSvc.UpdateAIModules(ctx, cfg)
}

// GetConfig returns the current AI configuration with the API key masked.
func (s *AIService) GetConfig(ctx context.Context) (AIConfig, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return AIConfig{}, err
	}
	// Mask the API key for display
	if cfg.APIKey != "" {
		if len(cfg.APIKey) > 8 {
			cfg.APIKey = cfg.APIKey[:4] + "****" + cfg.APIKey[len(cfg.APIKey)-4:]
		} else {
			cfg.APIKey = "****"
		}
	}
	return cfg, nil
}

// UpdateConfig persists the AI configuration to the DB.
func (s *AIService) UpdateConfig(ctx context.Context, cfg AIConfig) error {
	return s.settingSvc.SaveAIConfig(ctx, cfg)
}

// GetProvidersConfig returns the multi-provider AI configuration with API keys masked.
func (s *AIService) GetProvidersConfig(ctx context.Context) (AIProvidersConfig, error) {
	cfg, err := s.settingSvc.GetProvidersConfig(ctx)
	if err != nil {
		return AIProvidersConfig{}, err
	}
	// Mask API keys for display
	for i := range cfg.Providers {
		maskAPIKey(&cfg.Providers[i].APIKey)
	}
	return cfg, nil
}

// SaveProvidersConfig persists the multi-provider AI configuration to DB.
func (s *AIService) SaveProvidersConfig(ctx context.Context, cfg AIProvidersConfig) error {
	return s.settingSvc.SaveProvidersConfig(ctx, cfg)
}

// maskAPIKey masks an API key string in-place for safe display.
func maskAPIKey(key *string) {
	if *key == "" {
		return
	}
	if len(*key) > 8 {
		*key = (*key)[:4] + "****" + (*key)[len(*key)-4:]
	} else {
		*key = "****"
	}
}

// GenerateAlertReport generates an alert report using the configured LLM.
func (s *AIService) GenerateAlertReport(ctx context.Context, event *model.AlertEvent) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return "", fmt.Errorf("AI 功能未启用，请在系统设置中配置并启用 AI")
	}

	prompt := fmt.Sprintf(
		"You are an SRE assistant. Generate a concise incident report for the following alert:\n\n"+
			"Alert Name: %s\nSeverity: %s\nStatus: %s\nLabels: %v\nAnnotations: %v\nFired At: %s\nFire Count: %d\n\n"+
			"Please provide:\n1. Summary\n2. Impact Assessment\n3. Potential Root Causes\n4. Recommended Actions",
		event.AlertName,
		event.Severity,
		event.Status,
		event.Labels,
		event.Annotations,
		event.FiredAt.Format(time.RFC3339),
		event.FireCount,
	)

	return s.callLLM(ctx, cfg, prompt)
}

// SuggestSOP suggests Standard Operating Procedure steps for handling an alert.
func (s *AIService) SuggestSOP(ctx context.Context, event *model.AlertEvent) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return "", fmt.Errorf("AI 功能未启用，请在系统设置中配置并启用 AI")
	}

	prompt := fmt.Sprintf(
		"You are an SRE assistant. Suggest a step-by-step Standard Operating Procedure (SOP) for handling the following alert:\n\n"+
			"Alert Name: %s\nSeverity: %s\nLabels: %v\nAnnotations: %v\n\n"+
			"Please provide numbered steps that an on-call engineer should follow to diagnose and resolve this alert.",
		event.AlertName,
		event.Severity,
		event.Labels,
		event.Annotations,
	)

	return s.callLLM(ctx, cfg, prompt)
}

// AnalyzeAlert performs root cause analysis on an alert.
func (s *AIService) AnalyzeAlert(ctx context.Context, event *model.AlertEvent) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return "", fmt.Errorf("AI 功能未启用，请在系统设置中配置并启用 AI")
	}

	prompt := fmt.Sprintf(
		"You are an SRE assistant specialized in root cause analysis. Analyze the following alert:\n\n"+
			"Alert Name: %s\nSeverity: %s\nExpression/Labels: %v\nAnnotations: %v\nFired At: %s\n\n"+
			"Please provide:\n1. Likely Root Causes (ranked by probability)\n2. Diagnostic Commands to Run\n3. Correlation with Common Failure Patterns",
		event.AlertName,
		event.Severity,
		event.Labels,
		event.Annotations,
		event.FiredAt.Format(time.RFC3339),
	)

	return s.callLLM(ctx, cfg, prompt)
}

// TestConnection tests connectivity to the configured AI provider (default provider).
func (s *AIService) TestConnection(ctx context.Context) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return fmt.Errorf("AI is not enabled")
	}
	// Anthropic 有默认 base URL，不需要强制配置
	if cfg.Provider != "anthropic" && cfg.BaseURL == "" {
		return fmt.Errorf("AI base URL is not configured")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("AI API key is not configured")
	}

	// Try a minimal completion request to test connectivity
	_, err = s.callLLM(ctx, cfg, "Say hello in one word.")
	return err
}

// TestProviderConnection tests connectivity to a specific provider by key.
func (s *AIService) TestProviderConnection(ctx context.Context, providerKey string) error {
	provider, err := s.loadProviderConfig(ctx, providerKey)
	if err != nil {
		return fmt.Errorf("failed to load provider config: %w", err)
	}
	if !provider.Enabled {
		return fmt.Errorf("provider %q is not enabled", providerKey)
	}
	// Anthropic 有默认 base URL，不需要强制配置
	if provider.Provider != "anthropic" && provider.BaseURL == "" {
		return fmt.Errorf("provider %q base URL is not configured", providerKey)
	}

	cfg := providerToAIConfig(provider)
	_, err = s.callLLM(ctx, cfg, "Say hello in one word.")
	return err
}

// chatCompletionRequest represents an OpenAI-compatible chat completion request.
type chatCompletionRequest struct {
	Model       string                   `json:"model"`
	Messages    []ChatMessage            `json:"messages"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
	Temperature *float64                 `json:"temperature,omitempty"`
	MaxTokens   *int                     `json:"max_tokens,omitempty"`
	TopP        *float64                 `json:"top_p,omitempty"`
}

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool call from the LLM.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// chatCompletionResponse represents an OpenAI-compatible chat completion response.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ---- Anthropic API 类型定义 ----

// anthropicRequest represents an Anthropic Messages API request.
type anthropicRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	System      string           `json:"system,omitempty"`
	Messages    []ChatMessage    `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
}

// anthropicResponse represents an Anthropic Messages API response.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// anthropicDefaultMaxTokens is the default max_tokens for Anthropic API calls
// when the user has not configured a value. Anthropic requires max_tokens > 0.
const anthropicDefaultMaxTokens = 4096

// AnalyzeAlertWithContext performs LLM analysis with full metric context.
func (s *AIService) AnalyzeAlertWithContext(ctx context.Context, contextText string) (*AlertAnalysis, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("AI is not enabled")
	}

	// Truncate context to configured budget
	if cfg.ContextMaxChars > 0 && len(contextText) > cfg.ContextMaxChars {
		contextText = contextText[:cfg.ContextMaxChars] + "\n...(truncated)"
	}

	systemPrompt := `You are an expert SRE assistant for a monitoring platform. You analyze alerts with their associated metric data.
Your task is to:
1. Provide a brief summary of the situation
2. Assess the probable root causes based on the metric trends
3. Evaluate the impact
4. Recommend specific diagnostic and remediation steps

Respond in JSON format:
{
  "summary": "...",
  "severity": "critical|warning|info",
  "probable_causes": ["cause1", "cause2"],
  "impact": "...",
  "recommended_steps": ["step1", "step2"],
  "root_cause_hint": "..."
}

Respond in Chinese (简体中文).`

	var analysis AlertAnalysis
	if err := s.callLLMJSON(ctx, cfg, systemPrompt, contextText, &analysis); err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return &analysis, nil
}

// trimHistory drops the oldest messages from history so that the total character
// budget (system prompt + history + user message) stays within maxChars.
// It preserves tool-call pairs (assistant with tool_calls → matching tool responses).
// maxChars <= 0 means no trimming.
func trimHistory(systemPrompt string, history []ChatMessage, userMessage string, maxChars int) []ChatMessage {
	if maxChars <= 0 {
		return history
	}
	// Rough estimate: 1 token ~ 4 chars (English/Chinese mix). We work in chars for simplicity.
	budget := maxChars - len(systemPrompt) - len(userMessage)
	if budget <= 0 {
		return nil // no room for history
	}
	// Walk from the end, keeping messages that fit within budget.
	total := 0
	cutoff := len(history)
	for i := len(history) - 1; i >= 0; i-- {
		msgLen := len(history[i].Content) + len(history[i].ToolCallID)
		for _, tc := range history[i].ToolCalls {
			msgLen += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
		// Reserve ~100 chars overhead per message (role, JSON framing)
		msgLen += 100
		if total+msgLen > budget {
			// This message would exceed the budget. But if it's a tool response,
			// try to also drop the preceding assistant tool_call to keep pairs intact.
			if history[i].Role == "tool" && i > 0 && history[i-1].Role == "assistant" {
				i-- // skip the assistant tool_call message too
			}
			cutoff = i + 1
			break
		}
		total += msgLen
		cutoff = i
	}
	if cutoff <= 0 {
		return nil
	}
	return history[cutoff:]
}

// Chat sends a multi-turn conversation to the LLM and returns the assistant reply.
// The caller supplies the system prompt, conversation history, and the new user message.
// 根据 provider 类型分发：anthropic 走原生 Messages API，其余走 OpenAI 兼容协议。
func (s *AIService) Chat(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return "", fmt.Errorf("AI is not enabled")
	}

	// B8-17: Trim history to fit within context window budget.
	history = trimHistory(systemPrompt, history, userMessage, cfg.ContextMaxChars)

	// Anthropic 走原生 Messages API（system 独立字段）
	if cfg.Provider == "anthropic" {
		return s.chatAnthropic(ctx, cfg, systemPrompt, history, userMessage)
	}

	messages := make([]ChatMessage, 0, 1+len(history)+1)
	messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	messages = append(messages, history...)
	messages = append(messages, ChatMessage{Role: "user", Content: userMessage})

	reqBody := chatCompletionRequest{
		Model:    cfg.Model,
		Messages: messages,
	}
	if cfg.Temperature > 0 {
		reqBody.Temperature = &cfg.Temperature
	}
	if cfg.MaxTokens > 0 {
		reqBody.MaxTokens = &cfg.MaxTokens
	}
	if cfg.TopP > 0 && cfg.TopP < 1.0 {
		reqBody.TopP = &cfg.TopP
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	url := baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call AI API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB max
	if err != nil {
		return "", fmt.Errorf("failed to read AI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, truncateResp(respBody, 200))
	}

	var completionResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResp); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	if completionResp.Error != nil {
		return "", fmt.Errorf("AI API error: %s", completionResp.Error.Message)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("AI API returned no choices")
	}

	// Record token usage metrics
	if completionResp.Usage != nil {
		metrics.IncAITokensUsed(cfg.Provider, "prompt", completionResp.Usage.PromptTokens)
		metrics.IncAITokensUsed(cfg.Provider, "completion", completionResp.Usage.CompletionTokens)
	}

	return completionResp.Choices[0].Message.Content, nil
}

// callLLMJSON sends a prompt to the LLM and parses the JSON response into the target.
// It includes retry logic and handles markdown code block wrapping that LLMs sometimes add.
func (s *AIService) callLLMJSON(ctx context.Context, cfg AIConfig, systemPrompt, userPrompt string, target interface{}) error {
	maxRetries := cfg.RetryMax
	if maxRetries <= 0 {
		maxRetries = 2
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 3s, ...
			time.Sleep(time.Duration(attempt) * time.Second)
			s.logger.Info("retrying LLM JSON call",
				zap.Int("attempt", attempt+1),
				zap.Error(lastErr),
			)
		}

		raw, err := s.callLLMWithSystem(ctx, cfg, systemPrompt, userPrompt)
		if err != nil {
			lastErr = err
			continue
		}

		// Strip markdown code blocks if present (```json ... ``` or ``` ... ```)
		cleaned := stripMarkdownCodeBlock(raw)

		if err := json.Unmarshal([]byte(cleaned), target); err != nil {
			lastErr = fmt.Errorf("failed to parse LLM JSON response: %w (raw: %s)", err, truncateString(raw, 200))
			continue
		}

		return nil
	}

	return fmt.Errorf("LLM JSON call failed after %d attempts: %w", maxRetries+1, lastErr)
}

// stripMarkdownCodeBlock removes markdown code block wrapping from LLM output.
// Handles fenced blocks (3+ backticks with optional language tag),
// indented code blocks (4-space indent), nested backticks,
// mixed line endings, and empty code blocks.
func stripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Normalize line endings (\r\n, \r -> \n)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	// --- Try fenced code block (3+ backticks) ---
	firstLine := strings.TrimRight(lines[0], " \t")
	openTicks := countLeadingBackticks(firstLine)
	if openTicks >= 3 {
		tag := firstLine[openTicks:]
		// Opening fence must be only backticks + optional language tag (no backticks in tag)
		if !strings.Contains(tag, "`") {
			// Find matching closing fence: a line of only backticks with count >= opening
			closingIdx := -1
			for i := 1; i < len(lines); i++ {
				trimmed := strings.TrimRight(lines[i], " \t")
				if countLeadingBackticks(trimmed) >= openTicks && strings.TrimLeft(trimmed, "`") == "" {
					closingIdx = i
					break
				}
			}
			if closingIdx != -1 {
				return strings.TrimSpace(strings.Join(lines[1:closingIdx], "\n"))
			}
			// No closing fence found — strip opening line only
			return strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
	}

	// --- Try indented code block (all non-empty lines indented 4 spaces) ---
	allIndented := true
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "    ") {
			allIndented = false
			break
		}
	}
	if allIndented {
		for i, line := range lines {
			if strings.HasPrefix(line, "    ") {
				lines[i] = line[4:]
			}
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}

	return strings.TrimSpace(s)
}

// countLeadingBackticks returns the number of leading backtick characters in s.
func countLeadingBackticks(s string) int {
	count := 0
	for _, c := range s {
		if c == '`' {
			count++
		} else {
			break
		}
	}
	return count
}

// truncateString truncates a string to maxLen, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// callLLMWithSystem sends a prompt to the configured LLM API with a custom system prompt.
// 根据 provider 类型分发：anthropic 走原生 Messages API，其余走 OpenAI 兼容协议。
func (s *AIService) callLLMWithSystem(ctx context.Context, cfg AIConfig, systemPrompt, userPrompt string) (string, error) {
	if cfg.Provider == "anthropic" {
		return s.callLLMAnthropic(ctx, cfg, systemPrompt, userPrompt)
	}
	reqBody := chatCompletionRequest{
		Model: cfg.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	if cfg.Temperature > 0 {
		reqBody.Temperature = &cfg.Temperature
	}
	if cfg.MaxTokens > 0 {
		reqBody.MaxTokens = &cfg.MaxTokens
	}
	if cfg.TopP > 0 && cfg.TopP < 1.0 {
		reqBody.TopP = &cfg.TopP
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	url := baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call AI API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB max
	if err != nil {
		return "", fmt.Errorf("failed to read AI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, truncateResp(respBody, 200))
	}

	var completionResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &completionResp); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	if completionResp.Error != nil {
		return "", fmt.Errorf("AI API error: %s", completionResp.Error.Message)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("AI API returned no choices")
	}

	// Record token usage metrics
	if completionResp.Usage != nil {
		metrics.IncAITokensUsed(cfg.Provider, "prompt", completionResp.Usage.PromptTokens)
		metrics.IncAITokensUsed(cfg.Provider, "completion", completionResp.Usage.CompletionTokens)
	}

	return completionResp.Choices[0].Message.Content, nil
}

// callLLMAnthropic sends a prompt to the Anthropic Messages API.
// 与 OpenAI 兼容协议的主要区别：
//   - system prompt 是独立顶层字段，不在 messages 数组中
//   - 认证使用 x-api-key 头而非 Authorization: Bearer
//   - 响应格式为 content[0].text 而非 choices[0].message.content
//   - 必须显式指定 max_tokens（Anthropic 要求 > 0）
func (s *AIService) callLLMAnthropic(ctx context.Context, cfg AIConfig, systemPrompt, userPrompt string) (string, error) {
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = anthropicDefaultMaxTokens
	}

	reqBody := anthropicRequest{
		Model:     cfg.Model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []ChatMessage{
			{Role: "user", Content: userPrompt},
		},
	}
	if cfg.Temperature > 0 {
		reqBody.Temperature = &cfg.Temperature
	}
	if cfg.TopP > 0 && cfg.TopP < 1.0 {
		reqBody.TopP = &cfg.TopP
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal anthropic request: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	url := baseURL + "/v1/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create anthropic request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB max
	if err != nil {
		return "", fmt.Errorf("failed to read Anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API returned status %d: %s", resp.StatusCode, truncateResp(respBody, 200))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	if anthropicResp.Error != nil {
		return "", fmt.Errorf("anthropic API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("anthropic API returned empty content")
	}

	// 记录 token 用量指标
	if anthropicResp.Usage != nil {
		metrics.IncAITokensUsed(cfg.Provider, "prompt", anthropicResp.Usage.InputTokens)
		metrics.IncAITokensUsed(cfg.Provider, "completion", anthropicResp.Usage.OutputTokens)
	}

	return anthropicResp.Content[0].Text, nil
}

// chatAnthropic sends a multi-turn conversation to the Anthropic Messages API.
// Anthropic 要求 system prompt 作为独立顶层字段，messages 数组中只包含 user/assistant 角色。
func (s *AIService) chatAnthropic(ctx context.Context, cfg AIConfig, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = anthropicDefaultMaxTokens
	}

	// 构建消息列表：历史消息 + 新用户消息（不含 system，Anthropic 的 system 是独立字段）
	messages := make([]ChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, ChatMessage{Role: "user", Content: userMessage})

	reqBody := anthropicRequest{
		Model:     cfg.Model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  messages,
	}
	if cfg.Temperature > 0 {
		reqBody.Temperature = &cfg.Temperature
	}
	if cfg.TopP > 0 && cfg.TopP < 1.0 {
		reqBody.TopP = &cfg.TopP
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal anthropic chat request: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	url := baseURL + "/v1/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create anthropic chat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB max
	if err != nil {
		return "", fmt.Errorf("failed to read Anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API returned status %d: %s", resp.StatusCode, truncateResp(respBody, 200))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	if anthropicResp.Error != nil {
		return "", fmt.Errorf("anthropic API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("anthropic API returned empty content")
	}

	// 记录 token 用量指标
	if anthropicResp.Usage != nil {
		metrics.IncAITokensUsed(cfg.Provider, "prompt", anthropicResp.Usage.InputTokens)
		metrics.IncAITokensUsed(cfg.Provider, "completion", anthropicResp.Usage.OutputTokens)
	}

	return anthropicResp.Content[0].Text, nil
}

// defaultSystemPrompt is used by callLLM for general-purpose SRE assistance.
const defaultSystemPrompt = "You are a helpful SRE assistant for an alert management platform."

// callLLM sends a prompt to the configured OpenAI-compatible API using the
// default system prompt. It delegates to callLLMWithSystem.
func (s *AIService) callLLM(ctx context.Context, cfg AIConfig, prompt string) (string, error) {
	return s.callLLMWithSystem(ctx, cfg, defaultSystemPrompt, prompt)
}

// ToolCallRecord 记录一次工具调用的详情
type ToolCallRecord struct {
	ToolName string `json:"tool_name"`
	Params   string `json:"params"`
	Result   string `json:"result"`
}

// callLLMWithToolsCustom 与 callLLMWithTools 类似，但接受自定义工具执行器和工具定义，
// 返回最终文本回答和所有工具调用记录。供 RunUntilDone 等场景使用。
//
// TODO(B8-13): Anthropic tool calling is not supported. The Anthropic Messages API uses a different
// tool format (content blocks with type "tool_use" / "tool_result") than OpenAI's function calling.
// This method only implements the OpenAI-compatible chat/completions protocol. When provider is
// "anthropic" and tools are non-empty, we log a warning and fall back to a non-tool single-shot call.
func (s *AIService) callLLMWithToolsCustom(
	ctx context.Context,
	cfg AIConfig,
	systemPrompt, userPrompt string,
	tools []map[string]interface{},
	executor func(ctx context.Context, name string, params map[string]interface{}) (string, error),
	maxRounds int,
) (string, []ToolCallRecord, error) {
	// B8-13: Anthropic does not support OpenAI-format tool calling.
	// Fall back to non-tool mode with a warning.
	if cfg.Provider == "anthropic" && len(tools) > 0 {
		s.logger.Warn("Anthropic provider does not support OpenAI-format tool calling, falling back to non-tool mode",
			zap.String("provider", cfg.Provider),
			zap.Int("tools_count", len(tools)),
		)
		result, err := s.callLLMWithSystem(ctx, cfg, systemPrompt, userPrompt)
		return result, nil, err
	}
	if maxRounds <= 0 {
		maxRounds = 5
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	var records []ToolCallRecord

	for round := 0; round < maxRounds; round++ {
		reqBody := chatCompletionRequest{
			Model:    cfg.Model,
			Messages: messages,
			Tools:    tools,
		}
		if cfg.Temperature > 0 {
			reqBody.Temperature = &cfg.Temperature
		}
		if cfg.MaxTokens > 0 {
			reqBody.MaxTokens = &cfg.MaxTokens
		}
		if cfg.TopP > 0 && cfg.TopP < 1.0 {
			reqBody.TopP = &cfg.TopP
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return "", records, fmt.Errorf("failed to marshal request: %w", err)
		}

		baseURL := strings.TrimRight(cfg.BaseURL, "/")
		url := baseURL + "/chat/completions"

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", records, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return "", records, fmt.Errorf("failed to call AI API: %w", err)
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		_ = resp.Body.Close()
		if err != nil {
			return "", records, fmt.Errorf("failed to read AI response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", records, fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, truncateResp(respBody, 200))
		}

		var completionResp chatCompletionResponse
		if err := json.Unmarshal(respBody, &completionResp); err != nil {
			return "", records, fmt.Errorf("failed to parse AI response: %w", err)
		}
		if completionResp.Error != nil {
			return "", records, fmt.Errorf("AI API error: %s", completionResp.Error.Message)
		}
		if len(completionResp.Choices) == 0 {
			return "", records, fmt.Errorf("AI API returned no choices")
		}

		if completionResp.Usage != nil {
			metrics.IncAITokensUsed(cfg.Provider, "prompt", completionResp.Usage.PromptTokens)
			metrics.IncAITokensUsed(cfg.Provider, "completion", completionResp.Usage.CompletionTokens)
		}

		assistantMsg := completionResp.Choices[0].Message

		if len(assistantMsg.ToolCalls) == 0 {
			return assistantMsg.Content, records, nil
		}

		messages = append(messages, ChatMessage{
			Role:      "assistant",
			Content:   assistantMsg.Content,
			ToolCalls: assistantMsg.ToolCalls,
		})

		for _, tc := range assistantMsg.ToolCalls {
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				params = map[string]interface{}{}
			}

			result, execErr := executor(ctx, tc.Function.Name, params)
			if execErr != nil {
				result = fmt.Sprintf("工具执行失败: %v", execErr)
			}

			records = append(records, ToolCallRecord{
				ToolName: tc.Function.Name,
				Params:   tc.Function.Arguments,
				Result:   truncateString(result, 5000),
			})

			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	s.logger.Warn("AI 工具调用达到最大轮次", zap.Int("max_rounds", maxRounds))
	return "工具调用轮次已达上限，请基于已有信息生成报告。", records, nil
}
