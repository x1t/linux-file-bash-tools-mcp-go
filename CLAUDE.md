# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview 📋

This is a **MCP (Model Context Protocol) server** providing file operations and system command tools, optimized for Linux distributions (Alpine, Debian, Ubuntu). It implements various tools including file I/O, shell command execution, and web utilities through the MCP protocol.

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
go test -v tools/file_test.go tools/file.go

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
│   ├── tools.go       # Module registration
│   ├── file.go        # File operations (read, write, edit, glob, grep)
│   ├── bash.go        # Shell execution tools (bash, bash_output, kill_shell)
│   ├── todo.go        # Todo list management
│   ├── web.go         # Web utilities (web_search, web_fetch)
│   └── *_test.go      # Test files
go.mod                 # Go module and dependencies
BUILD.md               # Build documentation
```

### Core Components

**1. Main Server** (`main.go:1-70`)
- Initializes MCP server instance
- Registers all tool modules
- Uses `github.com/modelcontextprotocol/go-sdk/mcp` for protocol handling
- Supports stdio and HTTP transport

**2. File Tools** (`tools/file.go:1-300`)
- `read_file` - Read files with optional line ranges
- `write_file` - Write/overwrite files with automatic directory creation
- `edit_file` - Replace file content by line ranges
- `glob` - File pattern matching with `*` and `**` wildcards
- `grep` - Text search with regex, case sensitivity, and context options

**3. Bash Tools** (`tools/bash.go:1-200`)
- `bash` - Execute shell commands (sync/async with timeout)
- `bash_output` - Get output from background processes
- `kill_shell` - Terminate background processes (SIGINT/SIGKILL)

**4. Web Tools** (`tools/web.go:1-200`)
- `web_search` - Internet search with domain filters
- `web_fetch` - URL content fetching with AI processing

**5. Todo Tools** (`tools/todo.go:1-100`)
- `todo_write` - Create and manage task lists with status tracking

### Dependencies

**Primary**:
- `github.com/modelcontextprotocol/go-sdk v1.1.0` - MCP protocol implementation

**Transitive**:
- `github.com/google/jsonschema-go` - JSON schema generation
- `golang.org/x/oauth2` - OAuth authentication support

**Go Version**: Requires Go 1.25.3+ (according to go.mod)

## Key Implementation Details 🔍

### Tool Registration Pattern
Each tool module follows a consistent pattern:
```go
// Registration function called from main()
func AddXxxTools(server *mcp.Server) {
    server.AddTool(XxxInput{}, "description", handleXxxTool)
}
```

### Error Handling
- All tools use structured error responses
- File operations validate paths and permissions
- Shell commands respect timeout settings
- Background process cleanup is automatic

### Build System
- `build.ps1` - PowerShell script for multi-platform builds (Windows x64, Linux AMD64, Linux ARM64)
- `build-release.sh` - Bash script for static-linked production builds
- Build outputs to `dist/` directory
- Static linking with musl for Linux builds (`CGO_ENABLED=1`, `CC=musl-gcc`)

### Testing Strategy
- Unit tests for each tool module (`*_test.go`)
- Functional tests in `test_functions.sh`
- Integration tests via actual MCP protocol calls

## Development Workflow 👨‍💻

### Adding New Tools
1. Create new file in `tools/` directory
2. Define input/output struct types
3. Implement handler function
4. Register in `tools/tools.go`
5. Add tests in `*_test.go`
6. Update `todo.md` with TypeScript interface definitions

### MCP Integration
The server communicates via stdin/stdout using the MCP protocol. Each tool call:
1. Receives JSON input from MCP client
2. Validates parameters
3. Executes operation with proper error handling
4. Returns structured JSON response

### Platform Support
- **Primary**: Linux (Alpine, Debian, Ubuntu) - fully optimized
- **Build targets**: Windows x64, Linux AMD64, Linux ARM64
- **Static linking**: Enabled for Linux builds to avoid dependency issues

## Security Considerations ⚠️

1. **Shell Commands**: The `bash` tool can execute arbitrary shell commands - use with trusted clients only
2. **File Access**: Tools can read/write/modify files - verify file paths and permissions
3. **Background Processes**: Long-running processes consume resources - automatic cleanup recommended
4. **Environment Variables**: Avoid exposing sensitive data in environment variable settings

## Build Outputs 📦

After running `build.ps1`, the following files are generated in `dist/`:
- `mcp-server.exe` (~5.5 MB) - Windows x64
- `mcp-server-linux` (~5.3 MB) - Linux AMD64
- `mcp-server-linux-arm64` (~5.1 MB) - Linux ARM64
- `mcp-server-current` - Current platform build

## Documentation Files

- **README.md** - User-facing documentation with usage examples
- **BUILD.md** - Detailed build instructions and troubleshooting
- **AGENTS.md** - Contributor guidelines and development standards
- **todo.md** - TypeScript type definitions for all tool inputs/outputs

## Common Issues & Solutions 🔧

**Build fails**: Ensure Go 1.23+ is installed and `go mod tidy` has been run
**Tests timeout**: Check for hanging background processes in bash tests
**Permission errors**: Verify file permissions and user access rights
**Cross-compilation issues**: Use PowerShell build script instead of manual go build for multi-platform builds
