package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go-data-gateway/internal/datasource"
	"go.uber.org/zap"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Configure Dremio Arrow Flight connection
	// SECURITY: Always use environment variables for credentials!
	config := &datasource.DremioConfig{
		Host:     getEnv("DREMIO_HOST", "localhost"),
		Port:     32010, // Arrow Flight port (32010 is the correct port)
		Username: getEnv("DREMIO_USERNAME", ""),
		Password: getEnv("DREMIO_PASSWORD", ""),
		UseTLS:   false,
		Project:  "nessie_iceberg",
	}

	// Check if credentials are provided
	if config.Username == "" || config.Password == "" {
		log.Fatal("DREMIO_USERNAME and DREMIO_PASSWORD environment variables are required")
	}

	// Create Arrow Flight SQL client
	logger.Info("Creating Arrow Flight SQL client...")
	client, err := datasource.NewDremioArrowClient(config, logger)
	if err != nil {
		log.Fatalf("Failed to create Arrow client: %v", err)
	}
	defer client.Close()

	logger.Info("Successfully connected to Dremio Arrow Flight SQL")

	// Test query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Simple test query
	query := `SELECT 1 as test_col`
	logger.Info("Executing test query", zap.String("query", query))

	result, err := client.Query(ctx, query, nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	logger.Info("Query successful!",
		zap.Int("rows", result.Count),
		zap.Duration("query_time", result.QueryTime))

	// Display results
	fmt.Printf("\n=== Query Results ===\n")
	fmt.Printf("Rows returned: %d\n", result.Count)
	fmt.Printf("Query time: %v\n", result.QueryTime)
	fmt.Printf("Data: %+v\n\n", result.Data)

	// Try a real table query if it exists
	realQuery := `SELECT * FROM nessie_iceberg.tender_data LIMIT 5`
	logger.Info("Testing real table query", zap.String("query", realQuery))

	result, err = client.Query(ctx, realQuery, nil)
	if err != nil {
		logger.Warn("Real table query failed (table might not exist)", zap.Error(err))
		// Try alternative table name
		alternativeQuery := `SELECT * FROM nessie_iceberg.tender LIMIT 5`
		logger.Info("Trying alternative table name", zap.String("query", alternativeQuery))

		result, err = client.Query(ctx, alternativeQuery, nil)
		if err != nil {
			logger.Warn("Alternative query also failed", zap.Error(err))
		} else {
			displayResults(result, logger)
		}
	} else {
		displayResults(result, logger)
	}

	logger.Info("Arrow Flight SQL test completed successfully!")
}

func displayResults(result *datasource.QueryResult, logger *zap.Logger) {
	logger.Info("Real table query successful!",
		zap.Int("rows", result.Count),
		zap.Duration("query_time", result.QueryTime),
		zap.String("source", string(result.Source)),
		zap.Bool("cache_hit", result.CacheHit))

	fmt.Printf("\n=== Real Table Results ===\n")
	fmt.Printf("Count: %d\n", result.Count)
	fmt.Printf("Query time: %v\n", result.QueryTime)
	fmt.Printf("Source: %s\n", result.Source)
	fmt.Printf("Cache hit: %v\n\n", result.CacheHit)

	// Display first few rows
	for i, row := range result.Data {
		if i >= 3 {
			fmt.Printf("... and %d more rows\n", len(result.Data)-3)
			break
		}
		fmt.Printf("Row %d:\n", i+1)
		for key, value := range row {
			fmt.Printf("  %s: %v\n", key, value)
		}
		fmt.Println()
	}
}