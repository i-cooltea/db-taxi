package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"db-taxi/internal/config"
	"db-taxi/internal/server"
)

func main() {
	// Command line flags
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		host       = flag.String("host", "", "Database host")
		port       = flag.Int("port", 0, "Database port")
		username   = flag.String("user", "", "Database username")
		password   = flag.String("password", "", "Database password")
		database   = flag.String("database", "", "Database name")
		ssl        = flag.Bool("ssl", false, "Enable SSL connection")
		serverPort = flag.Int("server-port", 0, "Server port")
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
				SSL:      *ssl,
			},
			Server: config.ServerConfig{
				Port: *serverPort,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start server
	srv := server.New(cfg)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func printHelp() {
	log.Println("DB-Taxi MySQL Web Explorer")
	log.Println("")
	log.Println("Usage:")
	log.Println("  db-taxi [options]")
	log.Println("")
	log.Println("Configuration Options:")
	log.Println("  -config string      Path to configuration file")
	log.Println("  -host string        Database host")
	log.Println("  -port int           Database port")
	log.Println("  -user string        Database username")
	log.Println("  -password string    Database password")
	log.Println("  -database string    Database name")
	log.Println("  -ssl                Enable SSL connection")
	log.Println("  -server-port int    Server port")
	log.Println("  -help               Show this help message")
	log.Println("")
	log.Println("Examples:")
	log.Println("  # Use custom config file")
	log.Println("  db-taxi -config /path/to/config.yaml")
	log.Println("")
	log.Println("  # Override database connection via command line")
	log.Println("  db-taxi -host localhost -port 3306 -user root -password secret -database mydb")
	log.Println("")
	log.Println("  # Mix config file with command line overrides")
	log.Println("  db-taxi -config myconfig.yaml -password newsecret -server-port 9090")
	log.Println("")
	log.Println("Environment Variables:")
	log.Println("  All configuration options can be set via environment variables with DBT_ prefix:")
	log.Println("  DBT_DATABASE_HOST, DBT_DATABASE_PORT, DBT_DATABASE_USERNAME, etc.")
}
