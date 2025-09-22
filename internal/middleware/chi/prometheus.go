package chi

import (
	"fmt"
	"net/http"
	"time"
)

// Simple Prometheus metrics handler
func PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# HELP go_gateway_requests_total Total number of requests\n")
		fmt.Fprintf(w, "# TYPE go_gateway_requests_total counter\n")
		fmt.Fprintf(w, "go_gateway_requests_total %d\n", requestCount)
		fmt.Fprintf(w, "\n# HELP go_gateway_uptime_seconds Service uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE go_gateway_uptime_seconds gauge\n")
		fmt.Fprintf(w, "go_gateway_uptime_seconds %.0f\n", time.Since(startTime).Seconds())
	})
}

var (
	requestCount int64
	startTime    = time.Now()
)

// MetricsCollector middleware to count requests
func MetricsCollector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		next.ServeHTTP(w, r)
	})
}
