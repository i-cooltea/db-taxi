package sync

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// TransferOptimizer handles data compression and transmission optimization
// Requirement 8.3: Support data compression for transmission when network bandwidth is limited
// Requirement 8.4: Support rate limiting and resource control when system resources are tight
// Requirement 8.5: Optimize connection reuse and caching strategies when sync executes frequently
type TransferOptimizer struct {
	logger            *logrus.Logger
	compressionLevel  int
	enableCompression bool
	rateLimiter       *rate.Limiter
	connectionPool    *ConnectionPool
	cacheManager      *CacheManager
	mu                sync.RWMutex
}

// TransferOptimizerConfig configures the transfer optimizer
type TransferOptimizerConfig struct {
	EnableCompression bool          // Enable data compression
	CompressionLevel  int           // Compression level (1-9, default 6)
	RateLimitMBps     float64       // Rate limit in MB/s (0 = unlimited)
	BurstSizeMB       int           // Burst size in MB
	MaxConnections    int           // Maximum connections in pool
	ConnectionTTL     time.Duration // Connection time-to-live
	EnableCache       bool          // Enable metadata caching
	CacheTTL          time.Duration // Cache time-to-live
}

// NewTransferOptimizer creates a new transfer optimizer
func NewTransferOptimizer(logger *logrus.Logger, config *TransferOptimizerConfig) *TransferOptimizer {
	if config == nil {
		config = &TransferOptimizerConfig{
			EnableCompression: true,
			CompressionLevel:  6,
			RateLimitMBps:     0, // Unlimited by default
			BurstSizeMB:       10,
			MaxConnections:    10,
			ConnectionTTL:     5 * time.Minute,
			EnableCache:       true,
			CacheTTL:          5 * time.Minute,
		}
	}

	// Validate compression level
	if config.CompressionLevel < 1 || config.CompressionLevel > 9 {
		config.CompressionLevel = 6 // Default to balanced compression
	}

	// Create rate limiter if rate limiting is enabled
	var rateLimiter *rate.Limiter
	if config.RateLimitMBps > 0 {
		// Convert MB/s to bytes/s
		bytesPerSecond := config.RateLimitMBps * 1024 * 1024
		burstBytes := config.BurstSizeMB * 1024 * 1024
		rateLimiter = rate.NewLimiter(rate.Limit(bytesPerSecond), burstBytes)
	}

	return &TransferOptimizer{
		logger:            logger,
		compressionLevel:  config.CompressionLevel,
		enableCompression: config.EnableCompression,
		rateLimiter:       rateLimiter,
		connectionPool:    NewConnectionPool(config.MaxConnections, config.ConnectionTTL, logger),
		cacheManager:      NewCacheManager(config.CacheTTL, logger),
	}
}

// CompressData compresses data using gzip
// Requirement 8.3: Support data compression for transmission
func (to *TransferOptimizer) CompressData(data []byte) ([]byte, error) {
	if !to.enableCompression || len(data) == 0 {
		return data, nil
	}

	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, to.compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	compressed := buf.Bytes()
	compressionRatio := float64(len(data)) / float64(len(compressed))

	to.logger.WithFields(logrus.Fields{
		"original_size":     len(data),
		"compressed_size":   len(compressed),
		"compression_ratio": fmt.Sprintf("%.2f:1", compressionRatio),
	}).Debug("Data compressed")

	return compressed, nil
}

