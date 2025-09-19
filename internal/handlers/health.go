package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-data-gateway/internal/clients"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "go-data-gateway",
	})
}

func Ready(dremio *clients.DremioClient, bigquery *clients.BigQueryClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"status": "ready",
			"checks": gin.H{},
		}

		if dremio != nil {
			if err := dremio.TestConnection(c.Request.Context()); err != nil {
				status["checks"].(gin.H)["dremio"] = "unhealthy"
			} else {
				status["checks"].(gin.H)["dremio"] = "healthy"
			}
		}

		if bigquery != nil {
			if err := bigquery.TestConnection(c.Request.Context()); err != nil {
				status["checks"].(gin.H)["bigquery"] = "unhealthy"
			} else {
				status["checks"].(gin.H)["bigquery"] = "healthy"
			}
		}

		c.JSON(http.StatusOK, status)
	}
}