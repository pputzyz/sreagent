package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server            ServerConfig   `mapstructure:"server"`
	Database          DatabaseConfig `mapstructure:"database"`
	Redis             RedisConfig    `mapstructure:"redis"`
	JWT               JWTConfig      `mapstructure:"jwt"`
	OIDC              OIDCConfig     `mapstructure:"oidc"`
	Log               LogConfig      `mapstructure:"log"`
	Engine            EngineConfig   `mapstructure:"engine"`
	Task              TaskConfig     `mapstructure:"task"`
	MetricsToken      string         `mapstructure:"metrics_token"`
	CORSAllowedOrigins string        `mapstructure:"cors_allowed_origins"`
}

// TaskConfig holds configuration for SSH-based task execution.
type TaskConfig struct {
	SSHKnownHostsFile string `mapstructure:"ssh_known_hosts_file"` // path to known_hosts file; defaults to /etc/ssh/ssh_known_hosts
}

// EngineConfig holds configuration for the native alert evaluator.
type EngineConfig struct {
	Enabled            bool   `mapstructure:"enabled"`               // default true
	SyncInterval       int    `mapstructure:"sync_interval"`         // how often to sync rules from DB (seconds, default 30)
	PerDatasourceEval  bool   `mapstructure:"per_datasource_eval"`   // per-datasource bucket evaluation (default false = legacy)
	HeartbeatInterval  int    `mapstructure:"heartbeat_interval"`    // heartbeat check interval (seconds, default 60)
	HashRingEnabled    bool   `mapstructure:"hash_ring_enabled"`     // distribute rules across instances via consistent hash ring (default false = single-leader)
	HashRingReplicas   int    `mapstructure:"hash_ring_replicas"`    // virtual nodes per physical node in the hash ring (default 500)
	InstanceID         string `mapstructure:"instance_id"`           // unique identifier for this engine instance (default: hostname:pid)
}

type ServerConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Mode          string `mapstructure:"mode"`
	ExternalBase  string `mapstructure:"external_base"`  // external base URL for links in notifications
	WebhookSecret string `mapstructure:"webhook_secret"` // shared secret for /webhooks/* endpoints
}

func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// ExternalURL returns the external base URL for the platform.
// Falls back to http://host:port if not explicitly configured.
func (s *ServerConfig) ExternalURL() string {
	if s.ExternalBase != "" {
		return s.ExternalBase
	}
	return fmt.Sprintf("http://%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"` // unused; MySQL is hardcoded. Kept for config file compatibility.
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime"`
	Debug        bool   `mapstructure:"debug"` // enable GORM SQL logging (env: SREAGENT_DB_DEBUG=true)
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Database, d.Charset)
}

// SafeDSN returns the DSN with the password masked for safe logging.
// Useful for startup logs and health-check endpoints.
func (d *DatabaseConfig) SafeDSN() string {
	return fmt.Sprintf("%s:****@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.Username, d.Host, d.Port, d.Database, d.Charset)
}

// MigrateDSN returns a DSN with multiStatements=true, required by
// golang-migrate's MySQL driver which executes the entire migration file
// as a single db.ExecContext call. The main app connection uses DSN()
// without this flag to avoid security exposure.
func (d *DatabaseConfig) MigrateDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&multiStatements=true",
		d.Username, d.Password, d.Host, d.Port, d.Database, d.Charset)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret       string `mapstructure:"secret"`
	Expire       int    `mapstructure:"expire"`
	Issuer       string `mapstructure:"issuer"`
	RefreshGrace int    `mapstructure:"refresh_grace"` // Refresh grace window in seconds (default 1800 = 30min)
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
	File   string `mapstructure:"file"`
}

// OIDCConfig holds configuration for OIDC/Keycloak integration.
// When Enabled is true, the platform supports "Login with SSO" alongside local auth.
type OIDCConfig struct {
	Enabled       bool              `mapstructure:"enabled"`        // master switch
	IssuerURL     string            `mapstructure:"issuer_url"`     // e.g. https://keycloak.example.com/realms/sreagent
	ClientID      string            `mapstructure:"client_id"`      // OIDC client ID
	ClientSecret  string            `mapstructure:"client_secret"`  // OIDC client secret
	RedirectURL   string            `mapstructure:"redirect_url"`   // e.g. https://sreagent.example.com/api/v1/auth/oidc/callback
	Scopes        []string          `mapstructure:"scopes"`         // default: ["openid","profile","email"]
	RoleClaim     string            `mapstructure:"role_claim"`     // JWT claim path for roles, default "realm_access.roles"
	RoleMapping   map[string]string `mapstructure:"role_mapping"`   // Keycloak role → SREAgent role, e.g. {"sre-admin":"admin","sre-member":"member"}
	RoleStrategy  string            `mapstructure:"role_strategy"`  // "highest" (default) or "lowest" — how to resolve multiple OIDC role matches
	DefaultRole   string            `mapstructure:"default_role"`   // role when no mapping matches, default "viewer"
	AutoProvision bool              `mapstructure:"auto_provision"` // create user on first OIDC login, default true
}