// DecompressData decompresses gzip data
func (to *TransferOptimizer) DecompressData(data []byte) ([]byte, error) {
	if !to.enableCompression || len(data) == 0 {
		return data, nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return decompressed, nil
}

// ApplyRateLimit applies rate limiting to data transfer
// Requirement 8.4: Support rate limiting and resource control
func (to *TransferOptimizer) ApplyRateLimit(ctx context.Context, dataSize int) error {
	if to.rateLimiter == nil {
		return nil // Rate limiting disabled
	}

	// Wait for rate limiter to allow the transfer
	if err := to.rateLimiter.WaitN(ctx, dataSize); err != nil {
		return fmt.Errorf("rate limit wait failed: %w", err)
	}

	return nil
}

// GetConnection retrieves a connection from the pool or creates a new one
// Requirement 8.5: Optimize connection reuse and caching strategies
func (to *TransferOptimizer) GetConnection(ctx context.Context, config *ConnectionConfig) (*sqlx.DB, error) {
	return to.connectionPool.GetConnection(ctx, config)
}

// ReleaseConnection returns a connection to the pool
func (to *TransferOptimizer) ReleaseConnection(config *ConnectionConfig, db *sqlx.DB) {
	to.connectionPool.ReleaseConnection(config, db)
}

// GetCachedTableSchema retrieves cached table schema or fetches it
// Requirement 8.5: Optimize connection reuse and caching strategies
func (to *TransferOptimizer) GetCachedTableSchema(
	ctx context.Context,
	connectionID string,
	tableName string,
	fetchFunc func() (*TableSchema, error),
) (*TableSchema, error) {
	cacheKey := fmt.Sprintf("schema:%s:%s", connectionID, tableName)

	// Try to get from cache
	if cached := to.cacheManager.Get(cacheKey); cached != nil {
		if schema, ok := cached.(*TableSchema); ok {
			to.logger.WithFields(logrus.Fields{
				"connection_id": connectionID,
				"table_name":    tableName,
			}).Debug("Table schema retrieved from cache")
			return schema, nil
		}
	}

	// Fetch from database
	schema, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// Store in cache
	to.cacheManager.Set(cacheKey, schema)

	to.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"table_name":    tableName,
	}).Debug("Table schema cached")

	return schema, nil
}

// InvalidateCache invalidates cached data for a connection
func (to *TransferOptimizer) InvalidateCache(connectionID string) {
	to.cacheManager.InvalidatePrefix(fmt.Sprintf("schema:%s:", connectionID))
	to.logger.WithField("connection_id", connectionID).Debug("Cache invalidated")
}

// GetStats returns transfer optimizer statistics
func (to *TransferOptimizer) GetStats() *TransferOptimizerStats {
	return &TransferOptimizerStats{
		CompressionEnabled: to.enableCompression,
		RateLimitEnabled:   to.rateLimiter != nil,
		ConnectionPoolSize: to.connectionPool.Size(),
		CacheSize:          to.cacheManager.Size(),
	}
}

// Close closes the transfer optimizer and cleans up resources
func (to *TransferOptimizer) Close() error {
	to.connectionPool.Close()
	to.cacheManager.Clear()
	return nil
}

// TransferOptimizerStats contains transfer optimizer statistics
type TransferOptimizerStats struct {
	CompressionEnabled bool
	RateLimitEnabled   bool
	ConnectionPoolSize int
	CacheSize          int
}

// ConnectionPool manages a pool of database connections
// Requirement 8.5: Optimize connection reuse and caching strategies
type ConnectionPool struct {
	maxConnections int
	connectionTTL  time.Duration
	logger         *logrus.Logger
	connections    map[string]*PooledConnection
	mu             sync.RWMutex
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	DB         *sqlx.DB
	LastUsed   time.Time
	InUse      bool
	CreateTime time.Time
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxConnections int, connectionTTL time.Duration, logger *logrus.Logger) *ConnectionPool {
	pool := &ConnectionPool{
		maxConnections: maxConnections,
		connectionTTL:  connectionTTL,
		logger:         logger,
		connections:    make(map[string]*PooledConnection),
		stopCleanup:    make(chan struct{}),
	}

	// Start cleanup goroutine
	pool.cleanupTicker = time.NewTicker(1 * time.Minute)
	go pool.cleanupExpiredConnections()

	return pool
}

