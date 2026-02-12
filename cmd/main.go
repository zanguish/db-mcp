package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zanguish/db-mcp/internal/config"
	"github.com/zanguish/db-mcp/internal/database"
	"github.com/zanguish/db-mcp/internal/server"

	"github.com/spf13/cobra"
)

var logger *slog.Logger

var cfg = &config.Config{}

var rootCmd = &cobra.Command{
	Use:   "db-mcp",
	Short: "A read-only MCP server for database access",
	Long: `db-mcp is a Model Context Protocol server that provides read-only access to databases.

Supported databases: MySQL, PostgreSQL, SQLite, SQL Server, TiDB, GaussDB, ClickHouse

Supported transports: stdio, http

Only read operations are allowed: SELECT, SHOW, DESCRIBE, EXPLAIN, USE`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		logger = slog.New(handler)
		slog.SetDefault(logger)

		slog.Info("Starting MCP Database Server", "config", cfg.String())

		db, err := database.Connect(cfg)
		if err != nil {
			slog.Error("Failed to connect to database", "error", err)
			return err
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			slog.Error("Failed to ping database", "error", err)
			return err
		}

		slog.Info("Successfully connected to database")

		mcpServer := server.NewMCPServer(cfg, db.GormDB())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan
			slog.Info("Received shutdown signal")
			cancel()
		}()

		var runErr error
		switch cfg.Transport {
		case "stdio":
			runErr = mcpServer.StartStdio()
		case "http":
			runErr = mcpServer.StartHTTP(ctx)
		default:
			runErr = mcpServer.StartStdio()
		}

		if runErr != nil {
			slog.Error("Server error", "error", runErr)
			return runErr
		}

		slog.Info("Server shutdown complete")
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.Driver, "driver", "", "Database driver: mysql, postgres, sqlite, sqlserver, tidb, gaussdb, clickhouse (required)")
	rootCmd.Flags().StringVar(&cfg.DSN, "dsn", "", "Database connection string (required)")
	rootCmd.Flags().IntVar(&cfg.MaxResults, "max-results", 1000, "Maximum number of rows to return")
	rootCmd.Flags().IntVar(&cfg.Timeout, "timeout", 30, "Query timeout in seconds")
	rootCmd.Flags().StringVar(&cfg.Transport, "transport", "stdio", "Transport type: stdio, http")
	rootCmd.Flags().IntVar(&cfg.HTTPPort, "http-port", 8080, "HTTP server port (for http transport)")
	rootCmd.Flags().StringVar(&cfg.ServerName, "name", "db-mcp", "MCP server name")
	rootCmd.Flags().StringVar(&cfg.ServerVersion, "version", "1.0.0", "MCP server version")
	rootCmd.Flags().StringVar(&cfg.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
