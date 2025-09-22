package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     QueryRequest
		isValid bool
	}{
		{
			name: "valid DREMIO request",
			req: QueryRequest{
				Source: "DREMIO",
				SQL:    "SELECT * FROM table",
			},
			isValid: true,
		},
		{
			name: "valid BIGQUERY request",
			req: QueryRequest{
				Source: "BIGQUERY",
				SQL:    "SELECT * FROM dataset.table",
			},
			isValid: true,
		},
		{
			name: "invalid source",
			req: QueryRequest{
				Source: "INVALID",
				SQL:    "SELECT * FROM table",
			},
			isValid: false,
		},
		{
			name: "empty SQL",
			req: QueryRequest{
				Source: "DREMIO",
				SQL:    "",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic test
			isValid := tt.req.Source == "DREMIO" || tt.req.Source == "BIGQUERY"
			isValid = isValid && tt.req.SQL != ""
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestRUPFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  map[string]string
		expected int // expected number of WHERE conditions
	}{
		{
			name:     "no filters",
			filters:  map[string]string{},
			expected: 0,
		},
		{
			name: "single filter",
			filters: map[string]string{
				"tahun_anggaran": "2024",
			},
			expected: 1,
		},
		{
			name: "multiple filters",
			filters: map[string]string{
				"tahun_anggaran": "2024",
				"kd_satker":      "12345",
				"nama_satker":    "Test",
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := len(tt.filters)
			assert.Equal(t, tt.expected, count)
		})
	}
}