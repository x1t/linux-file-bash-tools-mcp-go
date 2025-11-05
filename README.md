# 🚀 MCP 文件操作与系统命令工具服务器

一个功能完整、企业级的 MCP（Model Context Protocol）服务器，为 AI 应用提供强大的文件操作和系统命令执行能力，完美适配 Linux 生态系统！💪

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/dl/)
[![MCP Protocol](https://img.shields.io/badge/MCP-2025-blue?style=flat-square)](https://modelcontextprotocol.io/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Platform](https://img.shields.io/badge/Platform-Linux-lightgrey?style=flat-square&logo=linux)](https://www.linux.org/)

---

## ✨ 核心特性

### 📁 文件操作工具套件

我们的文件工具套件提供了企业级的文件管理能力，让 AI 能够安全、高效地操作文件系统！

| 工具 | 功能描述 | 特色能力 |
|------|---------|---------|
| **read_file** 📖 | 读取文件内容 | ✅ 支持行范围选择 (`start:end`)<br>✅ 支持相对行号 (`-10:-1`)<br>✅ 大文件优化处理 |
| **write_file** ✍️ | 写入内容到文件 | ✅ 自动创建目录结构<br>✅ 原子性写入保证数据安全<br>✅ 支持覆盖和追加模式 |
| **edit_file** 🔧 | 编辑文件内容 | ✅ 精确的行范围替换<br>✅ 批量编辑支持<br>✅ 变更预览机制 |
| **glob** 🔍 | 文件模式匹配 | ✅ 通配符支持 (`*`, `**`, `?`)<br>✅ 递归目录搜索<br>✅ 过滤条件支持 |
| **grep** 🎯 | 智能文本搜索 | ✅ 正则表达式支持<br>✅ 大小写敏感/不敏感<br>✅ 多线程并行搜索<br>✅ 行号和上下文显示 |

### 🐚 系统命令工具集

强大的 shell 命令执行能力，支持同步、异步及后台进程管理！

| 工具 | 功能描述 | 核心优势 |
|------|---------|---------|
| **bash** ⚡ | 执行 Shell 命令 | ✅ 同步/异步执行模式<br>✅ 超时控制机制<br>✅ 环境变量自定义<br>✅ 工作目录指定 |
| **bash_output** 📊 | 获取后台进程输出 | ✅ 实时 stdout/stderr 捕获<br>✅ 增量输出读取<br>✅ 进程状态监控 |
| **kill_shell** 🛑 | 终止后台进程 | ✅ 优雅中断 (SIGINT)<br>✅ 强制终止 (SIGKILL)<br>✅ 自动资源清理 |

### 🌐 Web 工具集 (可选)

| 工具 | 功能描述 |
|------|---------|
| **web_search** 🔎 | 互联网搜索，支持域名过滤 |
| **web_fetch** 🌐 | URL 内容获取与 AI 智能处理 |

### ✅ Todo 工具

| 工具 | 功能描述 |
|------|---------|
| **todo_write** 📝 | 创建和管理任务列表，支持状态跟踪 |

---

## 🚀 快速开始

### 📋 前置要求

在开始之前，请确保您的环境满足以下要求：

- **Go 版本**: 1.23 或更高版本 [📥 下载](https://golang.org/dl/)
- **MCP 客户端**: Claude Desktop、VS Code Copilot 或其他兼容客户端
- **操作系统**: Linux (Alpine, Debian, Ubuntu) 优化支持 🐧

### 🛠️ 安装步骤

#### 方式一：从源码构建 (推荐)

```bash
# 1️⃣ 克隆项目
git clone <your-repository-url>
cd mcp-file-tools

# 2️⃣ 安装依赖
go mod tidy

# 3️⃣ 验证依赖
go mod verify

# 4️⃣ 编译项目
go build -o mcp-server .

# 5️⃣ 验证构建
./mcp-server --version
```

#### 方式二：使用构建脚本

**Windows (PowerShell):**
```powershell
pwsh.exe -File build.ps1
```

**Linux/macOS:**
```bash
# 多平台构建
chmod +x build-release.sh
./build-release.sh

# 或指定版本
VERSION=1.0.0 ./build-release.sh
```

构建完成后，可执行文件将位于 `dist/` 目录中：
- `mcp-server.exe` (~5.5 MB) - Windows x64
- `mcp-server-linux` (~5.3 MB) - Linux AMD64
- `mcp-server-linux-arm64` (~5.1 MB) - Linux ARM64

### ▶️ 运行服务器

#### 标准模式 (推荐用于 MCP 客户端)

```bash
# 基本运行
./mcp-server

# 带版本信息
./mcp-server --version

# 带帮助信息
./mcp-server --help
```

#### HTTP 模式 (可选)

```bash
# 启动 HTTP 服务器
./mcp-server -http :8080

# 自定义地址和端口
./mcp-server -http 192.168.1.100:9000
```

---

## 🔧 MCP 客户端配置

### Claude Desktop 配置

按照以下步骤配置 Claude Desktop：

#### 1️⃣ 定位配置文件

**macOS/Linux:**
```bash
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Windows:**
```bash
%APPDATA%\Claude\claude_desktop_config.json
```

#### 2️⃣ 编辑配置

在 `claude_desktop_config.json` 中添加：

```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/full/path/to/mcp-server",
      "args": [],
      "env": {}
    }
  }
}
```

#### 3️⃣ 重启客户端

重启 Claude Desktop，服务器将自动加载！🎉

### VS Code Copilot 配置

创建 `.vscode/mcp.json` 文件：

```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/full/path/to/mcp-server",
      "args": []
    }
  }
}
```

---

## 📖 实用示例

### 📂 文件操作示例

#### 读取文件特定行

```json
{
  "tool": "read_file",
  "arguments": {
    "filePath": "/path/to/project/main.go",
    "offset": 1,
    "limit": 50
  }
}
```

#### 创建新文件

```json
{
  "tool": "write_file",
  "arguments": {
    "filePath": "/path/to/newfile.txt",
    "content": "Hello, MCP World! 🌟\n这是第二行内容。"
  }
}
```

#### 编辑文件内容

```json
{
  "tool": "edit_file",
  "arguments": {
    "filePath": "/path/to/config.yaml",
    "old_string": "old_value: 123",
    "new_string": "old_value: 456",
    "replace_all": true
  }
}
```

#### 搜索项目文件

```json
{
  "tool": "glob",
  "arguments": {
    "pattern": "**/*.go",
    "path": "/path/to/project"
  }
}
```

#### 文本内容搜索

```json
{
  "tool": "grep",
  "arguments": {
    "pattern": "TODO|FIXME",
    "path": "/path/to/project",
    "caseSensitive": false,
    "output_mode": "content",
    "-n": true
  }
}
```

### 🐚 Shell 命令示例

#### 执行简单命令

```json
{
  "tool": "bash",
  "arguments": {
    "command": "ls -la /var/log",
    "timeout": 5000
  }
}
```

#### 项目构建

```json
{
  "tool": "bash",
  "arguments": {
    "command": "go build -o myapp .",
    "workingDir": "/path/to/project",
    "timeout": 30000
  }
}
```

#### 后台服务运行

```json
{
  "tool": "bash",
  "arguments": {
    "command": "python -m http.server 8000",
    "run_in_background": true,
    "timeout": 0
  }
}
```

#### 获取后台进程输出

```json
{
  "tool": "bash_output",
  "arguments": {
    "bash_id": "process-12345"
  }
}
```

#### 终止后台进程

```json
{
  "tool": "kill_shell",
  "arguments": {
    "shell_id": "process-12345"
  }
}
```

---

## 🧪 测试与验证

### 运行测试套件

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -v tools/file_test.go tools/file.go

# 生成覆盖率报告
go test -cover ./...

# 运行功能测试脚本
chmod +x test_functions.sh
./test_functions.sh
```

