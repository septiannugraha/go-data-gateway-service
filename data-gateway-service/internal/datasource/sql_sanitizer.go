package datasource

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLSanitizer provides methods to safely build SQL queries
type SQLSanitizer struct {
	// Whitelist of allowed table names (can be loaded from config)
	allowedTables map[string]bool
	// Pattern for valid identifier names
	identifierPattern *regexp.Regexp
}

// NewSQLSanitizer creates a new SQL sanitizer
func NewSQLSanitizer() *SQLSanitizer {
	return &SQLSanitizer{
		// Only allow alphanumeric, underscore, dash, and dots for schema.table format
		identifierPattern: regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`),
		allowedTables:     make(map[string]bool),
	}
}

// SetAllowedTables sets the whitelist of allowed table names
func (s *SQLSanitizer) SetAllowedTables(tables []string) {
	s.allowedTables = make(map[string]bool)
	for _, table := range tables {
		s.allowedTables[table] = true
	}
}

// ValidateTableName validates and sanitizes table names
func (s *SQLSanitizer) ValidateTableName(table string) (string, error) {
	// Remove any backticks or quotes
	table = strings.ReplaceAll(table, "`", "")
	table = strings.ReplaceAll(table, "'", "")
	table = strings.ReplaceAll(table, "\"", "")

	// Check against whitelist if configured
	if len(s.allowedTables) > 0 {
		if !s.allowedTables[table] {
			return "", fmt.Errorf("table '%s' is not in allowed list", table)
		}
	}

	// Validate format
	if !s.identifierPattern.MatchString(table) {
		return "", fmt.Errorf("invalid table name format: '%s'", table)
	}

	// Check for SQL injection patterns
	lowerTable := strings.ToLower(table)
	dangerousPatterns := []string{
		"select", "insert", "update", "delete", "drop", "create",
		"alter", "exec", "execute", "union", "--", "/*", "*/",
		";", "xp_", "sp_", "0x", "\\x",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerTable, pattern) {
			return "", fmt.Errorf("potential SQL injection detected in table name: '%s'", table)
		}
	}

	return table, nil
}

// ValidateColumnName validates column names for ORDER BY clauses
func (s *SQLSanitizer) ValidateColumnName(column string) (string, error) {
	// Remove any backticks or quotes
	column = strings.ReplaceAll(column, "`", "")
	column = strings.ReplaceAll(column, "'", "")
	column = strings.ReplaceAll(column, "\"", "")

	// Only allow simple column names (no functions or expressions)
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(column) {
		return "", fmt.Errorf("invalid column name: '%s'", column)
	}

	return column, nil
}

// ValidateOrderDirection validates ORDER BY direction
func (s *SQLSanitizer) ValidateOrderDirection(dir string) (string, error) {
	dir = strings.ToUpper(strings.TrimSpace(dir))
	if dir != "ASC" && dir != "DESC" && dir != "" {
		return "", fmt.Errorf("invalid order direction: '%s'", dir)
	}
	if dir == "" {
		dir = "ASC"
	}
	return dir, nil
}

// BuildSafeTableQuery builds a safe SELECT query with validation
func (s *SQLSanitizer) BuildSafeTableQuery(table string, opts *QueryOptions) (string, error) {
	// Validate table name
	safeTable, err := s.ValidateTableName(table)
	if err != nil {
		return "", fmt.Errorf("table validation failed: %w", err)
	}

	// Start building query
	query := fmt.Sprintf("SELECT * FROM %s", safeTable)

	if opts != nil {
		// Add ORDER BY if specified
		if opts.OrderBy != "" {
			safeColumn, err := s.ValidateColumnName(opts.OrderBy)
			if err != nil {
				return "", fmt.Errorf("order by validation failed: %w", err)
			}

			safeDir, err := s.ValidateOrderDirection(opts.OrderDir)
			if err != nil {
				return "", fmt.Errorf("order direction validation failed: %w", err)
			}

			query += fmt.Sprintf(" ORDER BY %s %s", safeColumn, safeDir)
		}

		// Add LIMIT (already safe as integer)
		if opts.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", opts.Limit)

			// Add OFFSET (already safe as integer)
			if opts.Offset > 0 {
				query += fmt.Sprintf(" OFFSET %d", opts.Offset)
			}
		}
	}

	return query, nil
}

// EscapeString escapes special characters in SQL strings
// Note: Prefer parameterized queries when possible
func (s *SQLSanitizer) EscapeString(input string) string {
	// Replace single quotes with escaped version
	escaped := strings.ReplaceAll(input, "'", "''")
	// Remove null bytes
	escaped = strings.ReplaceAll(escaped, "\x00", "")
	return escaped
}