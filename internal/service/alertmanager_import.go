package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
)

// AlertmanagerImportService parses Alertmanager YAML configuration and imports
// receivers as Channels and inhibit_rules as InhibitionRules.
type AlertmanagerImportService struct {
	channelSvc   *ChannelService
	inhibRuleSvc *InhibitionRuleService
	logger       *zap.Logger
}

// NewAlertmanagerImportService creates a new AlertmanagerImportService.
func NewAlertmanagerImportService(
	channelSvc *ChannelService,
	inhibRuleSvc *InhibitionRuleService,
	logger *zap.Logger,
) *AlertmanagerImportService {
	return &AlertmanagerImportService{
		channelSvc:   channelSvc,
		inhibRuleSvc: inhibRuleSvc,
		logger:       logger,
	}
}

// AlertmanagerImportResult holds the outcome of an Alertmanager config import.
type AlertmanagerImportResult struct {
	ChannelsCreated    int      `json:"channels_created"`
	InhibitionsCreated int      `json:"inhibitions_created"`
	Warnings           []string `json:"warnings"`
	Errors             []string `json:"errors"`
}

// --- Alertmanager YAML structures ---

// alertmanagerConfig is the top-level Alertmanager configuration.
type alertmanagerConfig struct {
	Global       interface{}        `yaml:"global"`
	Route        *alertRoute        `yaml:"route"`
	InhibitRules []alertInhibitRule `yaml:"inhibit_rules"`
	Receivers    []alertReceiver    `yaml:"receivers"`
}

// alertRoute represents a routing tree node.
type alertRoute struct {
	Receiver        string        `yaml:"receiver"`
	GroupBy         []string      `yaml:"group_by"`
	GroupWait       string        `yaml:"group_wait"`
	GroupInterval   string        `yaml:"group_interval"`
	RepeatInterval  string        `yaml:"repeat_interval"`
	Matchers        []string      `yaml:"matchers"`
	Match           interface{}   `yaml:"match"`
	Continue        bool          `yaml:"continue"`
	Routes          []*alertRoute `yaml:"routes"`
}

// alertReceiver represents a notification receiver.
type alertReceiver struct {
	Name           string                 `yaml:"name"`
	WebhookConfigs []alertWebhookConfig   `yaml:"webhook_configs"`
}

// alertWebhookConfig represents a webhook endpoint.
type alertWebhookConfig struct {
	URL           string `yaml:"url"`
	SendResolved  bool   `yaml:"send_resolved"`
}

// alertInhibitRule represents an Alertmanager inhibition rule.
type alertInhibitRule struct {
	SourceMatchers []string `yaml:"source_matchers"`
	TargetMatchers []string `yaml:"target_matchers"`
	Equal          []string `yaml:"equal"`
}

