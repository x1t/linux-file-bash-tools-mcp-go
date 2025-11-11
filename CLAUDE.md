# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview 📋

This is a **MCP (Model Context Protocol) server** providing file operations and system command tools, optimized for Linux distributions (Alpine, Debian, Ubuntu). It implements various tools including file I/O, shell command execution, and todo management through the MCP protocol.

## Common Commands 🔧

### Building the Project
```bash
# Multi-platform build (PowerShell on Windows)
pwsh.exe -File build.ps1

# Manual build for current platform
go build -o mcp-server .

# With build information
go build -ldflags "-X main.Version=dev" -o mcp-server .
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific test file
go test -v tools/bash_test.go tools/bash.go

# Run tests with coverage
go test -cover ./...

# Functional testing script
./test_functions.sh
```

### Development Tasks
```bash
# Install dependencies
go mod tidy

# Verify dependencies
go mod verify

# Format code
gofmt -l .

# Static analysis
go vet ./...

# Check build
go build -o /tmp/test-build .
```

### Running the Server
```bash
# Standard mode (recommended for MCP clients)
./mcp-server

# With version info
./mcp-server --version

# With help
./mcp-server --help
```

### Release Build
```bash
# Create production build with static linking (Linux)
./build-release.sh

# Or set VERSION and build
VERSION=1.0.0 ./build-release.sh
```

## Code Architecture 🏗️

### High-Level Structure
```
main.go                # MCP server entry point
├── tools/
│   ├── tools.go       # Module registration (currently minimal)
│   ├── bash.go        # Shell execution tools (bash, bash_output, kill_shell)
│   ├── file.go        # File operations (read, write, edit, glob, grep)
│   ├── todo.go        # Todo list management
│   └── *_test.go      # Test files (comprehensive test coverage)
go.mod                 # Go module and dependencies
README.md              # User-facing documentation
AGENTS.md              # Contributor guidelines
todo.md                # TypeScript type definitions
```

### Core Components

**1. Main Server** (`main.go:1-68`)
- Initializes MCP server instance
- Registers all tool modules
- Uses `github.com/modelcontextprotocol/go-sdk/mcp` for protocol handling
- Supports stdio transport
- Version: "dev" (default) with build-time injection

**2. File Tools** (`tools/file.go:1-100+`)
- `read_file` - Read files with optional line ranges, offset, and limit
- `write_file` - Write/overwrite files with automatic directory creation
- `edit_file` - Replace file content by string matching
- `glob` - File pattern matching with doublestar/v4 (supports `*` and `**` wildcards)
- `grep` - Text search with regex, case sensitivity, and context options
- **Key Feature**: All file operations validate absolute paths for security

**3. Bash Tools** (`tools/bash.go:1-100+`)
- `bash` - Execute shell commands (sync/async with configurable timeout)
- `bash_output` - Get output from background processes with optional regex filtering
- `kill_shell` - Terminate background processes (SIGINT/SIGKILL)
- **Background Process Management**: Uses `LimitedBuffer` to prevent memory leaks (100KB default)
- **Process Tracking**: Maintains active processes with cleanup on completion

**4. Todo Tools** (`tools/todo.go:1-100`)
- `todo_write` - Create and manage task lists with status tracking
- Supports status: pending, in_progress, completed
- Includes activeForm field for dynamic status display

### Dependencies

**Primary**:
- `github.com/modelcontextprotocol/go-sdk v1.1.0` - MCP protocol implementation
- `github.com/bmatcuk/doublestar/v4 v4.9.1` - Advanced glob pattern matching

**Testing & Development**:
- `github.com/stretchr/testify v1.11.1` - Testing framework
- `gopkg.in/yaml.v3 v3.0.1` - YAML parsing

**Transitive**:
- `github.com/google/jsonschema-go v0.3.0` - JSON schema generation
- `golang.org/x/oauth2 v0.30.0` - OAuth authentication support

**Go Version**: Requires Go 1.23.0+ (as specified in go.mod)

## Key Implementation Details 🔍

### Tool Registration Pattern
Tools follow MCP SDK pattern with parameter validation and structured responses:
```go
// Example from tools/bash.go:28-65
type BashParams struct {
    Command         string      `json:"command" jsonschema:"..."`
    Description     string      `json:"description,omitempty" jsonschema:"..."`
    Timeout         int         `json:"timeout,omitempty" jsonschema:"..."`
    RunInBackground bool        `json:"run_in_background,omitempty" jsonschema:"..."`
}
```

### Error Handling Strategy
- All tools use structured JSON error responses
- File operations validate paths, permissions, and UTF-8 encoding
- Shell commands respect timeout settings (default 30s, max 600s)
- Background process cleanup is automatic with `Done` channel pattern

### Background Process Management
**LimitedBuffer** (tools/bash.go:20-73)
- Prevents unbounded memory growth in long-running processes
- Default 100KB limit with automatic old data eviction
- Thread-safe with sync.Mutex
- Used by `ProcessInfo` for stdout/stderr collection

