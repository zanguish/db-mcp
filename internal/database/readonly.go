package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/zanguish/db-mcp/internal/config"

	"gorm.io/gorm"
)

type ReadOnlyDB struct {
	db        *gorm.DB
	config    *config.Config
	validator *SQLValidator
}

func NewReadOnlyDB(db *gorm.DB, cfg *config.Config) *ReadOnlyDB {
	return &ReadOnlyDB{
		db:        db,
		config:    cfg,
		validator: NewSQLValidator(),
	}
}

func (r *ReadOnlyDB) ListTables() ([]string, error) {
	var tables []string
	switch r.config.Driver {
	case "mysql", "tidb":
		// SHOW TABLES returns a driver/database-dependent column name
		// (for example, Tables_in_test-newapi). Scan by column position so
		// the result does not depend on that generated name.
		return r.scanSingleStringColumn("SHOW TABLES")
	case "postgres", "gaussdb":
		var result []struct {
			TableName string `gorm:"column:tablename"`
		}
		if err := r.db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = current_schema()").Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		for _, row := range result {
			tables = append(tables, row.TableName)
		}
	case "sqlite":
		var result []struct {
			TableName string `gorm:"column:name"`
		}
		if err := r.db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		for _, row := range result {
			tables = append(tables, row.TableName)
		}
	case "sqlserver":
		var result []struct {
			TableName string `gorm:"column:TABLE_NAME"`
		}
		if err := r.db.Raw("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE'").Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		for _, row := range result {
			tables = append(tables, row.TableName)
		}
	case "clickhouse":
		var result []struct {
			TableName string `gorm:"column:name"`
		}
		if err := r.db.Raw("SELECT name FROM system.tables WHERE database = currentDatabase()").Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		for _, row := range result {
			tables = append(tables, row.TableName)
		}
	default:
		return nil, fmt.Errorf("unsupported driver for listing tables: %s", r.config.Driver)
	}

	return tables, nil
}

