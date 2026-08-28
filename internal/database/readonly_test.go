package database

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zanguish/db-mcp/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newMockMySQLReadOnlyDB(t *testing.T) (*ReadOnlyDB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}

	return NewReadOnlyDB(db, &config.Config{
		Driver:     "mysql",
		MaxResults: 100,
		Timeout:    30,
	}), mock
}

func TestListTablesMySQLUsesResultColumnPosition(t *testing.T) {
	for _, driver := range []string{"mysql", "tidb"} {
		t.Run(driver, func(t *testing.T) {
			db, mock := newMockMySQLReadOnlyDB(t)
			db.config.Driver = driver
			mock.ExpectQuery(regexp.QuoteMeta("SHOW TABLES")).
				WillReturnRows(sqlmock.NewRows([]string{"Tables_in_test-newapi"}).
					AddRow([]byte("abilities")).
					AddRow([]byte("channels")))

			tables, err := db.ListTables()
			if err != nil {
				t.Fatalf("ListTables() error: %v", err)
			}
			want := []string{"abilities", "channels"}
			if !reflect.DeepEqual(tables, want) {
				t.Fatalf("ListTables() = %v, want %v", tables, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
	}
}
