package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"db-taxi/internal/config"
)

func TestNew_CreatesServerWithConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			Host:         "localhost",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			EnableHTTPS:  false,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	server := New(cfg)

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}
	if server.config != cfg {
		t.Error("Expected server config to match provided config")
	}
	if server.engine == nil {
		t.Error("Expected Gin engine to be initialized")
	}
	if server.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestNew_SetsGinModeBasedOnLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		// Note: We can't easily test Gin mode directly as it's a global setting
		// This test mainly ensures no panics occur with different log levels
	}{
		{"Debug mode", "debug"},
		{"Info mode", "info"},
		{"Warn mode", "warn"},
		{"Error mode", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Port: 8080,
					Host: "localhost",
				},
				Logging: config.LoggingConfig{
					Level:  tt.logLevel,
					Format: "json",
				},
			}

			server := New(cfg)
			if server == nil {
				t.Fatalf("Expected server to be created for log level %s", tt.logLevel)
			}
		})
	}
}

func TestServer_StartAndStop(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         0, // Use port 0 to get a random available port
			Host:         "localhost",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			EnableHTTPS:  false,
		},
		Logging: config.LoggingConfig{
			Level:  "error", // Reduce log noise during tests
			Format: "json",
		},
	}

	server := New(cfg)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is running by making a request to health endpoint
	// Note: Since we used port 0, we need to find the actual port
	// For this test, we'll just verify that Start() was called without immediate error
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("Server failed to start: %v", err)
		}
	default:
		// Server is running, now stop it
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatalf("Failed to stop server: %v", err)
		}

		// Wait for server to actually stop
		select {
		case err := <-errChan:
			if err != nil && err != http.ErrServerClosed {
				t.Fatalf("Server stopped with unexpected error: %v", err)
			}
		case <-time.After(6 * time.Second):
			t.Fatal("Server did not stop within timeout")
		}
	}
}

func TestServer_StartWithHTTPS(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         0,
			Host:         "localhost",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			EnableHTTPS:  true,
			CertFile:     "nonexistent.crt", // This will cause an error, which is expected
			KeyFile:      "nonexistent.key",
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}

	server := New(cfg)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Should get an error because cert files don't exist
	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected error when starting HTTPS server with nonexistent cert files")
		}
	case <-time.After(2 * time.Second):
		t.Error("Expected immediate error for missing cert files")
		// Try to stop server if it somehow started
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Stop(ctx)
	}
}

func TestServer_StopWithTimeout(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         0,
			Host:         "localhost",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			EnableHTTPS:  false,
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}

	server := New(cfg)

	// Start server
	go func() {
		server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop with very short timeout to test timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err := server.Stop(ctx)
	// Should either succeed quickly or return context deadline exceeded
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Expected nil or context deadline exceeded, got: %v", err)
	}
}

func TestServer_HealthCheckEndpoint(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 0,
			Host: "localhost",
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}

	server := New(cfg)

	// Test that routes are registered by checking if the engine has routes
	routes := server.engine.Routes()

	healthRouteFound := false
	for _, route := range routes {
		if route.Path == "/health" && route.Method == "GET" {
			healthRouteFound = true
			break
		}
	}

	if !healthRouteFound {
		t.Error("Expected /health route to be registered")
	}
}

func TestServer_APIRoutesRegistered(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 0,
			Host: "localhost",
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
	}

	server := New(cfg)

	// Test that API routes are registered
	routes := server.engine.Routes()

	statusRouteFound := false
	for _, route := range routes {
		if route.Path == "/api/status" && route.Method == "GET" {
			statusRouteFound = true
			break
		}
	}

	if !statusRouteFound {
		t.Error("Expected /api/status route to be registered")
	}
}

func TestServer_ConfigurationValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		valid  bool
	}{
		{
			name: "Valid configuration",
			config: &config.Config{
				Server: config.ServerConfig{
					Port:         8080,
					Host:         "localhost",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: config.LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			valid: true,
		},
		{
			name: "Zero port (should work - OS assigns port)",
			config: &config.Config{
				Server: config.ServerConfig{
					Port:         0,
					Host:         "localhost",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: config.LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := New(tt.config)
			if tt.valid && server == nil {
				t.Error("Expected valid server to be created")
			}
			if !tt.valid && server != nil {
				t.Error("Expected invalid configuration to fail")
			}
		})
	}
}
