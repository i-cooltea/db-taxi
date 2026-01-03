package sync

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferOptimizer_CompressData(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name             string
		config           *TransferOptimizerConfig
		data             []byte
		expectCompressed bool
	}{
		{
			name: "compress enabled with data",
			config: &TransferOptimizerConfig{
				EnableCompression: true,
				CompressionLevel:  6,
			},
			data:             bytes.Repeat([]byte("test data "), 100),
			expectCompressed: true,
		},
		{
			name: "compress disabled",
			config: &TransferOptimizerConfig{
				EnableCompression: false,
			},
			data:             bytes.Repeat([]byte("test data "), 100),
			expectCompressed: false,
		},
		{
			name: "empty data",
			config: &TransferOptimizerConfig{
				EnableCompression: true,
			},
			data:             []byte{},
			expectCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer := NewTransferOptimizer(logger, tt.config)

			compressed, err := optimizer.CompressData(tt.data)
			require.NoError(t, err)

			if tt.expectCompressed {
				// Compressed data should be smaller for repetitive data
				assert.Less(t, len(compressed), len(tt.data), "Compressed data should be smaller")

				// Verify we can decompress it back
				decompressed, err := optimizer.DecompressData(compressed)
				require.NoError(t, err)
				assert.Equal(t, tt.data, decompressed, "Decompressed data should match original")
			} else {
				// Data should be unchanged
				assert.Equal(t, tt.data, compressed, "Data should not be compressed")
			}
		})
	}
}

func TestTransferOptimizer_CompressionLevels(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	data := bytes.Repeat([]byte("test data with some repetition "), 1000)

	levels := []int{1, 6, 9}
	for _, level := range levels {
		t.Run(fmt.Sprintf("level_%d", level), func(t *testing.T) {
			config := &TransferOptimizerConfig{
				EnableCompression: true,
				CompressionLevel:  level,
			}
			optimizer := NewTransferOptimizer(logger, config)

			compressed, err := optimizer.CompressData(data)
			require.NoError(t, err)
			assert.Less(t, len(compressed), len(data))

			// Verify decompression
			decompressed, err := optimizer.DecompressData(compressed)
			require.NoError(t, err)
			assert.Equal(t, data, decompressed)
		})
	}
}

func TestTransferOptimizer_RateLimit(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("rate limit disabled", func(t *testing.T) {
		config := &TransferOptimizerConfig{
			RateLimitMBps: 0, // Disabled
		}
		optimizer := NewTransferOptimizer(logger, config)

		ctx := context.Background()
		start := time.Now()

		// Should complete immediately
		err := optimizer.ApplyRateLimit(ctx, 1024*1024) // 1 MB
		require.NoError(t, err)

		duration := time.Since(start)
		assert.Less(t, duration, 100*time.Millisecond, "Should complete immediately when rate limit disabled")
	})

	t.Run("rate limit enabled", func(t *testing.T) {
		config := &TransferOptimizerConfig{
			RateLimitMBps: 1.0, // 1 MB/s
			BurstSizeMB:   1,   // 1 MB burst
		}
		optimizer := NewTransferOptimizer(logger, config)

		ctx := context.Background()
		start := time.Now()

		// Transfer 1 MB (within burst) - should be fast
		err := optimizer.ApplyRateLimit(ctx, 1024*1024)
		require.NoError(t, err)

		// Transfer another 1 MB - should be rate limited
		err = optimizer.ApplyRateLimit(ctx, 1024*1024)
		require.NoError(t, err)

		duration := time.Since(start)
		// Second transfer should take at least 0.5 seconds due to rate limit
		assert.GreaterOrEqual(t, duration, 500*time.Millisecond, "Should be throttled by rate limit on second transfer")
	})

	t.Run("rate limit with context cancellation", func(t *testing.T) {
		config := &TransferOptimizerConfig{
			RateLimitMBps: 0.1, // Very slow: 0.1 MB/s
			BurstSizeMB:   1,
		}
		optimizer := NewTransferOptimizer(logger, config)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Try to transfer 10 MB - should be cancelled by context
		err := optimizer.ApplyRateLimit(ctx, 10*1024*1024)
		assert.Error(t, err, "Should fail due to context cancellation")
	})
}

