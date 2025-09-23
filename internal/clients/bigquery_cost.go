package clients

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

const (
	// BigQuery pricing constants (as of 2025)
	BytesPerTB          = 1099511627776 // 1TB in bytes
	CostPerTB           = 5.00          // $5 per TB scanned
	FreeMonthlyQuotaGB  = 1000          // 1TB free tier per month
	MaxBytesPerQuery    = BytesPerTB * 10 // 10TB max per query (safety limit)
)

// QueryCostEstimator provides BigQuery query cost estimation
type QueryCostEstimator struct {
	client   *bigquery.Client
	logger   *zap.Logger
	project  string
	monthlyUsage float64 // Track monthly usage in GB
}

// CostEstimate represents the estimated cost of a query
type CostEstimate struct {
	Query              string    `json:"query"`
	EstimatedBytes     int64     `json:"estimated_bytes"`
	EstimatedGB        float64   `json:"estimated_gb"`
	EstimatedCostUSD   float64   `json:"estimated_cost_usd"`
	ProcessingTimeMins float64   `json:"estimated_processing_mins"`
	CacheHit           bool      `json:"cache_hit"`
	Warning            string    `json:"warning,omitempty"`
	Error              string    `json:"error,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// NewQueryCostEstimator creates a new cost estimator
func NewQueryCostEstimator(client *bigquery.Client, projectID string, logger *zap.Logger) *QueryCostEstimator {
	return &QueryCostEstimator{
		client:  client,
		logger:  logger,
		project: projectID,
	}
}

// EstimateQueryCost estimates the cost of a BigQuery query without running it
func (e *QueryCostEstimator) EstimateQueryCost(ctx context.Context, query string) (*CostEstimate, error) {
	estimate := &CostEstimate{
		Query:     query,
		Timestamp: time.Now(),
	}

	// Create a dry run query to get statistics
	q := e.client.Query(query)
	q.DryRun = true // This makes BigQuery only estimate, not execute

	job, err := q.Run(ctx)
	if err != nil {
		estimate.Error = fmt.Sprintf("Failed to estimate query: %v", err)
		return estimate, err
	}

	// Get job statistics
	status, err := job.Wait(ctx)
	if err != nil {
		estimate.Error = fmt.Sprintf("Failed to get job status: %v", err)
		return estimate, err
	}

	if status.Err() != nil {
		estimate.Error = fmt.Sprintf("Query validation failed: %v", status.Err())
		return estimate, status.Err()
	}

	// Get statistics from the dry run
	stats := job.LastStatus().Statistics

	// Check statistics and extract query stats
	if stats != nil {
		// Extract bytes processed from statistics
		if stats.TotalBytesProcessed > 0 {
			estimate.EstimatedBytes = stats.TotalBytesProcessed
			estimate.EstimatedGB = float64(estimate.EstimatedBytes) / (1024 * 1024 * 1024)

			// Calculate cost (first TB per month is free)
			if !estimate.CacheHit {
				estimate.EstimatedCostUSD = e.calculateCost(estimate.EstimatedBytes)
			}

			// Estimate processing time (rough estimate: 1GB/sec for simple queries)
			estimate.ProcessingTimeMins = (estimate.EstimatedGB / 60.0)
		}

		// Add warnings for expensive queries
		if estimate.EstimatedBytes > BytesPerTB {
			estimate.Warning = fmt.Sprintf("Query will scan %.2f TB of data!",
				float64(estimate.EstimatedBytes)/float64(BytesPerTB))
		}

		if estimate.EstimatedCostUSD > 10.0 {
			if estimate.Warning != "" {
				estimate.Warning += " | "
			}
			estimate.Warning += fmt.Sprintf("High cost query: $%.2f USD", estimate.EstimatedCostUSD)
		}
	}

	e.logger.Info("Query cost estimated",
		zap.String("query", truncateQuery(query)),
		zap.Float64("gb_scanned", estimate.EstimatedGB),
		zap.Float64("cost_usd", estimate.EstimatedCostUSD),
		zap.Bool("cache_hit", estimate.CacheHit))

	return estimate, nil
}

// EstimateTableScan estimates the cost of scanning an entire table
func (e *QueryCostEstimator) EstimateTableScan(ctx context.Context, datasetID, tableID string) (*CostEstimate, error) {
	table := e.client.Dataset(datasetID).Table(tableID)

	metadata, err := table.Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table metadata: %w", err)
	}

	estimate := &CostEstimate{
		Query:          fmt.Sprintf("Full scan of %s.%s", datasetID, tableID),
		EstimatedBytes: int64(metadata.NumBytes),
		EstimatedGB:    float64(metadata.NumBytes) / (1024 * 1024 * 1024),
		Timestamp:      time.Now(),
	}

	estimate.EstimatedCostUSD = e.calculateCost(estimate.EstimatedBytes)

	// Estimate based on table complexity
	numColumns := len(metadata.Schema)
	estimate.ProcessingTimeMins = (estimate.EstimatedGB / 30.0) * (1 + float64(numColumns)/100)

	if estimate.EstimatedBytes > BytesPerTB {
		estimate.Warning = fmt.Sprintf("Table is %.2f TB! Consider using partitioning or clustering",
			float64(estimate.EstimatedBytes)/float64(BytesPerTB))
	}

	return estimate, nil
}

// BatchEstimate estimates costs for multiple queries
func (e *QueryCostEstimator) BatchEstimate(ctx context.Context, queries []string) ([]*CostEstimate, error) {
	estimates := make([]*CostEstimate, 0, len(queries))
	totalCost := 0.0
	totalBytes := int64(0)

	for _, query := range queries {
		estimate, err := e.EstimateQueryCost(ctx, query)
		if err != nil {
			e.logger.Warn("Failed to estimate query cost",
				zap.String("query", truncateQuery(query)),
				zap.Error(err))
			estimate = &CostEstimate{
				Query:     query,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		}

		estimates = append(estimates, estimate)
		totalCost += estimate.EstimatedCostUSD
		totalBytes += estimate.EstimatedBytes
	}

	// Log batch summary
	e.logger.Info("Batch cost estimation completed",
		zap.Int("queries", len(queries)),
		zap.Float64("total_gb", float64(totalBytes)/(1024*1024*1024)),
		zap.Float64("total_cost_usd", totalCost))

	return estimates, nil
}

// GetMonthlyUsage returns the current month's BigQuery usage
func (e *QueryCostEstimator) GetMonthlyUsage(ctx context.Context) (float64, error) {
	// Query the INFORMATION_SCHEMA to get monthly usage
	query := fmt.Sprintf(`
		SELECT
			SUM(total_bytes_processed) as total_bytes
		FROM %s.region-us.INFORMATION_SCHEMA.JOBS
		WHERE
			DATE(creation_time) >= DATE_TRUNC(CURRENT_DATE(), MONTH)
			AND job_type = 'QUERY'
			AND state = 'DONE'
	`, "`"+e.project+"`")

	q := e.client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query monthly usage: %w", err)
	}

	var row []bigquery.Value
	err = it.Next(&row)
	if err != nil && err != iterator.Done {
		return 0, fmt.Errorf("failed to read usage data: %w", err)
	}

	if len(row) > 0 && row[0] != nil {
		if bytes, ok := row[0].(int64); ok {
			return float64(bytes) / (1024 * 1024 * 1024), nil
		}
	}

	return 0, nil
}

// OptimizeQuery suggests optimizations to reduce query cost
func (e *QueryCostEstimator) OptimizeQuery(query string) []string {
	suggestions := []string{}
	upperQuery := strings.ToUpper(query)

	// Check for SELECT *
	if strings.Contains(upperQuery, "SELECT *") {
		suggestions = append(suggestions,
			"Avoid SELECT * - specify only required columns to reduce data scanned")
	}

	// Check for missing LIMIT
	if !strings.Contains(upperQuery, "LIMIT") && strings.Contains(upperQuery, "ORDER BY") {
		suggestions = append(suggestions,
			"Add LIMIT clause when using ORDER BY to reduce processing")
	}

	// Check for missing partition filter
	if strings.Contains(upperQuery, "_PARTITIONTIME") || strings.Contains(upperQuery, "_PARTITIONDATE") {
		if !strings.Contains(upperQuery, "WHERE") {
			suggestions = append(suggestions,
				"Add partition filter in WHERE clause to reduce data scanned")
		}
	}

	// Suggest using preview for exploration
	if !strings.Contains(upperQuery, "LIMIT") {
		suggestions = append(suggestions,
			"Use table preview or add LIMIT for data exploration")
	}

	// Check for potential JOINs that could be optimized
	joinCount := strings.Count(upperQuery, "JOIN")
	if joinCount > 2 {
		suggestions = append(suggestions,
			fmt.Sprintf("Query has %d JOINs - consider materializing intermediate results", joinCount))
	}

	// Suggest using clustered tables
	if strings.Contains(upperQuery, "WHERE") && strings.Contains(upperQuery, "ORDER BY") {
		suggestions = append(suggestions,
			"Consider using clustered tables for frequently filtered/sorted columns")
	}

	return suggestions
}

// calculateCost calculates the actual cost based on bytes processed
func (e *QueryCostEstimator) calculateCost(bytes int64) float64 {
	if bytes == 0 {
		return 0
	}

	// Convert bytes to TB
	tb := float64(bytes) / float64(BytesPerTB)

	// First TB per month is free (simplified - should track actual monthly usage)
	if e.monthlyUsage < FreeMonthlyQuotaGB/1000 {
		remainingFree := (FreeMonthlyQuotaGB / 1000) - e.monthlyUsage
		if tb <= remainingFree {
			return 0
		}
		tb -= remainingFree
	}

	// $5 per TB after free tier
	cost := tb * CostPerTB

	// Round to 4 decimal places for cents
	return math.Round(cost*10000) / 10000
}

// truncateQuery truncates long queries for logging
func truncateQuery(query string) string {
	query = strings.TrimSpace(query)
	if len(query) > 100 {
		return query[:97] + "..."
	}
	return query
}

// ValidateQueryBudget checks if a query exceeds a cost budget
func (e *QueryCostEstimator) ValidateQueryBudget(ctx context.Context, query string, budgetUSD float64) (bool, *CostEstimate, error) {
	estimate, err := e.EstimateQueryCost(ctx, query)
	if err != nil {
		return false, estimate, err
	}

	if estimate.EstimatedCostUSD > budgetUSD {
		estimate.Warning = fmt.Sprintf("Query exceeds budget: $%.2f > $%.2f",
			estimate.EstimatedCostUSD, budgetUSD)
		return false, estimate, nil
	}

	return true, estimate, nil
}

// GetCostReport generates a cost report for recent queries
func (e *QueryCostEstimator) GetCostReport(ctx context.Context, days int) (map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT
			DATE(creation_time) as query_date,
			COUNT(*) as query_count,
			SUM(total_bytes_processed) as total_bytes,
			SUM(total_bytes_billed) as total_bytes_billed,
			AVG(total_slot_ms) as avg_slot_ms
		FROM %s.region-us.INFORMATION_SCHEMA.JOBS
		WHERE
			DATE(creation_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
			AND job_type = 'QUERY'
			AND state = 'DONE'
		GROUP BY query_date
		ORDER BY query_date DESC
	`, "`"+e.project+"`", days)

	q := e.client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cost report: %w", err)
	}

	dailyCosts := []map[string]interface{}{}
	totalCost := 0.0

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(row) >= 3 {
			bytes := int64(0)
			if row[2] != nil {
				bytes = row[2].(int64)
			}

			cost := e.calculateCost(bytes)
			totalCost += cost

			dailyCosts = append(dailyCosts, map[string]interface{}{
				"date":        row[0],
				"query_count": row[1],
				"gb_scanned":  float64(bytes) / (1024 * 1024 * 1024),
				"cost_usd":    cost,
			})
		}
	}

	return map[string]interface{}{
		"period_days":    days,
		"total_cost_usd": totalCost,
		"daily_costs":    dailyCosts,
		"avg_daily_cost": totalCost / float64(days),
	}, nil
}