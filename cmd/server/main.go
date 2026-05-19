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
	defer zapLogger.Sync()

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
		&model.NotifyPolicy{},
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

	// Virtual pet system
	models = append(models, &model.Pet{}, &model.PetInteraction{})

	// Chat history
	models = append(models, &model.ChatHistory{})

	// Status page services
	models = append(models, &model.StatusService{})

	return db.AutoMigrate(models...)
}

func seedAdminUser(db *gorm.DB, logger *zap.Logger) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	defaultPwd := os.Getenv("SREAGENT_ADMIN_PASSWORD")
	if defaultPwd == "" {
		if os.Getenv("GIN_MODE") == "release" {
			logger.Fatal("SREAGENT_ADMIN_PASSWORD must be set in release mode")
		}
		defaultPwd = "admin123"
		logger.Warn("SREAGENT_ADMIN_PASSWORD not set, using default password — change it immediately after first login")
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
