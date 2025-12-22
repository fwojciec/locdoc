package sqlite

import (
	"fmt"
	"strings"
	"time"
)

// parseRFC3339 parses an RFC3339 formatted timestamp string.
// Returns an error if parsing fails with a descriptive message including the field name.
func parseRFC3339(value, fieldName string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse %s: %w", fieldName, err)
	}
	return t, nil
}

// appendPagination appends LIMIT and OFFSET clauses to a query builder if values are > 0.
func appendPagination(query *strings.Builder, args *[]any, limit, offset int) {
	if limit > 0 {
		query.WriteString(" LIMIT ?")
		*args = append(*args, limit)
	}
	if offset > 0 {
		query.WriteString(" OFFSET ?")
		*args = append(*args, offset)
	}
}
