package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth validates API keys
func APIKeyAuth(validKeys []string) gin.HandlerFunc {
	// Create a map for O(1) lookup
	keyMap := make(map[string]bool)
	for _, key := range validKeys {
		keyMap[key] = true
	}

	return func(c *gin.Context) {
		// Check for API key in header
		apiKey := c.GetHeader("X-API-Key")

		// Also check Authorization header
		if apiKey == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		// Check if key is valid
		if apiKey == "" || !keyMap[apiKey] {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing API key",
			})
			c.Abort()
			return
		}

		// Store API key in context for logging
		c.Set("api_key", apiKey)
		c.Next()
	}
}