// Load reads config from file and environment variables.
// Config file path can be specified, defaults to configs/config.yaml.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// Allow environment variable overrides
	// e.g. SREAGENT_DATABASE_HOST=xxx
	viper.SetEnvPrefix("SREAGENT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Explicitly bind env vars for sensitive fields that may be absent from
	// the config file. Viper's AutomaticEnv only works for keys already
	// registered in the config file; BindEnv ensures these are always read
	// from the environment regardless.
	_ = viper.BindEnv("database.password", "SREAGENT_DATABASE_PASSWORD")
	_ = viper.BindEnv("database.host", "SREAGENT_DATABASE_HOST")
	_ = viper.BindEnv("database.port", "SREAGENT_DATABASE_PORT")
	_ = viper.BindEnv("database.username", "SREAGENT_DATABASE_USERNAME")
	_ = viper.BindEnv("database.debug", "SREAGENT_DB_DEBUG")
	_ = viper.BindEnv("redis.password", "SREAGENT_REDIS_PASSWORD")
	_ = viper.BindEnv("redis.host", "SREAGENT_REDIS_HOST")
	_ = viper.BindEnv("redis.port", "SREAGENT_REDIS_PORT")
	_ = viper.BindEnv("jwt.secret", "SREAGENT_JWT_SECRET")
	_ = viper.BindEnv("jwt.refresh_grace", "SREAGENT_JWT_REFRESH_GRACE")
	_ = viper.BindEnv("metrics_token", "SREAGENT_METRICS_TOKEN")
	_ = viper.BindEnv("cors_allowed_origins", "SREAGENT_CORS_ALLOWED_ORIGINS")
	_ = viper.BindEnv("server.webhook_secret", "SREAGENT_WEBHOOK_SECRET")
	_ = viper.BindEnv("oidc.client_secret", "SREAGENT_OIDC_CLIENT_SECRET")
	_ = viper.BindEnv("oidc.client_id", "SREAGENT_OIDC_CLIENT_ID")
	_ = viper.BindEnv("oidc.issuer_url", "SREAGENT_OIDC_ISSUER_URL")

	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found — continue with env-var-only configuration.
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Defaults for fields that have no zero-value sentinel.
	if cfg.Database.Charset == "" {
		cfg.Database.Charset = "utf8mb4"
	}
	if cfg.JWT.RefreshGrace <= 0 {
		cfg.JWT.RefreshGrace = 1800 // 30 minutes
	}

	// Validate JWT secret strength.
	weakSecrets := map[string]bool{
		"secret": true, "password": true, "changeme": true,
		"your-secret-key": true, "jwt-secret": true, "sreagent": true,
	}
	if weakSecrets[strings.ToLower(cfg.JWT.Secret)] {
		return nil, fmt.Errorf("JWT secret uses a well-known weak value; please set a strong, random secret")
	}
	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 bytes, got %d", len(cfg.JWT.Secret))
	}

	// Backward compatibility: legacy env var names without SREAGENT_ prefix.
	// The new SREAGENT_-prefixed vars (bound above) take precedence.
	if cfg.MetricsToken == "" {
		if v := os.Getenv("METRICS_TOKEN"); v != "" {
			cfg.MetricsToken = v
			// #14: Deprecation warning
			fmt.Fprintf(os.Stderr, "WARNING: deprecated env var METRICS_TOKEN is ignored; use SREAGENT_METRICS_TOKEN instead\n")
		}
	}
	if cfg.CORSAllowedOrigins == "" {
		if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
			cfg.CORSAllowedOrigins = v
			fmt.Fprintf(os.Stderr, "WARNING: env var CORS_ALLOWED_ORIGINS is deprecated; please rename to SREAGENT_CORS_ALLOWED_ORIGINS\n")
		}
	}

	return &cfg, nil
}
