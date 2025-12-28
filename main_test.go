package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"db-taxi/internal/config"
	"db-taxi/internal/server"
	"github.com/spf13/viper"
)

func TestMain_ConfigurationLoading(t *testing.T) {
	// Test that configuration can be loaded without errors
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected configuration to be loaded, got nil")
	}

	// Verify that essential configuration fields are set
	if cfg.Server.Port <= 0 {
		t.Errorf("Expected valid server port, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host == "" {
		t.Error("Expected server host to be set")
	}
	if cfg.Server.ReadTimeout <= 0 {
		t.Errorf("Expected positive read timeout, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout <= 0 {
		t.Errorf("Expected positive write timeout, got %v", cfg.Server.WriteTimeout)
	}
}

func TestMain_ServerCreation(t *testing.T) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create server (same as in main function)
	srv := server.New(cfg)
	if srv == nil {
		t.Fatal("Failed to create server")
	}

	// Verify server can be created without panics
	// This tests the same path as main() function
}

func TestMain_ServerStartStop(t *testing.T) {
	// Use a test configuration to avoid conflicts
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         0, // Use random port
			Host:         "localhost",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			EnableHTTPS:  false,
		},
		Logging: config.LoggingConfig{
			Level:  "error", // Reduce log noise
			Format: "json",
		},
	}

	srv := server.New(cfg)

	// Test server startup (similar to main function)
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test graceful shutdown (similar to main function)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop server gracefully: %v", err)
	}

	// Wait for server to stop
	select {
	case err := <-errChan:
		if err != nil && err.Error() != "http: Server closed" {
			t.Fatalf("Server stopped with unexpected error: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}

func TestMain_ConfigurationWithEnvironmentVariables(t *testing.T) {
	// Set environment variables that would affect main() execution
	os.Setenv("DBT_SERVER_PORT", "9999")
	os.Setenv("DBT_LOGGING_LEVEL", "debug")
	defer func() {
		os.Unsetenv("DBT_SERVER_PORT")
		os.Unsetenv("DBT_LOGGING_LEVEL")
	}()

	// Load configuration (same as main function)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration with env vars: %v", err)
	}

	// Verify environment variables are applied
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port from env var 9999, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level from env var 'debug', got %s", cfg.Logging.Level)
	}

	// Verify server can be created with env var configuration
	srv := server.New(cfg)
	if srv == nil {
		t.Fatal("Failed to create server with env var configuration")
	}
}

func TestMain_ConfigurationError(t *testing.T) {
	// Save any existing config file
	var existingConfig []byte
	var hadExistingConfig bool
	if data, err := os.ReadFile("config.yaml"); err == nil {
		existingConfig = data
		hadExistingConfig = true
	}

	// Create an invalid config file to test error handling
	invalidConfig := `
server:
  port: "invalid"
  read_timeout: "invalid_duration"
`

	err := os.WriteFile("config.yaml", []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	// Cleanup function
	defer func() {
		if hadExistingConfig {
			os.WriteFile("config.yaml", existingConfig, 0644)
		} else {
			os.Remove("config.yaml")
		}
	}()

	// This simulates what would happen in main() with invalid config
	// Note: We can't easily test the actual main() function exit,
	// but we can test that config.Load() returns an error
	_, err = config.Load()
	if err == nil {
		t.Error("Expected error when loading invalid configuration")
	}
}

// Helper function to simulate the main function logic without os.Exit
func simulateMain() error {
	// Reset viper to ensure clean state
	viper.Reset()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create server
	srv := server.New(cfg)
	if srv == nil {
		return fmt.Errorf("failed to create server")
	}

	return nil
}

func TestMain_SimulateFullStartup(t *testing.T) {
	// Ensure no config file exists for this test
	os.Remove("config.yaml")

	// Test the main function logic without actually running main()
	err := simulateMain()
	if err != nil {
		t.Fatalf("Simulated main function failed: %v", err)
	}
}