### 代码质量检查

```bash
# 格式化代码
gofmt -l .

# 静态分析
go vet ./...

# 依赖验证
go mod verify

# 构建测试
go build -o /tmp/test-build .
```

---

## 📦 构建与发布

### 开发构建

```bash
# 编译当前平台
go build -o mcp-server .

# 带版本信息
go build -ldflags "-X main.Version=dev" -o mcp-server .
```

### 生产构建

创建静态链接的生产版本：

```bash
# Linux 静态构建
./build-release.sh

# 或指定版本
VERSION=1.0.0 ./build-release.sh
```

### 多平台交叉编译

```powershell
# Windows PowerShell (自动多平台构建)
pwsh.exe -File build.ps1
```

构建输出将位于 `dist/` 目录。

---

## ⚠️ 安全注意事项

在使用本服务器时，请务必注意以下安全事项：

### 🔒 关键安全点

1. **Shell 命令执行** ⚠️
   - `bash` 工具可以执行任意 shell 命令
   - **仅在受信任的环境中使用**
   - 建议配置命令白名单

2. **文件系统访问** 🔐
   - 工具可以读取、修改、删除文件系统中的文件
   - **注意权限控制**，避免访问敏感目录
   - 建议在沙盒环境中运行

3. **环境变量保护** 🛡️
   - 设置环境变量时避免泄露敏感信息
   - 不要在日志中输出密钥或令牌

4. **后台进程管理** 🔄
   - 长时间运行的进程会持续占用系统资源
   - **及时清理** 无用的后台进程
   - 监控进程资源使用情况

### 🛡️ 最佳实践

- 在容器或虚拟机中运行服务器
- 定期更新依赖和补丁
- 配置适当的文件权限（推荐 `700` 目录权限）
- 使用防火墙限制网络访问
- 启用审计日志记录操作

---

## 🏗️ 项目架构

### 目录结构

