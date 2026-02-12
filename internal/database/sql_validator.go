package database

import (
	"fmt"
	"regexp"
	"strings"
)

type SQLValidator struct{}

func NewSQLValidator() *SQLValidator {
	return &SQLValidator{}
}

func (v *SQLValidator) ValidateQuery(query string) error {
	query = strings.TrimSpace(query)
	query = removeComments(query)

	upperQuery := strings.ToUpper(query)

	prefixes := []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN", "USE"}
	isAllowedPrefix := false
	for _, op := range prefixes {
		if strings.HasPrefix(upperQuery, op) {
			isAllowedPrefix = true
			break
		}
	}

	if !isAllowedPrefix {
		return ErrQueryNotAllowed
	}

	dangerousKeywords := []string{
		"INTO OUTFILE",
		"INTO DUMPFILE",
		"INTO @",
		"INTO $",
		"COPY ",
		"COPY(",
		"INSERT ",
		"INSERT INTO",
		"UPDATE ",
		"DELETE ",
		"DROP ",
		"ALTER ",
		"CREATE ",
		"TRUNCATE ",
		"REPLACE ",
		"MERGE ",
		"GRANT ",
		"REVOKE ",
		"LOCK ",
		"UNLOCK ",
		"EXECUTE ",
		"EXEC ",
		"CALL ",
		"LOAD ",
		"RENAME ",
	}

	for _, kw := range dangerousKeywords {
		if strings.Contains(upperQuery, kw) {
			return &SQLQueryError{Message: fmt.Sprintf("unsafe operation: %s is not allowed", kw)}
		}
	}

	return nil
}

func removeComments(query string) string {
	query = regexp.MustCompile(`--.*$`).ReplaceAllString(query, "")
	query = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(query, "")
	return query
}

var ErrQueryNotAllowed = &SQLQueryError{Message: "query must start with SELECT, SHOW, DESCRIBE, EXPLAIN, or USE"}

type SQLQueryError struct {
	Message string
}

func (e *SQLQueryError) Error() string {
	return e.Message
}
