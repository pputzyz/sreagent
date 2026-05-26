package model

import "encoding/json"

// SeedBuiltinDashboards returns the default built-in dashboard definitions.
// Called by the service layer on first startup when the table is empty.
func SeedBuiltinDashboards() []BuiltinDashboard {
	return []BuiltinDashboard{
		hostMonitoringDashboard(),
		containerMonitoringDashboard(),
		httpMonitoringDashboard(),
		mysqlMonitoringDashboard(),
		redisMonitoringDashboard(),
	}
}

// ──────────────────────── Host Monitoring ────────────────────────

func hostMonitoringDashboard() BuiltinDashboard {
	panels := []panelConfig{
		{
			ID:    1,
			Title: "CPU Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle",instance="$instance"}[5m])) * 100)`, LegendFormat: "CPU Usage"},
			},
			FieldConfig: fieldConfig{Unit: "percent", Min: floatPtr(0), Max: floatPtr(100)},
		},
		{
			ID:    2,
			Title: "Memory Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `(1 - node_memory_MemAvailable_bytes{instance="$instance"} / node_memory_MemTotal_bytes{instance="$instance"}) * 100`, LegendFormat: "Memory Usage"},
			},
			FieldConfig: fieldConfig{Unit: "percent", Min: floatPtr(0), Max: floatPtr(100)},
		},
		{
			ID:    3,
			Title: "Disk Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 8, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `(1 - node_filesystem_avail_bytes{instance="$instance",fstype!~"tmpfs|fuse.lxcfs"} / node_filesystem_size_bytes) * 100`, LegendFormat: "{{mountpoint}}"},
			},
			FieldConfig: fieldConfig{Unit: "percent", Min: floatPtr(0), Max: floatPtr(100)},
		},
		{
			ID:    4,
			Title: "Network Traffic",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 8, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(node_network_receive_bytes_total{instance="$instance",device!~"lo|veth.*|docker.*|br-.*"}[5m])`, LegendFormat: "Receive {{device}}"},
				{Expr: `rate(node_network_transmit_bytes_total{instance="$instance",device!~"lo|veth.*|docker.*|br-.*"}[5m])`, LegendFormat: "Transmit {{device}}"},
			},
			FieldConfig: fieldConfig{Unit: "Bps"},
		},
		{
			ID:    5,
			Title: "System Load",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 16, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `node_load1{instance="$instance"}`, LegendFormat: "Load 1m"},
				{Expr: `node_load5{instance="$instance"}`, LegendFormat: "Load 5m"},
				{Expr: `node_load15{instance="$instance"}`, LegendFormat: "Load 15m"},
			},
		},
		{
			ID:    6,
			Title: "Disk I/O",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 16, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(node_disk_read_bytes_total{instance="$instance"}[5m])`, LegendFormat: "Read {{device}}"},
				{Expr: `rate(node_disk_written_bytes_total{instance="$instance"}[5m])`, LegendFormat: "Write {{device}}"},
			},
			FieldConfig: fieldConfig{Unit: "Bps"},
		},
	}

	variables := []variableConfig{
		{Name: "instance", Label: "Instance", Type: "query", Query: `label_values(node_uname_info, instance)`},
	}

	return BuiltinDashboard{
		Name:      "Host Monitoring",
		Ident:     "host-monitoring",
		Category:  "host",
		Component: "node-exporter",
		Tags:      "host,cpu,memory,disk,network",
		Config:    marshalDashboardConfig(panels, variables),
		Version:   1,
		BuiltIn:   true,
	}
}

// ──────────────────────── Container Monitoring ────────────────────────

func containerMonitoringDashboard() BuiltinDashboard {
	panels := []panelConfig{
		{
			ID:    1,
			Title: "Pod CPU Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(rate(container_cpu_usage_seconds_total{namespace="$namespace",pod=~"$pod",container!=""}[5m])) by(pod,container)`, LegendFormat: "{{pod}} - {{container}}"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    2,
			Title: "Pod Memory Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(container_memory_working_set_bytes{namespace="$namespace",pod=~"$pod",container!=""}) by(pod,container)`, LegendFormat: "{{pod}} - {{container}}"},
			},
			FieldConfig: fieldConfig{Unit: "bytes"},
		},
		{
			ID:    3,
			Title: "Pod Restart Count",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 8, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(kube_pod_container_status_restarts_total{namespace="$namespace",pod=~"$pod"}) by(pod)`, LegendFormat: "{{pod}}"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    4,
			Title: "Pod Status Phase",
			Type:  "stat",
			GridPos: gridPos{X: 12, Y: 8, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `count(kube_pod_status_phase{namespace="$namespace",phase="Running"})`, LegendFormat: "Running"},
				{Expr: `count(kube_pod_status_phase{namespace="$namespace",phase="Pending"})`, LegendFormat: "Pending"},
				{Expr: `count(kube_pod_status_phase{namespace="$namespace",phase="Failed"})`, LegendFormat: "Failed"},
			},
		},
		{
			ID:    5,
			Title: "Container Memory Limit Usage %",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 16, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `container_memory_working_set_bytes{namespace="$namespace",pod=~"$pod",container!=""} / container_spec_memory_limit_bytes{container!=""} * 100`, LegendFormat: "{{pod}} - {{container}}"},
			},
			FieldConfig: fieldConfig{Unit: "percent", Min: floatPtr(0)},
		},
		{
			ID:    6,
			Title: "Network Receive/Transmit",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 16, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(rate(container_network_receive_bytes_total{namespace="$namespace",pod=~"$pod"}[5m])) by(pod)`, LegendFormat: "Receive {{pod}}"},
				{Expr: `sum(rate(container_network_transmit_bytes_total{namespace="$namespace",pod=~"$pod"}[5m])) by(pod)`, LegendFormat: "Transmit {{pod}}"},
			},
			FieldConfig: fieldConfig{Unit: "Bps"},
		},
	}

	variables := []variableConfig{
		{Name: "namespace", Label: "Namespace", Type: "query", Query: `label_values(kube_pod_info, namespace)`},
		{Name: "pod", Label: "Pod", Type: "query", Query: `label_values(kube_pod_info{namespace="$namespace"}, pod)`},
	}

	return BuiltinDashboard{
		Name:      "Container Monitoring",
		Ident:     "container-monitoring",
		Category:  "container",
		Component: "kube-state-metrics",
		Tags:      "kubernetes,container,pod,cpu,memory",
		Config:    marshalDashboardConfig(panels, variables),
		Version:   1,
		BuiltIn:   true,
	}
}

