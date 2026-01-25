# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Linux-optimized **MCP (Model Context Protocol) server** providing file operations, shell execution, and todo management tools for Alpine, Debian, and Ubuntu.

## Common Commands

```bash
# Current platform build
go build -o mcp-server .

# With build info
go build -ldflags "-X main.Version=dev" -o mcp-server .

# Run all tests
go test ./...

# Run specific test
go test -v tools/bash_test.go tools/bash.go

# Test with coverage
go test -cover ./...

# Verify dependencies
go mod tidy && go mod verify

# Static analysis
go vet ./...

# Release build with static linking
./build-release.sh
VERSION=1.0.0 ./build-release.sh

# Run server
./mcp-server
./mcp-server --version
./mcp-server --help
```

## Code Architecture

### Structure
```
main.go              # MCP server entry point, graceful shutdown handling
tools/
├── bash.go          # Shell execution (bash, bash_output, kill_shell)
├── file.go          # File I/O (read_file, write_file, edit_file, glob, grep)
├── todo.go          # Todo list management (todo_write)
└── *_test.go        # Comprehensive test suite
go.mod               # Go module (mcp-file-tools, Go 1.23+)
```

### Core Components

**Server** (`main.go`)
- Initializes MCP server with stdio transport
- Registers tools: `AddFileTools`, `AddBashTools`, `AddTodoTools`
- Graceful shutdown via SIGINT/SIGTERM with resource cleanup

**File Tools** (`tools/file.go:125-156`)
- `read_file` - Read with line range/offset/limit, absolute paths required
- `write_file` - Write with atomic file writes, auto directory creation
- `edit_file` - String replacement (first or all occurrences)
- `glob` - Pattern matching with doublestar/v4 (`*`, `**`)
- `grep` - Regex/text search with context, ReDoS protection

**Bash Tools** (`tools/bash.go:238-260`)
- `bash` - Execute commands with timeout (max 600s), sync/async modes
- `bash_output` - Poll background process output with regex filter
- `kill_shell` - Graceful termination (SIGINT → SIGKILL)

**Todo Tools** (`tools/todo.go:44-51`)
- `todo_write` - Task list management with status tracking

### Key Implementation Patterns

**Tool Registration** (consistent across all tools):
```go
func AddXxxTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{Name: "...", Description: "..."}, handler)
}
```

**Handler Signature**:
```go
func handler(ctx context.Context, req *mcp.CallToolRequest, params XxxParams) 
    (*mcp.CallToolResult, XxxResult, error)
```

**Background Process Management**:
- `LimitedBuffer` (100KB default) prevents memory leaks in long-running processes
- Periodic cleanup every 30s removes completed processes
- Thread-safe with sync.Mutex and atomic.Value for status

**Security**:
- `isPathInSafeZone()` - Blocks access to system directories (`/proc`, `/sys`, `/etc`, etc.) and critical files
- `isRegexComplexitySafe()` - Prevents ReDoS attacks via nested quantifier detection
- `atomicWriteFile()` - Uses temp file + rename for atomic writes
- Absolute paths required for all file operations

### Dependencies

- `github.com/modelcontextprotocol/go-sdk v1.1.0` - MCP protocol
- `github.com/bmatcuk/doublestar/v4 v4.9.1` - Glob patterns
- `github.com/stretchr/testify v1.11.1` - Testing framework

## Development Workflow

### Adding a New Tool

1. Create `tools/your_tool.go`
2. Define `XxxParams`, `XxxResult` structs with JSON/jsonschema tags
3. Implement handler function with parameter validation
4. Register: `mcp.AddTool(server, &mcp.Tool{...}, handler)`
5. Call registration in `main.go:57-63`
6. Add tests in `tools/your_tool_test.go`

### MCP Protocol Flow

```
JSON input → Validate params → Execute operation → JSON output (via stdin/stdout)
```

## Platform Support

- **Primary**: Linux (Alpine, Debian, Ubuntu)
- **Static linking**: Enabled with musl-gcc (`CGO_ENABLED=1`)
- **Build output**: `dist/mcp-file-tools-linux-{amd64|arm64}` (via build-release.sh)

## Test Organization

**Core tests**: `bash_test.go`, `file_test.go`, `todo_test.go`
**Utility tests**: `read_file_test.go`, `write_file_test.go`, `edit_file_test.go`, `glob_test.go`, `grep_test.go`, `resolve_path_test.go`, `get_search_path_test.go`, `parse_range_test.go`, `parse_line_number_test.go`, `search_lines_test.go`, `truncate_by_tokens_test.go`

All use `github.com/stretchr/testify/assert`.
