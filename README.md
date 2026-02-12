# Database MCP Server

[![Go Version](https://img.shields.io/badge/Go-1.25.7-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

一个只读的 Model Context Protocol (MCP) 服务器，用于安全地访问数据库。

## 功能特性

- **多数据库支持**: MySQL, PostgreSQL, SQLite, SQL Server, TiDB, GaussDB, ClickHouse
- **只读访问**: 只允许 SELECT、SHOW、DESCRIBE、EXPLAIN、USE 操作
- **双传输模式**: stdio (本地) 和 Streamable HTTP (远程)
- **SQL 安全防护**: 注入防护、多语句阻止、危险函数检测
- **查询限制**: 结果集限制、超时保护

## 快速开始

### 从源码构建

```bash
make
```

### 使用

```bash
# Stdio 模式 (默认)
db-mcp --driver=mysql --dsn="user:password@tcp(host:3306)/database?parseTime=true"

# HTTP 模式
db-mcp --driver=postgres --dsn="host=localhost port=5432 user=user password=pass dbname=test sslmode=disable" --transport=http --http-port=8080
```

## 命令行参数

| 参数 | 说明 | 默认值 |
|-----|------|-------|
| `--driver` | 数据库驱动 (mysql/postgres/sqlite/sqlserver/tidb/gaussdb/clickhouse) | 必需 |
| `--dsn` | 数据库连接字符串 | 必需 |
| `--max-results` | 最大返回行数 | 1000 |
| `--timeout` | 查询超时秒数 | 30 |
| `--transport` | 传输方式 (stdio/http) | stdio |
| `--http-port` | HTTP 服务端口 | 8080 |
| `--name` | MCP 服务器名称 | db-mcp |
| `--version` | 服务器版本 | 1.0.0 |
| `--log-level` | 日志级别 (debug/info/warn/error) | info |

## 支持的数据库

| 数据库 | DSN 示例 |
|-------|---------|
| MySQL | `user:pass@tcp(host:3306)/db?parseTime=true` |
| TiDB | `user:pass@tcp(host:4000)/db?parseTime=true` |
| PostgreSQL | `host=localhost port=5432 user=user dbname=test sslmode=disable` |
| GaussDB | `host=localhost port=5432 user=user dbname=test sslmode=disable` |
| SQLite | `file:/path/to/database.db` 或 `:memory:` |
| SQL Server | `sqlserver://host:1433;database=db;user=user;password=pass;encrypt=disable` |
| ClickHouse | `tcp://host:9000` |

## MCP 工具

### 1. list_tables

列出数据库中的所有表。

```json
{
  "tool": "list_tables",
  "arguments": {}
}
```

### 2. describe_table

获取指定表的结构信息。

```json
{
  "tool": "describe_table",
  "arguments": {
    "table_name": "users"
  }
}
```

### 3. execute_query

执行只读 SQL 查询。

```json
{
  "tool": "execute_query",
  "arguments": {
    "query": "SELECT * FROM users WHERE active = 1 LIMIT 10"
  }
}
```

### 4. get_table_sample

获取表的示例数据。

```json
{
  "tool": "get_table_sample",
  "arguments": {
    "table_name": "users",
    "limit": 5
  }
}
```

### 5. get_table_count

统计表的行数。

```json
{
  "tool": "get_table_count",
  "arguments": {
    "table_name": "users"
  }
}
```

## IDE 配置

### Claude Desktop

```json
{
  "mcpServers": {
    "db-mcp": {
      "command": "/path/to/db-mcp",
      "args": [
        "--driver=mysql",
        "--dsn=root:password@tcp(localhost:3306)/testdb?parseTime=true"
      ]
    }
  }
}
```

### Cursor IDE

```json
{
  "mcpServers": {
    "db-mcp": {
      "command": "/path/to/db-mcp",
      "args": [
        "--driver=postgres",
        "--dsn=host=localhost port=5432 user=postgres dbname=test"
      ]
    }
  }
}
```

## HTTP 传输模式

使用 Streamable HTTP 模式可通过 HTTP 访问 MCP 服务：

```bash
db-mcp --driver=postgres --dsn="host=localhost port=5432 user=postgres dbname=test" --transport=http --http-port=8080
```

客户端访问端点: `http://localhost:8080/mcp`

## 安全性

### 只读访问限制

只允许以下 SQL 操作：
- `SELECT` - 查询数据
- `SHOW` - 显示数据库信息
- `DESCRIBE` / `EXPLAIN` - 查看表结构和查询计划
- `USE` - 切换数据库

### 自动阻止的操作

| 类型 | 关键词 |
|-----|-------|
| 数据修改 | INSERT, UPDATE, DELETE, REPLACE, MERGE |
| 表结构 | DROP, TRUNCATE, ALTER, CREATE, RENAME |
| 权限控制 | GRANT, REVOKE |
| 系统操作 | EXEC, EXECUTE, CALL, LOAD |
| 文件操作 | INTO OUTFILE, INTO DUMPFILE |
| 事务控制 | BEGIN, COMMIT, ROLLBACK, SAVEPOINT |
| 危险函数 | SLEEP, BENCHMARK, LOAD_FILE |

### SQL 注入防护

- 自动移除 SQL 注释 (`--` 和 `/* */`)
- 阻止多语句执行 (`;`)
- 检测可疑模式

### 查询限制

- 自动添加 `LIMIT` 限制结果集大小
- 可配置的查询超时

## 项目结构

```
db-mcp/
├── cmd/
│   └── main.go                    # 应用入口
├── internal/
│   ├── config/
│   │   └── config.go              # 配置管理
│   ├── database/
│   │   ├── connection.go          # 数据库连接
│   │   ├── readonly.go            # 只读查询封装
│   │   └── sql_validator.go      # SQL 验证器
│   ├── mcp/
│   │   └── tools.go               # MCP 工具定义
│   └── server/
│       └── server.go              # MCP 服务器封装
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 依赖

- Go 1.25.7+
- github.com/mark3labs/mcp-go v0.43.2
- gorm.io/gorm v1.31.1
- gorm.io/driver/mysql v1.6.0
- gorm.io/driver/postgres v1.5.4
- gorm.io/driver/sqlite v1.6.0
- gorm.io/driver/sqlserver v1.6.0
- gorm.io/driver/clickhouse v0.7.0
- github.com/spf13/cobra v1.9.1

## License

MIT