```
mcp-file-tools/
├── main.go                    # MCP 服务器入口点 🏁
├── go.mod                     # Go 模块定义
├── go.sum                     # 依赖校验文件
├── build.ps1                  # Windows 多平台构建脚本 🪟
├── build-release.sh           # Linux 静态构建脚本 🐧
├── test_functions.sh          # 功能测试脚本 🧪
├── README.md                  # 项目文档 📚
├── BUILD.md                   # 构建文档 🔨
├── AGENTS.md                  # 开发者指南 👨‍💻
├── todo.md                    # TypeScript 类型定义 📝
└── tools/                     # 工具模块目录 🧰
    ├── tools.go               # 模块注册中心
    ├── file.go                # 文件操作工具
    ├── file_test.go           # 文件工具测试
    ├── bash.go                # Shell 执行工具
    ├── bash_test.go           # Bash 工具测试
    ├── todo.go                # Todo 管理工具
    ├── todo_test.go           # Todo 工具测试
    ├── web.go                 # Web 工具 (可选)
    └── web_test.go            # Web 工具测试
```

### 核心组件说明

#### 1️⃣ Main Server (`main.go:1-70`)
- 初始化 MCP 服务器实例
- 注册所有工具模块
- 支持 stdio 和 HTTP 传输
- 遵循 [MCP 规范](https://modelcontextprotocol.io/)

#### 2️⃣ File Tools (`tools/file.go:1-300`)
- 完整的文件 I/O 操作
- 高级模式匹配和搜索
- 企业级错误处理

#### 3️⃣ Bash Tools (`tools/bash.go:1-200`)
- 安全的 shell 命令执行
- 后台进程生命周期管理
- 资源清理和超时控制

---

## 📊 性能特性

### 🚀 优化亮点

- **静态链接**: Linux 构建使用 musl-gcc，避免动态依赖问题
- **多线程并发**: grep 等工具支持并行搜索
- **内存优化**: 大文件分块读取，防止内存溢出
- **进程池**: 后台进程自动管理，避免资源泄漏

### 📈 基准测试

典型操作性能参考：

| 操作 | 文件大小 | 平均耗时 |
|------|---------|---------|
| read_file | 1MB | <10ms |
| grep | 10MB | <100ms |
| glob | 1000 files | <50ms |

*测试环境: Ubuntu 22.04, Go 1.23, SSD*

---

## 🤝 贡献指南

欢迎社区贡献！让我们一起让这个项目更强大！💪

### 🎯 贡献方式

- 🐛 提交 Bug 报告
- 💡 提出新功能建议
- 📝 改进文档和示例
- 🔧 提交代码修复或新功能
- 🧪 添加测试用例

### 📋 开发流程

1. **Fork** 项目到您的 GitHub 账户
2. **创建** 功能分支 (`git checkout -b feature/AmazingFeature`)
3. **提交** 更改 (`git commit -m 'Add some AmazingFeature'`)
4. **推送** 到分支 (`git push origin feature/AmazingFeature`)
5. **开启** Pull Request

### ✅ 代码规范

- 遵循 [Go 代码规范](https://golang.org/doc/effective_go.html)
- 添加适当的单元测试
- 确保所有测试通过 (`go test ./...`)
- 更新相关文档

---

## 🆘 常见问题

### Q: 构建失败，提示 "go: cannot find main module"？

**A:** 请确保您在项目根目录，并运行了 `go mod tidy`

```bash
go mod tidy
go build -o mcp-server .
```

### Q: 测试超时，如何解决？

**A:** 检查是否有挂起的后台进程，确保在测试后正确清理

```bash
# 查看后台进程
ps aux | grep mcp-server

# 清理测试进程
pkill -f mcp-server
```

### Q: 权限错误，如何解决？

**A:** 验证文件权限，确保用户有读写权限

```bash
# 检查文件权限
ls -la /path/to/file

# 设置权限 (谨慎使用)
chmod 644 /path/to/file
```

### Q: 如何查看详细日志？

**A:** 启用调试模式

```bash
./mcp-server --verbose
```

### Q: 支持 Windows 吗？

**A:** 主要针对 Linux 优化，但可以使用 Windows 子系统 (WSL2) 运行

---

## 📚 参考资源

- **MCP 官方规范**: [modelcontextprotocol.io](https://modelcontextprotocol.io/)
- **Go 官方文档**: [golang.org](https://golang.org/doc/)
- **MCP Go SDK**: [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- **最佳实践**: [MCP 安全指南](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)

---

## 📜 许可证

本项目采用 [MIT 许可证](LICENSE) 开源，详情请参阅许可证文件。

---

## 👏 致谢

感谢以下开源项目和社区：

- [Model Context Protocol](https://modelcontextprotocol.io/) - 强大的协议规范
- [Go 语言](https://golang.org/) - 优秀的系统编程语言
- Linux 社区 - 持续的测试和反馈

---

## 📞 联系我们

- 🐛 **问题报告**: [GitHub Issues](https://github.com/your-repo/issues)
- 💬 **讨论交流**: [GitHub Discussions](https://github.com/your-repo/discussions)
- 📧 **邮件联系**: your-email@example.com

---

<div align="center">

**⭐ 如果这个项目对您有帮助，请给我们一个 Star！ ⭐**

Made with ❤️ by 小C

</div>
