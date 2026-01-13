package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
	"db-taxi/internal/database"
	"db-taxi/internal/sync"
)

func main() {
	configFile := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("Job Engine 测试工具")
	fmt.Println("========================================")
	fmt.Println()

	// Load configuration
	fmt.Println("1. 加载配置...")
	cfg, err := config.LoadWithOptions(&config.LoadOptions{
		ConfigFile: *configFile,
	})
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	fmt.Printf("✓ 配置加载成功\n")
	fmt.Printf("  - Sync Enabled: %v\n", cfg.Sync.Enabled)
	fmt.Printf("  - Database: %s@%s:%d/%s\n", cfg.Database.Username, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)
	fmt.Println()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Create database connection
	fmt.Println("2. 连接数据库...")
	dbPool, err := database.NewConnectionPool(&cfg.Database, logger)
	if err != nil {
		log.Fatalf("❌ 创建数据库连接池失败: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.TestConnection(); err != nil {
		log.Fatalf("❌ 数据库连接测试失败: %v", err)
	}
	fmt.Println("✓ 数据库连接成功")
	fmt.Println()

	// Check if sync is enabled
	if !cfg.Sync.Enabled {
		log.Fatal("❌ Sync 系统被禁用，请在配置中启用")
	}

	// Create sync manager
	fmt.Println("3. 创建 Sync Manager...")
	syncManager, err := sync.NewManager(cfg, dbPool.GetDB(), logger)
	if err != nil {
		log.Fatalf("❌ 创建 Sync Manager 失败: %v", err)
	}
	fmt.Println("✓ Sync Manager 创建成功")
	fmt.Println()

	// Initialize sync system
	fmt.Println("4. 初始化 Sync 系统...")
	ctx := context.Background()
	if err := syncManager.Initialize(ctx); err != nil {
		log.Fatalf("❌ 初始化 Sync 系统失败: %v", err)
	}
	fmt.Println("✓ Sync 系统初始化成功")
	fmt.Println()

	// Check job engine status
	fmt.Println("5. 检查 Job Engine 状态...")
	jobEngine := syncManager.GetJobEngine()
	if jobEngine == nil {
		log.Fatal("❌ Job Engine 为 nil")
	}
	fmt.Println("✓ Job Engine 已创建")

	// Try to get job status (this will verify the engine is working)
	jobs, err := jobEngine.GetJobsByStatus(ctx, "running")
	if err != nil {
		log.Printf("⚠ 获取运行中任务失败: %v", err)
	} else {
		fmt.Printf("  - 运行中的任务数: %d\n", len(jobs))
	}
	fmt.Println()

	// Health check
	fmt.Println("6. 执行健康检查...")
	if err := syncManager.HealthCheck(ctx); err != nil {
		log.Fatalf("❌ 健康检查失败: %v", err)
	}
	fmt.Println("✓ 健康检查通过")
	fmt.Println()

	// Get stats
	fmt.Println("7. 获取系统统计...")
	stats, err := syncManager.GetStats(ctx)
	if err != nil {
		log.Fatalf("❌ 获取统计信息失败: %v", err)
	}
	fmt.Println("✓ 统计信息:")
	for key, value := range stats {
		fmt.Printf("  - %s: %v\n", key, value)
	}
	fmt.Println()

	// Keep running for a few seconds to observe logs
	fmt.Println("8. 保持运行 5 秒以观察日志...")
	time.Sleep(5 * time.Second)
	fmt.Println()

	// Shutdown
	fmt.Println("9. 关闭系统...")
	if err := syncManager.Shutdown(ctx); err != nil {
		log.Fatalf("❌ 关闭失败: %v", err)
	}
	fmt.Println("✓ 系统已关闭")
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("✓ 所有测试通过！Job Engine 工作正常")
	fmt.Println("========================================")
}