**Process Tracking** (tools/bash.go:100+)
- Global process registry with cleanup
- Non-blocking status checks via `Done` channel
- Graceful termination (SIGINT → SIGKILL)

### Build System
- `build.ps1` - PowerShell script for multi-platform builds (Windows x64, Linux AMD64, Linux ARM64)
- `build-release.sh` - Bash script for static-linked production builds
- Build outputs to `dist/` directory
- **Static linking**: Enabled for Linux builds with musl-gcc (`CGO_ENABLED=1`)
- **Version injection**: Build-time ldflags for Version, BuildTime, GitCommit

### Testing Strategy
- **Unit tests**: Per-module test files (`*_test.go`)
- **Coverage**: Comprehensive testing with 15+ test files
- **Functional tests**: Shell script integration tests
- **Test Files**:
  - `bash_test.go` - Shell execution tests
  - `file_test.go` - File I/O tests
  - `todo_test.go` - Task management tests
  - `*_test.go` - 10+ additional specialized test files

### Web Tools Status
⚠️ **Note**: Web tools (web_search, web_fetch) mentioned in some documentation have been **removed**. Current implementation focuses on file operations, shell execution, and todo management only.

## Development Workflow 👨‍💻

### Adding New Tools
1. Create new file in `tools/` directory
2. Define input struct with JSON tags and jsonschema annotations
3. Implement handler function with parameter validation
4. Register in main.go `tools.AddXxxTools(server)` pattern
5. Add tests in `*_test.go`
6. Update `todo.md` with TypeScript interface definitions

### MCP Integration
The server communicates via stdin/stdout using the MCP protocol. Each tool call:
1. Receives JSON input from MCP client
2. Validates parameters (types, required fields, constraints)
3. Executes operation with proper error handling
4. Returns structured JSON response with success/error

### Platform Support
- **Primary**: Linux (Alpine, Debian, Ubuntu) - fully optimized
- **Build targets**: Windows x64, Linux AMD64, Linux ARM64
- **Static linking**: Enabled for Linux builds to avoid dependency issues
- **Go 1.23+**: Required for proper module support

## Security Considerations ⚠️

1. **Path Validation**: All file paths must be absolute to prevent directory traversal
2. **Shell Commands**: The `bash` tool can execute arbitrary shell commands - use only with trusted clients
3. **Timeout Protection**: Default 30s timeout, max 600s (10 minutes)
4. **Background Process Cleanup**: Automatic cleanup prevents resource leaks
5. **Environment Variables**: Avoid exposing sensitive data (tokens, keys, passwords)
6. **UTF-8 Validation**: File operations validate UTF-8 encoding

## Build Outputs 📦

After running `build.ps1`, the following files are generated in `dist/`:
- `mcp-server.exe` (~5.5 MB) - Windows x64 PE32+
- `mcp-server-linux` (~5.3 MB) - Linux AMD64
- `mcp-server-linux-arm64` (~5.1 MB) - Linux ARM64 aarch64
- `mcp-server-current` - Current platform build (symlink or copy)

## Documentation Files

- **README.md** - User-facing documentation with usage examples
- **AGENTS.md** - Contributor guidelines and development standards
- **todo.md** - TypeScript type definitions for all tool inputs/outputs
- **CLAUDE.md** - This file, guidance for Claude Code

## Common Issues & Solutions 🔧

**Build fails**:
- Ensure Go 1.23+ is installed (`go version`)
- Run `go mod tidy` to sync dependencies
- Check `go.mod` and verify module name: `mcp-file-tools`

**Tests timeout**:
- Check for hanging background processes in bash tests
- Look for test files with infinite loops or missing cleanup
- Run tests with verbose output: `go test -v -timeout 30s ./...`

**Permission errors**:
- Verify file permissions and user access rights
- File operations require absolute paths
- Check directory write permissions for `write_file`

**Cross-compilation issues**:
- Use PowerShell build script (`build.ps1`) for multi-platform builds
- For manual builds, use GOOS/GOARCH environment variables
- Ensure `CGO_ENABLED=1` for static linking

**"web" tools not found**:
- Web tools have been removed from the codebase
- Current tools: bash, file, todo
- Use shell commands for web interactions instead

## Test File Organization 📝

**Core Test Files**:
- `tools/bash_test.go` - Shell execution and process management
- `tools/file_test.go` - File I/O operations (read/write/edit/glob/grep)
- `tools/todo_test.go` - Task list management

**Utility Test Files**:
- `tools/read_file_test.go`, `tools/write_file_test.go`, `tools/edit_file_test.go`
- `tools/glob_test.go`, `tools/grep_test.go`
- `tools/resolve_path_test.go`, `tools/get_search_path_test.go`
- `tools/parse_range_test.go`, `tools/parse_line_number_test.go`
- `tools/search_lines_test.go`, `tools/truncate_by_tokens_test.go`

All tests use `github.com/stretchr/testify/assert` and maintain >80% coverage.
