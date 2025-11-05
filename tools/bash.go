package tools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BashParams 定义bash命令参数 (完全符合todo.md标准)
type BashParams struct {
	Command         string      `json:"command" jsonschema:"Shell command to execute"`
	Description     string      `json:"description,omitempty" jsonschema:"5-10 word brief description of command functionality"`
	Timeout         int         `json:"timeout,omitempty" jsonschema:"Optional timeout in milliseconds (max 600000)"`
	RunInBackground interface{} `json:"run_in_background,omitempty" jsonschema:"Set to true to run command in background"`
}

// BashOutputParams 定义获取bash输出参数 (完全符合todo.md标准)
type BashOutputParams struct {
	BashID string `json:"bash_id" jsonschema:"Shell ID of the background process"`
	Filter string `json:"filter,omitempty" jsonschema:"Optional regex to filter output lines"`
}

// KillShellParams 定义终止进程参数 (完全符合todo.md标准)
type KillShellParams struct {
	ShellID string `json:"shell_id" jsonschema:"Shell ID of the background process to kill"`
}

// BashResult 定义bash执行结果 (完全符合todo.md标准)
type BashResult struct {
	Output   string `json:"output"`     // Combined stdout and stderr
	ExitCode int    `json:"exitCode"`   // Command exit code
	Killed   bool   `json:"killed"`     // Whether command was killed due to timeout
	ShellID  string `json:"shellId,omitempty"` // Shell ID for background processes
}

// BashOutputResult 定义获取输出的结果 (完全符合todo.md标准)
type BashOutputResult struct {
	Output   string `json:"output"`                     // New output since last check
	Status   string `json:"status"`                     // 'running' | 'completed' | 'failed'
	ExitCode int    `json:"exitCode,omitempty"`         // Exit code when completed
}

// KillShellResult 定义终止进程结果 (完全符合todo.md标准)
type KillShellResult struct {
	Message string `json:"message"`   // Success message
	ShellID string `json:"shell_id"` // Shell ID of the killed process
}

// ProcessInfo 存储后台进程信息
type ProcessInfo struct {
	Cmd       *exec.Cmd
	Stdout    bytes.Buffer
	Stderr    bytes.Buffer
	StartTime time.Time
	Mutex     sync.Mutex
}

// 全局变量来跟踪后台进程
var (
	backgroundProcesses = make(map[int]*ProcessInfo)
	processMutex        sync.Mutex
	// ANSI转义序列正则表达式（ESC字符 + [ + 数字/分号 + m）
	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// cleanANSI 清理ANSI转义序列
func cleanANSI(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}

// parseBool 解析布尔参数，支持字符串和布尔类型
func parseBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true"
	default:
		return false
	}
}

// AddBashTools 注册所有bash工具
func AddBashTools(server *mcp.Server) {
	// Bash工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bash",
		Description: "执行shell命令 - file-bash-tools.bash (MCP)(command: \"ls -la\", run_in_background: \"false\") - 支持后台执行",
	}, bashHandler)

	// BashOutput工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bash_output",
		Description: "获取后台进程输出 - file-bash-tools.bash_output (MCP)(bash_id: \"12345\", filter: \"error\") - 支持正则过滤",
	}, bashOutputHandler)

	// KillShell工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "kill_shell",
		Description: "终止后台进程 - file-bash-tools.kill_shell (MCP)(shell_id: \"12345\") - 优雅终止进程",
	}, killShellHandler)
}

// getShellCommand 根据操作系统获取正确的shell命令和参数
func getShellCommand(command string) (string, []string) {
	os := runtime.GOOS

	switch os {
	case "windows":
		// Windows: 优先使用PowerShell，如果不存在则使用cmd
		// 首先尝试PowerShell (pwsh)
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			return "pwsh.exe", []string{"-Command", command}
		}
		// 然后尝试PowerShell Core (powershell)
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			return "powershell.exe", []string{"-Command", command}
		}
		// 最后使用cmd
		return "cmd.exe", []string{"/C", command}

	case "darwin":
		// macOS: 使用bash或zsh
		return "bash", []string{"-c", command}

	default:
		// Linux和其他Unix系统: 使用bash
		return "bash", []string{"-c", command}
	}
}

// bashHandler 处理bash命令执行 (完全符合todo.md标准)
func bashHandler(ctx context.Context, req *mcp.CallToolRequest, params BashParams) (*mcp.CallToolResult, BashResult, error) {
	// 验证必需参数
	if params.Command == "" {
		return nil, BashResult{}, fmt.Errorf("command parameter is required")
	}

	// 设置超时时间（todo.md标准使用毫秒，最大600000毫秒=600秒）
	var timeout time.Duration
	if params.Timeout > 0 {
		if params.Timeout > 600000 {
			timeout = 600000 * time.Millisecond // 最大600秒
		} else {
			timeout = time.Duration(params.Timeout) * time.Millisecond
		}
	} else {
		timeout = 30000 * time.Millisecond // 默认30秒
	}

	startTime := time.Now()

	// 根据操作系统获取正确的shell命令
	shell, args := getShellCommand(params.Command)
	cmd := exec.Command(shell, args...)

	// 检查是否后台执行
	if parseBool(params.RunInBackground) {
		// 创建管道来捕获输出
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, BashResult{}, fmt.Errorf("Failed to create stdout pipe: %w", err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, BashResult{}, fmt.Errorf("Failed to create stderr pipe: %w", err)
		}

		// 创建进程信息对象
		processInfo := &ProcessInfo{
			Cmd:       cmd,
			StartTime: startTime,
		}

		// 启动命令
		if err := cmd.Start(); err != nil {
			return nil, BashResult{}, fmt.Errorf("Failed to start command: %w", err)
		}

		// 记录后台进程
		processMutex.Lock()
		backgroundProcesses[cmd.Process.Pid] = processInfo
		processMutex.Unlock()

		// 异步读取输出并缓存
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				processInfo.Mutex.Lock()
				processInfo.Stdout.WriteString(scanner.Text() + "\n")
				processInfo.Mutex.Unlock()
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				processInfo.Mutex.Lock()
				processInfo.Stderr.WriteString(scanner.Text() + "\n")
				processInfo.Mutex.Unlock()
			}
		}()

		return nil, BashResult{
			Output:   "",
			ExitCode: -1,
			Killed:   false,
			ShellID:  fmt.Sprintf("%d", cmd.Process.Pid),
		}, nil
	}

	// 同步执行
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时控制
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd.Start()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	var killed bool
	select {
	case err = <-done:
		// 命令正常完成
	case <-ctx.Done():
		// 超时，终止进程
		cmd.Process.Kill()
		err = fmt.Errorf("Command execution timed out")
		killed = true
	}

	// 清理ANSI转义序列
	stdoutText := cleanANSI(stdout.String())
	stderrText := cleanANSI(stderr.String())

	// 合并stdout和stderr为单个output字段
	output := stdoutText
	if stderrText != "" {
		if output != "" {
			output += "\n" + stderrText
		} else {
			output = stderrText
		}
	}

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else if killed {
			exitCode = -1
		}
	}

	return nil, BashResult{
		Output:   output,
		ExitCode: exitCode,
		Killed:   killed,
		ShellID:  "", // 同步执行没有shell ID
	}, nil
}