func TestConnectionPool_GetAndRelease(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	pool := NewConnectionPool(5, 5*time.Minute, logger)
	defer pool.Close()

	config := &ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "test",
		Password: "test",
		Database: "testdb",
	}

	t.Run("connection key generation", func(t *testing.T) {
		key1 := pool.getConnectionKey(config)
		key2 := pool.getConnectionKey(config)
		assert.Equal(t, key1, key2, "Same config should generate same key")

		config2 := &ConnectionConfig{
			Host:     "localhost",
			Port:     3307, // Different port
			Username: "test",
			Password: "test",
			Database: "testdb",
		}
		key3 := pool.getConnectionKey(config2)
		assert.NotEqual(t, key1, key3, "Different config should generate different key")
	})

	t.Run("pool size tracking", func(t *testing.T) {
		initialSize := pool.Size()
		assert.GreaterOrEqual(t, initialSize, 0)
	})
}

func TestCacheManager_Operations(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cache := NewCacheManager(1*time.Second, logger)

	t.Run("set and get", func(t *testing.T) {
		cache.Set("key1", "value1")
		value := cache.Get("key1")
		assert.Equal(t, "value1", value)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		value := cache.Get("non-existent")
		assert.Nil(t, value)
	})

	t.Run("cache expiration", func(t *testing.T) {
		cache.Set("expiring-key", "expiring-value")

		// Should exist immediately
		value := cache.Get("expiring-key")
		assert.Equal(t, "expiring-value", value)

		// Wait for expiration
		time.Sleep(1500 * time.Millisecond)

		// Should be expired
		value = cache.Get("expiring-key")
		assert.Nil(t, value)
	})

	t.Run("delete", func(t *testing.T) {
		cache.Set("key-to-delete", "value")
		assert.NotNil(t, cache.Get("key-to-delete"))

		cache.Delete("key-to-delete")
		assert.Nil(t, cache.Get("key-to-delete"))
	})

	t.Run("invalidate prefix", func(t *testing.T) {
		cache.Set("prefix:key1", "value1")
		cache.Set("prefix:key2", "value2")
		cache.Set("other:key3", "value3")

		cache.InvalidatePrefix("prefix:")

		assert.Nil(t, cache.Get("prefix:key1"))
		assert.Nil(t, cache.Get("prefix:key2"))
		assert.NotNil(t, cache.Get("other:key3"))
	})

	t.Run("clear", func(t *testing.T) {
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")
		assert.Greater(t, cache.Size(), 0)

		cache.Clear()
		assert.Equal(t, 0, cache.Size())
	})

	t.Run("size tracking", func(t *testing.T) {
		cache.Clear()
		assert.Equal(t, 0, cache.Size())

		cache.Set("key1", "value1")
		assert.Equal(t, 1, cache.Size())

		cache.Set("key2", "value2")
		assert.Equal(t, 2, cache.Size())

		cache.Delete("key1")
		assert.Equal(t, 1, cache.Size())
	})
}

func TestTransferOptimizer_GetCachedTableSchema(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := &TransferOptimizerConfig{
		EnableCache: true,
		CacheTTL:    1 * time.Second,
	}
	optimizer := NewTransferOptimizer(logger, config)

	schema := &TableSchema{
		Name: "test_table",
		Columns: []*ColumnInfo{
			{Name: "id", Type: "INT"},
			{Name: "name", Type: "VARCHAR(255)"},
		},
	}

	fetchCount := 0
	fetchFunc := func() (*TableSchema, error) {
		fetchCount++
		return schema, nil
	}

	ctx := context.Background()

	t.Run("first fetch", func(t *testing.T) {
		result, err := optimizer.GetCachedTableSchema(ctx, "conn1", "test_table", fetchFunc)
		require.NoError(t, err)
		assert.Equal(t, schema, result)
		assert.Equal(t, 1, fetchCount, "Should fetch from database")
	})

	t.Run("cached fetch", func(t *testing.T) {
		result, err := optimizer.GetCachedTableSchema(ctx, "conn1", "test_table", fetchFunc)
		require.NoError(t, err)
		assert.Equal(t, schema, result)
		assert.Equal(t, 1, fetchCount, "Should use cached value, not fetch again")
	})

	t.Run("cache expiration", func(t *testing.T) {
		time.Sleep(1500 * time.Millisecond)

		result, err := optimizer.GetCachedTableSchema(ctx, "conn1", "test_table", fetchFunc)
		require.NoError(t, err)
		assert.Equal(t, schema, result)
		assert.Equal(t, 2, fetchCount, "Should fetch again after expiration")
	})

	t.Run("cache invalidation", func(t *testing.T) {
		// Fetch and cache
		_, err := optimizer.GetCachedTableSchema(ctx, "conn1", "test_table", fetchFunc)
		require.NoError(t, err)

		// Invalidate cache
		optimizer.InvalidateCache("conn1")

		// Should fetch again
		_, err = optimizer.GetCachedTableSchema(ctx, "conn1", "test_table", fetchFunc)
		require.NoError(t, err)
		assert.Equal(t, 3, fetchCount, "Should fetch again after invalidation")
	})
}

