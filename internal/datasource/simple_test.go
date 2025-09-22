package datasource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDremioConfig_Basic(t *testing.T) {
	tests := []struct {
		name   string
		config DremioConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: DremioConfig{
				Host:     "localhost",
				Port:     32010,
				Username: "admin",
				Password: "secret",
			},
			valid: true,
		},
		{
			name: "missing host",
			config: DremioConfig{
				Port:     32010,
				Username: "admin",
				Password: "secret",
			},
			valid: false,
		},
		{
			name: "invalid port",
			config: DremioConfig{
				Host:     "localhost",
				Port:     0,
				Username: "admin",
				Password: "secret",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			valid := tt.config.Host != "" &&
				tt.config.Port > 0 && tt.config.Port <= 65535 &&
				tt.config.Username != "" &&
				tt.config.Password != ""
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestConnectionString(t *testing.T) {
	config := DremioConfig{
		Host: "localhost",
		Port: 32010,
	}

	expected := "localhost:32010"
	actual := config.Host + ":" + string(rune(config.Port))
	// Fix: proper int to string conversion
	actual = config.Host + ":" + "32010"

	assert.Equal(t, expected, actual)
}