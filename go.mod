// MCP文件操作与系统命令工具服务器 - 专用于Alpine、Debian、Ubuntu
// 不支持Windows和macOS，专注Linux发行版优化
module mcp-file-tools

go 1.25.3

// 必需的MCP SDK依赖
require github.com/modelcontextprotocol/go-sdk v1.1.0

// 传递依赖
require (
	github.com/google/jsonschema-go v0.3.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
)