func TestTransferOptimizer_Stats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := &TransferOptimizerConfig{
		EnableCompression: true,
		RateLimitMBps:     1.0,
		EnableCache:       true,
	}
	optimizer := NewTransferOptimizer(logger, config)

	stats := optimizer.GetStats()
	assert.True(t, stats.CompressionEnabled)
	assert.True(t, stats.RateLimitEnabled)
	assert.GreaterOrEqual(t, stats.ConnectionPoolSize, 0)
	assert.GreaterOrEqual(t, stats.CacheSize, 0)
}

func TestResourceController_Throttling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("no throttling by default", func(t *testing.T) {
		rc := NewResourceController(80.0, 1024, 1000, logger)
		defer rc.Close()

		assert.False(t, rc.ShouldThrottle())
		assert.Empty(t, rc.GetThrottleReason())
	})

	t.Run("throttle reason", func(t *testing.T) {
		rc := NewResourceController(80.0, 1024, 1000, logger)
		defer rc.Close()

		// Manually set throttle flags for testing
		rc.mu.Lock()
		rc.memoryThrottle = true
		rc.mu.Unlock()

		assert.True(t, rc.ShouldThrottle())
		assert.NotEmpty(t, rc.GetThrottleReason())
	})
}

func TestTransferOptimizer_CompressionRatio(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := &TransferOptimizerConfig{
		EnableCompression: true,
		CompressionLevel:  6,
	}
	optimizer := NewTransferOptimizer(logger, config)

	tests := []struct {
		name                string
		data                []byte
		minCompressionRatio float64
	}{
		{
			name:                "highly repetitive data",
			data:                bytes.Repeat([]byte("A"), 10000),
			minCompressionRatio: 10.0, // Should compress very well
		},
		{
			name:                "moderately repetitive data",
			data:                bytes.Repeat([]byte("test data "), 1000),
			minCompressionRatio: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := optimizer.CompressData(tt.data)
			require.NoError(t, err)

			ratio := float64(len(tt.data)) / float64(len(compressed))
			assert.GreaterOrEqual(t, ratio, tt.minCompressionRatio,
				"Compression ratio should be at least %.2f:1", tt.minCompressionRatio)

			t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2f:1",
				len(tt.data), len(compressed), ratio)
		})
	}
}

func TestTransferOptimizer_Close(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	optimizer := NewTransferOptimizer(logger, nil)

	// Add some cached data
	optimizer.cacheManager.Set("test-key", "test-value")
	assert.Greater(t, optimizer.cacheManager.Size(), 0)

	// Close should clean up resources
	err := optimizer.Close()
	assert.NoError(t, err)

	// Cache should be cleared
	assert.Equal(t, 0, optimizer.cacheManager.Size())
}

// Benchmark tests
func BenchmarkCompression(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := &TransferOptimizerConfig{
		EnableCompression: true,
		CompressionLevel:  6,
	}
	optimizer := NewTransferOptimizer(logger, config)

	data := bytes.Repeat([]byte("benchmark test data "), 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.CompressData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompression(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := &TransferOptimizerConfig{
		EnableCompression: true,
		CompressionLevel:  6,
	}
	optimizer := NewTransferOptimizer(logger, config)

	data := bytes.Repeat([]byte("benchmark test data "), 1000)
	compressed, _ := optimizer.CompressData(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.DecompressData(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}
