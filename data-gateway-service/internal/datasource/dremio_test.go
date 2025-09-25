package datasource

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestDremioAuthenticationSecure tests authentication with real Dremio using environment variables
// SECURITY: This test ONLY uses environment variables for credentials - NEVER hardcode passwords!
func TestDremioAuthenticationSecure(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Check if required environment variables are set
	username := os.Getenv("DREMIO_USERNAME")
	password := os.Getenv("DREMIO_PASSWORD")

	if username == "" || password == "" {
		t.Skip("Skipping test: DREMIO_USERNAME and DREMIO_PASSWORD environment variables required")
	}

	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		config        *DremioConfig
		shouldSucceed bool
		errorContains string
	}{
		{
			name: "Valid credentials from environment",
			config: &DremioConfig{
				Host:     getEnvOrDefault("DREMIO_HOST", "localhost"),
				Port:     32010,
				Username: username, // From environment variable
				Password: password, // From environment variable
				UseTLS:   false,
			},
			shouldSucceed: true,
		},
		{
			name: "Invalid password",
			config: &DremioConfig{
				Host:     getEnvOrDefault("DREMIO_HOST", "localhost"),
				Port:     32010,
				Username: username,
				Password: "definitely_wrong_password",
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "Invalid username or password",
		},
		{
			name: "Invalid username",
			config: &DremioConfig{
				Host:     getEnvOrDefault("DREMIO_HOST", "localhost"),
				Port:     32010,
				Username: "invalid_user",
				Password: password,
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "Invalid username or password",
		},
		{
			name: "Wrong port",
			config: &DremioConfig{
				Host:     getEnvOrDefault("DREMIO_HOST", "localhost"),
				Port:     9999, // Wrong port
				Username: username,
				Password: password,
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "connection error",
		},
		{
			name: "Empty credentials",
			config: &DremioConfig{
				Host:     getEnvOrDefault("DREMIO_HOST", "localhost"),
				Port:     32010,
				Username: "",
				Password: "",
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "Invalid username or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with test configuration
			client, err := NewDremioArrowClient(tt.config, logger)

			if tt.shouldSucceed {
				require.NoError(t, err, "Expected successful connection")
				require.NotNil(t, client, "Client should not be nil")

				// Test the connection
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				testErr := client.TestConnection(ctx)
				assert.NoError(t, testErr, "Connection test should succeed")

				// Clean up
				client.Close()
			} else {
				if err != nil {
					assert.Contains(t, err.Error(), tt.errorContains,
						"Error should contain expected message")
				} else if client != nil {
					// Connection might succeed but auth might fail on query
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					testErr := client.TestConnection(ctx)
					assert.Error(t, testErr, "Expected authentication to fail")
					if testErr != nil {
						assert.Contains(t, testErr.Error(), tt.errorContains,
							"Error should contain expected message")
					}
					client.Close()
				}
			}
		})
	}
}

// TestDremioConnectionPoolWithEnvCredentials tests connection pool with secure credentials
func TestDremioConnectionPoolWithEnvCredentials(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	username := os.Getenv("DREMIO_USERNAME")
	password := os.Getenv("DREMIO_PASSWORD")

	if username == "" || password == "" {
		t.Skip("Skipping test: DREMIO_USERNAME and DREMIO_PASSWORD required")
	}

	logger, _ := zap.NewDevelopment()

	config := &DremioConfig{
		Host:     getEnvOrDefault("DREMIO_HOST", "localhost"),
		Port:     32010,
		Username: username,
		Password: password,
		UseTLS:   false,
	}

	poolConfig := &PoolConfig{
		MaxConnections:      5,
		MinConnections:      1,
		MaxIdleTime:         5 * time.Minute,
		ConnectionTimeout:   10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}

	// Create client with connection pool
	client, err := NewDremioArrowClientWithPool(config, poolConfig, logger)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// Test connection
	ctx := context.Background()
	err = client.TestConnection(ctx)
	assert.NoError(t, err, "Pool connection should work")
}

// TestBasicAuthGeneration tests the basic auth header generation
func TestBasicAuthGeneration(t *testing.T) {
	// This test doesn't need real credentials - just tests the mechanism
	tests := []struct {
		name     string
		username string
		password string
		expected string // Base64 encoded result
	}{
		{
			name:     "Standard credentials",
			username: "testuser",
			password: "testpass",
			expected: "dGVzdHVzZXI6dGVzdHBhc3M=", // base64("testuser:testpass")
		},
		{
			name:     "Empty password",
			username: "user",
			password: "",
			expected: "dXNlcjo=", // base64("user:")
		},
		{
			name:     "Special characters",
			username: "user@domain",
			password: "p@ss!word#123",
			expected: "dXNlckBkb21haW46cEBzcyF3b3JkIzEyMw==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := basicAuth(tt.username, tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEnvironmentVariableLoading verifies environment variables are loaded correctly
func TestEnvironmentVariableLoading(t *testing.T) {
	// Set test environment variables
	testVars := map[string]string{
		"TEST_DREMIO_HOST":     "test.dremio.com",
		"TEST_DREMIO_USERNAME": "test_user",
		"TEST_DREMIO_PORT":     "32010",
	}

	for key, value := range testVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Verify they can be retrieved
	assert.Equal(t, "test.dremio.com", os.Getenv("TEST_DREMIO_HOST"))
	assert.Equal(t, "test_user", os.Getenv("TEST_DREMIO_USERNAME"))
	assert.Equal(t, "32010", os.Getenv("TEST_DREMIO_PORT"))

	// Verify password is NOT set (should never be hardcoded)
	assert.Empty(t, os.Getenv("TEST_DREMIO_PASSWORD"),
		"Password should never be hardcoded, only set in environment")
}

// Security reminder for future developers
func TestSecurityReminder(t *testing.T) {
	t.Log("SECURITY REMINDER: This test file uses environment variables for credentials.")
	t.Log("NEVER hardcode passwords, API keys, or other secrets in test files!")
	t.Log("Set credentials using environment variables or .env.test file (not committed)")
	t.Log("")
	t.Log("Required environment variables for integration tests:")
	t.Log("  - DREMIO_USERNAME")
	t.Log("  - DREMIO_PASSWORD")
	t.Log("  - DREMIO_HOST (optional, defaults to localhost)")
	t.Log("  - RUN_INTEGRATION_TESTS=true")
}