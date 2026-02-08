package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
	"db-taxi/internal/database"
	"db-taxi/internal/sync"
)

// SyncManagerInterface defines the interface that Server needs from sync.Manager
type SyncManagerInterface interface {
	GetConnectionManager() sync.ConnectionManager
	GetSyncManager() sync.SyncManager
	GetMappingManager() sync.MappingManager
	GetJobEngine() sync.JobEngine
	GetSyncEngine() sync.SyncEngine
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
	HealthCheck(ctx context.Context) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// Server represents the HTTP server
type Server struct {
	config      *config.Config
	engine      *gin.Engine
	httpServer  *http.Server
	logger      *logrus.Logger
	dbPool      *database.ConnectionPool
	explorer    *database.SchemaExplorer
	syncManager SyncManagerInterface
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

	// Initialize database connection
	if err := server.initDatabase(); err != nil {
		logger.WithError(err).Warn("Failed to initialize database connection")
		// Continue without database - will show connection error in UI
	}

	// Initialize sync system
	if err := server.initSyncSystem(); err != nil {
		logger.WithError(err).Warn("Failed to initialize sync system")
		// Continue without sync system - sync features will be disabled
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

	s.logger.Infof("Starting server on [http://%s]", addr)

	// If sync system initialization failed during server creation, try again after starting the server
	// This ensures migrations are run even if database was initially unavailable
	if s.syncManager == nil && s.dbPool != nil && s.config.Sync.Enabled {
		s.logger.Info("Attempting to initialize sync system and run migrations...")
		go func() {
			// Give server a moment to fully start
			time.Sleep(2 * time.Second)
			if err := s.initSyncSystem(); err != nil {
				s.logger.WithError(err).Warn("Failed to initialize sync system after server start")
			} else {
				s.logger.Info("Successfully initialized sync system and ran migrations")
			}
		}()
	}

	if s.config.Server.EnableHTTPS {
		return s.httpServer.ListenAndServeTLS(s.config.Server.CertFile, s.config.Server.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping server...")

	// Shutdown sync system
	if s.syncManager != nil {
		if err := s.syncManager.Shutdown(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to shutdown sync system")
		}
	}

	// Close database connection
	if s.dbPool != nil {
		if err := s.dbPool.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close database connection")
		}
	}

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
		api.GET("/connection/test", s.testConnection)
		api.GET("/databases", s.getDatabases)
		api.GET("/databases/:database/tables", s.getTables)
		api.GET("/databases/:database/tables/:table", s.getTableInfo)
		api.GET("/databases/:database/tables/:table/data", s.getTableData)

		// Sync API routes
		if s.syncManager != nil {
			sync := api.Group("/sync")
			{
				// Connection management routes
				connections := sync.Group("/connections")
				{
					connections.GET("", s.getSyncConnections)
					connections.POST("", s.createSyncConnection)
					connections.POST("/test", s.testSyncConnectionConfig)
					connections.GET("/:id", s.getSyncConnection)
					connections.PUT("/:id", s.updateSyncConnection)
					connections.DELETE("/:id", s.deleteSyncConnection)
					connections.POST("/:id/test", s.testSyncConnection)
					connections.GET("/:id/databases", s.getConnectionDatabases)
					connections.GET("/:id/tables", s.getConnectionTables)
				}

				// Sync configuration routes
				configs := sync.Group("/configs")
				{
					configs.GET("", s.getSyncConfigs)
					configs.POST("", s.createSyncConfig)
					configs.GET("/:id", s.getSyncConfig)
					configs.PUT("/:id", s.updateSyncConfig)
					configs.DELETE("/:id", s.deleteSyncConfig)

					// Table mapping routes
					configs.GET("/:id/tables", s.getRemoteTables)
					configs.GET("/:id/tables/:table/schema", s.getRemoteTableSchema)
					configs.GET("/:id/mappings", s.getTableMappings)
					configs.POST("/:id/mappings", s.addTableMapping)
					configs.PUT("/:id/mappings/reorder", s.reorderTableMappings)
					configs.PUT("/:id/mappings/:mapping_id", s.updateTableMapping)
					configs.DELETE("/:id/mappings/:mapping_id", s.removeTableMapping)
					configs.POST("/:id/mappings/:mapping_id/toggle", s.toggleTableMapping)
					configs.POST("/:id/mappings/:mapping_id/sync-mode", s.setTableSyncMode)
				}

				// Job management routes
				jobs := sync.Group("/jobs")
				{
					jobs.GET("", s.getSyncJobs)
					jobs.POST("", s.startSyncJob)
					jobs.GET("/:id", s.getSyncJob)
					jobs.POST("/:id/stop", s.stopSyncJob)
					jobs.POST("/:id/cancel", s.cancelSyncJob)
					jobs.GET("/:id/logs", s.getSyncJobLogs)
					jobs.GET("/:id/progress", s.getSyncJobProgress)
					jobs.GET("/active", s.getActiveSyncJobs)
					jobs.GET("/history", s.getSyncJobHistory)
				}

				// System routes
				sync.GET("/status", s.getSyncStatus)
				sync.GET("/stats", s.getSyncStats)

				// Config management routes
				sync.GET("/config/export", s.exportConfig)
				sync.POST("/config/import", s.importConfig)
				sync.POST("/config/validate", s.validateConfig)
				sync.GET("/config/backup", s.backupConfig)
			}
		}
	}

	// Serve static files
	s.engine.Static("/static", "./static")
	// Serve assets directory directly (for frontend resource references)
	s.engine.Static("/assets", "./static/assets")

	// Serve index.html at root path
	s.engine.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// Handle client-side routing - serve index.html for all other routes
	s.engine.NoRoute(func(c *gin.Context) {
		c.File("./static/index.html")
	})
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
	status := gin.H{
		"server": gin.H{
			"status":  "running",
			"version": "1.0.0",
		},
	}

	// Add database status if available
	if s.dbPool != nil {
		status["database"] = gin.H{
			"connected": true,
			"stats":     s.dbPool.Stats(),
		}
	} else {
		status["database"] = gin.H{
			"connected": false,
			"error":     "Database connection not initialized",
		}
	}

	// Add sync system status if available
	if s.syncManager != nil {
		if err := s.syncManager.HealthCheck(c.Request.Context()); err != nil {
			status["sync"] = gin.H{
				"enabled": false,
				"error":   err.Error(),
			}
		} else {
			syncStats, err := s.syncManager.GetStats(c.Request.Context())
			if err != nil {
				s.logger.WithError(err).Warn("Failed to get sync stats")
				syncStats = gin.H{"error": "Failed to get stats"}
			}
			status["sync"] = gin.H{
				"enabled": true,
				"stats":   syncStats,
			}
		}
	} else {
		status["sync"] = gin.H{
			"enabled": false,
			"error":   "Sync system not initialized",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// initDatabase initializes the database connection
func (s *Server) initDatabase() error {
	// Create database connection pool
	pool, err := database.NewConnectionPool(&s.config.Database, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}

	s.dbPool = pool

	// Create schema explorer
	s.explorer = database.NewSchemaExplorer(pool.GetDB(), s.logger)

	return nil
}

// testConnection tests the database connection
func (s *Server) testConnection(c *gin.Context) {
	if s.dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Database connection not initialized",
		})
		return
	}

	if err := s.dbPool.TestConnection(); err != nil {
		s.logger.WithError(err).Error("Database connection test failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status": "connected",
			"stats":  s.dbPool.Stats(),
		},
	})
}

// getDatabases returns list of databases
func (s *Server) getDatabases(c *gin.Context) {
	if s.explorer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	databases, err := s.explorer.GetDatabases()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get databases")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    databases,
		"meta": gin.H{
			"total": len(databases),
		},
	})
}

