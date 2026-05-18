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
)

func init() {
	prometheus.MustRegister(alertsEvaluatedTotal)
	prometheus.MustRegister(notificationsSentTotal)
	prometheus.MustRegister(escalationStepsTotal)
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