// bashOutputHandler 处理获取后台进程输出 (完全符合todo.md标准)
func bashOutputHandler(ctx context.Context, req *mcp.CallToolRequest, params BashOutputParams) (*mcp.CallToolResult, BashOutputResult, error) {
	// 验证必需参数
	if params.BashID == "" {
		return nil, BashOutputResult{}, fmt.Errorf("bash_id parameter is required")
	}

	// 从bash_id解析PID
	pid, err := strconv.Atoi(params.BashID)
	if err != nil {
		return nil, BashOutputResult{}, fmt.Errorf("Invalid bash_id format: %s", params.BashID)
	}

	processMutex.Lock()
	processInfo, exists := backgroundProcesses[pid]
	if !exists {
		processMutex.Unlock()
		return nil, BashOutputResult{}, fmt.Errorf("Background process with bash_id %s not found", params.BashID)
	}

	stdout := cleanANSI(processInfo.Stdout.String())
	stderr := cleanANSI(processInfo.Stderr.String())

	// 合并stdout和stderr为单个output字段
	output := stdout
	if stderr != "" {
		if output != "" {
			output += "\n" + stderr
		} else {
			output = stderr
		}
	}

	// 应用过滤器
	if params.Filter != "" {
		if re, err := regexp.Compile(params.Filter); err == nil {
			lines := strings.Split(output, "\n")
			var filteredLines []string
			for _, line := range lines {
				if re.MatchString(line) {
					filteredLines = append(filteredLines, line)
				}
			}
			output = strings.Join(filteredLines, "\n")
		}
	}

	// 获取进程状态
	process := processInfo.Cmd.Process
	status := "running"
	var exitCode int

	if process == nil {
		// 进程已经结束，清理并返回
		delete(backgroundProcesses, pid)
		processMutex.Unlock()
		return nil, BashOutputResult{
			Output:   output,
			Status:   "completed",
			ExitCode: 0,
		}, nil
	}

	// 检查进程是否已经结束
	processMutex.Unlock()
	err = processInfo.Cmd.Wait()
	if err == nil {
		// 进程正常结束
		processMutex.Lock()
		delete(backgroundProcesses, pid)
		processMutex.Unlock()
		status = "completed"
	} else if exitError, ok := err.(*exec.ExitError); ok {
		// 进程异常结束
		processMutex.Lock()
		delete(backgroundProcesses, pid)
		processMutex.Unlock()
		status = "failed"
		exitCode = exitError.ExitCode()
	}

	return nil, BashOutputResult{
		Output:   output,
		Status:   status,
		ExitCode: exitCode,
	}, nil
}

// killShellHandler 处理终止进程 (完全符合todo.md标准)
func killShellHandler(ctx context.Context, req *mcp.CallToolRequest, params KillShellParams) (*mcp.CallToolResult, KillShellResult, error) {
	// 验证必需参数
	if params.ShellID == "" {
		return nil, KillShellResult{}, fmt.Errorf("shell_id parameter is required")
	}

	// 从shell_id解析PID
	pid, err := strconv.Atoi(params.ShellID)
	if err != nil {
		return nil, KillShellResult{}, fmt.Errorf("Invalid shell_id format: %s", params.ShellID)
	}

	processMutex.Lock()
	processInfo, exists := backgroundProcesses[pid]
	if !exists {
		processMutex.Unlock()
		return nil, KillShellResult{}, fmt.Errorf("Background process with shell_id %s not found", params.ShellID)
	}

	// 尝试终止进程 (先尝试优雅终止，失败则强制终止)
	if err := processInfo.Cmd.Process.Signal(os.Interrupt); err != nil {
		// 如果中断信号失败，强制终止
		if err := processInfo.Cmd.Process.Kill(); err != nil {
			processMutex.Unlock()
			return nil, KillShellResult{}, fmt.Errorf("Failed to kill process: %w", err)
		}
	}

	// 等待进程结束
	processInfo.Cmd.Wait()

	// 从跟踪列表中删除
	delete(backgroundProcesses, pid)
	processMutex.Unlock()

	return nil, KillShellResult{
		Message: fmt.Sprintf("Successfully killed background process with shell_id: %s", params.ShellID),
		ShellID: params.ShellID,
	}, nil
}
