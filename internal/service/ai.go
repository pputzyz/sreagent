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
	settingSvc *SystemSettingService
	client     *http.Client
	logger     *zap.Logger
}

// NewAIService creates a new AIService backed by DB-stored configuration.
func NewAIService(settingSvc *SystemSettingService, logger *zap.Logger) *AIService {
	return &AIService{
		settingSvc: settingSvc,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// loadConfig fetches the current AI config from the DB.
func (s *AIService) loadConfig(ctx context.Context) (AIConfig, error) {
	return s.settingSvc.GetAIConfig(ctx)
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

// GenerateAlertReport generates an alert report using the configured LLM.
func (s *AIService) GenerateAlertReport(ctx context.Context, event *model.AlertEvent) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		s.logger.Warn("AI is not enabled, returning placeholder report")
		return fmt.Sprintf("[AI Disabled] Alert report placeholder for event: %s (severity: %s)", event.AlertName, event.Severity), nil
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
		s.logger.Warn("AI is not enabled, returning placeholder SOP")
		return fmt.Sprintf("[AI Disabled] SOP suggestion placeholder for event: %s", event.AlertName), nil
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
		s.logger.Warn("AI is not enabled, returning placeholder analysis")
		return fmt.Sprintf("[AI Disabled] Root cause analysis placeholder for event: %s", event.AlertName), nil
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

// TestConnection tests connectivity to the configured AI provider.
func (s *AIService) TestConnection(ctx context.Context) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return fmt.Errorf("AI is not enabled")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("AI base URL is not configured")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("AI API key is not configured")
	}

	// Try a minimal completion request to test connectivity
	_, err = s.callLLM(ctx, cfg, "Say hello in one word.")
	return err
}

// chatCompletionRequest represents an OpenAI-compatible chat completion request.
type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse represents an OpenAI-compatible chat completion response.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// AnalyzeAlertWithContext performs LLM analysis with full metric context.
func (s *AIService) AnalyzeAlertWithContext(ctx context.Context, contextText string) (*AlertAnalysis, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("AI is not enabled")
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

// Chat sends a multi-turn conversation to the LLM and returns the assistant reply.
// The caller supplies the system prompt, conversation history, and the new user message.
func (s *AIService) Chat(ctx context.Context, systemPrompt string, history []ChatMessage, userMessage string) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AI config: %w", err)
	}
	if !cfg.Enabled {
		return "", fmt.Errorf("AI is not enabled")
	}

	messages := make([]ChatMessage, 0, 1+len(history)+1)
	messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	messages = append(messages, history...)
	messages = append(messages, ChatMessage{Role: "user", Content: userMessage})

	reqBody := chatCompletionRequest{
		Model:    cfg.Model,
		Messages: messages,
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read AI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(respBody))
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

	return completionResp.Choices[0].Message.Content, nil
}

// callLLMJSON sends a prompt to the LLM and parses the JSON response into the target.
// It includes retry logic and handles markdown code block wrapping that LLMs sometimes add.
func (s *AIService) callLLMJSON(ctx context.Context, cfg AIConfig, systemPrompt, userPrompt string, target interface{}) error {
	const maxRetries = 2

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
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
func stripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)

	// Handle ```json ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (with optional language tag)
		idx := strings.Index(s, "\n")
		if idx != -1 {
			s = s[idx+1:]
		}
		// Remove closing fence
		if lastIdx := strings.LastIndex(s, "```"); lastIdx != -1 {
			s = s[:lastIdx]
		}
		s = strings.TrimSpace(s)
	}

	return s
}

// truncateString truncates a string to maxLen, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// callLLMWithSystem sends a prompt to the configured OpenAI-compatible API with a custom system prompt.
func (s *AIService) callLLMWithSystem(ctx context.Context, cfg AIConfig, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatCompletionRequest{
		Model: cfg.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read AI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(respBody))
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

	return completionResp.Choices[0].Message.Content, nil
}

// defaultSystemPrompt is used by callLLM for general-purpose SRE assistance.
const defaultSystemPrompt = "You are a helpful SRE assistant for an alert management platform."

// callLLM sends a prompt to the configured OpenAI-compatible API using the
// default system prompt. It delegates to callLLMWithSystem.
func (s *AIService) callLLM(ctx context.Context, cfg AIConfig, prompt string) (string, error) {
	return s.callLLMWithSystem(ctx, cfg, defaultSystemPrompt, prompt)
}