// ──────────────────────── HTTP Monitoring ────────────────────────

func httpMonitoringDashboard() BuiltinDashboard {
	panels := []panelConfig{
		{
			ID:    1,
			Title: "Request Rate",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(rate(http_requests_total{job="$job"}[5m])) by(status)`, LegendFormat: "{{status}}"},
			},
			FieldConfig: fieldConfig{Unit: "reqps"},
		},
		{
			ID:    2,
			Title: "Error Rate (5xx)",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(rate(http_requests_total{job="$job",status=~"5.."}[5m])) / sum(rate(http_requests_total{job="$job"}[5m])) * 100`, LegendFormat: "5xx Error %"},
			},
			FieldConfig: fieldConfig{Unit: "percent", Min: floatPtr(0)},
		},
		{
			ID:    3,
			Title: "Request Latency P50/P90/P99",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 8, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket{job="$job"}[5m])) by(le))`, LegendFormat: "P50"},
				{Expr: `histogram_quantile(0.90, sum(rate(http_request_duration_seconds_bucket{job="$job"}[5m])) by(le))`, LegendFormat: "P90"},
				{Expr: `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job="$job"}[5m])) by(le))`, LegendFormat: "P99"},
			},
			FieldConfig: fieldConfig{Unit: "s"},
		},
		{
			ID:    4,
			Title: "Request Duration Heatmap",
			Type:  "heatmap",
			GridPos: gridPos{X: 12, Y: 8, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(rate(http_request_duration_seconds_bucket{job="$job"}[5m])) by(le)`, LegendFormat: "{{le}}"},
			},
		},
		{
			ID:    5,
			Title: "Active Connections",
			Type:  "stat",
			GridPos: gridPos{X: 0, Y: 16, W: 6, H: 4},
			Targets: []targetConfig{
				{Expr: `sum(http_connections_active{job="$job"})`, LegendFormat: "Active"},
			},
		},
		{
			ID:    6,
			Title: "Total Requests (24h)",
			Type:  "stat",
			GridPos: gridPos{X: 6, Y: 16, W: 6, H: 4},
			Targets: []targetConfig{
				{Expr: `sum(increase(http_requests_total{job="$job"}[24h]))`, LegendFormat: "Total"},
			},
		},
		{
			ID:    7,
			Title: "Status Code Distribution",
			Type:  "piechart",
			GridPos: gridPos{X: 12, Y: 16, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `sum(increase(http_requests_total{job="$job"}[1h])) by(status)`, LegendFormat: "{{status}}"},
			},
		},
	}

	variables := []variableConfig{
		{Name: "job", Label: "Job", Type: "query", Query: `label_values(http_requests_total, job)`},
	}

	return BuiltinDashboard{
		Name:      "HTTP Monitoring",
		Ident:     "http-monitoring",
		Category:  "application",
		Component: "app",
		Tags:      "http,request,latency,error,availability",
		Config:    marshalDashboardConfig(panels, variables),
		Version:   1,
		BuiltIn:   true,
	}
}

