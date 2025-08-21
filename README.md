# MCP Simple-Memory Server

A Model Context Protocol (MCP) server implementation in Go that provides persistent memory functionality for AI agents. This server enables agents to store, retrieve, search, and manage memories across sessions, making them more intelligent and context-aware.

## Overview

The MCP Simple-Memory Server is built using the [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) library and provides a SQLite-based persistent memory system. It's designed to help AI agents maintain context, learn from interactions, and provide personalized experiences.

## Features

- **Persistent Simple-Memory Storage**: SQLite database with WAL mode for optimal concurrency
- **Full-Text Search**: Find memories using substring matching
- **Simple-Memory Management**: Add, list, search, and delete operations
- **Multiple Transport Options**: Support for stdio, HTTP, and SSE transports
- **Logging**: Configurable rolling log files for debugging and monitoring
- **Cross-Session Persistence**: Memories survive server restarts and session changes

## Installation

### Prerequisites

- Go 1.24.4 or later
- SQLite3 (included via CGO)

### Build from Source

```bash
git clone <repository-url>
cd mcp-simple-memory
go mod tidy
go build -o simple-memory-server main.go
```

## Usage

### Basic Usage (stdio transport)

```bash
./simple-memory-server
```

### HTTP Transport

```bash
MCP_USE_HTTP=true PORT=3002 ./simple-memory-server
```

### SSE Transport

```bash
MCP_USE_SSE=true PORT=3002 ./simple-memory-server
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SIMPLE_MEMORY_DB_PATH` | Path to SQLite database file | `$HOME/simple-memories.db` |
| `DISABLE_SIMPLE_MEMORY_LOGGING` | Disable logging (true/false) | `false` |
| `MCP_USE_HTTP` | Enable HTTP transport | `false` |
| `MCP_USE_SSE` | Enable SSE transport | `false` |
| `PORT` | Port for HTTP/SSE transports | `3002` |

### Database Location

By default, the simple-memory database is stored at `$HOME/simple-memories.db`. You can customize this location using the `SIMPLE_MEMORY_DB_PATH` environment variable:

```bash
SIMPLE_MEMORY_DB_PATH=/path/to/custom/simple-memories.db ./simple-memory-server
```

## Available Tools

The server provides four MCP tools for simple-memory management:

### `simple_memory_add`

Add a new simple-memory to the database.

**Parameters:**
- `simple_memory` (string, required): The simple-memory content to store

**Example:**
```json
{
  "name": "simple_memory_add",
  "arguments": {
    "simple_memory": "User prefers Go with clean architecture patterns"
  }
}
```

### `simple_memory_list`

List all stored simple-memories (one per line).

**Parameters:** None

**Example:**
```json
{
  "name": "simple_memory_list",
  "arguments": {}
}
```

### `simple_memory_search`

Search for simple-memories containing a specific substring.

**Parameters:**
- `query` (string, required): Substring to search for in simple-memories

**Example:**
```json
{
  "name": "simple_memory_search",
  "arguments": {
    "query": "Go"
  }
}
```

### `simple_memory_delete`

Delete all simple-memories containing a specific substring.

**Parameters:**
- `query` (string, required): Substring to match for deletion

**Example:**
```json
{
  "name": "simple_memory_delete",
  "arguments": {
    "query": "temporary"
  }
}
```

## Testing

### Manual Testing

You can test the server manually using JSON-RPC over stdio:

```bash
# Add a simple-memory
echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "simple_memory_add", "arguments": {"simple_memory": "Test simple-memory entry"}}, "id": 1}' | ./simple-memory-server

# List simple-memories
echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "simple_memory_list", "arguments": {}}, "id": 2}' | ./simple-memory-server

# Search simple-memories
echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "simple_memory_search", "arguments": {"query": "Test"}}, "id": 3}' | ./simple-memory-server
```

### Integration with AI Clients

This server is designed to work with MCP-compatible AI clients like:
- Claude (via MCP)
- Custom agents using MCP client libraries
- MCPHost CLI tool

All memory references should be interpreted as simple-memory in this context.

## Simple-Memory Categories for Agents

The simple-memory system supports various types of information storage:

### 1. Factual Simple-Memory
- User profile information
- Project specifications
- Technical constraints
- Business requirements

### 2. Procedural Simple-Memory
- User's preferred workflows
- Code patterns and conventions
- Testing approaches
- Deployment procedures

### 3. Episodic Simple-Memory
- Previous conversations
- Past decisions and outcomes
- Problem-solving approaches
- Learning experiences

### 4. Semantic Simple-Memory
- Domain knowledge
- Best practices
- Library preferences
- Architecture patterns

## Use Cases

### Development Assistant

```
Simple-Memory Examples:
- "User prefers Chi router over Gin for Go HTTP servers"
- "Project uses PostgreSQL with GORM for data persistence"
- "User follows clean architecture with dependency injection"
- "Testing strategy: unit tests with testify, integration tests with Docker"
```

### Project Context Management

```
Simple-Memory Examples:
- "Current project: E-commerce microservice in Go"
- "Completed: User authentication service with JWT"
- "Next milestone: Implement product catalog API"
- "Issue: Need to optimize database queries for product search"
```

### Learning and Adaptation

```
Simple-Memory Examples:
- "User learning Go from Python background"
- "Confusion resolved: Go interfaces vs Python duck typing"
- "User successfully implemented channels for concurrency"
- "Prefers explicit error handling over exceptions"
```

## Why Simple-Memory is Essential for Agents

### Context Continuity
Without simple-memory, agents lose all context between sessions, making them ineffective for ongoing projects and relationships.

### Learning and Improvement
Simple-memory enables agents to learn user preferences, successful patterns, and avoid repeating mistakes.

### Personalization
Agents can provide tailored responses based on user history, preferences, and past interactions.

### Complex Task Management
Multi-step projects require persistent state tracking and progress management across sessions.

## Database Schema

The server uses a simple SQLite schema:

```sql
CREATE TABLE IF NOT EXISTS simple_memories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
```

## Logging

Logs are written to `/tmp/mcp-simple-memory-server.log` by default with the following configuration:
- Maximum file size: 10MB
- Maximum backup files: 2
- Maximum age: 7 days
- Compression: disabled

To disable logging entirely:
```bash
DISABLE_SIMPLE_MEMORY_LOGGING=true ./simple-memory-server
```

## Performance Considerations

- **SQLite WAL Mode**: Enabled for better concurrent access
- **Connection Pooling**: Handled by Go's `sql.DB`
- **Simple-Memory Efficiency**: Streaming results for large datasets
- **Index Optimization**: Automatic SQLite optimizations

## Security Considerations

- **SQL Injection**: Parameterized queries prevent SQL injection
- **File Permissions**: Database created with 0755 permissions
- **Input Validation**: All inputs are validated and sanitized
- **No Network Exposure**: stdio transport by default (HTTP/SSE optional)

## Contributing

1. Ensure Go 1.24.4+ is installed
2. Run `golangci-lint run` to check code quality
3. Add tests for new features
4. Update documentation as needed

## License

[Add your license information here]

## Related Projects

- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) - Go implementation of MCP
- [Model Context Protocol](https://modelcontextprotocol.io/) - Official MCP specification
- [MCPHost](https://github.com/mark3labs/mcphost) - CLI host for MCP servers

## Support

For issues and questions:
1. Check the logs at `/tmp/mcp-simple-memory-server.log`
2. Verify database permissions and location
3. Test with manual JSON-RPC calls
4. Check environment variable configuration