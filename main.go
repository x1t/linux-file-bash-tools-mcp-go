package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"mcp-file-tools/tools"
)

// 构建信息变量，由构建脚本注入
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func showVersion() {
	fmt.Printf("MCP File-Bash-Tools\n")
	fmt.Printf("版本: %s\n", Version)
	fmt.Printf("构建时间: %s\n", BuildTime)
	fmt.Printf("Git 提交: %s\n", GitCommit)
	os.Exit(0)
}

func main() {
	// 检查命令行参数
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			showVersion()
		case "-h", "--help", "help":
			fmt.Printf("MCP File-Bash-Tools %s\n\n", Version)
			fmt.Printf("用法: %s [选项]\n\n", os.Args[0])
			fmt.Printf("选项:\n")
			fmt.Printf("  -v, --version   显示版本信息\n")
			fmt.Printf("  -h, --help      显示此帮助信息\n\n")
			fmt.Printf("这是一个 MCP (Model Context Protocol) 服务器，\n")
			fmt.Printf("提供文件操作和系统命令工具。\n")
			os.Exit(0)
		}
	}

	// 创建服务器实例
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "file-tools-server",
		Version: Version,
	}, &mcp.ServerOptions{
		Instructions: "文件操作和系统命令工具服务器，提供读写文件、搜索、glob匹配和shell命令执行功能",
	})

	// 注册文件操作工具
	tools.AddFileTools(server)

	// 注册系统命令工具
	tools.AddBashTools(server)

	// 注册待办事项工具
	tools.AddTodoTools(server)

	// 注册Web工具
	tools.AddWebTools(server)

	// 启动服务器
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
