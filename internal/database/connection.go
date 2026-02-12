package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/zanguish/db-mcp/internal/config"

	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type TableColumn struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue sql.NullString
	Key          string
	Extra        string
	Comment      string
}

type Database struct {
	*gorm.DB
	config *config.Config
}

func Connect(cfg *config.Config) (*Database, error) {
	var db *gorm.DB
	var err error

	switch cfg.Driver {
	case "mysql", "tidb":
		db, err = gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	case "postgres", "gaussdb":
		db, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{})
	case "sqlserver":
		db, err = gorm.Open(sqlserver.Open(cfg.DSN), &gorm.Config{})
	case "clickhouse":
		db, err = gorm.Open(clickhouse.Open(cfg.DSN), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Timeout) * time.Second)

	return &Database{
		DB:     db,
		config: cfg,
	}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (d *Database) GetConfig() *config.Config {
	return d.config
}

func (d *Database) GormDB() *gorm.DB {
	return d.DB
}
