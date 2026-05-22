package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/bcrypt"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/dbmigrate"
	"github.com/sreagent/sreagent/internal/router"
)

func main() {
	cfgFile := flag.String("config", "", "config file path")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	zapLogger := initLogger(cfg.Log)
	defer func() { _ = zapLogger.Sync() }()

	zapLogger.Info("starting SREAgent server",
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
	)

	// Initialize database
	db, err := initDatabase(cfg.Database)
	if err != nil {
		zapLogger.Fatal("failed to initialize database", zap.Error(err))
	}

	// Run database migrations (golang-migrate, version-tracked).
	migrateDB, err := sql.Open("mysql", cfg.Database.MigrateDSN())
	if err != nil {
		zapLogger.Fatal("failed to open migration db connection", zap.Error(err))
	}
	if err := dbmigrate.RunMigrations(migrateDB, cfg.Database.Database, zapLogger); err != nil {
		_ = migrateDB.Close()
		zapLogger.Fatal("database migration failed", zap.Error(err))
	}
	_ = migrateDB.Close()

	// Auto-migrate any models not covered by SQL migrations (development safety net)
	if err := autoMigrate(db); err != nil {
		zapLogger.Fatal("failed to auto-migrate", zap.Error(err))
	}

	// Seed default admin user
	seedAdminUser(db, zapLogger)

	// Seed built-in preset rules
	seedPresetRules(db, zapLogger)

	// Initialize all dependencies (repos, services, handlers, engine)
	deps, err := initDependencies(cfg, db, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to initialize dependencies", zap.Error(err))
	}

	// Start label registry sync worker (cancels on shutdown via deps.appCtx)
	go deps.LabelRegistrySvc.StartSyncWorker(deps.appCtx, 10*time.Minute)

	// Setup router
	r := router.Setup(cfg, deps.Handlers, zapLogger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Error("failed to start server", zap.Error(err))
			os.Exit(1)
		}
	}()

	zapLogger.Info("server started", zap.String("addr", cfg.Server.Addr()))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down server...")

	// Stop all background workers in the correct order
	deps.Shutdown()

	// Shutdown HTTP server (drain in-flight requests)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("server exited")
}

func initLogger(cfg config.LogConfig) *zap.Logger {
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var zapCfg zap.Config
	if cfg.Format == "console" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	zapCfg.Level.SetLevel(level)

	logger, err := zapCfg.Build()
	if err != nil {
		// Fallback to a basic production logger if config is invalid.
		fallback, _ := zap.NewProduction()
		fallback.Error("failed to build logger from config, using fallback", zap.Error(err))
		return fallback
	}
	return logger
}

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	gormLogLevel := logger.Silent
	if os.Getenv("SREAGENT_DB_DEBUG") == "true" {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(gormmysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	// Phase 1 models
	models := []interface{}{
		&model.User{},
		&model.Team{},
		&model.DataSource{},
		&model.AlertRule{},
		&model.AlertRuleHistory{},
		&model.AlertEvent{},
		&model.AlertTimeline{},
		&model.Schedule{},
		&model.ScheduleParticipant{},
		&model.ScheduleOverride{},
		&model.OnCallShift{},
		&model.EscalationPolicy{},
		&model.EscalationStep{},
		&model.NotifyChannel{},
		&model.NotifyRecord{},
		&model.MuteRule{},
	}

	// Audit log
	models = append(models, &model.AuditLog{})

	// Phase 2 notification v2 models
	models = append(models, model.NotificationV2Models()...)

	// Dispatch models (alert channels + user notify configs)
	models = append(models, model.DispatchModels()...)

	// Platform settings
	models = append(models, &model.SystemSetting{})

	// Inhibition rules (alert suppression)
	models = append(models, &model.InhibitionRule{})

	// Label registry (autocomplete for match_labels)
	models = append(models, &model.LabelRegistry{})

	// Dashboards (v2 — panel/variable config stored in JSON)
	models = append(models, &model.Dashboard{})

	// V2 feature models (alerts, channels, incidents, integrations, dispatch, templates)
	models = append(models, model.V2Models()...)

	// Chat history
	models = append(models, &model.ChatHistory{})

	// Status page services
	models = append(models, &model.StatusService{})

	// User preferences — table created by migration 000044, skip AutoMigrate
	// (MySQL strict mode rejects DEFAULT on JSON columns)

	return db.AutoMigrate(models...)
}

func seedAdminUser(db *gorm.DB, logger *zap.Logger) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	// SREAGENT_ADMIN_PASSWORD is mandatory — no weak fallback passwords allowed.
	defaultPwd := os.Getenv("SREAGENT_ADMIN_PASSWORD")
	if defaultPwd == "" {
		logger.Fatal("SREAGENT_ADMIN_PASSWORD environment variable must be set")
	}
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(defaultPwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash password", zap.Error(err))
		return
	}

	admin := &model.User{
		Username:    "admin",
		Password:    string(hashedPwd),
		DisplayName: "Administrator",
		Email:       "admin@sreagent.local",
		Role:        model.RoleAdmin,
		IsActive:    true,
	}

	if err := db.Create(admin).Error; err != nil {
		logger.Error("failed to seed admin user", zap.Error(err))
		return
	}

	logger.Info("seeded default admin user — change password immediately after first login")
}

