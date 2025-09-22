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
		Port:     32010, // Arrow Flight port
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
		log.Fatalf("Failed to create client: %v", err)
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

	// Print results
	fmt.Printf("\nQuery Results:\n")
	fmt.Printf("Count: %d\n", result.Count)
	fmt.Printf("Query Time: %s\n", result.QueryTime)
	fmt.Printf("Data: %+v\n", result.Data)

	// Try a real table query if it exists
	realQuery := `SELECT * FROM nessie_iceberg.tender LIMIT 5`
	logger.Info("Trying real table query", zap.String("query", realQuery))

	result, err = client.Query(ctx, realQuery, nil)
	if err != nil {
		logger.Warn("Real table query failed (table might not exist)", zap.Error(err))
	} else {
		logger.Info("Real table query successful!",
			zap.Int("rows", result.Count),
			zap.Duration("query_time", result.QueryTime))
		fmt.Printf("\nReal Table Results:\n")
		fmt.Printf("Count: %d\n", result.Count)
		fmt.Printf("First row: %+v\n", result.Data[0])
	}
}