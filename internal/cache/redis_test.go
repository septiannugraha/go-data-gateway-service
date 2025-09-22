package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	cache := &RedisCache{}

	tests := []struct {
		name   string
		source string
		query  string
		want   string
	}{
		{
			name:   "simple query",
			source: "BIGQUERY",
			query:  "SELECT * FROM table",
			want:   "query:BIGQUERY:",
		},
		{
			name:   "same query different source",
			source: "DREMIO",
			query:  "SELECT * FROM table",
			want:   "query:DREMIO:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.GenerateKey(tt.source, tt.query)
			assert.Contains(t, got, tt.want)
			assert.Len(t, got, len(tt.want)+16) // 8 bytes hex = 16 chars
		})
	}
}

func TestNoOpCache(t *testing.T) {
	cache := &NoOpCache{}
	ctx := context.Background()

	t.Run("Get returns no data", func(t *testing.T) {
		data, hit, err := cache.Get(ctx, "test-key")
		assert.Nil(t, data)
		assert.False(t, hit)
		assert.Nil(t, err)
	})

	t.Run("Set does nothing", func(t *testing.T) {
		err := cache.Set(ctx, "test-key", "test-data", 5*time.Minute)
		assert.Nil(t, err)
	})

	t.Run("Stats returns noop info", func(t *testing.T) {
		stats, err := cache.Stats(ctx)
		require.NoError(t, err)
		assert.Equal(t, false, stats["connected"])
		assert.Equal(t, "noop", stats["type"])
	})
}

func TestMetrics(t *testing.T) {
	m := NewMetrics()

	t.Run("RecordHit increments counter", func(t *testing.T) {
		m.RecordHit(100 * time.Millisecond)
		stats := m.GetStats()
		assert.Equal(t, int64(1), stats["hits"])
		assert.Equal(t, float64(100), stats["hit_rate"])
	})

	t.Run("RecordMiss increments counter", func(t *testing.T) {
		m.RecordMiss(200 * time.Millisecond)
		stats := m.GetStats()
		assert.Equal(t, int64(1), stats["hits"])
		assert.Equal(t, int64(1), stats["misses"])
		assert.Equal(t, float64(50), stats["hit_rate"])
	})

	t.Run("Reset clears all metrics", func(t *testing.T) {
		m.Reset()
		stats := m.GetStats()
		assert.Equal(t, int64(0), stats["hits"])
		assert.Equal(t, int64(0), stats["misses"])
		assert.Equal(t, float64(0), stats["hit_rate"])
	})
}
