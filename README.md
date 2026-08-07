# 📦 File-Bash-Tools Linux

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.23.0+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)
![Test Coverage](https://img.shields.io/badge/Test%20Coverage-86%2B-brightgreen?style=for-the-badge)
![Platform](https://img.shields.io/badge/Platform-Linux-lightgrey?style=for-the-badge&logo=linux&logoColor=white)

一个强大的 **MCP（Model Context Protocol）** 工具集，专为文件操作和bash命令执行而设计！🚀

[✨ 特性](#-特性) • [🛠️ 功能列表](#️-功能列表) • [🚀 快速开始](#-快速开始) • [📋 项目结构](#-项目结构) • [🧪 测试](#-测试) • [🔒 安全](#-安全) • [🤝 贡献](#-贡献)

</div>

---

## ✨ 特性

| 特性                       | 描述                                             |
| -------------------------- | ------------------------------------------------ |
| 🗂️**文件操作工具** | 读取、写入、编辑文件，支持行范围选择、偏移和限制 |
| 🔍**搜索工具**       | glob模式匹配和grep文本搜索功能，支持高级通配符   |
| 💻**Bash命令执行**   | 支持同步和后台执行，带超时控制和进程管理         |
| ⚡**进程管理**       | 后台进程输出获取和优雅终止功能                   |
| 📝**待办事项管理**   | 任务状态跟踪工具，支持多种状态                   |
| ✅**高测试覆盖率**   | 超过**86%** 的代码覆盖率，确保可靠性 🛡️  |
| 🔒**安全加固**       | 路径验证、超时保护、UTF-8编码验证、ReDoS防护     |
| 🛡️**优雅关闭**      | 服务器优雅关闭机制，支持信号处理和资源清理       |
| ⚡**性能优化**      | 线程安全、原子操作、内存保护机制                 |

---

## 🛠️ 功能列表

### 1️⃣ Bash 工具 💻

| 工具            | 功能                           | 示例                                                                           |
| --------------- | ------------------------------ | ------------------------------------------------------------------------------ |
| `bash`        | 执行shell命令，支持后台执行    | `file-bash-tools.bash` (command: `"ls -la"`, run_in_background: `false`) |
| `bash_output` | 获取后台进程输出，支持正则过滤 | `file-bash-tools.bash_output` (bash_id: `"12345"`, filter: `"error"`)    |
| `kill_shell`  | 终止后台进程（优雅终止）       | `file-bash-tools.kill_shell` (shell_id: `"12345"`)                         |

#### 💡 Bash 工具亮点

- ⏱️ **超时控制**: 默认30秒，**必需参数**，最大600秒
- 🔄 **后台执行**: 支持长时间运行命令
- 🛡️ **内存保护**: 100KB输出缓冲区，防止内存泄漏
- 🎯 **进程跟踪**: 自动清理完成的进程
- ⚡ **线程安全**: 原子操作管理进程状态
- 🔒 **增强安全**: 路径验证和正则复杂度检查

---

### 2️⃣ 文件 工具 🗂️

| 工具           | 功能                         | 示例                                                                                                                   |
| -------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `read_file`  | 读取文件内容，支持指定行范围 | `file-bash-tools.read_file` (file_path: `"/path/to/file.txt"`)                                                     |
| `write_file` | 写入内容到文件（完全覆盖）   | `file-bash-tools.write_file` (file_path: `"/path/to/file.txt"`, content: `"内容"`)                               |
| `edit_file`  | 编辑文件内容（字符串替换）   | `file-bash-tools.edit_file` (file_path: `"/path/to/file.txt"`, old_string: `"旧文本"`, new_string: `"新文本"`) |
| `glob`       | 根据模式匹配文件路径         | `file-bash-tools.glob` (pattern: `"**/*.go"`)                                                                      |
| `grep`       | 在文件或目录中搜索文本内容   | `file-bash-tools.grep` (pattern: `"func"`, path: `"/src"`)                                                       |

#### 🌟 文件工具特色

- 📍 **绝对路径**: 所有操作需要绝对路径，安全可靠
- 🎨 **高级通配符**: 支持 `*` 和 `**` 模式（基于doublestar/v4）
- 📊 **分页支持**: `head_limit` 和 `offset` 参数
- 🔤 **编码验证**: 自动验证UTF-8编码
- 📝 **行号显示**: 可选显示行号（默认开启）
- 🔒 **安全加固**: 防止路径遍历，保护系统关键目录
- ⚡ **原子写入**: 确保文件写入操作的原子性
- 📏 **文件大小限制**: 10MB限制，防止内存问题

---

### 3️⃣ 任务管理 工具 📝

| 工具           | 功能                       | 示例                                                                               |
| -------------- | -------------------------- | ---------------------------------------------------------------------------------- |
| `todo_write` | 更新待办事项列表，状态管理 | `file-bash-tools.todo_write` (todos: `[{content: "任务", status: "pending"}]`) |

#### 📋 Todo 工具支持的状态

- `pending` - 待处理
- `in_progress` - 进行中
- `completed` - 已完成

---

## 🚀 快速开始

### 📋 前置要求

| 依赖     | 版本要求                       |
| -------- | ------------------------------ |
| Go       | 1.25.0+                        |
| 操作系统 | Linux (Alpine, Debian, Ubuntu) |
| MCP SDK  | v1.7.0                         |

### 💾 安装

```bash
# 克隆仓库
git clone <repository-url>
cd file-bash-tools-linux

# 安装依赖
go mod tidy
```

### 🔨 构建

#### 方式一：手动构建（推荐开发环境）

```bash
# 当前平台
go build -o mcp-server .

# 包含版本信息
go build -ldflags "-X main.Version=dev" -o mcp-server .

# 静态链接构建（Linux）
CGO_ENABLED=1 CC=musl-gcc go build -ldflags "-X main.Version=dev" -o mcp-server .
```

#### 方式二：多平台构建

```powershell
# PowerShell（Windows）
pwsh.exe -File build.ps1

# 或Linux/macOS Bash
./build-release.sh
```

### ▶️ 运行

```bash
# 标准模式
./mcp-server

# 查看版本
./mcp-server --version

# 查看帮助
./mcp-server --help
```

---

## 📋 项目结构

```
file-bash-tools-linux/
├── 📄 main.go                      # MCP服务器入口
├── 📄 go.mod                       # Go模块定义
│
├── 🛠️ tools/                       # 工具实现目录
│   ├── bash.go                     # Bash命令工具
│   ├── file.go                     # 文件操作工具
│   ├── todo.go                     # 任务管理工具
│   ├── tools.go                    # 工具注册（当前为空）
│   │
│   ├── 🧪 bash_test.go             # Bash工具测试
│   ├── 🧪 read_file_test.go        # ReadFile工具测试
│   ├── 🧪 write_file_test.go       # WriteFile工具测试
│   ├── 🧪 edit_file_test.go        # EditFile工具测试
│   ├── 🧪 glob_test.go             # Glob工具测试
│   ├── 🧪 grep_test.go             # Grep工具测试
│   ├── 🧪 todo_test.go             # Todo工具测试
│   ├── 🧪 resolve_path_test.go     # 路径解析测试
│   ├── 🧪 truncate_by_tokens_test.go # Token截断测试
│   ├── 🧪 get_search_path_test.go  # 搜索路径测试
│   ├── 🧪 parse_range_test.go      # 范围解析测试
│   ├── 🧪 parse_line_number_test.go # 行号解析测试
│   ├── 🧪 search_lines_test.go     # 搜索行测试
│   ├── 🧪 get_shell_command_test.go # Shell命令测试
│   │
│   └── 📊 testdata/                # 测试数据目录
│
├── 📜 BUILD.md                     # 构建指南
├── 📚 CLAUDE.md                    # Claude Code开发指南
├── 👥 AGENTS.md                    # 贡献者指南
├── 📝 todo.md                      # TypeScript类型定义
└── 📖 README.md                    # 项目文档
```

---

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试文件
go test -v tools/bash_test.go tools/bash.go

# 生成测试覆盖率报告
go test -cover ./...

# 输出覆盖率到文件
go test -coverprofile=coverage.out ./...
go go tool cover -html=coverage.out -o coverage.html
```

### 📊 测试统计

- **测试文件数量**: 15+ 个
- **核心测试覆盖**:
  - ✅ Bash命令执行
  - ✅ 文件读写操作
  - ✅ Todo任务管理
  - ✅ 工具函数
- **测试覆盖率**: **86%+** 🎯
- **功能测试**: `./test_functions.sh`

### 🎯 测试特性

- 使用 `github.com/stretchr/testify/assert`
- 完整单元测试覆盖
- 集成测试支持
- 性能测试（可选）

---

## 🔒 安全

### 🛡️ 安全特性

1. **路径验证**

   - ✅ 所有文件路径必须为绝对路径
   - ✅ 防止路径遍历攻击和符号链接检查
   - ✅ 智能路径安全验证，保护系统关键文件
   - ✅ 安全区域路径：`/dev`、`/lib`、`/lib64`、`/run`、`/var/run`、`/var/tmp`
2. **命令执行保护**

   - ⏱️ 默认30秒超时（可配置，必需参数）
   - ⏱️ 最大超时限制：600秒
   - 🔒 仅受信任客户端使用
   - 🔄 增强的后台进程超时控制
3. **进程管理**

   - 🧹 自动清理后台进程，防止内存泄漏
   - 💾 100KB输出缓冲区限制
   - 🔄 优雅终止（SIGINT → SIGKILL）
   - ⚡ 原子操作确保线程安全
4. **编码安全**

   - ✅ UTF-8编码验证
   - ✅ 防止无效字符和ReDoS攻击
5. **环境变量**

   - 🚫 避免暴露敏感信息
   - 🔐 安全处理API密钥和令牌

### ⚠️ 安全建议

- 仅在**受信任环境**中使用
- **定期更新**依赖库
- 监控**长时间运行**的进程
- 使用**最小权限**原则
- ⚠️ **注意**: bash工具可执行任意shell命令，仅限受信任客户端使用
- 🔒 **路径安全**: 所有文件操作都会进行路径安全验证
- 📝 **超时设置**: timeout参数现在是必需的，请务必指定合理的超时时间

---

## 🤝 贡献

我们欢迎所有形式的贡献！🎉

### 📝 贡献流程

1. **Fork** 仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 **Pull Request**

### ✅ 贡献要求

- [ ] 添加**适当的测试**覆盖
- [ ] 通过 `go vet` 和 `go fmt` 检查
- [ ] 更新**相关文档**
- [ ] 遵循项目**编码标准**
- [ ] 确保**跨平台兼容性**

### 📋 编码规范

- 使用 `gofmt` 自动格式化
- 完整的**类型注释**
- 包含**错误处理**
- 添加**测试用例**
- 编写**清晰注释**

### 💬 提交信息格式

```
type(scope): description

feat(bash): add timeout configuration
fix(file): handle UTF-8 encoding errors
docs(readme): update installation guide
test(tools): add integration tests
```

---

## 📦 构建输出

使用 `build.ps1` 构建后，`dist/` 目录将包含：

| 文件名                     | 平台        | 大小    | 描述               |
| -------------------------- | ----------- | ------- | ------------------ |
| `mcp-server.exe`         | Windows x64 | ~5.5 MB | PE32+ 可执行文件   |
| `mcp-server-linux`       | Linux AMD64 | ~5.3 MB | x86-64 二进制文件  |
| `mcp-server-linux-arm64` | Linux ARM64 | ~5.1 MB | aarch64 二进制文件 |
| `mcp-server-current`     | 当前平台    | -       | 本机构建版本       |

### 🔨 静态链接

Linux版本使用**musl-gcc**静态链接，避免依赖问题：

```bash
CGO_ENABLED=1
CC=musl-gcc
```

---

## 📖 文档

| 文档                  | 描述                         |
| --------------------- | ---------------------------- |
| 📄**README.md** | 项目说明和使用指南（本文档） |
| 🔨**BUILD.md**  | 详细构建说明和故障排除       |
| 🤖**CLAUDE.md** | Claude Code开发指南          |
| 👥**AGENTS.md** | 贡献者指南和开发标准         |
| 📝**todo.md**   | TypeScript类型定义           |

---

## 🐛 故障排除

### 常见问题

#### Q: 构建失败？

**A:** 确保：

- Go 1.23+ 已安装 (`go version`)
- 运行 `go mod tidy` 同步依赖
- 检查模块名称：`mcp-file-tools`

#### Q: 测试超时？

**A:** 检查：

- 后台进程是否未正确清理
- 查看测试文件是否有无限循环
- 使用：`go test -v -timeout 30s ./...`

#### Q: 权限错误？

**A:** 验证：

- 文件权限和用户访问权限
- 文件操作需要绝对路径
- 目录写权限检查

#### Q: 跨平台编译问题？

**A:** 使用：

- PowerShell构建脚本（推荐）
- 或设置 `GOOS`/`GOARCH` 环境变量
- 确保 `CGO_ENABLED=1` 静态链接

---

## 📄 许可证

本项目采用 **MIT** 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## 🙏 致谢

感谢以下开源项目：

- [MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - 协议实现
- [doublestar/v4](https://github.com/bmatcuk/doublestar) - 高级glob匹配
- [stretchr/testify](https://github.com/stretchr/testify) - 测试框架

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给我们一个Star！⭐**

Made with ❤️ by MCP Tools Team

</div>
