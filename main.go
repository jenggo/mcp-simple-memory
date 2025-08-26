package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	trueString = "true"
)

// Memory represents a single memory entry in the database.
type Memory struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Tags      string    `json:"tags"`
	Status    string    `json:"status"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SimpleMemoryServer manages SQLite3 DB and logging for memory operations.
type SimpleMemoryServer struct {
	db             *sql.DB
	logger         *log.Logger
	disableLogging bool
}

// NewSimpleMemoryServer creates a new SimpleMemoryServer with rolling log and SQLite3 DB.
func NewSimpleMemoryServer(dbPath string) (*SimpleMemoryServer, error) {
	disable := strings.ToLower(os.Getenv("DISABLE_SIMPLE_MEMORY_LOGGING")) == trueString
	lj := &lumberjack.Logger{
		Filename:   "/tmp/mcp-simple-memory-server.log",
		MaxSize:    10,
		MaxBackups: 2,
		MaxAge:     7,
		Compress:   false,
	}
	logger := log.New(lj, "", log.LstdFlags|log.Lmicroseconds)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite3 db: %w", err)
	}
	// Set WAL mode for better concurrency
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	// Create schema if not exists
	schema := `
	CREATE TABLE IF NOT EXISTS simple_memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		tags TEXT,
		status TEXT,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Ensure new columns exist (for migrations)
	columns := map[string]string{
		"title":  "ALTER TABLE simple_memories ADD COLUMN title TEXT;",
		"tags":   "ALTER TABLE simple_memories ADD COLUMN tags TEXT;",
		"status": "ALTER TABLE simple_memories ADD COLUMN status TEXT;",
	}
	for col, stmt := range columns {
		var found bool
		rows, err := db.Query("PRAGMA table_info(simple_memories);")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cid int
				var name, ctype string
				var notnull, pk int
				var dfltValue sql.NullString
				if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
					if name == col {
						found = true
						break
					}
				}
			}
			if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("failed to check columns: %w", err)
			}
		}
		if !found {
			_, _ = db.Exec(stmt)
		}
	}

	return &SimpleMemoryServer{
		db:             db,
		logger:         logger,
		disableLogging: disable,
	}, nil
}

// --- MCP Tool Handlers ---

// SimpleMemoryAdd inserts a new memory into the database.
func (s *SimpleMemoryServer) SimpleMemoryAdd(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := req.GetString("title", "")
	tags := req.GetString("tags", "")
	status := req.GetString("status", "")
	memory, err := req.RequireString("memory")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid params: %v", err)), nil
	}
	content := strings.TrimSpace(memory)
	if content == "" {
		return mcp.NewToolResultError("memory cannot be empty"), nil
	}
	_, err = s.db.Exec(
		"INSERT INTO simple_memories (title, tags, status, content) VALUES (?, ?, ?, ?)",
		strings.TrimSpace(title), strings.TrimSpace(tags), strings.TrimSpace(status), content,
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add memory: %v", err)), nil
	}
	if !s.disableLogging {
		s.logger.Printf("[INFO] Added simple-memory: title=%q tags=%q status=%q content=%q", title, tags, status, content)
	}
	return mcp.NewToolResultText("Simple-memory added."), nil
}

// SimpleMemoryList returns all simple-memories, one per line.
func (s *SimpleMemoryServer) SimpleMemoryList(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rows, err := s.db.Query("SELECT id, title, tags, status, content, created_at FROM simple_memories ORDER BY id ASC")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read simple-memories: %v", err)), nil
	}
	defer rows.Close()
	var memories []string
	for rows.Next() {
		var (
			id        int64
			title     sql.NullString
			tags      sql.NullString
			status    sql.NullString
			content   string
			createdAt string
		)
		if err := rows.Scan(&id, &title, &tags, &status, &content, &createdAt); err == nil && strings.TrimSpace(content) != "" {
			mem := fmt.Sprintf(
				`{"id":%d,"title":%q,"tags":%q,"status":%q,"content":%q,"created_at":%q}`,
				id,
				title.String,
				tags.String,
				status.String,
				content,
				createdAt,
			)
			memories = append(memories, mem)
		}
	}
	if err := rows.Err(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read simple-memories: %v", err)), nil
	}
	if len(memories) == 0 {
		return mcp.NewToolResultText(""), nil
	}
	return mcp.NewToolResultText(strings.Join(memories, "\n")), nil
}