// scanSingleStringColumn reads a one-column result set without relying on its
// database-generated column name. MySQL's SHOW TABLES column is named after the
// selected database (for example, Tables_in_test-newapi), so a static GORM
// struct tag cannot match it. Scanning through database/sql also normalizes
// driver-returned []byte values to strings.
func (r *ReadOnlyDB) scanSingleStringColumn(query string) ([]string, error) {
	rows, err := r.db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		tables = append(tables, tableName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	return tables, nil
}

func (r *ReadOnlyDB) DescribeTable(tableName string) ([]TableColumn, error) {
	var columns []TableColumn

	safeTableName := sanitizeIdentifier(tableName, r.config.Driver)

	switch r.config.Driver {
	case "mysql":
		var result []struct {
			Field    string
			Type     string
			Nullable string
			Key      string
			Default  *string
			Extra    string
			Comment  string
		}
		query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s", safeTableName)
		if err := r.db.Raw(query).Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}
		for _, col := range result {
			nullable := strings.ToLower(col.Nullable) == "yes"
			var defaultVal sql.NullString
			if col.Default != nil {
				defaultVal = sql.NullString{String: *col.Default, Valid: true}
			}
			columns = append(columns, TableColumn{
				Name:         col.Field,
				Type:         col.Type,
				Nullable:     nullable,
				DefaultValue: defaultVal,
				Key:          col.Key,
				Extra:        col.Extra,
				Comment:      col.Comment,
			})
		}
	case "sqlite":
		var result []struct {
			Cid       int
			Name      string
			Type      string
			NotNull   int
			DfltValue *string
			Pk        int
		}
		query := "PRAGMA table_info(?)"
		if err := r.db.Raw(query, tableName).Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}
		for _, col := range result {
			nullable := col.NotNull == 0
			var defaultVal sql.NullString
			if col.DfltValue != nil {
				defaultVal = sql.NullString{String: *col.DfltValue, Valid: true}
			}
			key := ""
			if col.Pk == 1 {
				key = "PRI"
			}
			columns = append(columns, TableColumn{
				Name:         col.Name,
				Type:         col.Type,
				Nullable:     nullable,
				DefaultValue: defaultVal,
				Key:          key,
				Comment:      "",
			})
		}
	case "postgres", "gaussdb":
		var result []struct {
			OrdinalPosition int     `gorm:"column:ordinal_position"`
			ColumnName      string  `gorm:"column:column_name"`
			DataType        string  `gorm:"column:data_type"`
			IsNullable      string  `gorm:"column:is_nullable"`
			ColumnDefault   *string `gorm:"column:column_default"`
		}

		query := `SELECT ordinal_position, column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ?
			ORDER BY ordinal_position`
		if err := r.db.Raw(query, tableName).Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}

		commentMap := make(map[string]string)
		if r.config.Driver == "postgres" {
			var commentResult []struct {
				ColumnName string `gorm:"column:column_name"`
				Comment    string `gorm:"column:description"`
			}
			r.db.Raw(`SELECT a.attname AS column_name, COALESCE(d.description, '') AS description
				FROM pg_attribute a
				LEFT JOIN pg_class c ON a.attrelid = c.oid
				LEFT JOIN pg_namespace n ON c.relnamespace = n.oid
				LEFT JOIN pg_description d ON d.objoid = c.oid AND d.objsubid = a.attnum
				WHERE n.nspname = current_schema() AND c.relname = ? AND a.attnum > 0
				ORDER BY a.attnum`, tableName).Scan(&commentResult)
			for _, c := range commentResult {
				commentMap[c.ColumnName] = c.Comment
			}
		} else if r.config.Driver == "gaussdb" {
			var commentResult []struct {
				ColumnName string `gorm:"column:column_name"`
				Comment    string `gorm:"column:comments"`
			}
			r.db.Raw(`SELECT column_name, COALESCE(comments, '') AS comments
				FROM DBA_TAB_COLUMNS WHERE owner = current_schema() AND table_name = ?`,
				tableName).Scan(&commentResult)
			for _, c := range commentResult {
				commentMap[c.ColumnName] = c.Comment
			}
		}

		for _, col := range result {
			nullable := strings.ToLower(col.IsNullable) == "yes"
			var defaultVal sql.NullString
			if col.ColumnDefault != nil {
				defaultVal = sql.NullString{String: *col.ColumnDefault, Valid: true}
			}
			columns = append(columns, TableColumn{
				Name:         col.ColumnName,
				Type:         col.DataType,
				Nullable:     nullable,
				DefaultValue: defaultVal,
				Comment:      commentMap[col.ColumnName],
			})
		}
	case "sqlserver":
		var result []struct {
			ColumnName    string  `gorm:"column:COLUMN_NAME"`
			DataType      string  `gorm:"column:DATA_TYPE"`
			IsNullable    string  `gorm:"column:IS_NULLABLE"`
			ColumnDefault *string `gorm:"column:COLUMN_DEFAULT"`
		}
		query := `SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`
		if err := r.db.Raw(query, tableName).Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}

		var commentResult []struct {
			ColumnName string `gorm:"column:COLUMN_NAME"`
			Comment    string `gorm:"column:COMMENT"`
		}
		commentQuery := `SELECT objname AS COLUMN_NAME, value AS COMMENT
			FROM fn_listextendedproperty(NULL, 'SCHEMA', 'dbo', 'TABLE', ?, 'COLUMN', NULL)`
		r.db.Raw(commentQuery, tableName).Scan(&commentResult)

		commentMap := make(map[string]string)
		for _, c := range commentResult {
			commentMap[c.ColumnName] = c.Comment
		}

		for _, col := range result {
			nullable := strings.ToUpper(col.IsNullable) == "YES"
			var defaultVal sql.NullString
			if col.ColumnDefault != nil {
				defaultVal = sql.NullString{String: *col.ColumnDefault, Valid: true}
			}
			columns = append(columns, TableColumn{
				Name:         col.ColumnName,
				Type:         col.DataType,
				Nullable:     nullable,
				DefaultValue: defaultVal,
				Comment:      commentMap[col.ColumnName],
			})
		}
	case "tidb":
		var result []struct {
			Field    string
			Type     string
			Nullable string
			Key      string
			Default  *string
			Extra    string
			Comment  string
		}
		query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s", safeTableName)
		if err := r.db.Raw(query).Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}
		for _, col := range result {
			nullable := strings.ToLower(col.Nullable) == "yes"
			var defaultVal sql.NullString
			if col.Default != nil {
				defaultVal = sql.NullString{String: *col.Default, Valid: true}
			}
			columns = append(columns, TableColumn{
				Name:         col.Field,
				Type:         col.Type,
				Nullable:     nullable,
				DefaultValue: defaultVal,
				Key:          col.Key,
				Extra:        col.Extra,
				Comment:      col.Comment,
			})
		}
	case "clickhouse":
		var result []struct {
			Name              string  `gorm:"column:name"`
			Type              string  `gorm:"column:type"`
			DefaultType       string  `gorm:"column:default_type"`
			DefaultExpression *string `gorm:"column:default_expression"`
		}
		query := fmt.Sprintf("DESCRIBE TABLE %s", safeTableName)
		if err := r.db.Raw(query).Scan(&result).Error; err != nil {
			return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}
		for _, col := range result {
			nullable := !strings.Contains(col.DefaultType, "NOT")
			var defaultVal sql.NullString
			if col.DefaultExpression != nil {
				defaultVal = sql.NullString{String: *col.DefaultExpression, Valid: true}
			}
			columns = append(columns, TableColumn{
				Name:         col.Name,
				Type:         col.Type,
				Nullable:     nullable,
				DefaultValue: defaultVal,
				Comment:      "",
			})
		}
	default:
		return nil, fmt.Errorf("unsupported driver for describing table: %s", r.config.Driver)
	}

	return columns, nil
}

func (r *ReadOnlyDB) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	if err := r.validator.ValidateQuery(query); err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)

	var results []map[string]interface{}

	rows, err := r.db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			rowMap[col] = val
		}

		results = append(results, rowMap)

		if len(results) >= r.config.MaxResults {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

func (r *ReadOnlyDB) GetTableSample(tableName string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > r.config.MaxResults {
		limit = r.config.MaxResults
	}

	var results []map[string]interface{}

	rows, err := r.db.Table(tableName).Limit(limit).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get table sample: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			rowMap[col] = val
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

func (r *ReadOnlyDB) GetTableCount(tableName string) (int64, error) {
	var count int64
	err := r.db.Table(tableName).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count table %s: %w", tableName, err)
	}
	return count, nil
}

func (r *ReadOnlyDB) GetValidator() *SQLValidator {
	return r.validator
}

func (r *ReadOnlyDB) GetConfig() *config.Config {
	return r.config
}

func sanitizeIdentifier(id string, driver string) string {
	id = strings.Trim(id, "`'\"")
	if strings.ContainsAny(id, "`'\"") {
		return ""
	}
	switch driver {
	case "mysql", "tidb", "clickhouse":
		return "`" + id + "`"
	default:
		return id
	}
}
