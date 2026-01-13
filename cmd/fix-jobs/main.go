package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
	"db-taxi/internal/database"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Show what would be fixed without making changes")
	timeoutMinutes := flag.Int("timeout", 30, "Timeout in minutes for stuck jobs")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Connect to database
	dbPool, err := database.NewPool(cfg.Database, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create database pool")
	}
	defer dbPool.Close()

	db := dbPool.GetDB()

	// Run diagnostics and fix
	ctx := context.Background()
	if err := diagnoseAndFix(ctx, db, logger, *dryRun, *timeoutMinutes); err != nil {
		logger.WithError(err).Fatal("Failed to diagnose and fix jobs")
	}

	logger.Info("Job fix completed successfully")
}

func diagnoseAndFix(ctx context.Context, db *sqlx.DB, logger *logrus.Logger, dryRun bool, timeoutMinutes int) error {
	logger.Info("=== 数据库同步任务诊断工具 ===")
	logger.Info("")

	// 1. Check for stuck jobs
	logger.Info("1. 检查卡住的任务...")

	type StuckJob struct {
		ID              string    `db:"id"`
		ConfigID        string    `db:"config_id"`
		Status          string    `db:"status"`
		StartTime       time.Time `db:"start_time"`
		RunningMinutes  int       `db:"running_minutes"`
		TotalTables     int       `db:"total_tables"`
		CompletedTables int       `db:"completed_tables"`
		ErrorMessage    string    `db:"error_message"`
	}

	query := `
		SELECT 
			id,
			config_id,
			status,
			start_time,
			TIMESTAMPDIFF(MINUTE, start_time, NOW()) as running_minutes,
			total_tables,
			completed_tables,
			COALESCE(error_message, '') as error_message
		FROM sync_jobs 
		WHERE status = 'running'
		AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > ?
		ORDER BY start_time
	`

	var stuckJobs []StuckJob
	if err := db.SelectContext(ctx, &stuckJobs, query, timeoutMinutes); err != nil {
		return fmt.Errorf("failed to query stuck jobs: %w", err)
	}

	if len(stuckJobs) == 0 {
		logger.Info("✅ 没有发现卡住的任务")
		return nil
	}

	logger.Warnf("❌ 发现 %d 个卡住的任务:", len(stuckJobs))
	for _, job := range stuckJobs {
		logger.Warnf("  - Job ID: %s, Config: %s, Running: %d minutes, Progress: %d/%d tables",
			job.ID, job.ConfigID, job.RunningMinutes, job.CompletedTables, job.TotalTables)
	}
	logger.Info("")

	// 2. Fix stuck jobs
	if dryRun {
		logger.Info("🔍 DRY RUN 模式 - 不会进行实际修改")
		logger.Infof("将会修复 %d 个卡住的任务", len(stuckJobs))
		return nil
	}

	logger.Info("2. 修复卡住的任务...")

	updateQuery := `
		UPDATE sync_jobs 
		SET 
			status = 'failed',
			error_message = CONCAT('Task timeout - automatically marked as failed after ', ?, ' minutes. Original error: ', COALESCE(error_message, 'none')),
			end_time = NOW()
		WHERE status = 'running' 
		AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > ?
	`

	result, err := db.ExecContext(ctx, updateQuery, timeoutMinutes, timeoutMinutes)
	if err != nil {
		return fmt.Errorf("failed to update stuck jobs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	logger.Infof("✅ 成功修复 %d 个卡住的任务", rowsAffected)
	logger.Info("")

	// 3. Show statistics
	logger.Info("3. 任务状态统计:")

	type StatusStat struct {
		Status   string     `db:"status"`
		Count    int        `db:"count"`
		Earliest *time.Time `db:"earliest"`
		Latest   *time.Time `db:"latest"`
	}

	statsQuery := `
		SELECT 
			status,
			COUNT(*) as count,
			MIN(start_time) as earliest,
			MAX(start_time) as latest
		FROM sync_jobs
		GROUP BY status
		ORDER BY status
	`

	var stats []StatusStat
	if err := db.SelectContext(ctx, &stats, statsQuery); err != nil {
		return fmt.Errorf("failed to query statistics: %w", err)
	}

	for _, stat := range stats {
		earliestStr := "N/A"
		latestStr := "N/A"
		if stat.Earliest != nil {
			earliestStr = stat.Earliest.Format("2006-01-02 15:04:05")
		}
		if stat.Latest != nil {
			latestStr = stat.Latest.Format("2006-01-02 15:04:05")
		}
		logger.Infof("  %s: %d jobs (earliest: %s, latest: %s)",
			stat.Status, stat.Count, earliestStr, latestStr)
	}
	logger.Info("")

	// 4. Show recent failures
	logger.Info("4. 最近失败的任务:")

	type FailedJob struct {
		ID           string     `db:"id"`
		ConfigID     string     `db:"config_id"`
		StartTime    time.Time  `db:"start_time"`
		EndTime      *time.Time `db:"end_time"`
		ErrorMessage string     `db:"error_message"`
	}

	failedQuery := `
		SELECT 
			id,
			config_id,
			start_time,
			end_time,
			COALESCE(error_message, '') as error_message
		FROM sync_jobs
		WHERE status = 'failed'
		ORDER BY start_time DESC
		LIMIT 5
	`

	var failedJobs []FailedJob
	if err := db.SelectContext(ctx, &failedJobs, failedQuery); err != nil {
		return fmt.Errorf("failed to query failed jobs: %w", err)
	}

	if len(failedJobs) == 0 {
		logger.Info("  没有失败的任务")
	} else {
		for _, job := range failedJobs {
			endTimeStr := "N/A"
			if job.EndTime != nil {
				endTimeStr = job.EndTime.Format("2006-01-02 15:04:05")
			}
			logger.Infof("  - Job ID: %s, Config: %s", job.ID, job.ConfigID)
			logger.Infof("    Start: %s, End: %s", job.StartTime.Format("2006-01-02 15:04:05"), endTimeStr)
			if job.ErrorMessage != "" {
				logger.Infof("    Error: %s", job.ErrorMessage)
			}
		}
	}
	logger.Info("")

	// 5. Recommendations
	logger.Info("=== 建议 ===")
	logger.Info("1. 检查应用程序日志以了解任务失败的原因")
	logger.Info("2. 验证远程数据库连接是否正常")
	logger.Info("3. 检查是否有网络问题或防火墙阻止连接")
	logger.Info("4. 考虑增加任务超时时间或优化同步配置")
	logger.Info("5. 如果问题持续，重启应用程序以重新初始化JobEngine")

	return nil
}
