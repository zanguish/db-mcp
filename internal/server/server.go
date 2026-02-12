package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/zanguish/db-mcp/internal/config"
	"github.com/zanguish/db-mcp/internal/database"
	"github.com/zanguish/db-mcp/internal/mcp"

	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"
)

type MCPServer struct {
	cfg        *config.Config
	readonlyDB *database.ReadOnlyDB
	mcpServer  *server.MCPServer
}

func NewMCPServer(cfg *config.Config, gormDB *gorm.DB) *MCPServer {
	readonlyDB := database.NewReadOnlyDB(gormDB, cfg)

	mcpServer := server.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
	)

	tools := mcp.NewTools(readonlyDB)
	tools.RegisterTools(mcpServer)

	return &MCPServer{
		cfg:        cfg,
		readonlyDB: readonlyDB,
		mcpServer:  mcpServer,
	}
}

func (s *MCPServer) StartStdio() error {
	slog.Info("Starting MCP Server in stdio mode", "name", s.cfg.ServerName, "version", s.cfg.ServerVersion)
	return server.ServeStdio(s.mcpServer)
}

func (s *MCPServer) StartHTTP(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.cfg.HTTPPort)
	slog.Info("Starting MCP Server in HTTP mode", "address", addr, "name", s.cfg.ServerName, "version", s.cfg.ServerVersion)

	httpServer := server.NewStreamableHTTPServer(s.mcpServer)

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down HTTP server...")
		httpServer.Shutdown(ctx)
	}()

	return httpServer.Start(addr)
}

func (s *MCPServer) GetServer() *server.MCPServer {
	return s.mcpServer
}

func (s *MCPServer) GetReadOnlyDB() *database.ReadOnlyDB {
	return s.readonlyDB
}