// SimpleMemorySearch returns simple-memories matching query in title, tags, status, or content.
func (s *SimpleMemoryServer) SimpleMemorySearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	queryParam, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid params: %v", err)), nil
	}
	query := strings.TrimSpace(queryParam)
	if query == "" {
		return mcp.NewToolResultError("query cannot be empty"), nil
	}
	sqlQuery := `
		SELECT id, title, tags, status, content, created_at
		FROM simple_memories
		WHERE title LIKE ? OR tags LIKE ? OR status LIKE ? OR content LIKE ?
		ORDER BY id ASC
	`
	rows, err := s.db.Query(sqlQuery, "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search simple-memories: %v", err)), nil
	}
	defer rows.Close()
	var matches []string
	for rows.Next() {
		var (
			id        int64
			title     sql.NullString
			tags      sql.NullString
			status    sql.NullString
			content   string
			createdAt string
		)
		if err := rows.Scan(&id, &title, &tags, &status, &content, &createdAt); err == nil && strings.TrimSpace(content) != "" {
			mem := fmt.Sprintf(
				`{"id":%d,"title":%q,"tags":%q,"status":%q,"content":%q,"created_at":%q}`,
				id,
				title.String,
				tags.String,
				status.String,
				content,
				createdAt,
			)
			matches = append(matches, mem)
		}
	}
	if err := rows.Err(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search simple-memories: %v", err)), nil
	}
	if len(matches) == 0 {
		return mcp.NewToolResultText("No matching simple-memories found."), nil
	}
	return mcp.NewToolResultText(strings.Join(matches, "\n")), nil
}

// SimpleMemoryDelete deletes all simple-memories containing the query substring.
func (s *SimpleMemoryServer) SimpleMemoryDelete(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	queryParam, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid params: %v", err)), nil
	}
	query := strings.TrimSpace(queryParam)
	if query == "" {
		return mcp.NewToolResultError("query cannot be empty"), nil
	}
	sqlQuery := `
		DELETE FROM simple_memories
		WHERE title LIKE ? OR tags LIKE ? OR status LIKE ? OR content LIKE ?
	`
	res, err := s.db.Exec(sqlQuery, "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete simple-memories: %v", err)), nil
	}
	n, _ := res.RowsAffected()
	if !s.disableLogging {
		s.logger.Printf("[INFO] Deleted %d simple-memories matching %q in any field", n, query)
	}
	if n == 0 {
		return mcp.NewToolResultText("No simple-memories deleted (no match)."), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Deleted %d simple-memories.", n)), nil
}

func main() {
	// Store DB in $HOME/simple_memories.db by default
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get $HOME: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(homeDir, "simple_memories.db")
	if envPath := os.Getenv("SIMPLE_MEMORY_DB_PATH"); envPath != "" {
		dbPath = envPath
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create simple-memory DB directory: %v\n", err)
		os.Exit(1)
	}

	simpleMemServer, err := NewSimpleMemoryServer(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start simple-memory server: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"simple-memory-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	// Register tools
	s.AddTool(
		mcp.NewTool(
			"simple_memory_add",
			mcp.WithDescription("Append a memory string to the simple-memory database."),
			mcp.WithString("memory", mcp.Required(), mcp.Description("The memory to add (string).")),
			mcp.WithString("title", mcp.Description("Optional title for the memory.")),
			mcp.WithString("tags", mcp.Description("Optional tags for the memory (comma-separated).")),
			mcp.WithString("status", mcp.Description("Optional status for the memory (e.g., completed, issue, etc.).")),
		),
		simpleMemServer.SimpleMemoryAdd,
	)
	s.AddTool(
		mcp.NewTool(
			"simple_memory_list",
			mcp.WithDescription("List all simple-memories (one per line, as JSON)."),
		),
		simpleMemServer.SimpleMemoryList,
	)
	s.AddTool(
		mcp.NewTool(
			"simple_memory_search",
			mcp.WithDescription("Search for simple-memories by substring in title, tags, status, or content."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Substring to search for in title, tags, status, or content.")),
		),
		simpleMemServer.SimpleMemorySearch,
	)
	s.AddTool(
		mcp.NewTool(
			"simple_memory_delete",
			mcp.WithDescription("Delete all simple-memories matching the query substring in title, tags, status, or content."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Substring to match for deletion in title, tags, status, or content.")),
		),
		simpleMemServer.SimpleMemoryDelete,
	)

	// Transport selection: stdio, SSE, or HTTP
	const defaultPort = "3002"
	sseEnable := strings.ToLower(os.Getenv("MCP_USE_SSE")) == trueString
	httpEnable := strings.ToLower(os.Getenv("MCP_USE_HTTP")) == trueString

	switch {
	case sseEnable:
		port := os.Getenv("PORT")
		if port == "" {
			port = defaultPort
		}
		addr := ":" + port
		log.Printf("MCP simple-memory server running in SSE mode on %s\n", addr)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(addr); err != nil {
			log.Fatalf("Fatal error running SSE server: %v\n", err)
		}
	case httpEnable:
		port := os.Getenv("PORT")
		if port == "" {
			port = defaultPort
		}
		addr := ":" + port
		log.Printf("MCP simple-memory server running in HTTP mode on %s\n", addr)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(addr); err != nil {
			log.Fatalf("Fatal error running HTTP server: %v\n", err)
		}
	default:
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error running stdio server: %v\n", err)
			os.Exit(1)
		}
	}
}
