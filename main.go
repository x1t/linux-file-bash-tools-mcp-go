package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	// 设置优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器协程
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx, &mcp.StdioTransport{})
	}()

	// 等待服务器结束或收到信号
	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
	case sig := <-sigChan:
		log.Printf("收到信号 %s，正在优雅关闭...", sig)
		cancel()
		// 停止bash工具相关资源
		tools.StopBashTools()
		// 等待服务器结束
		if err := <-serverErr; err != nil && err != context.Canceled {
			log.Printf("服务器关闭错误: %v", err)
		}
	}
}
