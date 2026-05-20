// Package metrics provides Prometheus counters for SREAgent business metrics.
// It tracks alert evaluations, notification deliveries, and escalation steps.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// alertsEvaluatedTotal counts the total number of alert rule evaluations.
	// Labels: rule_id (string), result (string: "firing", "resolved", "nodata", "error", "ok")
	alertsEvaluatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sreagent_alerts_evaluated_total",
			Help: "Total number of alert rule evaluations performed",
		},
		[]string{"rule_id", "result"},
	)

	// notificationsSentTotal counts the total number of notifications sent.
	// Labels: channel_type (string: "lark", "email", "webhook", etc.), status (string: "success", "failure")
	notificationsSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sreagent_notifications_sent_total",
			Help: "Total number of notifications sent by the system",
		},
		[]string{"channel_type", "status"},
	)

	// escalationStepsTotal counts the total number of escalation steps executed.
	// Labels: policy_id (string), status (string: "success", "failure")
	escalationStepsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sreagent_escalation_steps_total",
			Help: "Total number of escalation steps executed",
		},
		[]string{"policy_id", "status"},
	)

	// aiTokensUsedTotal counts the total number of LLM tokens consumed.
	// Labels: provider (string), direction (string: "prompt", "completion")
	aiTokensUsedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sreagent_ai_tokens_used_total",
			Help: "Total number of LLM tokens consumed",
		},
		[]string{"provider", "direction"},
	)

	// engineLeaderStatus indicates whether this instance is the engine leader (1) or not (0).
	engineLeaderStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sreagent_engine_leader_status",
			Help: "Whether this instance is the engine leader (1) or follower (0)",
		},
	)

	// heartbeatChecksTotal counts heartbeat check passes.
	// Labels: result (string: "ok", "missed", "resolved", "error")
	heartbeatChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sreagent_heartbeat_checks_total",
			Help: "Total number of heartbeat checks performed",
		},
		[]string{"result"},
	)

	// heartbeatActiveRules gauges the number of active heartbeat rules being monitored.
	heartbeatActiveRules = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sreagent_heartbeat_active_rules",
			Help: "Number of active heartbeat rules being monitored",
		},
	)
)

func init() {
	prometheus.MustRegister(alertsEvaluatedTotal)
	prometheus.MustRegister(notificationsSentTotal)
	prometheus.MustRegister(escalationStepsTotal)
	prometheus.MustRegister(aiTokensUsedTotal)
	prometheus.MustRegister(engineLeaderStatus)
	prometheus.MustRegister(heartbeatChecksTotal)
	prometheus.MustRegister(heartbeatActiveRules)
}

// IncAlertsEvaluated increments the alert evaluation counter.
// ruleID is the string representation of the rule ID.
// result is one of: "firing", "resolved", "nodata", "error", "ok".
func IncAlertsEvaluated(ruleID, result string) {
	alertsEvaluatedTotal.WithLabelValues(ruleID, result).Inc()
}

// IncNotificationsSent increments the notification counter.
// channelType is the notification channel type (e.g., "lark", "email", "webhook").
// status is "success" or "failure".
func IncNotificationsSent(channelType, status string) {
	notificationsSentTotal.WithLabelValues(channelType, status).Inc()
}

// IncEscalationSteps increments the escalation step counter.
// policyID is the string representation of the escalation policy ID.
// status is "success" or "failure".
func IncEscalationSteps(policyID, status string) {
	escalationStepsTotal.WithLabelValues(policyID, status).Inc()
}

// IncAITokensUsed increments the AI token usage counter.
// provider is the AI provider name (e.g. "openai", "azure").
// direction is "prompt" or "completion".
// count is the number of tokens consumed.
func IncAITokensUsed(provider, direction string, count int) {
	if count > 0 {
		aiTokensUsedTotal.WithLabelValues(provider, direction).Add(float64(count))
	}
}

// SetEngineLeaderStatus sets the engine leader status gauge (1 = leader, 0 = follower).
func SetEngineLeaderStatus(isLeader bool) {
	if isLeader {
		engineLeaderStatus.Set(1)
	} else {
		engineLeaderStatus.Set(0)
	}
}

// IncHeartbeatChecks increments the heartbeat check counter.
// result is one of: "ok", "missed", "resolved", "error".
func IncHeartbeatChecks(result string) {
	heartbeatChecksTotal.WithLabelValues(result).Inc()
}

// SetHeartbeatActiveRules sets the number of active heartbeat rules being monitored.
func SetHeartbeatActiveRules(count int) {
	heartbeatActiveRules.Set(float64(count))
}
