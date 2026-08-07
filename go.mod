// MCP文件操作与系统命令工具服务器 - 专用于Alpine、Debian、Ubuntu
// 不支持Windows和macOS，专注Linux发行版优化
module mcp-file-tools

go 1.25.0

// 必需的MCP SDK依赖
require github.com/modelcontextprotocol/go-sdk v1.7.0

// 传递依赖
require github.com/google/jsonschema-go v0.4.3 // indirect

require github.com/bmatcuk/doublestar/v4 v4.9.1

require (
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
