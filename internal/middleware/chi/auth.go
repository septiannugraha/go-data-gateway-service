package chi

import (
	"net/http"
	"strings"

	"go-data-gateway/internal/response"
)

// APIKeyAuth validates API keys for Chi router
func APIKeyAuth(validKeys []string) func(next http.Handler) http.Handler {
	// Create map for O(1) lookup
	keyMap := make(map[string]bool)
	for _, key := range validKeys {
		keyMap[key] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for API key in header
			apiKey := r.Header.Get("X-API-Key")

			// Also check Authorization header
			if apiKey == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			// Validate key
			if apiKey == "" || !keyMap[apiKey] {
				response.Error(w, "Invalid or missing API key", http.StatusUnauthorized)
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}
