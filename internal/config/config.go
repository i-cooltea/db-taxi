package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Security SecurityConfig `mapstructure:"security"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Sync     SyncConfig     `mapstructure:"sync"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	EnableHTTPS  bool          `mapstructure:"enable_https"`
	CertFile     string        `mapstructure:"cert_file"`
	KeyFile      string        `mapstructure:"key_file"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSL             bool          `mapstructure:"ssl"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	QueryTimeout    time.Duration `mapstructure:"query_timeout"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	SessionTimeout time.Duration `mapstructure:"session_timeout"`
	ReadOnlyMode   bool          `mapstructure:"read_only_mode"`
	EnableAudit    bool          `mapstructure:"enable_audit"`
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// SyncConfig holds synchronization-related configuration
type SyncConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	BatchSize      int           `mapstructure:"batch_size"`
	RetryAttempts  int           `mapstructure:"retry_attempts"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
	JobTimeout     time.Duration `mapstructure:"job_timeout"`
	CleanupAge     time.Duration `mapstructure:"cleanup_age"`
}

// LoadOptions contains options for loading configuration
type LoadOptions struct {
	ConfigFile string  // Path to configuration file
	Overrides  *Config // Command line overrides
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	return LoadWithOptions(nil)
}

// LoadWithOptions loads configuration with additional options
func LoadWithOptions(opts *LoadOptions) (*Config, error) {
	// Set default values
	setDefaults()

	// Set config file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// If specific config file is provided, use it
	if opts != nil && opts.ConfigFile != "" {
		viper.SetConfigFile(opts.ConfigFile)
	} else {
		// Use default search paths
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/db-taxi")
	}

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("DBT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply command line overrides
	if opts != nil && opts.Overrides != nil {
		applyOverrides(&config, opts.Overrides)
	}

	return &config, nil
}

// applyOverrides applies command line overrides to the configuration
func applyOverrides(config *Config, overrides *Config) {
	// Database overrides
	if overrides.Database.Host != "" {
		config.Database.Host = overrides.Database.Host
	}
	if overrides.Database.Port != 0 {
		config.Database.Port = overrides.Database.Port
	}
	if overrides.Database.Username != "" {
		config.Database.Username = overrides.Database.Username
	}
	if overrides.Database.Password != "" {
		config.Database.Password = overrides.Database.Password
	}
	if overrides.Database.Database != "" {
		config.Database.Database = overrides.Database.Database
	}
	if overrides.Database.SSL {
		config.Database.SSL = overrides.Database.SSL
	}

	// Server overrides
	if overrides.Server.Port != 0 {
		config.Server.Port = overrides.Server.Port
	}
	if overrides.Server.Host != "" {
		config.Server.Host = overrides.Server.Host
	}
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.enable_https", false)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.username", "root")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.database", "")
	viper.SetDefault("database.ssl", false)
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("database.query_timeout", "30s")

	// Security defaults
	viper.SetDefault("security.session_timeout", "30m")
	viper.SetDefault("security.read_only_mode", false)
	viper.SetDefault("security.enable_audit", true)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)

	// Sync defaults
	viper.SetDefault("sync.enabled", true)
	viper.SetDefault("sync.max_concurrency", 5)
	viper.SetDefault("sync.batch_size", 1000)
	viper.SetDefault("sync.retry_attempts", 3)
	viper.SetDefault("sync.retry_delay", "30s")
	viper.SetDefault("sync.job_timeout", "1h")
	viper.SetDefault("sync.cleanup_age", "720h")
}
