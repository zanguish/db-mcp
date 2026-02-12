package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Driver        string
	DSN           string
	MaxResults    int
	Timeout       int
	Transport     string
	HTTPPort      int
	ServerName    string
	ServerVersion string
	LogLevel      string
}

func (c *Config) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("--driver is required")
	}

	validDrivers := []string{"mysql", "postgres", "sqlite", "sqlserver", "tidb", "gaussdb", "clickhouse"}
	driverValid := false
	for _, d := range validDrivers {
		if strings.ToLower(c.Driver) == d {
			driverValid = true
			c.Driver = d
			break
		}
	}
	if !driverValid {
		return fmt.Errorf("--driver must be one of: mysql, postgres, sqlite, sqlserver, tidb, gaussdb, clickhouse")
	}

	if c.DSN == "" {
		return fmt.Errorf("--dsn is required")
	}

	if c.MaxResults <= 0 {
		return fmt.Errorf("--max-results must be greater than 0")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("--timeout must be greater than 0")
	}

	if c.Transport == "http" && c.HTTPPort <= 0 {
		return fmt.Errorf("--http-port must be greater than 0 for http transport")
	}

	return nil
}

func (c *Config) MaskedDSN() string {
	if strings.Contains(c.DSN, "@") {
		parts := strings.SplitN(c.DSN, "@", 2)
		if len(parts) == 2 {
			return "***@" + parts[1]
		}
	}
	return "***"
}

func (c *Config) String() string {
	return fmt.Sprintf("Driver: %s, DSN: %s, MaxResults: %d, Timeout: %d, Transport: %s, HTTPPort: %d, Name: %s, Version: %s, LogLevel: %s",
		c.Driver, c.MaskedDSN(), c.MaxResults, c.Timeout, c.Transport, c.HTTPPort, c.ServerName, c.ServerVersion, c.LogLevel)
}