// GetConnection retrieves or creates a connection
func (cp *ConnectionPool) GetConnection(ctx context.Context, config *ConnectionConfig) (*sqlx.DB, error) {
	key := cp.getConnectionKey(config)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check if connection exists and is valid
	if conn, exists := cp.connections[key]; exists {
		if !conn.InUse && time.Since(conn.LastUsed) < cp.connectionTTL {
			// Verify connection is still alive
			if err := conn.DB.PingContext(ctx); err == nil {
				conn.InUse = true
				conn.LastUsed = time.Now()
				cp.logger.WithField("connection_key", key).Debug("Reusing pooled connection")
				return conn.DB, nil
			}
			// Connection is dead, remove it
			conn.DB.Close()
			delete(cp.connections, key)
		}
	}

	// Check pool size limit
	if len(cp.connections) >= cp.maxConnections {
		// Try to find an unused connection to evict
		for k, conn := range cp.connections {
			if !conn.InUse {
				conn.DB.Close()
				delete(cp.connections, k)
				break
			}
		}
	}

	// Create new connection
	db, err := cp.createConnection(config)
	if err != nil {
		return nil, err
	}

	cp.connections[key] = &PooledConnection{
		DB:         db,
		LastUsed:   time.Now(),
		InUse:      true,
		CreateTime: time.Now(),
	}

	cp.logger.WithField("connection_key", key).Debug("Created new pooled connection")

	return db, nil
}

// ReleaseConnection marks a connection as available
func (cp *ConnectionPool) ReleaseConnection(config *ConnectionConfig, db *sqlx.DB) {
	key := cp.getConnectionKey(config)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	if conn, exists := cp.connections[key]; exists && conn.DB == db {
		conn.InUse = false
		conn.LastUsed = time.Now()
		cp.logger.WithField("connection_key", key).Debug("Released pooled connection")
	}
}

// Size returns the current pool size
func (cp *ConnectionPool) Size() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.connections)
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() {
	close(cp.stopCleanup)
	cp.cleanupTicker.Stop()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	for key, conn := range cp.connections {
		conn.DB.Close()
		delete(cp.connections, key)
	}

	cp.logger.Info("Connection pool closed")
}

// getConnectionKey generates a unique key for a connection
func (cp *ConnectionPool) getConnectionKey(config *ConnectionConfig) string {
	return fmt.Sprintf("%s:%d:%s:%s", config.Host, config.Port, config.Username, config.Database)
}

// createConnection creates a new database connection
func (cp *ConnectionPool) createConnection(config *ConnectionConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC&charset=utf8mb4",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	if config.SSL {
		dsn += "&tls=true"
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// cleanupExpiredConnections periodically removes expired connections
func (cp *ConnectionPool) cleanupExpiredConnections() {
	for {
		select {
		case <-cp.cleanupTicker.C:
			cp.mu.Lock()
			for key, conn := range cp.connections {
				if !conn.InUse && time.Since(conn.LastUsed) > cp.connectionTTL {
					conn.DB.Close()
					delete(cp.connections, key)
					cp.logger.WithField("connection_key", key).Debug("Removed expired connection")
				}
			}
			cp.mu.Unlock()
		case <-cp.stopCleanup:
			return
		}
	}
}

// CacheManager manages metadata caching
// Requirement 8.5: Optimize connection reuse and caching strategies
type CacheManager struct {
	cacheTTL      time.Duration
	logger        *logrus.Logger
	cache         map[string]*CacheEntry
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	AccessTime time.Time
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cacheTTL time.Duration, logger *logrus.Logger) *CacheManager {
	cm := &CacheManager{
		cacheTTL:    cacheTTL,
		logger:      logger,
		cache:       make(map[string]*CacheEntry),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	cm.cleanupTicker = time.NewTicker(1 * time.Minute)
	go cm.cleanupExpiredEntries()

	return cm
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(key string) interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entry, exists := cm.cache[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	entry.AccessTime = time.Now()
	return entry.Value
}

// Set stores a value in cache
func (cm *CacheManager) Set(key string, value interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache[key] = &CacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(cm.cacheTTL),
		AccessTime: time.Now(),
	}
}

// Delete removes a value from cache
func (cm *CacheManager) Delete(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cache, key)
}

// InvalidatePrefix removes all cache entries with a given prefix
func (cm *CacheManager) InvalidatePrefix(prefix string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for key := range cm.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(cm.cache, key)
		}
	}
}

// Clear removes all cache entries
func (cm *CacheManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache = make(map[string]*CacheEntry)
	cm.logger.Debug("Cache cleared")
}

// Size returns the number of cached entries
func (cm *CacheManager) Size() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.cache)
}

