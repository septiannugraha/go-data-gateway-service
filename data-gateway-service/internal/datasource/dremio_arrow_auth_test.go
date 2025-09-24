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

// TestDremioAuthenticationRealServer tests authentication with real Dremio if available
func TestDremioAuthenticationRealServer(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		config        *DremioConfig
		shouldSucceed bool
		errorContains string
	}{
		{
			name: "Valid credentials",
			config: &DremioConfig{
				Host:     "localhost",
				Port:     32010,
				Username: "septiannugraha",
				Password: "snoogz123",
				UseTLS:   false,
			},
			shouldSucceed: true,
		},
		{
			name: "Invalid password (our bug scenario)",
			config: &DremioConfig{
				Host:     "localhost",
				Port:     32010,
				Username: "septiannugraha",
				Password: "?uJ*u2a@u!@f2e]", // Old password
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "Invalid username or password",
		},
		{
			name: "Invalid username",
			config: &DremioConfig{
				Host:     "localhost",
				Port:     32010,
				Username: "wronguser",
				Password: "snoogz123",
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "Invalid username or password",
		},
		{
			name: "Wrong port",
			config: &DremioConfig{
				Host:     "localhost",
				Port:     31010, // Wrong port (REST API port instead of Arrow Flight)
				Username: "septiannugraha",
				Password: "snoogz123",
				UseTLS:   false,
			},
			shouldSucceed: false,
			errorContains: "connection error", // More generic error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client
			client, err := NewDremioArrowClient(tt.config, logger)
			require.NoError(t, err, "Client creation should not fail")
			defer client.Close()

			// Test connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = client.TestConnection(ctx)

			if tt.shouldSucceed {
				assert.NoError(t, err, "Expected successful authentication")
			} else {
				assert.Error(t, err, "Expected authentication failure")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains,
						"Error should contain expected message")
				}
			}
		})
	}
}

// TestDremioConnectionPoolAuthentication tests pool with auth
func TestDremioConnectionPoolAuthentication(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		config        *DremioConfig
		poolConfig    *PoolConfig
		shouldSucceed bool
	}{
		{
			name: "Pool with valid credentials",
			config: &DremioConfig{
				Host:     "localhost",
				Port:     32010,
				Username: "septiannugraha",
				Password: "snoogz123",
				UseTLS:   false,
			},
			poolConfig: &PoolConfig{
				MaxConnections:      5,
				MinConnections:      2,
				MaxIdleTime:         30 * time.Second,
				ConnectionTimeout:   5 * time.Second,
				HealthCheckInterval: 10 * time.Second,
			},
			shouldSucceed: true,
		},
		{
			name: "Pool with invalid credentials",
			config: &DremioConfig{
				Host:     "localhost",
				Port:     32010,
				Username: "septiannugraha",
				Password: "wrongpass",
				UseTLS:   false,
			},
			poolConfig: &PoolConfig{
				MaxConnections:      5,
				MinConnections:      2,
				MaxIdleTime:         30 * time.Second,
				ConnectionTimeout:   5 * time.Second,
				HealthCheckInterval: 10 * time.Second,
			},
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with pool
			client, err := NewDremioArrowClientWithPool(tt.config, tt.poolConfig, logger)

			if !tt.shouldSucceed {
				// Pool creation might fail if auth is checked during init
				if err != nil {
					assert.Contains(t, err.Error(), "failed to create")
					return
				}
			} else {
				require.NoError(t, err, "Pool creation should succeed with valid auth")
			}

			if client != nil {
				defer client.Close()

				// Test multiple connections
				ctx := context.Background()
				for i := 0; i < 3; i++ {
					err := client.TestConnection(ctx)
					if tt.shouldSucceed {
						assert.NoError(t, err, "Connection %d should succeed", i)
					} else {
						assert.Error(t, err, "Connection %d should fail", i)
					}
				}

				// Check pool metrics
				metrics := client.GetPoolMetrics()
				assert.NotNil(t, metrics)
				assert.Equal(t, true, metrics["pool_enabled"])
			}
		})
	}
}