// ──────────────────────── MySQL Monitoring ────────────────────────

func mysqlMonitoringDashboard() BuiltinDashboard {
	panels := []panelConfig{
		{
			ID:    1,
			Title: "MySQL Up",
			Type:  "stat",
			GridPos: gridPos{X: 0, Y: 0, W: 6, H: 4},
			Targets: []targetConfig{
				{Expr: `mysql_up{instance="$instance"}`, LegendFormat: "Up"},
			},
		},
		{
			ID:    2,
			Title: "Current Connections",
			Type:  "stat",
			GridPos: gridPos{X: 6, Y: 0, W: 6, H: 4},
			Targets: []targetConfig{
				{Expr: `mysql_global_status_threads_connected{instance="$instance"}`, LegendFormat: "Connected"},
			},
		},
		{
			ID:    3,
			Title: "Queries Per Second",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(mysql_global_status_queries{instance="$instance"}[5m])`, LegendFormat: "QPS"},
				{Expr: `rate(mysql_global_status_questions{instance="$instance"}[5m])`, LegendFormat: "Questions/s"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    4,
			Title: "Connection Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 4, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `mysql_global_status_threads_connected{instance="$instance"}`, LegendFormat: "Connected"},
				{Expr: `mysql_global_status_threads_running{instance="$instance"}`, LegendFormat: "Running"},
				{Expr: `mysql_global_variables_max_connections{instance="$instance"}`, LegendFormat: "Max"},
			},
		},
		{
			ID:    5,
			Title: "Slow Queries",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 12, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(mysql_global_status_slow_queries{instance="$instance"}[5m])`, LegendFormat: "Slow Queries/s"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    6,
			Title: "Replication Lag",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 12, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `mysql_slave_status_seconds_behind_master{instance="$instance"}`, LegendFormat: "Lag (s)"},
			},
			FieldConfig: fieldConfig{Unit: "s"},
		},
		{
			ID:    7,
			Title: "Command Breakdown",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 20, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(mysql_global_status_commands_total{instance="$instance",command=~"select|insert|update|delete"}[5m])`, LegendFormat: "{{command}}"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    8,
			Title: "Buffer Pool Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 20, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `mysql_global_status_innodb_buffer_pool_pages_total{instance="$instance"} * mysql_global_variables_innodb_page_size{instance="$instance"}`, LegendFormat: "Total"},
				{Expr: `mysql_global_status_innodb_buffer_pool_pages_data{instance="$instance"} * mysql_global_variables_innodb_page_size{instance="$instance"}`, LegendFormat: "Data"},
				{Expr: `mysql_global_status_innodb_buffer_pool_pages_dirty{instance="$instance"} * mysql_global_variables_innodb_page_size{instance="$instance"}`, LegendFormat: "Dirty"},
			},
			FieldConfig: fieldConfig{Unit: "bytes"},
		},
	}

	variables := []variableConfig{
		{Name: "instance", Label: "Instance", Type: "query", Query: `label_values(mysql_up, instance)`},
	}

	return BuiltinDashboard{
		Name:      "MySQL Monitoring",
		Ident:     "mysql-monitoring",
		Category:  "database",
		Component: "mysqld-exporter",
		Tags:      "mysql,database,connections,queries,replication",
		Config:    marshalDashboardConfig(panels, variables),
		Version:   1,
		BuiltIn:   true,
	}
}

// ──────────────────────── Redis Monitoring ────────────────────────

func redisMonitoringDashboard() BuiltinDashboard {
	panels := []panelConfig{
		{
			ID:    1,
			Title: "Redis Up",
			Type:  "stat",
			GridPos: gridPos{X: 0, Y: 0, W: 6, H: 4},
			Targets: []targetConfig{
				{Expr: `redis_up{instance="$instance"}`, LegendFormat: "Up"},
			},
		},
		{
			ID:    2,
			Title: "Connected Clients",
			Type:  "stat",
			GridPos: gridPos{X: 6, Y: 0, W: 6, H: 4},
			Targets: []targetConfig{
				{Expr: `redis_connected_clients{instance="$instance"}`, LegendFormat: "Clients"},
			},
		},
		{
			ID:    3,
			Title: "Memory Usage",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 0, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `redis_memory_used_bytes{instance="$instance"}`, LegendFormat: "Used"},
				{Expr: `redis_memory_max_bytes{instance="$instance"}`, LegendFormat: "Max"},
				{Expr: `redis_memory_used_rss_bytes{instance="$instance"}`, LegendFormat: "RSS"},
			},
			FieldConfig: fieldConfig{Unit: "bytes"},
		},
		{
			ID:    4,
			Title: "Commands Per Second",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 4, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(redis_commands_processed_total{instance="$instance"}[5m])`, LegendFormat: "Commands/s"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    5,
			Title: "Hit/Miss Rate",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 12, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(redis_keyspace_hits_total{instance="$instance"}[5m])`, LegendFormat: "Hits/s"},
				{Expr: `rate(redis_keyspace_misses_total{instance="$instance"}[5m])`, LegendFormat: "Misses/s"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    6,
			Title: "Keys per DB",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 12, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `redis_db_keys{instance="$instance"}`, LegendFormat: "db{{db}}"},
			},
			FieldConfig: fieldConfig{Unit: "short"},
		},
		{
			ID:    7,
			Title: "Network I/O",
			Type:  "timeseries",
			GridPos: gridPos{X: 0, Y: 20, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `rate(redis_net_input_bytes_total{instance="$instance"}[5m])`, LegendFormat: "Input"},
				{Expr: `rate(redis_net_output_bytes_total{instance="$instance"}[5m])`, LegendFormat: "Output"},
			},
			FieldConfig: fieldConfig{Unit: "Bps"},
		},
		{
			ID:    8,
			Title: "Command Latency",
			Type:  "timeseries",
			GridPos: gridPos{X: 12, Y: 20, W: 12, H: 8},
			Targets: []targetConfig{
				{Expr: `redis_commands_duration_seconds_total{instance="$instance",cmd="get"} / redis_commands_total{instance="$instance",cmd="get"}`, LegendFormat: "GET"},
				{Expr: `redis_commands_duration_seconds_total{instance="$instance",cmd="set"} / redis_commands_total{instance="$instance",cmd="set"}`, LegendFormat: "SET"},
			},
			FieldConfig: fieldConfig{Unit: "s"},
		},
	}

	variables := []variableConfig{
		{Name: "instance", Label: "Instance", Type: "query", Query: `label_values(redis_up, instance)`},
	}

	return BuiltinDashboard{
		Name:      "Redis Monitoring",
		Ident:     "redis-monitoring",
		Category:  "database",
		Component: "redis-exporter",
		Tags:      "redis,database,memory,keys,connections",
		Config:    marshalDashboardConfig(panels, variables),
		Version:   1,
		BuiltIn:   true,
	}
}

// ──────────────────────── Config helpers ────────────────────────

type dashboardConfig struct {
	Panels    []panelConfig    `json:"panels"`
	Variables []variableConfig `json:"variables"`
}

type panelConfig struct {
	ID          int            `json:"id"`
	Title       string         `json:"title"`
	Type        string         `json:"type"`
	GridPos     gridPos        `json:"gridPos"`
	Targets     []targetConfig `json:"targets"`
	FieldConfig  fieldConfig    `json:"fieldConfig,omitempty"`
}

type gridPos struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type targetConfig struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat,omitempty"`
}

type fieldConfig struct {
	Unit string   `json:"unit,omitempty"`
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
}

type variableConfig struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Query string `json:"query"`
}

func floatPtr(f float64) *float64 { return &f }

func marshalDashboardConfig(panels []panelConfig, variables []variableConfig) string {
	cfg := dashboardConfig{Panels: panels, Variables: variables}
	b, _ := json.Marshal(cfg)
	return string(b)
}
