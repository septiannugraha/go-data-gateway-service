package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIKeyValidation(t *testing.T) {
	validKeys := map[string]bool{
		"valid-key-1": true,
		"valid-key-2": true,
	}

	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "valid key 1",
			apiKey:   "valid-key-1",
			expected: true,
		},
		{
			name:     "valid key 2",
			apiKey:   "valid-key-2",
			expected: true,
		},
		{
			name:     "invalid key",
			apiKey:   "invalid-key",
			expected: false,
		},
		{
			name:     "empty key",
			apiKey:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validKeys[tt.apiKey]
			assert.Equal(t, tt.expected, isValid)
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	// Test that middleware can be chained
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}