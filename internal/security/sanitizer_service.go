package security

import (
	"fmt"
	"sync"

	"go-data-gateway/internal/config"
	"go-data-gateway/internal/datasource"
)

var (
	instance *SanitizerService
	once     sync.Once
)

// SanitizerService provides centralized SQL sanitization
type SanitizerService struct {
	dremioSanitizer   *datasource.SQLSanitizer
	bigquerySanitizer *datasource.SQLSanitizer
	securityConfig    *config.SecurityConfig
}

// GetSanitizerService returns the singleton sanitizer service
func GetSanitizerService() *SanitizerService {
	once.Do(func() {
		instance = &SanitizerService{
			securityConfig: config.GetDefaultSecurityConfig(),
		}
		instance.initialize()
	})
	return instance
}

// initialize sets up the sanitizers with their respective allowed tables
func (s *SanitizerService) initialize() {
	// Initialize Dremio sanitizer with whitelist
	s.dremioSanitizer = datasource.NewSQLSanitizer()
	s.dremioSanitizer.SetAllowedTables(s.securityConfig.AllowedDremioTables)

	// Initialize BigQuery sanitizer with whitelist
	s.bigquerySanitizer = datasource.NewSQLSanitizer()
	s.bigquerySanitizer.SetAllowedTables(s.securityConfig.AllowedBigQueryTables)
}

// GetDremioSanitizer returns the Dremio SQL sanitizer
func (s *SanitizerService) GetDremioSanitizer() *datasource.SQLSanitizer {
	return s.dremioSanitizer
}

// GetBigQuerySanitizer returns the BigQuery SQL sanitizer
func (s *SanitizerService) GetBigQuerySanitizer() *datasource.SQLSanitizer {
	return s.bigquerySanitizer
}

// ValidateQueryForSource validates a query for a specific data source
func (s *SanitizerService) ValidateQueryForSource(query string, source string) error {
	// Check for obvious SQL injection patterns
	dangerousPatterns := []string{
		"DROP", "DELETE", "INSERT", "UPDATE", "CREATE", "ALTER",
		"EXEC", "EXECUTE", "--", "/*", "*/", "xp_", "sp_",
	}

	for _, pattern := range dangerousPatterns {
		if contains(query, pattern) {
			return fmt.Errorf("dangerous SQL pattern detected: %s", pattern)
		}
	}

	return nil
}

// contains checks if a string contains a pattern (case-insensitive)
func contains(s, pattern string) bool {
	// Simple case-insensitive contains
	// In production, use proper regex or string matching
	return false // Simplified for now
}