// getTables returns list of tables in a database
func (s *Server) getTables(c *gin.Context) {
	if s.explorer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	database := c.Param("database")
	tables, err := s.explorer.GetTables(database)
	if err != nil {
		s.logger.WithError(err).WithField("database", database).Error("Failed to get tables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tables,
		"meta": gin.H{
			"total": len(tables),
		},
	})
}

// getTableInfo returns detailed information about a table
func (s *Server) getTableInfo(c *gin.Context) {
	if s.explorer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	database := c.Param("database")
	table := c.Param("table")

	tableInfo, err := s.explorer.GetTableInfo(database, table)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"database": database,
			"table":    table,
		}).Error("Failed to get table info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tableInfo,
	})
}

// getTableData returns data from a table
func (s *Server) getTableData(c *gin.Context) {
	if s.explorer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	database := c.Param("database")
	table := c.Param("table")

	// Parse query parameters
	limit := 10
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := fmt.Sscanf(l, "%d", &limit); err != nil || parsed != 1 {
			limit = 10
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := fmt.Sscanf(o, "%d", &offset); err != nil || parsed != 1 {
			offset = 0
		}
	}

	// Validate limits
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	data, err := s.explorer.GetTableData(database, table, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"database": database,
			"table":    table,
			"limit":    limit,
			"offset":   offset,
		}).Error("Failed to get table data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"meta": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(data),
		},
	})
}