func seedPresetRules(db *gorm.DB, logger *zap.Logger) {
	var count int64
	db.Model(&model.PresetRule{}).Where("is_builtin = ?", true).Count(&count)
	if count > 0 {
		return
	}

	type preset struct {
		Name, DisplayName, Category, SubCategory, Component string
		Expression, ForDuration, Severity, AlertType        string
		Labels, Annotations                                 model.JSONLabels
		Description                                         string
	}

	presets := []preset{
		// ── Host / System ──
		{
			Name: "HostHighCpuUsage", DisplayName: "主机 CPU 使用率过高", Category: "host", SubCategory: "cpu", Component: "node-exporter",
			Expression: `100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 90`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P0"}, Description: "主机 CPU 使用率持续 5 分钟超过 90%",
		},
		{
			Name: "HostHighMemoryUsage", DisplayName: "主机内存使用率过高", Category: "host", SubCategory: "memory", Component: "node-exporter",
			Expression: `(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 90`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P0"}, Description: "主机内存使用率持续 5 分钟超过 90%",
		},
		{
			Name: "HostDiskSpaceRunningLow", DisplayName: "主机磁盘空间不足", Category: "host", SubCategory: "disk", Component: "node-exporter",
			Expression: `(1 - node_filesystem_avail_bytes{fstype!~"tmpfs|fuse.lxcfs"} / node_filesystem_size_bytes) * 100 > 85`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P1"}, Description: "主机磁盘使用率持续 10 分钟超过 85%",
		},
		{
			Name: "HostDiskSpaceCritical", DisplayName: "主机磁盘空间严重不足", Category: "host", SubCategory: "disk", Component: "node-exporter",
			Expression: `(1 - node_filesystem_avail_bytes{fstype!~"tmpfs|fuse.lxcfs"} / node_filesystem_size_bytes) * 100 > 95`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P0"}, Description: "主机磁盘使用率持续 5 分钟超过 95%，即将写满",
		},
		{
			Name: "HostHighLoadAverage", DisplayName: "主机负载过高", Category: "host", SubCategory: "load", Component: "node-exporter",
			Expression: `node_load15 / count without(cpu, mode) (node_cpu_seconds_total{mode="idle"}) > 2`, ForDuration: "15m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P1"}, Description: "主机 15 分钟平均负载持续超过 CPU 核数的 2 倍",
		},
		{
			Name: "NodeExporterDown", DisplayName: "Node Exporter 离线", Category: "host", SubCategory: "availability", Component: "node-exporter",
			Expression: `up{job="node-exporter"} == 0`, ForDuration: "3m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P0"}, Description: "Node Exporter 无法采集，主机可能宕机",
		},
		{
			Name: "HostHighCpuIOWait", DisplayName: "主机 IO 等待过高", Category: "host", SubCategory: "disk", Component: "node-exporter",
			Expression: `avg by(instance) (rate(node_cpu_seconds_total{mode="iowait"}[5m])) * 100 > 20`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P1"}, Description: "CPU IO 等待比例持续 10 分钟超过 20%，磁盘可能成为瓶颈",
		},
		{
			Name: "HostNetworkErrors", DisplayName: "主机网络错误", Category: "host", SubCategory: "network", Component: "node-exporter",
			Expression: `rate(node_network_receive_errs_total[5m]) > 10 or rate(node_network_transmit_errs_total[5m]) > 10`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "host", "severity": "P1"}, Description: "网卡收发错误率持续偏高",
		},

		// ── Kubernetes / Container ──
		{
			Name: "KubePodCrashLooping", DisplayName: "Pod 反复重启", Category: "kubernetes", SubCategory: "pod", Component: "kube-state-metrics",
			Expression: `rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 0`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P0"}, Description: "Pod 容器在 15 分钟内发生重启",
		},
		{
			Name: "KubePodNotReady", DisplayName: "Pod 未就绪", Category: "kubernetes", SubCategory: "pod", Component: "kube-state-metrics",
			Expression: `kube_pod_status_phase{phase=~"Pending|Unknown"} > 0`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P1"}, Description: "Pod 持续 10 分钟处于 Pending 或 Unknown 状态",
		},
		{
			Name: "KubeNodeNotReady", DisplayName: "K8s 节点 NotReady", Category: "kubernetes", SubCategory: "node", Component: "kube-state-metrics",
			Expression: `kube_node_status_condition{condition="Ready",status="true"} == 0`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P0"}, Description: "Kubernetes 节点持续 5 分钟处于 NotReady 状态",
		},
		{
			Name: "KubeContainerOOMKilled", DisplayName: "容器 OOM Killed", Category: "kubernetes", SubCategory: "pod", Component: "kube-state-metrics",
			Expression: `kube_pod_container_status_last_terminated_reason{reason="OOMKilled"} > 0`, ForDuration: "0s", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P0"}, Description: "容器因内存超限被 OOM Killed",
		},
		{
			Name: "KubeDeploymentReplicasMismatch", DisplayName: "Deployment 副本不足", Category: "kubernetes", SubCategory: "deployment", Component: "kube-state-metrics",
			Expression: `kube_deployment_spec_replicas != kube_deployment_status_ready_replicas`, ForDuration: "15m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P1"}, Description: "Deployment 就绪副本数与期望不一致超过 15 分钟",
		},
		{
			Name: "KubeContainerHighCpu", DisplayName: "容器 CPU 使用率过高", Category: "kubernetes", SubCategory: "container", Component: "cadvisor",
			Expression: `sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by(pod, container) / sum(container_spec_cpu_quota{container!=""}/container_spec_cpu_period{container!=""}) by(pod, container) * 100 > 90`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P1"}, Description: "容器 CPU 使用率持续 5 分钟超过 90%",
		},
		{
			Name: "KubeContainerHighMemory", DisplayName: "容器内存使用率过高", Category: "kubernetes", SubCategory: "container", Component: "cadvisor",
			Expression: `container_memory_working_set_bytes{container!=""} / container_spec_memory_limit_bytes{container!=""} * 100 > 90`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "container", "severity": "P1"}, Description: "容器内存使用率持续 5 分钟超过 Limit 的 90%",
		},

		// ── MySQL ──
		{
			Name: "MysqlDown", DisplayName: "MySQL 实例宕机", Category: "database", SubCategory: "mysql", Component: "mysqld-exporter",
			Expression: `mysql_up == 0`, ForDuration: "1m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "mysql", "severity": "P0"}, Description: "MySQL 实例无法连接",
		},
		{
			Name: "MysqlHighConnections", DisplayName: "MySQL 连接数过高", Category: "database", SubCategory: "mysql", Component: "mysqld-exporter",
			Expression: `mysql_global_status_threads_connected / mysql_global_variables_max_connections * 100 > 80`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "mysql", "severity": "P1"}, Description: "MySQL 活跃连接数超过最大连接数的 80%",
		},
		{
			Name: "MysqlSlowQueries", DisplayName: "MySQL 慢查询激增", Category: "database", SubCategory: "mysql", Component: "mysqld-exporter",
			Expression: `rate(mysql_global_status_slow_queries[5m]) > 0.1`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "mysql", "severity": "P1"}, Description: "MySQL 慢查询速率持续偏高",
		},
		{
			Name: "MysqlReplicationLag", DisplayName: "MySQL 主从复制延迟", Category: "database", SubCategory: "mysql", Component: "mysqld-exporter",
			Expression: `mysql_slave_status_seconds_behind_master > 30`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "mysql", "severity": "P1"}, Description: "MySQL 主从复制延迟超过 30 秒",
		},

		// ── Redis ──
		{
			Name: "RedisDown", DisplayName: "Redis 实例宕机", Category: "database", SubCategory: "redis", Component: "redis-exporter",
			Expression: `redis_up == 0`, ForDuration: "1m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "redis", "severity": "P0"}, Description: "Redis 实例无法连接",
		},
		{
			Name: "RedisHighMemory", DisplayName: "Redis 内存使用率过高", Category: "database", SubCategory: "redis", Component: "redis-exporter",
			Expression: `redis_memory_used_bytes / redis_memory_max_bytes * 100 > 90`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "redis", "severity": "P1"}, Description: "Redis 内存使用超过 maxmemory 的 90%",
		},
		{
			Name: "RedisHighLatency", DisplayName: "Redis 延迟过高", Category: "database", SubCategory: "redis", Component: "redis-exporter",
			Expression: `redis_commands_duration_seconds_total{cmd="get"} / redis_commands_total{cmd="get"} > 0.01`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "redis", "severity": "P1"}, Description: "Redis GET 命令平均延迟超过 10ms",
		},

		// ── MongoDB ──
		{
			Name: "MongoDBDown", DisplayName: "MongoDB 实例宕机", Category: "database", SubCategory: "mongodb", Component: "mongodb-exporter",
			Expression: `mongodb_up == 0`, ForDuration: "1m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "mongodb", "severity": "P0"}, Description: "MongoDB 实例无法连接",
		},
		{
			Name: "MongoDBReplicationLag", DisplayName: "MongoDB 复制延迟", Category: "database", SubCategory: "mongodb", Component: "mongodb-exporter",
			Expression: `mongodb_mongod_replset_member_replication_lag > 30`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "mongodb", "severity": "P1"}, Description: "MongoDB 副本集复制延迟超过 30 秒",
		},

		// ── Elasticsearch ──
		{
			Name: "ElasticsearchClusterRed", DisplayName: "ES 集群 Red", Category: "database", SubCategory: "elasticsearch", Component: "es-exporter",
			Expression: `elasticsearch_cluster_health_status{color="red"} == 1`, ForDuration: "2m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "elasticsearch", "severity": "P0"}, Description: "Elasticsearch 集群健康状态为 Red",
		},
		{
			Name: "ElasticsearchClusterYellow", DisplayName: "ES 集群 Yellow", Category: "database", SubCategory: "elasticsearch", Component: "es-exporter",
			Expression: `elasticsearch_cluster_health_status{color="yellow"} == 1`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "elasticsearch", "severity": "P1"}, Description: "Elasticsearch 集群健康状态为 Yellow",
		},
		{
			Name: "ElasticsearchHighJVMHeap", DisplayName: "ES JVM 堆内存过高", Category: "database", SubCategory: "elasticsearch", Component: "es-exporter",
			Expression: `elasticsearch_jvm_memory_used_bytes{area="heap"} / elasticsearch_jvm_memory_max_bytes{area="heap"} * 100 > 85`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "elasticsearch", "severity": "P1"}, Description: "ES JVM 堆内存使用率超过 85%",
		},

		// ── Kafka ──
		{
			Name: "KafkaBrokerDown", DisplayName: "Kafka Broker 宕机", Category: "middleware", SubCategory: "kafka", Component: "kafka-exporter",
			Expression: `kafka_brokers < 3`, ForDuration: "2m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "kafka", "severity": "P0"}, Description: "Kafka 集群存活 Broker 数不足",
		},
		{
			Name: "KafkaConsumerLag", DisplayName: "Kafka 消费延迟", Category: "middleware", SubCategory: "kafka", Component: "kafka-exporter",
			Expression: `kafka_consumergroup_lag_sum > 10000`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "kafka", "severity": "P1"}, Description: "Kafka 消费组积压超过 10000 条",
		},
		{
			Name: "KafkaTopicUnderReplicated", DisplayName: "Kafka Topic 副本不足", Category: "middleware", SubCategory: "kafka", Component: "kafka-exporter",
			Expression: `kafka_topic_partition_under_replicated_partition > 0`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "kafka", "severity": "P1"}, Description: "Kafka Topic 存在副本不足的分区",
		},

		// ── RabbitMQ ──
		{
			Name: "RabbitMQDown", DisplayName: "RabbitMQ 实例宕机", Category: "middleware", SubCategory: "rabbitmq", Component: "rabbitmq-exporter",
			Expression: `rabbitmq_up == 0`, ForDuration: "1m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "rabbitmq", "severity": "P0"}, Description: "RabbitMQ 实例无法连接",
		},
		{
			Name: "RabbitMQHighQueueDepth", DisplayName: "RabbitMQ 队列积压", Category: "middleware", SubCategory: "rabbitmq", Component: "rabbitmq-exporter",
			Expression: `rabbitmq_queue_messages > 10000`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "rabbitmq", "severity": "P1"}, Description: "RabbitMQ 队列消息积压超过 10000",
		},

		// ── Nginx ──
		{
			Name: "NginxHighHttp5xxRate", DisplayName: "Nginx 5xx 错误率过高", Category: "middleware", SubCategory: "nginx", Component: "nginx-exporter",
			Expression: `rate(nginx_http_requests_total{status=~"5.."}[5m]) / rate(nginx_http_requests_total[5m]) * 100 > 5`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "nginx", "severity": "P0"}, Description: "Nginx 5xx 错误率超过 5%",
		},
		{
			Name: "NginxHighHttp4xxRate", DisplayName: "Nginx 4xx 错误率过高", Category: "middleware", SubCategory: "nginx", Component: "nginx-exporter",
			Expression: `rate(nginx_http_requests_total{status=~"4.."}[5m]) / rate(nginx_http_requests_total[5m]) * 100 > 20`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "nginx", "severity": "P1"}, Description: "Nginx 4xx 错误率超过 20%",
		},

		// ── Blackbox / Probe ──
		{
			Name: "BlackboxHttpProbeFailed", DisplayName: "HTTP 探测失败", Category: "network", SubCategory: "probe", Component: "blackbox-exporter",
			Expression: `probe_success{job="blackbox-http"} == 0`, ForDuration: "2m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "probe", "severity": "P0"}, Description: "HTTP 探测连续失败，服务可能不可用",
		},
		{
			Name: "BlackboxHttpProbeLatencyHigh", DisplayName: "HTTP 探测延迟过高", Category: "network", SubCategory: "probe", Component: "blackbox-exporter",
			Expression: `probe_duration_seconds{job="blackbox-http"} > 5`, ForDuration: "5m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "probe", "severity": "P1"}, Description: "HTTP 探测响应时间超过 5 秒",
		},
		{
			Name: "BlackboxTcpProbeFailed", DisplayName: "TCP 探测失败", Category: "network", SubCategory: "probe", Component: "blackbox-exporter",
			Expression: `probe_success{job="blackbox-tcp"} == 0`, ForDuration: "2m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "probe", "severity": "P0"}, Description: "TCP 端口探测连续失败",
		},

		// ── Application / HTTP ──
		{
			Name: "HttpHighErrorRate", DisplayName: "HTTP 5xx 错误率过高", Category: "application", SubCategory: "http", Component: "app",
			Expression: `rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100 > 5`, ForDuration: "5m", Severity: "critical", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "application", "severity": "P0"}, Description: "应用 HTTP 5xx 错误率超过 5%",
		},
		{
			Name: "HttpHighLatencyP99", DisplayName: "HTTP P99 延迟过高", Category: "application", SubCategory: "http", Component: "app",
			Expression: `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 2`, ForDuration: "10m", Severity: "warning", AlertType: "threshold",
			Labels: model.JSONLabels{"category": "application", "severity": "P1"}, Description: "HTTP 请求 P99 延迟超过 2 秒",
		},

		// ── Inhibition Presets (14 rules matching Alertmanager config) ──
		// 1. Host severity cascade — P0 suppresses P1/P2/P3
		{
			Name: "inhibit-host-p0-cascade", DisplayName: "主机 P0 抑制 P1/P2/P3", Category: "inhibition", SubCategory: "severity", Component: "engine",
			Expression: `{"source_match":{"severity":"P0"},"target_match":{"severity":"~P1|P2|P3"},"equal_labels":["biz_project","category","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "severity=P0", "target_match": "severity=~P1|P2|P3", "equal_labels": "biz_project,category,instance,project"},
			Description: "P0 告警触发时，抑制同业务线同实例的 P1/P2/P3 告警",
		},
		// 2. Host severity cascade — P1 suppresses P2/P3
		{
			Name: "inhibit-host-p1-cascade", DisplayName: "主机 P1 抑制 P2/P3", Category: "inhibition", SubCategory: "severity", Component: "engine",
			Expression: `{"source_match":{"severity":"P1"},"target_match":{"severity":"~P2|P3"},"equal_labels":["biz_project","category","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "severity=P1", "target_match": "severity=~P2|P3", "equal_labels": "biz_project,category,instance,project"},
			Description: "P1 告警触发时，抑制同业务线同实例的 P2/P3 告警",
		},
		// 3. Container severity cascade — P0 suppresses P1/P2/P3
		{
			Name: "inhibit-container-p0-cascade", DisplayName: "容器 P0 抑制 P1/P2/P3", Category: "inhibition", SubCategory: "severity", Component: "engine",
			Expression: `{"source_match":{"severity":"P0","category":"container"},"target_match":{"severity":"~P1|P2|P3","category":"container"},"equal_labels":["biz_project","namespace","pod","container","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "severity=P0,category=container", "target_match": "severity=~P1|P2|P3,category=container", "equal_labels": "biz_project,namespace,pod,container,project"},
			Description: "容器 P0 告警触发时，抑制同 Pod 的 P1/P2/P3 容器告警",
		},
		// 4. Container severity cascade — P1 suppresses P2/P3
		{
			Name: "inhibit-container-p1-cascade", DisplayName: "容器 P1 抑制 P2/P3", Category: "inhibition", SubCategory: "severity", Component: "engine",
			Expression: `{"source_match":{"severity":"P1","category":"container"},"target_match":{"severity":"~P2|P3","category":"container"},"equal_labels":["biz_project","namespace","pod","container","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "severity=P1,category=container", "target_match": "severity=~P2|P3,category=container", "equal_labels": "biz_project,namespace,pod,container,project"},
			Description: "容器 P1 告警触发时，抑制同 Pod 的 P2/P3 容器告警",
		},
		// 5. NodeExporterDown suppresses all severities
		{
			Name: "inhibit-node-down-cascade", DisplayName: "主机宕机抑制所有告警", Category: "inhibition", SubCategory: "availability", Component: "engine",
			Expression: `{"source_match":{"alertname":"NodeExporterDown"},"target_match":{"severity":"~P0|P1|P2|P3"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=NodeExporterDown", "target_match": "severity=~P0|P1|P2|P3", "equal_labels": "biz_project,instance,project"},
			Description: "NodeExporterDown 时抑制该主机的所有严重等级告警",
		},
		// 6. KubeNodeNotReady suppresses container alerts
		{
			Name: "inhibit-kube-node-notready-container", DisplayName: "K8s 节点 NotReady 抑制容器告警", Category: "inhibition", SubCategory: "kubernetes", Component: "engine",
			Expression: `{"source_match":{"alertname":"KubeNodeNotReady"},"target_match":{"category":"container"},"equal_labels":["biz_project","node","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=KubeNodeNotReady", "target_match": "category=container", "equal_labels": "biz_project,node,project"},
			Description: "K8s 节点 NotReady 时抑制该节点上的容器告警",
		},
		// 7. KubeNodeNotReady suppresses pod alerts
		{
			Name: "inhibit-kube-node-notready-pod", DisplayName: "K8s 节点 NotReady 抑制 Pod 告警", Category: "inhibition", SubCategory: "kubernetes", Component: "engine",
			Expression: `{"source_match":{"alertname":"KubeNodeNotReady"},"target_match":{"category":"pod"},"equal_labels":["biz_project","node","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=KubeNodeNotReady", "target_match": "category=pod", "equal_labels": "biz_project,node,project"},
			Description: "K8s 节点 NotReady 时抑制该节点上的 Pod 告警",
		},
		// 8. KafkaExporterDown suppresses kafka alerts
		{
			Name: "inhibit-kafka-down-cascade", DisplayName: "Kafka Down 抑制同实例告警", Category: "inhibition", SubCategory: "middleware", Component: "engine",
			Expression: `{"source_match":{"alertname":"KafkaExporterDown"},"target_match":{"category":"kafka"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=KafkaExporterDown", "target_match": "category=kafka", "equal_labels": "biz_project,instance,project"},
			Description: "Kafka Down 时抑制同实例的 Kafka 类告警",
		},
		// 9. RedisDown suppresses redis alerts
		{
			Name: "inhibit-redis-down-cascade", DisplayName: "Redis Down 抑制同实例告警", Category: "inhibition", SubCategory: "database", Component: "engine",
			Expression: `{"source_match":{"alertname":"RedisDown"},"target_match":{"category":"redis"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=RedisDown", "target_match": "category=redis", "equal_labels": "biz_project,instance,project"},
			Description: "Redis Down 时抑制同实例的 Redis 类告警",
		},
		// 10. ElasticsearchClusterRed suppresses Yellow
		{
			Name: "inhibit-es-red-cascade", DisplayName: "ES Red 抑制 Yellow 告警", Category: "inhibition", SubCategory: "database", Component: "engine",
			Expression: `{"source_match":{"alertname":"ElasticsearchClusterRed"},"target_match":{"alertname":"ElasticsearchClusterYellow"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=ElasticsearchClusterRed", "target_match": "alertname=ElasticsearchClusterYellow", "equal_labels": "biz_project,instance,project"},
			Description: "ES 集群 Red 时抑制同实例的 Yellow 告警",
		},
		// 11. MongoDBDown suppresses mongodb alerts
		{
			Name: "inhibit-mongodb-down-cascade", DisplayName: "MongoDB Down 抑制同实例告警", Category: "inhibition", SubCategory: "database", Component: "engine",
			Expression: `{"source_match":{"alertname":"MongoDBDown"},"target_match":{"category":"mongodb"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=MongoDBDown", "target_match": "category=mongodb", "equal_labels": "biz_project,instance,project"},
			Description: "MongoDB Down 时抑制同实例的 MongoDB 类告警",
		},
		// 12. RabbitMQDown suppresses rabbitmq alerts
		{
			Name: "inhibit-rabbitmq-down-cascade", DisplayName: "RabbitMQ Down 抑制同实例告警", Category: "inhibition", SubCategory: "middleware", Component: "engine",
			Expression: `{"source_match":{"alertname":"RabbitMQDown"},"target_match":{"category":"rabbitmq"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=RabbitMQDown", "target_match": "category=rabbitmq", "equal_labels": "biz_project,instance,project"},
			Description: "RabbitMQ Down 时抑制同实例的 RabbitMQ 类告警",
		},
		// 13. NacosDown suppresses nacos alerts
		{
			Name: "inhibit-nacos-down-cascade", DisplayName: "Nacos Down 抑制同实例告警", Category: "inhibition", SubCategory: "middleware", Component: "engine",
			Expression: `{"source_match":{"alertname":"NacosDown"},"target_match":{"category":"nacos"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=NacosDown", "target_match": "category=nacos", "equal_labels": "biz_project,instance,project"},
			Description: "Nacos Down 时抑制同实例的 Nacos 类告警",
		},
		// 14. RocketMQExporterDown suppresses rocketmq alerts
		{
			Name: "inhibit-rocketmq-down-cascade", DisplayName: "RocketMQ Down 抑制同实例告警", Category: "inhibition", SubCategory: "middleware", Component: "engine",
			Expression: `{"source_match":{"alertname":"RocketMQExporterDown"},"target_match":{"category":"rocketmq"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=RocketMQExporterDown", "target_match": "category=rocketmq", "equal_labels": "biz_project,instance,project"},
			Description: "RocketMQ Down 时抑制同实例的 RocketMQ 类告警",
		},
		// 15. BlackboxHttpProbeFailed suppresses latency/status/DNS alerts
		{
			Name: "inhibit-http-probe-failed-cascade", DisplayName: "HTTP 探测失败级联抑制", Category: "inhibition", SubCategory: "probe", Component: "engine",
			Expression: `{"source_match":{"alertname":"BlackboxHttpProbeFailed"},"target_match":{"alertname":"~BlackboxHttpProbeLatency.*|BlackboxHttpStatus5xx|BlackboxHttpDnsLatencyHigh"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=BlackboxHttpProbeFailed", "target_match": "alertname=~BlackboxHttpProbeLatency.*|BlackboxHttpStatus5xx|BlackboxHttpDnsLatencyHigh", "equal_labels": "biz_project,instance,project"},
			Description: "HTTP 探测失败时抑制同实例的延迟、状态码和 DNS 延迟告警",
		},
		// 16. BlackboxTcpProbeFailed suppresses latency alerts
		{
			Name: "inhibit-tcp-probe-failed-cascade", DisplayName: "TCP 探测失败级联抑制", Category: "inhibition", SubCategory: "probe", Component: "engine",
			Expression: `{"source_match":{"alertname":"BlackboxTcpProbeFailed"},"target_match":{"alertname":"~BlackboxTcpProbeLatency.*"},"equal_labels":["biz_project","instance","project"]}`, Severity: "info", AlertType: "inhibition",
			Labels: model.JSONLabels{"source_match": "alertname=BlackboxTcpProbeFailed", "target_match": "alertname=~BlackboxTcpProbeLatency.*", "equal_labels": "biz_project,instance,project"},
			Description: "TCP 探测失败时抑制同实例的延迟告警",
		},
	}

	var rules []model.PresetRule
	for _, p := range presets {
		rules = append(rules, model.PresetRule{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Category:    p.Category,
			SubCategory: p.SubCategory,
			Component:   p.Component,
			Expression:  p.Expression,
			ForDuration: p.ForDuration,
			Severity:    p.Severity,
			AlertType:   p.AlertType,
			Labels:      p.Labels,
			Annotations: p.Annotations,
			Source:      "builtin",
			IsBuiltin:   true,
			Description: p.Description,
		})
	}

	if err := db.CreateInBatches(rules, 100).Error; err != nil {
		logger.Error("failed to seed preset rules", zap.Error(err))
		return
	}

	logger.Info("seeded built-in preset rules", zap.Int("count", len(rules)))
}
