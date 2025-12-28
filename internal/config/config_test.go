package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestLoad_WithDefaults(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	// Load configuration with defaults only
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got %s", cfg.Server.Host)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Expected default read timeout 30s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Expected default max open conns 25, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Security.SessionTimeout != 30*time.Minute {
		t.Errorf("Expected default session timeout 30m, got %v", cfg.Security.SessionTimeout)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got %s", cfg.Logging.Level)
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	// Set environment variables with proper nested structure
	os.Setenv("DBT_SERVER_PORT", "9090")
	os.Setenv("DBT_SERVER_HOST", "127.0.0.1")
	os.Setenv("DBT_DATABASE_MAX_OPEN_CONNS", "50")
	os.Setenv("DBT_SECURITY_READ_ONLY_MODE", "true")
	os.Setenv("DBT_LOGGING_LEVEL", "debug")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("DBT_SERVER_PORT")
		os.Unsetenv("DBT_SERVER_HOST")
		os.Unsetenv("DBT_DATABASE_MAX_OPEN_CONNS")
		os.Unsetenv("DBT_SECURITY_READ_ONLY_MODE")
		os.Unsetenv("DBT_LOGGING_LEVEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify environment variables override defaults
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port from env var 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host from env var '127.0.0.1', got %s", cfg.Server.Host)
	}
	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("Expected max open conns from env var 50, got %d", cfg.Database.MaxOpenConns)
	}
	if !cfg.Security.ReadOnlyMode {
		t.Errorf("Expected read only mode from env var true, got %v", cfg.Security.ReadOnlyMode)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level from env var 'debug', got %s", cfg.Logging.Level)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Create a temporary config file
	configContent := `
server:
  port: 3000
  host: "localhost"
  read_timeout: "60s"
  enable_https: true

database:
  max_open_conns: 100
  query_timeout: "45s"

security:
  session_timeout: "60m"
  read_only_mode: true

logging:
  level: "warn"
  format: "text"
`

	// Create config.yaml in current directory
	configFile := "config.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify config file values
	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port from config file 3000, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host from config file 'localhost', got %s", cfg.Server.Host)
	}
	if cfg.Server.ReadTimeout != 60*time.Second {
		t.Errorf("Expected read timeout from config file 60s, got %v", cfg.Server.ReadTimeout)
	}
	if !cfg.Server.EnableHTTPS {
		t.Errorf("Expected HTTPS enabled from config file true, got %v", cfg.Server.EnableHTTPS)
	}
	if cfg.Database.MaxOpenConns != 100 {
		t.Errorf("Expected max open conns from config file 100, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Security.SessionTimeout != 60*time.Minute {
		t.Errorf("Expected session timeout from config file 60m, got %v", cfg.Security.SessionTimeout)
	}
	if !cfg.Security.ReadOnlyMode {
		t.Errorf("Expected read only mode from config file true, got %v", cfg.Security.ReadOnlyMode)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Expected log level from config file 'warn', got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Expected log format from config file 'text', got %s", cfg.Logging.Format)
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	// Create an invalid config file
	invalidConfigContent := `
server:
  port: "invalid_port"  # Should be integer
  read_timeout: invalid_duration
`

	configFile := "config.yaml"
	err := os.WriteFile(configFile, []byte(invalidConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}
	defer os.Remove(configFile)

	_, err = Load()
	if err == nil {
		t.Error("Expected error for invalid config file, got nil")
	}
}

func TestLoad_MissingConfigFile(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	// Set a non-existent config file path
	viper.SetConfigFile("/non/existent/path/config.yaml")

	// Should not error when config file is missing, should use defaults
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error for missing config file, got %v", err)
	}

	// Should have default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoad_EnvironmentOverridesConfigFile(t *testing.T) {
	// Create config file
	configContent := `
server:
  port: 3000
  host: "localhost"
`

	configFile := "config.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	// Set environment variable that should override config file
	os.Setenv("DBT_SERVER_PORT", "4000")
	defer os.Unsetenv("DBT_SERVER_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Environment variable should override config file
	if cfg.Server.Port != 4000 {
		t.Errorf("Expected port from env var 4000 (overriding config file), got %d", cfg.Server.Port)
	}
	// Config file value should still be used for non-overridden values
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host from config file 'localhost', got %s", cfg.Server.Host)
	}
}
