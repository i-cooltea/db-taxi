package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	engine     *gin.Engine
	httpServer *http.Server
	logger     *logrus.Logger
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Set Gin mode based on log level
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if cfg.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Create Gin engine
	engine := gin.New()

	// Add middleware
	engine.Use(gin.Recovery())
	engine.Use(LoggerMiddleware(logger))
	engine.Use(CORSMiddleware())

	server := &Server{
		config: cfg,
		engine: engine,
		logger: logger,
	}

	// Register routes
	server.registerRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	s.logger.Infof("Starting server on %s", addr)

	if s.config.Server.EnableHTTPS {
		return s.httpServer.ListenAndServeTLS(s.config.Server.CertFile, s.config.Server.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping server...")
	return s.httpServer.Shutdown(ctx)
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// Health check endpoint
	s.engine.GET("/health", s.healthCheck)

	// API routes group
	api := s.engine.Group("/api")
	{
		api.GET("/status", s.getStatus)
	}

	// Serve static files (will be implemented later)
	s.engine.Static("/static", "./static")
	s.engine.StaticFile("/", "./static/index.html")
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// getStatus handles status requests
func (s *Server) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"server": gin.H{
				"status":  "running",
				"uptime":  time.Since(time.Now()).String(), // Will be properly calculated later
				"version": "1.0.0",
			},
		},
	})
}