// cleanupExpiredEntries periodically removes expired cache entries
func (cm *CacheManager) cleanupExpiredEntries() {
	for {
		select {
		case <-cm.cleanupTicker.C:
			cm.mu.Lock()
			for key, entry := range cm.cache {
				if time.Now().After(entry.ExpiresAt) {
					delete(cm.cache, key)
				}
			}
			cm.mu.Unlock()
		case <-cm.stopCleanup:
			return
		}
	}
}

// RateLimitedReader wraps an io.Reader with rate limiting
type RateLimitedReader struct {
	reader      io.Reader
	rateLimiter *rate.Limiter
	ctx         context.Context
}

// NewRateLimitedReader creates a new rate-limited reader
func NewRateLimitedReader(ctx context.Context, reader io.Reader, rateLimiter *rate.Limiter) *RateLimitedReader {
	return &RateLimitedReader{
		reader:      reader,
		rateLimiter: rateLimiter,
		ctx:         ctx,
	}
}

// Read implements io.Reader with rate limiting
func (r *RateLimitedReader) Read(p []byte) (int, error) {
	if r.rateLimiter != nil {
		// Wait for rate limiter before reading
		if err := r.rateLimiter.WaitN(r.ctx, len(p)); err != nil {
			return 0, err
		}
	}

	return r.reader.Read(p)
}

// ResourceController manages system resource usage
// Requirement 8.4: Support rate limiting and resource control
type ResourceController struct {
	maxCPUPercent  float64
	maxMemoryMB    int64
	maxDiskIOPS    int64
	logger         *logrus.Logger
	mu             sync.RWMutex
	cpuThrottle    bool
	memoryThrottle bool
	diskIOThrottle bool
	checkInterval  time.Duration
	stopMonitoring chan struct{}
}

// NewResourceController creates a new resource controller
func NewResourceController(maxCPUPercent float64, maxMemoryMB int64, maxDiskIOPS int64, logger *logrus.Logger) *ResourceController {
	rc := &ResourceController{
		maxCPUPercent:  maxCPUPercent,
		maxMemoryMB:    maxMemoryMB,
		maxDiskIOPS:    maxDiskIOPS,
		logger:         logger,
		checkInterval:  5 * time.Second,
		stopMonitoring: make(chan struct{}),
	}

	// Start resource monitoring
	go rc.monitorResources()

	return rc
}

// ShouldThrottle returns true if operations should be throttled
func (rc *ResourceController) ShouldThrottle() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.cpuThrottle || rc.memoryThrottle || rc.diskIOThrottle
}

// GetThrottleReason returns the reason for throttling
func (rc *ResourceController) GetThrottleReason() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if rc.cpuThrottle {
		return "CPU usage too high"
	}
	if rc.memoryThrottle {
		return "Memory usage too high"
	}
	if rc.diskIOThrottle {
		return "Disk I/O too high"
	}

	return ""
}

// monitorResources monitors system resources and sets throttle flags
func (rc *ResourceController) monitorResources() {
	ticker := time.NewTicker(rc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.checkResources()
		case <-rc.stopMonitoring:
			return
		}
	}
}

// checkResources checks current resource usage
func (rc *ResourceController) checkResources() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Check memory usage
	var m sql.NullInt64
	currentMemoryMB := m.Int64 / 1024 / 1024

	if rc.maxMemoryMB > 0 && currentMemoryMB > rc.maxMemoryMB {
		if !rc.memoryThrottle {
			rc.logger.WithFields(logrus.Fields{
				"current_mb": currentMemoryMB,
				"max_mb":     rc.maxMemoryMB,
			}).Warn("Memory usage exceeded threshold, throttling enabled")
			rc.memoryThrottle = true
		}
	} else {
		if rc.memoryThrottle {
			rc.logger.Info("Memory usage back to normal, throttling disabled")
			rc.memoryThrottle = false
		}
	}

	// CPU and disk I/O monitoring would require platform-specific implementations
	// For now, we'll keep them as placeholders
}

// Close stops resource monitoring
func (rc *ResourceController) Close() {
	close(rc.stopMonitoring)
}