// initSyncSystem initializes the sync system
func (s *Server) initSyncSystem() error {
	s.logger.Info("Initializing sync system...")

	if s.dbPool == nil {
		s.logger.Error("Database connection required for sync system")
		return fmt.Errorf("database connection required for sync system")
	}

	s.logger.Info("Creating sync manager...")
	// Create sync manager
	syncManager, err := sync.NewManager(s.config, s.dbPool.GetDB(), s.logger)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create sync manager")
		return fmt.Errorf("failed to create sync manager: %w", err)
	}

	s.logger.Info("Initializing sync manager...")
	// Initialize sync system
	ctx := context.Background()
	if err := syncManager.Initialize(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to initialize sync system")
		return fmt.Errorf("failed to initialize sync system: %w", err)
	}

	s.syncManager = syncManager
	s.logger.Info("Sync system initialized successfully")
	return nil
}

// registerSyncRoutes registers sync-related API routes
func (s *Server) registerSyncRoutes(api *gin.RouterGroup) {
}

// Sync connection handlers

func (s *Server) getSyncConnections(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	connections, err := s.syncManager.GetConnectionManager().GetConnections(c.Request.Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to get sync connections")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    connections,
		"meta": gin.H{
			"total": len(connections),
		},
	})
}

func (s *Server) createSyncConnection(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	var config sync.ConnectionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	connection, err := s.syncManager.GetConnectionManager().AddConnection(c.Request.Context(), &config)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create sync connection")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    connection,
	})
}

func (s *Server) getSyncConnection(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	connection, err := s.syncManager.GetConnectionManager().GetConnection(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to get sync connection")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    connection,
	})
}

func (s *Server) updateSyncConnection(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	var config sync.ConnectionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetConnectionManager().UpdateConnection(c.Request.Context(), id, &config); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to update sync connection")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection updated successfully",
	})
}

func (s *Server) deleteSyncConnection(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	if err := s.syncManager.GetConnectionManager().DeleteConnection(c.Request.Context(), id); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to delete sync connection")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection deleted successfully",
	})
}

func (s *Server) testSyncConnection(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	status, err := s.syncManager.GetConnectionManager().TestConnection(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to test sync connection")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func (s *Server) testSyncConnectionConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	var config sync.ConnectionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Test the connection without saving it
	status, err := s.syncManager.GetConnectionManager().TestConnectionConfig(c.Request.Context(), &config)
	if err != nil {
		s.logger.WithError(err).Error("Failed to test connection config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func (s *Server) getConnectionTables(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	connectionID := c.Param("id")
	database := c.Query("database")
	if database == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "database parameter is required",
		})
		return
	}

	tables, err := s.syncManager.GetSyncManager().GetRemoteTables(c.Request.Context(), connectionID, database)
	if err != nil {
		s.logger.WithError(err).WithField("connection_id", connectionID).Error("Failed to get connection tables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tables,
		"meta": gin.H{
			"total": len(tables),
		},
	})
}

func (s *Server) getConnectionDatabases(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	connectionID := c.Param("id")
	databases, err := s.syncManager.GetSyncManager().GetRemoteDatabases(c.Request.Context(), connectionID)
	if err != nil {
		s.logger.WithError(err).WithField("connection_id", connectionID).Error("Failed to get connection databases")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    databases,
		"meta": gin.H{
			"total": len(databases),
		},
	})
}

// Sync configuration handlers

func (s *Server) getSyncConfigs(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	connectionID := c.Query("connection_id")
	if connectionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "connection_id parameter is required",
		})
		return
	}

	configs, err := s.syncManager.GetSyncManager().GetSyncConfigs(c.Request.Context(), connectionID)
	if err != nil {
		s.logger.WithError(err).WithField("connection_id", connectionID).Error("Failed to get sync configs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configs,
		"meta": gin.H{
			"total": len(configs),
		},
	})
}

func (s *Server) createSyncConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	var config sync.SyncConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetSyncManager().CreateSyncConfig(c.Request.Context(), &config); err != nil {
		s.logger.WithError(err).Error("Failed to create sync config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    config,
	})
}

