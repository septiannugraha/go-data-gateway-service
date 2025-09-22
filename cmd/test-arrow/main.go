package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-data-gateway/internal/datasource"
	"go.uber.org/zap"
)

func main() {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Configure Dremio Arrow Flight connection
	config := &datasource.DremioConfig{
		Host:     "localhost",
		Port:     31010, // Arrow Flight port
		Username: "septiannugraha",
		Password: "?uJ*u2a@u!@f2e]",
		UseTLS:   false,
		Project:  "nessie_iceberg",
	}

	// Create Arrow Flight SQL client
	logger.Info("Creating Arrow Flight SQL client...")
	client, err := datasource.NewDremioArrowClient(config, logger)
	if err != nil {
		log.Fatalf("Failed to create Arrow client: %v", err)
	}
	defer client.Close()

	// Test connection
	logger.Info("Testing connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.TestConnection(ctx)
	if err != nil {
		log.Fatalf("Connection test failed: %v", err)
	}
	logger.Info("Connection successful!")

	// Test query execution
	logger.Info("Testing query execution...")
	result, err := client.ExecuteQuery(ctx, "SELECT * FROM nessie_iceberg.tender_data LIMIT 5", nil)
	if err != nil {
		log.Fatalf("Query execution failed: %v", err)
	}

	// Display results
	fmt.Printf("\n=== Query Results ===\n")
	fmt.Printf("Rows returned: %d\n", result.Count)
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

	logger.Info("Arrow Flight SQL test completed successfully!")
}