// ImportConfig parses an Alertmanager YAML config and imports receivers as Channels
// and inhibit_rules as InhibitionRules.
func (s *AlertmanagerImportService) ImportConfig(ctx context.Context, yamlContent []byte, createdBy uint) (*AlertmanagerImportResult, error) {
	var cfg alertmanagerConfig
	if err := yaml.Unmarshal(yamlContent, &cfg); err != nil {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "invalid Alertmanager YAML: "+err.Error())
	}

	result := &AlertmanagerImportResult{}

	// --- Import receivers as Channels ---
	for _, recv := range cfg.Receivers {
		if recv.Name == "" {
			result.Warnings = append(result.Warnings, "skipping receiver with empty name")
			continue
		}

		// Collect webhook URLs
		var webhookURLs []string
		for _, wh := range recv.WebhookConfigs {
			if wh.URL != "" {
				webhookURLs = append(webhookURLs, wh.URL)
			}
		}
		if len(webhookURLs) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("receiver %q has no webhook_configs, skipped", recv.Name))
			continue
		}

		// Store webhook URLs in aggregation_config JSON for reference
		webhookData, err := json.Marshal(map[string]interface{}{
			"webhook_urls": webhookURLs,
			"source":       "alertmanager_import",
		})
		if err != nil {
			s.logger.Error("failed to marshal webhook data", zap.String("receiver", recv.Name), zap.Error(err))
			result.Errors = append(result.Errors, fmt.Sprintf("failed to marshal webhook data for %q: %v", recv.Name, err))
			continue
		}

		ch := &model.Channel{
			Name:             recv.Name,
			Description:      fmt.Sprintf("Imported from Alertmanager receiver %q", recv.Name),
			Status:           model.ChannelStatusActive,
			AccessLevel:      model.ChannelAccessPublic,
			AggregationConfig: string(webhookData),
		}

		if err := s.channelSvc.Create(ctx, ch); err != nil {
			// Treat duplicate name as a warning, not a hard error
			if appErr, ok := err.(*apperr.AppError); ok && appErr.Code == apperr.ErrDuplicateName.Code {
				result.Warnings = append(result.Warnings, fmt.Sprintf("channel %q already exists, skipped", recv.Name))
				continue
			}
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create channel %q: %v", recv.Name, err))
			continue
		}
		result.ChannelsCreated++
	}

	// --- Import inhibit_rules as InhibitionRules ---
	for i, rule := range cfg.InhibitRules {
		sourceMatch, err := parseAlertmanagerMatchers(rule.SourceMatchers)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("inhibit_rule[%d] source_matchers parse error: %v", i, err))
			continue
		}
		targetMatch, err := parseAlertmanagerMatchers(rule.TargetMatchers)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("inhibit_rule[%d] target_matchers parse error: %v", i, err))
			continue
		}

		equalLabels := strings.Join(rule.Equal, ",")

		// Generate a descriptive name
		name := fmt.Sprintf("imported-inhibit-%d", i+1)
		if len(rule.SourceMatchers) > 0 {
			name = fmt.Sprintf("inhibit: %s", rule.SourceMatchers[0])
		}

		description := fmt.Sprintf("Imported from Alertmanager inhibit_rule[%d]", i)

		ir := &model.InhibitionRule{
			Name:        name,
			Description: description,
			SourceMatch: sourceMatch,
			TargetMatch: targetMatch,
			EqualLabels: equalLabels,
			IsEnabled:   true,
			CreatedBy:   createdBy,
		}

		if err := s.inhibRuleSvc.Create(ctx, ir); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("inhibit_rule[%d] create error: %v", i, err))
			continue
		}
		result.InhibitionsCreated++
	}

	s.logger.Info("alertmanager config imported",
		zap.Int("channels_created", result.ChannelsCreated),
		zap.Int("inhibitions_created", result.InhibitionsCreated),
		zap.Int("warnings", len(result.Warnings)),
		zap.Int("errors", len(result.Errors)),
	)

	return result, nil
}

// matcherRegex matches Alertmanager matcher expressions like:
//   - label="value"
//   - label=~"regex"
//   - label!="value"
//   - label!~"regex"
var matcherRegex = regexp.MustCompile(`^(!?[\w.]+)\s*(=~?|!~?)\s*["']?(.+?)["']?\s*$`)

// parseAlertmanagerMatchers converts a list of Alertmanager matcher strings
// into a model.JSONLabels map.
//
// Alertmanager matchers support 4 operators:
//   - =  : exact match  (stored as literal value)
//   - != : not equal     (stored with "!" prefix)
//   - =~ : regex match   (stored with "~" prefix)
//   - !~ : not regex     (stored with "!~" prefix)
func parseAlertmanagerMatchers(matchers []string) (model.JSONLabels, error) {
	labels := make(model.JSONLabels)
	for _, m := range matchers {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}

		matches := matcherRegex.FindStringSubmatch(m)
		if matches == nil {
			return nil, fmt.Errorf("cannot parse matcher: %q", m)
		}

		label := strings.TrimSpace(matches[1])
		op := strings.TrimSpace(matches[2])
		value := strings.TrimSpace(matches[3])

		switch op {
		case "=":
			labels[label] = value
		case "!=":
			labels[label] = "!" + value
		case "=~":
			labels[label] = "~" + value
		case "!~":
			labels[label] = "!~" + value
		default:
			return nil, fmt.Errorf("unsupported matcher operator %q in: %q", op, m)
		}
	}
	return labels, nil
}

