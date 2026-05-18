package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// PresetRuleOverride holds optional overrides when applying a preset rule.
type PresetRuleOverride struct {
	DatasourceID uint              `json:"datasource_id"`
	ChannelID    uint              `json:"channel_id"`
	Labels       map[string]string `json:"labels"`   // merge with preset labels
	Severity     string            `json:"severity"` // override severity
}

type PresetRuleService struct {
	repo     *repository.PresetRuleRepository
	ruleRepo *repository.AlertRuleRepository
	dsRepo   *repository.DataSourceRepository
	logger   *zap.Logger
}

func NewPresetRuleService(
	repo *repository.PresetRuleRepository,
	ruleRepo *repository.AlertRuleRepository,
	dsRepo *repository.DataSourceRepository,
	logger *zap.Logger,
) *PresetRuleService {
	return &PresetRuleService{repo: repo, ruleRepo: ruleRepo, dsRepo: dsRepo, logger: logger}
}

func (s *PresetRuleService) List(ctx context.Context, category, search string, page, pageSize int) ([]model.PresetRule, int64, error) {
	return s.repo.List(ctx, category, search, page, pageSize)
}

func (s *PresetRuleService) GetByID(ctx context.Context, id uint) (*model.PresetRule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PresetRuleService) Create(ctx context.Context, rule *model.PresetRule) error {
	if rule.Name == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "name is required")
	}
	if rule.Expression == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "expression is required")
	}
	return s.repo.Create(ctx, rule)
}

func (s *PresetRuleService) Delete(ctx context.Context, id uint) error {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if rule.IsBuiltin {
		return apperr.WithMessage(apperr.ErrBuiltinDelete, "built-in preset rules cannot be deleted")
	}
	return s.repo.Delete(ctx, id)
}

func (s *PresetRuleService) Categories(ctx context.Context) ([]string, error) {
	return s.repo.Categories(ctx)
}

// Apply creates an AlertRule from a preset rule with optional overrides.
func (s *PresetRuleService) Apply(ctx context.Context, presetID uint, override *PresetRuleOverride) (*model.AlertRule, error) {
	preset, err := s.repo.GetByID(ctx, presetID)
	if err != nil {
		return nil, apperr.ErrNotFound
	}

	// Validate datasource if provided
	if override != nil && override.DatasourceID > 0 {
		if _, err := s.dsRepo.GetByID(ctx, override.DatasourceID); err != nil {
			return nil, apperr.WithMessage(apperr.ErrDSNotFound, fmt.Sprintf("datasource ID %d not found", override.DatasourceID))
		}
	}

	rule := &model.AlertRule{
		Name:        preset.Name,
		DisplayName: preset.DisplayName,
		Description: preset.Description,
		Expression:  preset.Expression,
		ForDuration: preset.ForDuration,
		Severity:    model.AlertSeverity(preset.Severity),
		Labels:      preset.Labels,
		Annotations: preset.Annotations,
		Category:    preset.Category,
	}

	// Apply overrides
	if override != nil {
		if override.DatasourceID > 0 {
			dsID := override.DatasourceID
			rule.DataSourceID = &dsID
		}
		if override.ChannelID > 0 {
			chID := override.ChannelID
			rule.ChannelID = &chID
		}
		if override.Severity != "" {
			rule.Severity = model.AlertSeverity(override.Severity)
		}
		if len(override.Labels) > 0 {
			if rule.Labels == nil {
				rule.Labels = make(model.JSONLabels)
			}
			for k, v := range override.Labels {
				rule.Labels[k] = v
			}
		}
	}

	rule.Version = 1
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create alert rule from preset", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Increment usage count (best effort)
	_ = s.repo.IncrementUsage(ctx, presetID)

	return rule, nil
}

// BatchCreate inserts multiple PresetRules at once.
func (s *PresetRuleService) BatchCreate(ctx context.Context, rules []model.PresetRule) error {
	if len(rules) == 0 {
		return nil
	}
	return s.repo.BatchCreate(ctx, rules)
}

