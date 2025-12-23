package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/regstrava/regstrava/internal/api"
	"github.com/regstrava/regstrava/internal/config"
	"github.com/regstrava/regstrava/internal/repository"
	"github.com/regstrava/regstrava/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize repositories
	invoiceRepo, err := repository.NewInvoiceRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer invoiceRepo.Close()

	funderRepo, err := repository.NewFunderRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize funder repository: %v", err)
	}
	defer funderRepo.Close()

	// Initialize services
	hashService := service.NewHashService(cfg.HMACKey)
	authService := service.NewAuthService(funderRepo, cfg.JWTSecret)
	invoiceService := service.NewInvoiceService(invoiceRepo, hashService)
	rateLimitService, err := service.NewRateLimitService(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Set up router
	router := api.NewRouter(
		invoiceService,
		authService,
		hashService,
		rateLimitService,
		funderRepo,
	)

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting RegStrava server on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
