package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"

	"go-data-gateway/internal/config"
)

// DremioClient handles connections to Dremio for Iceberg queries
type DremioClient struct {
	config config.DremioConfig
	client *http.Client
	cache  *cache.Cache
	logger *zap.Logger
	token  string
}

// NewDremioClient creates a new Dremio client
func NewDremioClient(cfg config.DremioConfig, logger *zap.Logger) (*DremioClient, error) {
	client := &DremioClient{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		cache:  cache.New(5*time.Minute, 10*time.Minute),
		logger: logger,
	}

	// Authenticate and get token if username/password provided
	if cfg.Username != "" && cfg.Password != "" {
		if err := client.authenticate(); err != nil {
			return nil, fmt.Errorf("dremio authentication failed: %w", err)
		}
	} else if cfg.Token != "" {
		client.token = cfg.Token
	}

	return client, nil
}

// authenticate gets a token from Dremio
func (c *DremioClient) authenticate() error {
	url := fmt.Sprintf("http://%s:%d/apiv2/login", c.config.Host, c.config.Port)

	payload := map[string]string{
		"userName": c.config.Username,
		"password": c.config.Password,
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if token, ok := result["token"].(string); ok {
		c.token = token
		c.logger.Info("Dremio authentication successful")
		return nil
	}

	return fmt.Errorf("no token in response")
}

// Query executes a SQL query against Dremio
func (c *DremioClient) Query(ctx context.Context, sqlQuery string, args ...interface{}) ([]map[string]interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("dremio:%s:%v", sqlQuery, args)
	if cached, found := c.cache.Get(cacheKey); found {
		c.logger.Debug("Cache hit", zap.String("query", sqlQuery))
		return cached.([]map[string]interface{}), nil
	}

	// Log query execution
	c.logger.Info("Executing Dremio query",
		zap.String("sql", sqlQuery),
		zap.Any("args", args))

	start := time.Now()

	// Build SQL API request
	url := fmt.Sprintf("http://%s:%d/api/v3/sql", c.config.Host, c.config.Port)

	payload := map[string]interface{}{
		"sql": sqlQuery,
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("_dremio%s", c.token))

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("Query request failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Query failed", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("query failed with status: %d", resp.StatusCode)
	}

	// Parse job response
	var jobResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return nil, err
	}

	// Wait a moment for job to complete
	time.Sleep(500 * time.Millisecond)

	// Get job results
	resultsURL := fmt.Sprintf("http://%s:%d/api/v3/job/%s/results", c.config.Host, c.config.Port, jobResp.ID)
	resultsReq, err := http.NewRequestWithContext(ctx, "GET", resultsURL, nil)
	if err != nil {
		return nil, err
	}
	resultsReq.Header.Set("Authorization", fmt.Sprintf("_dremio%s", c.token))

	resultsResp, err := c.client.Do(resultsReq)
	if err != nil {
		c.logger.Error("Failed to get job results", zap.Error(err))
		return nil, err
	}
	defer resultsResp.Body.Close()

	// Parse results
	var result struct {
		RowCount int                      `json:"rowCount"`
		Rows     []map[string]interface{} `json:"rows"`
	}

	if err := json.NewDecoder(resultsResp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Log performance metrics
	c.logger.Info("Dremio query completed",
		zap.Duration("duration", time.Since(start)),
		zap.Int("rows", len(result.Rows)))

	// Cache the results
	c.cache.Set(cacheKey, result.Rows, cache.DefaultExpiration)

	return result.Rows, nil
}

// ExecuteQuery is a simpler interface for executing queries
func (c *DremioClient) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	// Validate query is read-only
	if !isReadOnlyDremioSQL(query) {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	results, err := c.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"data":   results,
		"count":  len(results),
		"source": "dremio",
	}, nil
}

// TestConnection verifies the Dremio connection
func (c *DremioClient) TestConnection(ctx context.Context) error {
	_, err := c.Query(ctx, "SELECT 1")
	return err
}

// isReadOnlyDremioSQL checks if a SQL query is read-only for Dremio
func isReadOnlyDremioSQL(sql string) bool {
	sql = strings.ToUpper(strings.TrimSpace(sql))

	// List of forbidden keywords
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "GRANT", "REVOKE"}

	for _, keyword := range forbidden {
		if strings.Contains(sql, keyword) {
			return false
		}
	}

	// Must start with SELECT or WITH
	return strings.HasPrefix(sql, "SELECT") || strings.HasPrefix(sql, "WITH")
}