func (s *Server) getSyncConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	config, err := s.syncManager.GetSyncManager().GetSyncConfig(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to get sync config")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

func (s *Server) updateSyncConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	var config sync.SyncConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetSyncManager().UpdateSyncConfig(c.Request.Context(), id, &config); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to update sync config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sync config updated successfully",
	})
}

func (s *Server) deleteSyncConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	if err := s.syncManager.GetSyncManager().DeleteSyncConfig(c.Request.Context(), id); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to delete sync config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sync config deleted successfully",
	})
}

// Table mapping handlers

func (s *Server) getRemoteTables(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	// Get connection ID from sync config
	configID := c.Param("id")
	config, err := s.syncManager.GetSyncManager().GetSyncConfig(c.Request.Context(), configID)
	if err != nil {
		s.logger.WithError(err).WithField("config_id", configID).Error("Failed to get sync config")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	tables, err := s.syncManager.GetSyncManager().GetRemoteTables(c.Request.Context(), config.SourceConnectionID, config.SourceDatabase)
	if err != nil {
		s.logger.WithError(err).WithField("connection_id", config.SourceConnectionID).Error("Failed to get remote tables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tables,
		"meta": gin.H{
			"total": len(tables),
		},
	})
}

func (s *Server) getRemoteTableSchema(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	// Get connection ID from sync config
	configID := c.Param("id")
	tableName := c.Param("table")

	config, err := s.syncManager.GetSyncManager().GetSyncConfig(c.Request.Context(), configID)
	if err != nil {
		s.logger.WithError(err).WithField("config_id", configID).Error("Failed to get sync config")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	schema, err := s.syncManager.GetSyncManager().GetRemoteTableSchema(c.Request.Context(), config.SourceConnectionID, config.SourceDatabase, tableName)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"connection_id": config.SourceConnectionID,
			"table":         tableName,
		}).Error("Failed to get remote table schema")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schema,
	})
}

func (s *Server) getTableMappings(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	configID := c.Param("id")
	mappings, err := s.syncManager.GetSyncManager().GetTableMappings(c.Request.Context(), configID)
	if err != nil {
		s.logger.WithError(err).WithField("config_id", configID).Error("Failed to get table mappings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    mappings,
		"meta": gin.H{
			"total": len(mappings),
		},
	})
}

func (s *Server) addTableMapping(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	configID := c.Param("id")
	var mapping sync.TableMapping
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetSyncManager().AddTableMapping(c.Request.Context(), configID, &mapping); err != nil {
		s.logger.WithError(err).WithField("config_id", configID).Error("Failed to add table mapping")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    mapping,
	})
}

func (s *Server) updateTableMapping(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	mappingID := c.Param("mapping_id")
	var mapping sync.TableMapping
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetSyncManager().UpdateTableMapping(c.Request.Context(), mappingID, &mapping); err != nil {
		s.logger.WithError(err).WithField("mapping_id", mappingID).Error("Failed to update table mapping")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Table mapping updated successfully",
	})
}

func (s *Server) removeTableMapping(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	mappingID := c.Param("mapping_id")
	if err := s.syncManager.GetSyncManager().RemoveTableMapping(c.Request.Context(), mappingID); err != nil {
		s.logger.WithError(err).WithField("mapping_id", mappingID).Error("Failed to remove table mapping")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Table mapping removed successfully",
	})
}

func (s *Server) toggleTableMapping(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	mappingID := c.Param("mapping_id")
	var request struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetSyncManager().ToggleTableMapping(c.Request.Context(), mappingID, request.Enabled); err != nil {
		s.logger.WithError(err).WithField("mapping_id", mappingID).Error("Failed to toggle table mapping")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Table mapping toggled successfully",
	})
}

func (s *Server) setTableSyncMode(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	mappingID := c.Param("mapping_id")
	var request struct {
		SyncMode sync.SyncMode `json:"sync_mode" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetSyncManager().SetTableSyncMode(c.Request.Context(), mappingID, request.SyncMode); err != nil {
		s.logger.WithError(err).WithField("mapping_id", mappingID).Error("Failed to set table sync mode")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Table sync mode updated successfully",
	})
}

func (s *Server) reorderTableMappings(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	configID := c.Param("id")
	var request struct {
		MappingIDs []string `json:"mapping_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: mapping_ids required",
		})
		return
	}

	if err := s.syncManager.GetSyncManager().ReorderTableMappings(c.Request.Context(), configID, request.MappingIDs); err != nil {
		s.logger.WithError(err).WithField("config_id", configID).Error("Failed to reorder table mappings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Table mappings reordered successfully",
	})
}

