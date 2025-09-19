package chi

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"go-data-gateway/internal/response"
)

// visitor holds rate limiter for each visitor
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.RWMutex
)

// RateLimiter creates a Chi middleware for rate limiting
func RateLimiter(rps int) func(next http.Handler) http.Handler {
	// Start cleanup goroutine
	go cleanupVisitors()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			// Get or create limiter for this IP
			limiter := getVisitor(ip, rps)

			if !limiter.Allow() {
				response.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getVisitor gets or creates a rate limiter for the given IP
func getVisitor(ip string, rps int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(rps), rps*2) // Allow burst of 2x RPS
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes old visitors from the map
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}