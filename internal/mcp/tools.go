package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zanguish/db-mcp/internal/database"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Tools struct {
	db *database.ReadOnlyDB
}

func NewTools(db *database.ReadOnlyDB) *Tools {
	return &Tools{
		db: db,
	}
}

func toJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

func (t *Tools) RegisterTools(srv *server.MCPServer) {
	listTablesTool := mcp.NewTool(
		"list_tables",
		mcp.WithDescription("List all tables in the database"),
	)

	describeTableTool := mcp.NewTool(
		"describe_table",
		mcp.WithDescription("Get the structure of a specific table"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("Name of the table to describe"),
		),
	)

	executeQueryTool := mcp.NewTool(
		"execute_query",
		mcp.WithDescription("Execute a read-only SQL query (SELECT, SHOW, DESCRIBE, EXPLAIN, USE)"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("SQL query to execute"),
		),
	)

	getTableSampleTool := mcp.NewTool(
		"get_table_sample",
		mcp.WithDescription("Get sample rows from a table"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("Name of the table to sample from"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of rows to return (default: 100)"),
		),
	)

	getTableCountTool := mcp.NewTool(
		"get_table_count",
		mcp.WithDescription("Get the total row count of a table"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("Name of the table to count"),
		),
	)

	srv.AddTool(listTablesTool, t.handleListTables)
	srv.AddTool(describeTableTool, t.handleDescribeTable)
	srv.AddTool(executeQueryTool, t.handleExecuteQuery)
	srv.AddTool(getTableSampleTool, t.handleGetTableSample)
	srv.AddTool(getTableCountTool, t.handleGetTableCount)
}

func (t *Tools) handleListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tables, err := t.db.ListTables()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list tables: %v", err)), nil
	}

	var result struct {
		Tables []string `json:"tables"`
		Count  int      `json:"count"`
	}
	result.Tables = tables
	result.Count = len(tables)

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (t *Tools) handleDescribeTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError("table_name is required"), nil
	}

	columns, err := t.db.DescribeTable(tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to describe table: %v", err)), nil
	}

	var result struct {
		Table   string                 `json:"table"`
		Columns []database.TableColumn `json:"columns"`
		Count   int                    `json:"column_count"`
	}
	result.Table = tableName
	result.Columns = columns
	result.Count = len(columns)

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (t *Tools) handleExecuteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	query = strings.TrimSpace(query)

	results, err := t.db.ExecuteQuery(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}

	var result struct {
		Query    string                   `json:"query"`
		RowCount int                      `json:"row_count"`
		Rows     []map[string]interface{} `json:"rows"`
	}
	result.Query = query
	result.RowCount = len(results)
	result.Rows = results

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (t *Tools) handleGetTableSample(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError("table_name is required"), nil
	}

	limit := 100
	if l, ok := request.GetArguments()["limit"].(float64); ok {
		limit = int(l)
	}

	results, err := t.db.GetTableSample(tableName, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get table sample: %v", err)), nil
	}

	var result struct {
		Table    string                   `json:"table"`
		Limit    int                      `json:"limit"`
		RowCount int                      `json:"row_count"`
		Rows     []map[string]interface{} `json:"rows"`
	}
	result.Table = tableName
	result.Limit = limit
	result.RowCount = len(results)
	result.Rows = results

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (t *Tools) handleGetTableCount(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError("table_name is required"), nil
	}

	count, err := t.db.GetTableCount(tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to count table: %v", err)), nil
	}

	var result struct {
		Table string `json:"table"`
		Count int64  `json:"count"`
	}
	result.Table = tableName
	result.Count = count

	return mcp.NewToolResultText(toJSON(result)), nil
}
