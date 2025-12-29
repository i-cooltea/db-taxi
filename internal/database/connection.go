package database

import (
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
)

// ConnectionPool manages MySQL database connections
type ConnectionPool struct {
	db     *sqlx.DB
	config *config.DatabaseConfig
	logger *logrus.Logger
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(cfg *config.DatabaseConfig, logger *logrus.Logger) (*ConnectionPool, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Build MySQL DSN
	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	// Open database connection
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &ConnectionPool{
		db:     db,
		config: cfg,
		logger: logger,
	}

	logger.WithFields(logrus.Fields{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"database": cfg.Database,
		"ssl":      cfg.SSL,
	}).Info("MySQL connection pool created successfully")

	return pool, nil
}

// GetDB returns the database connection
func (cp *ConnectionPool) GetDB() *sqlx.DB {
	return cp.db
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() error {
	if cp.db == nil {
		return nil
	}

	cp.logger.Info("Closing MySQL connection pool")
	return cp.db.Close()
}

// Stats returns connection pool statistics
func (cp *ConnectionPool) Stats() map[string]interface{} {
	if cp.db == nil {
		return map[string]interface{}{}
	}

	stats := cp.db.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
	}
}

// TestConnection tests the database connection
func (cp *ConnectionPool) TestConnection() error {
	if cp.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	return cp.db.Ping()
}

// buildDSN builds a MySQL Data Source Name from configuration
func buildDSN(cfg *config.DatabaseConfig) (string, error) {
	mysqlConfig := mysql.Config{
		User:                 cfg.Username,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		DBName:               cfg.Database,
		Timeout:              cfg.QueryTimeout,
		ReadTimeout:          cfg.QueryTimeout,
		WriteTimeout:         cfg.QueryTimeout,
		AllowNativePasswords: true,
		ParseTime:            true,
		Loc:                  time.UTC,
	}

	// Configure SSL
	if cfg.SSL {
		mysqlConfig.TLSConfig = "true"
	} else {
		mysqlConfig.TLSConfig = "false"
	}

	return mysqlConfig.FormatDSN(), nil
}
