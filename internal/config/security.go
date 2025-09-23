package config

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// AllowedDremioTables - whitelist of tables that can be queried from Dremio
	AllowedDremioTables []string
	// AllowedBigQueryTables - whitelist of tables that can be queried from BigQuery
	AllowedBigQueryTables []string
}

// GetDefaultSecurityConfig returns default security configuration
func GetDefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		// Only allow specific tables to be queried
		AllowedDremioTables: []string{
			"nessie_iceberg.tender_data",
			"nessie_iceberg.tender_2024",
			"nessie_iceberg.tender_2025",
			"procurement.tender_master",
			"procurement.vendor_list",
		},
		AllowedBigQueryTables: []string{
			"gtp-data-prod.layer_isb.rup_kromaster",
			"gtp-data-prod.analytics.events",
			"spse-prod-sa.public.tender_data",
			"spse-prod-sa.public.rup_data",
		},
	}
}

// IsTableAllowed checks if a table is in the allowed list
func (s *SecurityConfig) IsTableAllowed(table string, source string) bool {
	var allowedTables []string

	switch source {
	case "dremio":
		allowedTables = s.AllowedDremioTables
	case "bigquery":
		allowedTables = s.AllowedBigQueryTables
	default:
		return false
	}

	for _, allowed := range allowedTables {
		if allowed == table {
			return true
		}
	}
	return false
}