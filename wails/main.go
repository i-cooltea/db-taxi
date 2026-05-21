//go:build desktop

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"db-taxi/internal/config"
	"db-taxi/internal/server"
)

// desktopDefaultPort is the default HTTP port used by the embedded server
// when running as a desktop app.
const desktopDefaultPort = 8080

func main() {
	// Load configuration (file + env). For desktop we ignore CLI flags
	// and rely on config + environment variables.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Make sure the server listens only on localhost in desktop mode.
	if cfg.Server.Host == "" || cfg.Server.Host == "0.0.0.0" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = desktopDefaultPort
	}

	// Create and start the existing HTTP server (Gin + APIs + static).
	srv := server.New(cfg)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start embedded HTTP server: %v", err)
		}
	}()

	// Give the server a short moment to start listening.
	time.Sleep(200 * time.Millisecond)

	backendURL, err := url.Parse(fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		log.Fatalf("invalid backend URL: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Run Wails desktop app. AssetServer.Handler simply反向代理到已有的 Gin HTTP 服务，
	// 这样前端的所有请求（包括 / 与 /api/...）都走你现有的 Web 入口。
	err = wails.Run(&options.App{
		Title:     "DB-Taxi",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proxy.ServeHTTP(w, r)
			}),
		},
		OnStartup: func(ctx context.Context) {
			log.Printf("DB-Taxi desktop started, proxying to %s", backendURL.String())
		},
		OnShutdown: func(ctx context.Context) {
			// Gracefully stop the HTTP server when the window closes.
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := srv.Stop(shutdownCtx); err != nil {
				log.Printf("error shutting down HTTP server: %v", err)
			}
		},
	})
	if err != nil {
		log.Fatalf("failed to run Wails app: %v", err)
	}
}