// ImportPresetInhibitions creates the 13 built-in inhibition preset rule templates
// based on the traditional platform's Alertmanager inhibit_rules.
// These are stored in preset_rules with category "inhibition".
func (s *PresetRuleService) ImportPresetInhibitions(ctx context.Context) error {
	type presetInhibition struct {
		Name         string
		DisplayName  string
		SourceLabels model.JSONLabels
		TargetLabels model.JSONLabels
		EqualLabels  []string
		Description  string
	}

	presets := []presetInhibition{
		{
			Name:         "host-severity-p0-cascade",
			DisplayName:  "主机严重等级级联 (P0)",
			SourceLabels: model.JSONLabels{"severity": "P0"},
			TargetLabels: model.JSONLabels{"severity": "~P1|P2|P3"},
			EqualLabels:  []string{"biz_project", "category", "instance", "project"},
			Description:  "P0 告警触发时，抑制同实例的 P1/P2/P3 告警",
		},
		{
			Name:         "host-severity-p1-cascade",
			DisplayName:  "主机严重等级级联 (P1)",
			SourceLabels: model.JSONLabels{"severity": "P1"},
			TargetLabels: model.JSONLabels{"severity": "~P2|P3"},
			EqualLabels:  []string{"biz_project", "category", "instance", "project"},
			Description:  "P1 告警触发时，抑制同实例的 P2/P3 告警",
		},
		{
			Name:         "container-severity-p0-cascade",
			DisplayName:  "容器严重等级级联 (P0)",
			SourceLabels: model.JSONLabels{"severity": "P0", "category": "container"},
			TargetLabels: model.JSONLabels{"severity": "~P1|P2|P3", "category": "container"},
			EqualLabels:  []string{"biz_project", "namespace", "pod", "container", "project"},
			Description:  "容器 P0 告警触发时，抑制同 Pod 的 P1/P2/P3 容器告警",
		},
		{
			Name:         "container-severity-p1-cascade",
			DisplayName:  "容器严重等级级联 (P1)",
			SourceLabels: model.JSONLabels{"severity": "P1", "category": "container"},
			TargetLabels: model.JSONLabels{"severity": "~P2|P3", "category": "container"},
			EqualLabels:  []string{"biz_project", "namespace", "pod", "container", "project"},
			Description:  "容器 P1 告警触发时，抑制同 Pod 的 P2/P3 容器告警",
		},
		{
			Name:         "node-exporter-down-cascade",
			DisplayName:  "主机 Down 抑制所有告警",
			SourceLabels: model.JSONLabels{"alertname": "NodeExporterDown"},
			TargetLabels: model.JSONLabels{"severity": "~P0|P1|P2|P3"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "NodeExporterDown 时抑制该主机的所有严重等级告警",
		},
		{
			Name:         "kube-node-notready-container",
			DisplayName:  "K8s 节点 NotReady 抑制容器告警",
			SourceLabels: model.JSONLabels{"alertname": "KubeNodeNotReady"},
			TargetLabels: model.JSONLabels{"category": "container"},
			EqualLabels:  []string{"biz_project", "node", "project"},
			Description:  "K8s 节点 NotReady 时抑制该节点上的容器告警",
		},
		{
			Name:         "kube-node-notready-pod",
			DisplayName:  "K8s 节点 NotReady 抑制 Pod 告警",
			SourceLabels: model.JSONLabels{"alertname": "KubeNodeNotReady"},
			TargetLabels: model.JSONLabels{"category": "pod"},
			EqualLabels:  []string{"biz_project", "node", "project"},
			Description:  "K8s 节点 NotReady 时抑制该节点上的 Pod 告警",
		},
		{
			Name:         "kafka-down-cascade",
			DisplayName:  "Kafka Down 抑制同实例告警",
			SourceLabels: model.JSONLabels{"alertname": "KafkaExporterDown"},
			TargetLabels: model.JSONLabels{"category": "kafka"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "Kafka Down 时抑制同实例的 Kafka 类告警",
		},
		{
			Name:         "redis-down-cascade",
			DisplayName:  "Redis Down 抑制同实例告警",
			SourceLabels: model.JSONLabels{"alertname": "RedisDown"},
			TargetLabels: model.JSONLabels{"category": "redis"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "Redis Down 时抑制同实例的 Redis 类告警",
		},
		{
			Name:         "es-cluster-red-cascade",
			DisplayName:  "ES 集群 Red 抑制 Yellow 告警",
			SourceLabels: model.JSONLabels{"alertname": "ElasticsearchClusterRed"},
			TargetLabels: model.JSONLabels{"alertname": "ElasticsearchClusterYellow"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "ES 集群 Red 时抑制同实例的 Yellow 告警",
		},
		{
			Name:         "mongodb-down-cascade",
			DisplayName:  "MongoDB Down 抑制同实例告警",
			SourceLabels: model.JSONLabels{"alertname": "MongoDBDown"},
			TargetLabels: model.JSONLabels{"category": "mongodb"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "MongoDB Down 时抑制同实例的 MongoDB 类告警",
		},
		{
			Name:         "rabbitmq-down-cascade",
			DisplayName:  "RabbitMQ Down 抑制同实例告警",
			SourceLabels: model.JSONLabels{"alertname": "RabbitMQDown"},
			TargetLabels: model.JSONLabels{"category": "rabbitmq"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "RabbitMQ Down 时抑制同实例的 RabbitMQ 类告警",
		},
		{
			Name:         "probe-failed-cascade",
			DisplayName:  "探测失败抑制延迟/状态码告警",
			SourceLabels: model.JSONLabels{"alertname": "BlackboxHttpProbeFailed"},
			TargetLabels: model.JSONLabels{"alertname": "~BlackboxHttpProbeLatency.*|BlackboxHttpStatus5xx|BlackboxHttpDnsLatencyHigh"},
			EqualLabels:  []string{"biz_project", "instance", "project"},
			Description:  "HTTP 探测失败时抑制同实例的延迟和状态码告警",
		},
	}

	var presetRules []model.PresetRule
	for _, p := range presets {
		// Build expression JSON encoding the inhibition rule structure
		exprData, err := json.Marshal(map[string]interface{}{
			"source_match": p.SourceLabels,
			"target_match": p.TargetLabels,
			"equal_labels": p.EqualLabels,
		})
		if err != nil {
			s.logger.Error("failed to marshal inhibition preset expression", zap.String("name", p.Name), zap.Error(err))
			return fmt.Errorf("marshal preset %q: %w", p.Name, err)
		}

		// Labels encode the raw matcher info for display
		labels := model.JSONLabels{
			"source_match": encodeLabelsToString(p.SourceLabels),
			"target_match": encodeLabelsToString(p.TargetLabels),
			"equal_labels":  strings.Join(p.EqualLabels, ","),
		}

		annotations := model.JSONLabels{
			"expression": string(exprData),
		}

		presetRules = append(presetRules, model.PresetRule{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Category:    "inhibition",
			Expression:  string(exprData),
			Severity:    "info",
			Labels:      labels,
			Annotations: annotations,
			Source:      "preset_inhibition",
			IsBuiltin:   true,
			Description: p.Description,
		})
	}

	if err := s.repo.BatchCreate(ctx, presetRules); err != nil {
		s.logger.Error("failed to import preset inhibitions", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("imported preset inhibition rules", zap.Int("count", len(presetRules)))
	return nil
}

// encodeLabelsToString converts a JSONLabels map to a readable "k=v,k2=v2" string.
func encodeLabelsToString(labels model.JSONLabels) string {
	if len(labels) == 0 {
		return ""
	}
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

// ImportFromYAML parses Prometheus rule YAML and batch creates PresetRules.
func (s *PresetRuleService) ImportFromYAML(ctx context.Context, yamlContent []byte) (int, error) {
	var ruleFile model.PrometheusRuleFile
	if err := yaml.Unmarshal(yamlContent, &ruleFile); err != nil {
		return 0, apperr.WithMessage(apperr.ErrInvalidParam, "invalid YAML: "+err.Error())
	}

	var presets []model.PresetRule
	for _, group := range ruleFile.Groups {
		for _, rule := range group.Rules {
			if rule.Alert == "" {
				continue // skip recording rules
			}

			severity := "warning"
			if s, ok := rule.Labels["severity"]; ok {
				severity = s
			}

			preset := model.PresetRule{
				Name:        rule.Alert,
				DisplayName: rule.Alert,
				Expression:  rule.Expr,
				ForDuration: rule.For,
				Severity:    severity,
				Source:      "yaml_import",
				IsBuiltin:   false,
			}

			if rule.Labels != nil {
				preset.Labels = model.JSONLabels(rule.Labels)
			}
			if rule.Annotations != nil {
				preset.Annotations = model.JSONLabels(rule.Annotations)
			}

			// Check for description in annotations
			if desc, ok := rule.Annotations["description"]; ok {
				preset.Description = desc
			}

			presets = append(presets, preset)
		}
	}

	if len(presets) == 0 {
		return 0, apperr.WithMessage(apperr.ErrInvalidParam, "no valid alerting rules found in YAML")
	}

	if err := s.repo.BatchCreate(ctx, presets); err != nil {
		s.logger.Error("failed to batch import preset rules", zap.Error(err))
		return 0, apperr.Wrap(apperr.ErrDatabase, err)
	}

	return len(presets), nil
}
