# Repository Guidelines

## 项目概述 📋

这是一个MCP (Model Context Protocol) 文件操作与系统命令工具服务器，专门为Linux发行版（Alpine、Debian、Ubuntu）优化设计。提供文件操作、Shell命令执行、待办事项管理和Web工具等核心功能。

### 🎯 主要特性
- **文件操作**: 读取、写入、编辑、glob匹配、文本搜索
- **系统命令**: Shell命令执行、后台进程管理
- **工具集成**: 待办事项、Web搜索与抓取
- **跨平台编译**: 支持Windows、Linux AMD64/ARM64

## 项目结构 🏗️

### 核心文件
- `main.go` - 主程序入口，MCP服务器初始化
- `go.mod` - Go模块定义和依赖管理
- `BUILD.md` - 构建指南文档

### 工具模块
- `tools/` - 核心工具模块目录
  - 文件操作工具（read_file、write_file、edit_file、glob、grep）
  - 系统命令工具（bash、bash_output、kill_shell）
  - 业务工具（todo、web_search、web_fetch）

### 构建脚本
- `build.ps1` - PowerShell多平台构建脚本
- `build-release.sh` - Linux构建脚本
- `test_functions.sh` - 功能测试脚本

### 文档
- `README.md` - 项目说明和使用指南
- `todo.md` - 工具输入输出类型定义
- `AGENTS.md` - 本贡献者指南

## 构建与测试 🔧

### 环境要求
- **Go**: 1.23+ (当前使用1.25.3)
- **平台**: Linux专用优化（支持Alpine、Debian、Ubuntu）
- **构建工具**: PowerShell或Bash

### 快速构建
```bash
# PowerShell构建（推荐）
.\build.ps1

# 或手动构建
go mod tidy
go build -o mcp-server .
```

### 测试命令
```bash
# 运行所有测试
go test ./...

# 功能测试脚本
.\test_functions.sh

# 版本检查
go version
```

### 编译目标
- `dist/mcp-server.exe` - Windows PE32+ x64 (5.5 MB)
- `dist/mcp-server-linux` - Linux AMD64 x64 (5.3 MB)  
- `dist/mcp-server-linux-arm64` - Linux ARM64 aarch64 (5.1 MB)
- `dist/mcp-server-current` - 当前平台版本

## 开发规范 👨‍💻

### 代码风格
- **语言**: Go (使用最新稳定版本)
- **包管理**: Go Modules (go.mod)
- **格式规范**: 使用 `gofmt` 自动格式化
- **类型注释**: 完整的TypeScript风格接口定义

### 文件命名
- 工具文件名采用功能描述性命名
- 模块文件使用小写字母和下划线
- 测试文件以 `_test.go` 结尾

### 错误处理
- 所有工具函数必须包含完整的错误处理
- 返回结构化的错误信息
- 重要操作需要日志记录

### 安全规范
- **命令执行**: 谨慎处理shell命令，防止任意代码执行
- **文件访问**: 严格控制文件系统访问权限
- **环境变量**: 避免泄露敏感信息

## 工具开发指南 🛠️

### 新增工具流程
1. 在 `tools/` 目录创建新模块文件
2. 实现工具函数和参数结构体
3. 注册到MCP服务器
4. 更新 `todo.md` 类型定义
5. 编写相应测试用例

### 工具接口要求
- 完整的TypeScript接口定义
- 参数验证和错误处理
- 返回值结构标准化
- 文档字符串说明

## 贡献流程 🤝

### Pull Request要求
1. **代码检查**: 确保通过 `go vet` 和 `go fmt`
2. **测试覆盖**: 新功能必须包含测试
3. **文档更新**: 更新相关文档和注释
4. **构建验证**: 在多个平台测试构建

### 提交信息规范
```
type(scope): description

feat(tools): add new file operation tool
fix(bash): handle timeout errors in shell execution
docs(readme): update installation instructions
```

### 版本管理
- 使用语义化版本号
- 构建时自动注入版本信息
- Git标签用于发布管理

## 调试与故障排除 🐛

### 常见问题
- **Go环境**: 确保Go版本 >= 1.23
- **依赖问题**: 运行 `go mod tidy` 清理依赖
- **权限错误**: 检查文件访问权限
- **平台兼容性**: 验证目标平台支持

### 调试工具
```bash
# 代码格式检查
gofmt -l .

# 静态分析
go vet ./...

# 性能分析
go test -cpuprofile=cpu.prof -memprofile=mem.prof
```

## 发布管理 📦

### 发布前检查
- [ ] 所有测试通过
- [ ] 文档完整更新
- [ ] 版本号更新
- [ ] 构建脚本验证
- [ ] 跨平台兼容性测试

### 文档维护
- 保持README.md和BUILD.md同步更新
- 为新功能添加使用示例
- 更新版本发布日志

---

**维护者**: MCP工具开发团队  
**最后更新**: 2024年  
**兼容性**: Go 1.23+, Linux发行版