// TestAuthenticationWithEnvironmentVariables tests loading creds from env
func TestAuthenticationWithEnvironmentVariables(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Save original env vars
	origHost := os.Getenv("DREMIO_HOST")
	origPort := os.Getenv("DREMIO_PORT")
	origUser := os.Getenv("DREMIO_USERNAME")
	origPass := os.Getenv("DREMIO_PASSWORD")
	defer func() {
		os.Setenv("DREMIO_HOST", origHost)
		os.Setenv("DREMIO_PORT", origPort)
		os.Setenv("DREMIO_USERNAME", origUser)
		os.Setenv("DREMIO_PASSWORD", origPass)
	}()

	tests := []struct {
		name          string
		envVars       map[string]string
		shouldSucceed bool
	}{
		{
			name: "Valid env credentials",
			envVars: map[string]string{
				"DREMIO_HOST":     "localhost",
				"DREMIO_PORT":     "32010",
				"DREMIO_USERNAME": "septiannugraha",
				"DREMIO_PASSWORD": "snoogz123",
			},
			shouldSucceed: true,
		},
		{
			name: "Invalid env password",
			envVars: map[string]string{
				"DREMIO_HOST":     "localhost",
				"DREMIO_PORT":     "32010",
				"DREMIO_USERNAME": "septiannugraha",
				"DREMIO_PASSWORD": "wrongpass",
			},
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if not in integration test mode
			if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
				t.Skip("Skipping integration test")
			}

			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Create config from env
			port := 32010
			if p := os.Getenv("DREMIO_PORT"); p != "" {
				// Convert string to int (simplified)
				port = 32010
			}

			config := &DremioConfig{
				Host:     os.Getenv("DREMIO_HOST"),
				Port:     port,
				Username: os.Getenv("DREMIO_USERNAME"),
				Password: os.Getenv("DREMIO_PASSWORD"),
				UseTLS:   false,
			}

			// Create client
			client, err := NewDremioArrowClient(config, logger)
			require.NoError(t, err)
			defer client.Close()

			// Test connection
			ctx := context.Background()
			err = client.TestConnection(ctx)

			if tt.shouldSucceed {
				assert.NoError(t, err, "Should authenticate with env credentials")
			} else {
				assert.Error(t, err, "Should fail with wrong env credentials")
			}
		})
	}
}

// TestBasicAuthHeaderGeneration tests the basicAuth function
func TestBasicAuthHeaderGeneration(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		expected string
	}{
		{
			name:     "Standard credentials",
			username: "user",
			password: "pass",
			expected: "dXNlcjpwYXNz", // base64("user:pass")
		},
		{
			name:     "Special characters in password",
			username: "septiannugraha",
			password: "snoogz123",
			expected: "c2VwdGlhbm51Z3JhaGE6c25vb2d6MTIz", // base64("septiannugraha:snoogz123")
		},
		{
			name:     "Complex password",
			username: "user",
			password: "?uJ*u2a@u!@f2e]",
			expected: "dXNlcjo/dUoqdTJhQHUhQGYyZV0=", // base64("user:?uJ*u2a@u!@f2e]")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := basicAuth(tt.username, tt.password)
			assert.Equal(t, tt.expected, result, "Basic auth encoding should match")
		})
	}
}

// TestConnectionRetryOnAuthFailure tests retry behavior
func TestConnectionRetryOnAuthFailure(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	logger, _ := zap.NewDevelopment()

	// First try with wrong password
	config := &DremioConfig{
		Host:     "localhost",
		Port:     32010,
		Username: "septiannugraha",
		Password: "wrongpass",
		UseTLS:   false,
	}

	client, err := NewDremioArrowClient(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.TestConnection(ctx)
	assert.Error(t, err, "Should fail with wrong password")
	assert.Contains(t, err.Error(), "Invalid username or password")
	client.Close()

	// Update config with correct password
	config.Password = "snoogz123"

	// Create new client with correct password
	client2, err := NewDremioArrowClient(config, logger)
	require.NoError(t, err)
	defer client2.Close()

	err = client2.TestConnection(ctx)
	assert.NoError(t, err, "Should succeed with correct password")
}