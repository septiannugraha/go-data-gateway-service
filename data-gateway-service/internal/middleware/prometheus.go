package middleware

import (
	"github.com/gin-gonic/gin"
)

func PrometheusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(200, "# HELP go_gateway_requests_total Total requests\n# TYPE go_gateway_requests_total counter\ngo_gateway_requests_total 1\n")
	}
}
