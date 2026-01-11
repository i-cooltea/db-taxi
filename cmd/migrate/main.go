package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"db-taxi/internal/config"
	"db-taxi/internal/migration"
)

func main() {
	// Command line flags
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		command    = flag.String("command", "migrate", "Command to run: migrate, status, version")
		host       = flag.String("host", "", "Database host")
		port       = flag.Int("port", 0, "Database port")
		username   = flag.String("user", "", "Database username")
		password   = flag.String("password", "", "Database password")
		database   = flag.String("database", "", "Database name")
		version    = flag.Int("version", 0, "Target version for migration (0 = latest)")
		help       = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Load configuration
	cfg, err := config.LoadWithOptions(&config.LoadOptions{
		ConfigFile: *configFile,
		Overrides: &config.Config{
			Database: config.DatabaseConfig{
				Host:     *host,
				Port:     *port,
				Username: *username,
				Password: *password,
				Database: *database,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create database connection
	db, err := connectDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Create migration manager
	mgr := migration.NewManager(db, logger)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Execute command
	switch *command {
	case "migrate":
		if *version > 0 {
			if err := mgr.MigrateToVersion(ctx, *version); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}
		} else {
			if err := mgr.Migrate(ctx); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}
		}
		fmt.Println("Migration completed successfully")

	case "status":
		status, err := mgr.Status(ctx)
		if err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}
		fmt.Println(status)

	case "version":
		currentVersion, err := mgr.GetCurrentVersion(ctx)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", currentVersion)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}

func connectDatabase(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func printHelp() {
	fmt.Println("DB-Taxi Migration Tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  migrate [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  -command migrate    Run all pending migrations (default)")
	fmt.Println("  -command status     Show migration status")
	fmt.Println("  -command version    Show current migration version")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -config string      Path to configuration file")
	fmt.Println("  -host string        Database host")
	fmt.Println("  -port int           Database port")
	fmt.Println("  -user string        Database username")
	fmt.Println("  -password string    Database password")
	fmt.Println("  -database string    Database name")
	fmt.Println("  -version int        Target version for migration (0 = latest)")
	fmt.Println("  -help               Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Run all pending migrations")
	fmt.Println("  migrate -host localhost -user root -password secret -database mydb")
	fmt.Println("")
	fmt.Println("  # Check migration status")
	fmt.Println("  migrate -command status -config config.yaml")
	fmt.Println("")
	fmt.Println("  # Migrate to specific version")
	fmt.Println("  migrate -version 2 -host localhost -user root -database mydb")
	fmt.Println("")
	fmt.Println("Environment Variables:")
	fmt.Println("  All configuration options can be set via environment variables with DBT_ prefix")
}
