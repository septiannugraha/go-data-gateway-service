package cache

import (
	"sync"
	"time"
)

// Metrics tracks cache performance
type Metrics struct {
	mu        sync.RWMutex
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Errors    int64
	TotalTime time.Duration
	HitTime   time.Duration
	MissTime  time.Duration
	LastReset time.Time
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		LastReset: time.Now(),
	}
}

// RecordHit records a cache hit
func (m *Metrics) RecordHit(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Hits++
	m.HitTime += duration
	m.TotalTime += duration
}

// RecordMiss records a cache miss
func (m *Metrics) RecordMiss(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Misses++
	m.MissTime += duration
	m.TotalTime += duration
}

// RecordSet records a cache set operation
func (m *Metrics) RecordSet() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sets++
}

// RecordDelete records a cache delete operation
func (m *Metrics) RecordDelete() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Deletes++
}

// RecordError records a cache error
func (m *Metrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors++
}

// GetStats returns current statistics
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.Hits + m.Misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(m.Hits) / float64(total) * 100
	}

	avgHitTime := time.Duration(0)
	if m.Hits > 0 {
		avgHitTime = m.HitTime / time.Duration(m.Hits)
	}

	avgMissTime := time.Duration(0)
	if m.Misses > 0 {
		avgMissTime = m.MissTime / time.Duration(m.Misses)
	}

	return map[string]interface{}{
		"hits":          m.Hits,
		"misses":        m.Misses,
		"sets":          m.Sets,
		"deletes":       m.Deletes,
		"errors":        m.Errors,
		"hit_rate":      hitRate,
		"total_time":    m.TotalTime.String(),
		"avg_hit_time":  avgHitTime.String(),
		"avg_miss_time": avgMissTime.String(),
		"uptime":        time.Since(m.LastReset).String(),
	}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Hits = 0
	m.Misses = 0
	m.Sets = 0
	m.Deletes = 0
	m.Errors = 0
	m.TotalTime = 0
	m.HitTime = 0
	m.MissTime = 0
	m.LastReset = time.Now()
}