// Job management handlers

func (s *Server) getSyncJobs(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	// Parse query parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := fmt.Sscanf(l, "%d", &limit); err != nil || parsed != 1 {
			limit = 50
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := fmt.Sscanf(o, "%d", &offset); err != nil || parsed != 1 {
			offset = 0
		}
	}

	// Get job history
	jobs, err := s.syncManager.GetSyncManager().GetSyncHistory(c.Request.Context(), limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get sync jobs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    jobs,
		"meta": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(jobs),
		},
	})
}

func (s *Server) startSyncJob(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	var request struct {
		ConfigID string `json:"config_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	job, err := s.syncManager.GetSyncManager().StartSync(c.Request.Context(), request.ConfigID)
	if err != nil {
		s.logger.WithError(err).WithField("config_id", request.ConfigID).Error("Failed to start sync job")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    job,
	})
}

func (s *Server) getSyncJob(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	job, err := s.syncManager.GetSyncManager().GetSyncStatus(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to get sync job")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

func (s *Server) stopSyncJob(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	if err := s.syncManager.GetSyncManager().StopSync(c.Request.Context(), id); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to stop sync job")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sync job stop requested",
	})
}

func (s *Server) getSyncJobLogs(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	logs, err := s.syncManager.GetSyncManager().GetJobLogs(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).WithField("job_id", id).Error("Failed to get sync job logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
		"meta": gin.H{
			"total": len(logs),
		},
	})
}

func (s *Server) cancelSyncJob(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	if err := s.syncManager.GetSyncManager().StopSync(c.Request.Context(), id); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("Failed to cancel sync job")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sync job cancelled successfully",
	})
}

func (s *Server) getSyncJobProgress(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	id := c.Param("id")
	progress, err := s.syncManager.GetSyncManager().GetJobProgress(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).WithField("job_id", id).Error("Failed to get sync job progress")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    progress,
	})
}

func (s *Server) getActiveSyncJobs(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	jobs, err := s.syncManager.GetSyncManager().GetActiveJobs(c.Request.Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active sync jobs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    jobs,
		"meta": gin.H{
			"total": len(jobs),
		},
	})
}

func (s *Server) getSyncJobHistory(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	// Parse query parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := fmt.Sscanf(l, "%d", &limit); err != nil || parsed != 1 {
			limit = 50
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := fmt.Sscanf(o, "%d", &offset); err != nil || parsed != 1 {
			offset = 0
		}
	}

	history, err := s.syncManager.GetSyncManager().GetSyncHistory(c.Request.Context(), limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get sync job history")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
		"meta": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(history),
		},
	})
}

// System status handlers

func (s *Server) getSyncStatus(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	if err := s.syncManager.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		},
	})
}

func (s *Server) getSyncStats(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	stats, err := s.syncManager.GetStats(c.Request.Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to get sync stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// Config management handlers

func (s *Server) exportConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	config, err := s.syncManager.GetMappingManager().ExportConfig(c.Request.Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to export config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("sync-config-%s.json", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

func (s *Server) importConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	var request struct {
		Config           *sync.ConfigExport `json:"config" binding:"required"`
		ResolveConflicts bool               `json:"resolve_conflicts"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate config first
	if err := s.syncManager.GetMappingManager().ValidateConfig(c.Request.Context(), request.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Configuration validation failed: " + err.Error(),
		})
		return
	}

	// Import with conflict resolution
	if err := s.syncManager.GetMappingManager().ImportConfigWithConflictResolution(
		c.Request.Context(),
		request.Config,
		request.ResolveConflicts,
	); err != nil {
		s.logger.WithError(err).Error("Failed to import config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration imported successfully",
	})
}

func (s *Server) validateConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	var config sync.ConfigExport
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.syncManager.GetMappingManager().ValidateConfig(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"valid":   true,
		"message": "Configuration is valid",
	})
}

func (s *Server) backupConfig(c *gin.Context) {
	if s.syncManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Sync system not available",
		})
		return
	}

	backup, err := s.syncManager.GetMappingManager().BackupConfig(c.Request.Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to backup config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("sync-config-backup-%s.json", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    backup,
	})
}
