package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"go-data-gateway/internal/cache"
	"go-data-gateway/internal/clients"
	"go-data-gateway/internal/config"
	"go-data-gateway/internal/datasource"
	v1 "go-data-gateway/internal/handlers/v1"
	custommw "go-data-gateway/internal/middleware/chi"
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
		zap.String("env", cfg.Environment))

	// Initialize cache
	cacheService := initializeCache(cfg, logger)
	if cacheService != nil {
		defer cacheService.Close()
	}

	// Initialize data sources with caching
	dataSources := initializeDataSources(cfg, logger, cacheService)
	defer closeDataSources(dataSources)

	// Create router with Chi
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommw.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(custommw.CORS())
	r.Use(middleware.Compress(5))

	// Health endpoints (no auth)
	r.Get("/health", healthCheck)
	r.Get("/ready", readyCheck(dataSources))

	// Metrics endpoint
	r.Handle("/metrics", custommw.PrometheusHandler())

	// Cache stats endpoint (no auth for monitoring)
	r.Get("/cache/stats", getCacheStats(cacheService, dataSources))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// API middleware
		r.Use(custommw.APIKeyAuth(cfg.APIKeys))
		r.Use(custommw.RateLimiter(cfg.RateLimit))
		r.Use(middleware.Timeout(30 * time.Second))

		// Create handlers
		queryHandler := v1.NewQueryHandler(dataSources, logger)
		tenderHandler := v1.NewTenderHandler(dataSources["dremio"], logger)
		batchHandler := v1.NewBatchHandler(dataSources, logger)
		streamHandler := v1.NewStreamHandler(dataSources, logger)

		// Create BigQuery client for RUP handler and cost estimator
		var rupHandler *v1.RUPHandler
		var costEstimator *clients.QueryCostEstimator
		if cfg.BigQuery.ProjectID != "" {
			bigQueryClient, err := clients.NewBigQueryClient(cfg.BigQuery, logger)
			if err != nil {
				logger.Warn("BigQuery client initialization failed", zap.Error(err))
			} else {
				rupHandler = v1.NewRUPHandler(bigQueryClient, logger)
				costEstimator = clients.NewQueryCostEstimator(bigQueryClient.GetClient(), cfg.BigQuery.ProjectID, logger)
				logger.Info("BigQuery client initialized for RUP handler and cost estimation")
			}
		}

		// Query endpoints
		r.Post("/query", queryHandler.Execute)
		r.Post("/batch", batchHandler.Execute)
		r.Post("/batch/stream", batchHandler.Stream)
		r.Post("/stream", streamHandler.Stream)
		r.Post("/stream/sse", streamHandler.StreamSSE)

		// Cost estimation endpoint (BigQuery only)
		if costEstimator != nil {
			r.Post("/estimate-cost", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Query string `json:"query"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}

				estimate, err := costEstimator.EstimateQueryCost(r.Context(), req.Query)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(estimate)
			})
		}

		// Tender endpoints (Dremio)
		r.Route("/tender", func(r chi.Router) {
			r.Get("/", tenderHandler.List)
			r.Get("/{id}", tenderHandler.GetByID)
			r.Post("/search", tenderHandler.Search)
		})

		// RUP endpoints (BigQuery)
		if rupHandler != nil {
			r.Route("/rup", func(r chi.Router) {
				r.Get("/", rupHandler.List)
				r.Get("/{id}", rupHandler.GetByID)
				r.Post("/search", rupHandler.Search)
			})
		}

		// Add more resource endpoints here
	})

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in goroutine
	go func() {
		logger.Info("Server starting with Chi router", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped gracefully")
}

// initializeCache creates cache service
func initializeCache(cfg *config.Config, logger *zap.Logger) cache.Cache {
	if cfg.Redis.Host == "" {
		logger.Info("Redis not configured, using no-op cache")
		return &cache.NoOpCache{}
	}

	cacheService, err := cache.NewRedisCache(cfg.Redis, logger)
	if err != nil {
		logger.Warn("Failed to initialize Redis cache, using no-op cache", zap.Error(err))
		return &cache.NoOpCache{}
	}

	return cacheService
}

// initializeDataSources creates all configured data sources with caching
func initializeDataSources(cfg *config.Config, logger *zap.Logger, cacheService cache.Cache) map[string]datasource.DataSource {
	sources := make(map[string]datasource.DataSource)

	// Initialize Dremio client
	if cfg.Dremio.Host != "" {
		// Arrow Flight SQL is now working with Apache Arrow Go v18!
		useArrowFlight := true
		if useArrowFlight { // Arrow Flight SQL on port 32010
			// Arrow Flight SQL configuration (port 32010)
			arrowConfig := &datasource.DremioConfig{
				Host:     cfg.Dremio.Host,
				Port:     32010, // Arrow Flight SQL port
				Username: cfg.Dremio.Username,
				Password: cfg.Dremio.Password,
				UseTLS:   false,
				Project:  "nessie_iceberg",
			}

			// Configure connection pool for Arrow Flight
			poolConfig := &datasource.PoolConfig{
				MaxConnections:      10,
				MinConnections:      2,
				MaxIdleTime:         30 * time.Minute,
				ConnectionTimeout:   10 * time.Second,
				HealthCheckInterval: 1 * time.Minute,
			}

			arrowClient, err := datasource.NewDremioArrowClientWithPool(arrowConfig, poolConfig, logger)
			if err != nil {
				logger.Warn("Arrow Flight SQL initialization failed", zap.Error(err))
			} else {
				// Wrap with caching
				sources["dremio"] = cache.NewCachedDataSource(arrowClient, cacheService, logger)
				logger.Info("Dremio Arrow Flight SQL client initialized with connection pool and caching",
					zap.Int("max_connections", poolConfig.MaxConnections))
			}
		} else {
			// Use REST client (default)
			dremioClient, err := datasource.NewDremioRESTClient(
				cfg.Dremio.Host,
				cfg.Dremio.Port,
				cfg.Dremio.Username,
				cfg.Dremio.Password,
				logger,
			)
			if err != nil {
				logger.Warn("Dremio REST client initialization failed", zap.Error(err))
			} else {
				// Wrap with caching
				sources["dremio"] = cache.NewCachedDataSource(dremioClient, cacheService, logger)
				logger.Info("Dremio REST client initialized with caching")
			}
		}
	}

	// Initialize BigQuery client
	if cfg.BigQuery.ProjectID != "" {
		bigQueryWrapper, err := datasource.NewBigQueryWrapper(cfg.BigQuery, logger)
		if err != nil {
			logger.Warn("BigQuery client initialization failed", zap.Error(err))
		} else {
			// Wrap with caching
			sources["BIGQUERY"] = cache.NewCachedDataSource(bigQueryWrapper, cacheService, logger)
			logger.Info("BigQuery client initialized with caching", zap.String("project", cfg.BigQuery.ProjectID))
		}
	}

	return sources
}

// closeDataSources closes all data source connections
func closeDataSources(sources map[string]datasource.DataSource) {
	for name, source := range sources {
		if err := source.Close(); err != nil {
			zap.L().Error("Failed to close data source",
				zap.String("name", name),
				zap.Error(err))
		}
	}
}

// getCacheStats returns cache statistics
func getCacheStats(cacheService cache.Cache, dataSources map[string]datasource.DataSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := make(map[string]interface{})

		// Get cache service stats
		if cacheService != nil {
			cacheStats, err := cacheService.Stats(r.Context())
			if err == nil {
				stats["cache"] = cacheStats
			}
		}

		// Get metrics from each cached data source
		sourceMetrics := make(map[string]interface{})
		for name, source := range dataSources {
			if cached, ok := source.(*cache.CachedDataSource); ok {
				sourceMetrics[name] = cached.GetMetrics()
			}
		}
		stats["sources"] = sourceMetrics

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// healthCheck returns service health status
func healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":  "healthy",
		"service": "go-data-gateway",
		"version": "2.0.0", // Chi + Arrow version
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// readyCheck checks if all data sources are ready
func readyCheck(sources map[string]datasource.DataSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		checks := make(map[string]string)

		for name, source := range sources {
			if err := source.TestConnection(ctx); err != nil {
				checks[name] = "unhealthy: " + err.Error()
			} else {
				checks[name] = "healthy"
			}
		}

		response := map[string]interface{}{
			"status": "ready",
			"checks": checks,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
