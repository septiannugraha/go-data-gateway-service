package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"go-data-gateway/internal/clients"
	"go-data-gateway/internal/config"
	"go-data-gateway/internal/handlers"
	"go-data-gateway/internal/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		println("No .env file found")
	}

	// Initialize logger
	logger, _ := zap.NewProduction()
	if os.Getenv("ENV") == "development" {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// Load configuration
	cfg := config.Load()
	logger.Info("Configuration loaded",
		zap.String("port", cfg.Port),
		zap.String("env", cfg.Environment),
	)

	// Initialize clients
	var dremioClient *clients.DremioClient
	var bigQueryClient *clients.BigQueryClient
	var err error

	// Initialize Dremio client if configured
	if cfg.Dremio.Host != "" {
		dremioClient, err = clients.NewDremioClient(cfg.Dremio, logger)
		if err != nil {
			logger.Warn("Dremio client initialization failed", zap.Error(err))
		} else {
			logger.Info("Dremio client initialized")
		}
	}

	// Initialize BigQuery client if configured
	if cfg.BigQuery.ProjectID != "" {
		bigQueryClient, err = clients.NewBigQueryClient(cfg.BigQuery, logger)
		if err != nil {
			logger.Warn("BigQuery client initialization failed", zap.Error(err))
		} else {
			logger.Info("BigQuery client initialized")
		}
	}

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())

	// Health check endpoint (no auth)
	router.GET("/health", handlers.Health)
	router.GET("/ready", handlers.Ready(dremioClient, bigQueryClient))

	// API v1 routes with authentication
	v1 := router.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(cfg.APIKeys))
	v1.Use(middleware.RateLimiter(cfg.RateLimit))

	// Initialize handlers
	tenderHandler := handlers.NewTenderHandler(dremioClient, logger)
	rupHandler := handlers.NewRUPHandler(bigQueryClient, logger)
	queryHandler := handlers.NewQueryHandler(dremioClient, bigQueryClient, logger)

	// Tender endpoints (Iceberg/Dremio)
	if dremioClient != nil {
		v1.GET("/tender", tenderHandler.List)
		v1.GET("/tender/:id", tenderHandler.GetByID)
		v1.POST("/tender/search", tenderHandler.Search)
	}

	// RUP endpoints (BigQuery)
	if bigQueryClient != nil {
		v1.GET("/rup", rupHandler.List)
		v1.GET("/rup/:id", rupHandler.GetByID)
		// v1.POST("/rup/search", rupHandler.Search) // TODO: Implement search
	}

	// Generic query endpoint (for custom queries)
	v1.POST("/query", queryHandler.Execute)

	// Metrics endpoint
	router.GET("/metrics", middleware.PrometheusHandler())

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in goroutine
	go func() {
		logger.Info("Server starting", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}