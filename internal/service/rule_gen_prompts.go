package service

import "fmt"

// fewShotAlertRule returns a few-shot prompt section for alert rule generation.
func fewShotAlertRule(labels []string) string {
	labelHint := ""
	if len(labels) > 0 {
		labelHint = fmt.Sprintf("\n可用标签（从数据源同步）: %v\n建议在 expression 和 labels 中使用这些标签。", labels)
	}
	return fmt.Sprintf(`## 示例

用户需求: "监控 Redis 内存使用率超过 80%% 持续 5 分钟"
输出:
{
  "name": "RedisHighMemoryUsage",
  "expression": "redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"component": "redis", "team": "infra"},
  "annotations": {
    "summary": "Redis 内存使用率超过 80%%",
    "description": "实例 {{ $labels.instance }} 内存使用率 {{ $value | printf \"%%.1f\" }}%%"
  }
}

用户需求: "CPU 使用率超过 90%% 告警"
输出:
{
  "name": "HighCPUUsage",
  "expression": "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 90",
  "for_duration": "5m",
  "severity": "warning",
  "labels": {"component": "host"},
  "annotations": {
    "summary": "CPU 使用率超过 90%%",
    "description": "实例 {{ $labels.instance }} CPU 使用率 {{ $value | printf \"%%.1f\" }}%%"
  }
}
%s`, labelHint)
}

// fewShotInhibition returns a few-shot prompt section for inhibition rule generation.
func fewShotInhibition() string {
	return `## 示例

用户需求: "当节点宕机时，抑制该节点上的所有告警"
输出:
{
  "name": "NodeDownSuppressesNodeAlerts",
  "source_match": {"alertname": "NodeDown"},
  "target_match_re": {"instance": "{{ $labels.instance }}.*"},
  "equal": ["instance"],
  "description": "节点宕机时抑制同实例的其他告警"
}

用户需求: "集群级别告警抑制 Pod 级别告警"
输出:
{
  "name": "ClusterAlertSuppressesPod",
  "source_match": {"alertname": "ClusterUnhealthy"},
  "target_match_re": {"namespace": ".+"},
  "equal": ["cluster"],
  "description": "集群异常时抑制该集群下所有 Pod 告警"
}`
}

// fewShotMute returns a few-shot prompt section for mute rule generation.
func fewShotMute() string {
	return `## 示例

用户需求: "每天凌晨 2-4 点静默所有告警，做维护窗口"
输出:
{
  "name": "DailyMaintenanceWindow",
  "matchers": [],
  "time_periods": [{"start": "02:00", "end": "04:00", "weekdays": ["mon","tue","wed","thu","fri","sat","sun"]}],
  "description": "每日凌晨维护窗口"
}

用户需求: "每周六晚 10 点到周日早 6 点静默 staging 环境告警"
输出:
{
  "name": "StagingWeekendSilence",
  "matchers": [{"name": "env", "value": "staging", "isRegex": false}],
  "time_periods": [{"start": "22:00", "end": "06:00+1d", "weekdays": ["sat"]}],
  "description": "staging 环境周末静默"
}`
}
