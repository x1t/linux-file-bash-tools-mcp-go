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
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LimitedBuffer 实现一个带限制的缓冲区，防止无限内存增长
type LimitedBuffer struct {
	data     []byte
	maxSize  int
	mutex    sync.Mutex
}

// BashParams 定义bash命令参数 (完全符合todo.md标准)
type BashParams struct {
	Command         string      `json:"command" jsonschema:"Shell command to execute"`
	Description     string      `json:"description,omitempty" jsonschema:"5-10 word brief description of command functionality"`
	Timeout         int         `json:"timeout" jsonschema:"Required timeout in milliseconds (max 600000)"`
	RunInBackground bool        `json:"run_in_background,omitempty" jsonschema:"Set to true to run command in background"`
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

// NewLimitedBuffer 创建一个新的带限制的缓冲区
func NewLimitedBuffer(maxSize int) *LimitedBuffer {
	return &LimitedBuffer{
		data:    make([]byte, 0, maxSize),
		maxSize: maxSize,
	}
}

// Write 向缓冲区写入数据，如果超过最大尺寸则丢弃旧数据
func (lb *LimitedBuffer) Write(p []byte) (n int, err error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	newLen := len(lb.data) + len(p)
	if newLen <= lb.maxSize {
		// 如果新数据长度未超过限制，直接追加
		lb.data = append(lb.data, p...)
	} else {
		// 如果超过限制，则丢弃旧数据，保留最新的数据
		keepSize := lb.maxSize - len(p)
		if keepSize <= 0 {
			// 如果单次写入就超过了最大尺寸，只保留最新的最大尺寸数据
			if len(p) >= lb.maxSize {
				lb.data = make([]byte, lb.maxSize)
				copy(lb.data, p[len(p)-lb.maxSize:])
			} else {
				lb.data = make([]byte, len(p))
				copy(lb.data, p)
			}
		} else {
			// 移动旧数据到前面，释放空间
			copy(lb.data, lb.data[len(lb.data)-keepSize:])
			lb.data = lb.data[:keepSize]
			lb.data = append(lb.data, p...)
		}
	}
	
	return len(p), nil
}

// String 返回缓冲区内容的字符串表示
func (lb *LimitedBuffer) String() string {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	// 创建副本以避免数据竞争
	result := make([]byte, len(lb.data))
	copy(result, lb.data)
	return string(result)
}

// Bytes 返回缓冲区内容的字节副本
func (lb *LimitedBuffer) Bytes() []byte {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	result := make([]byte, len(lb.data))
	copy(result, lb.data)
	return result
}

// Reset 清空缓冲区
func (lb *LimitedBuffer) Reset() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	lb.data = lb.data[:0]
}

// cleanupCompletedProcesses 清理已完成的进程，防止内存泄漏
func cleanupCompletedProcesses() {
	processMutex.Lock()
	defer processMutex.Unlock()
	
	completedPids := []int{}
	for pid, processInfo := range backgroundProcesses {
		select {
		case <-processInfo.Done:
			// 进程已完成，标记为待删除
			completedPids = append(completedPids, pid)
		default:
			// 进程仍在运行
		}
	}
	
	// 删除已完成的进程
	for _, pid := range completedPids {
		delete(backgroundProcesses, pid)
	}
	
	if len(completedPids) > 0 {
		// 记录清理信息（生产环境中可以记录日志）
		_ = len(completedPids) // 避免未使用变量警告
	}
}

// startCleanupRoutine 启动定期清理协程
func startCleanupRoutine() {
	cleanupTicker = time.NewTicker(30 * time.Second) // 每30秒清理一次
	cleanupDone = make(chan struct{})
	
	go func() {
		for {
			select {
			case <-cleanupTicker.C:
				cleanupCompletedProcesses()
			case <-cleanupDone:
				cleanupTicker.Stop()
				return
			}
		}
	}()
}

// stopCleanupRoutine 停止清理协程
func stopCleanupRoutine() {
	if cleanupDone != nil {
		close(cleanupDone)
	}
}

// ProcessInfo 存储后台进程信息
type ProcessInfo struct {
	Cmd       *exec.Cmd
	Stdout    *LimitedBuffer
	Stderr    *LimitedBuffer
	StartTime time.Time
	Mutex     sync.Mutex
	Done      chan struct{} // 用于通知进程结束
	status    atomic.Value  // 原子状态："running", "completed", "failed"
}

// setStatus 设置进程状态（原子操作）
func (pi *ProcessInfo) setStatus(status string) {
	pi.status.Store(status)
}

// getStatus 获取进程状态（原子操作）
func (pi *ProcessInfo) getStatus() string {
	if status := pi.status.Load(); status != nil {
		return status.(string)
	}
	return "running" // 默认状态
}

// 全局变量来跟踪后台进程
var (
	backgroundProcesses = make(map[int]*ProcessInfo)
	processMutex        sync.Mutex
	// ANSI转义序列正则表达式（ESC字符 + [ + 数字/分号 + m）
	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	// 清理定时器
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
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
	// 启动定期清理协程
	startCleanupRoutine()
	
	// Bash工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bash",
		Description: "执行shell命令 - file-bash-tools.bash (MCP)(command: \"ls -la\", timeout: 30000, run_in_background: \"false\") - 支持后台执行，timeout是必需参数（毫秒）",
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

// StopBashTools 停止bash工具相关资源
func StopBashTools() {
	stopCleanupRoutine()
	
	// 清理所有后台进程
	processMutex.Lock()
	defer processMutex.Unlock()
	
	for pid, processInfo := range backgroundProcesses {
		if processInfo.Cmd.Process != nil {
			processInfo.Cmd.Process.Kill()
		}
		delete(backgroundProcesses, pid)
	}
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

	// 验证timeout参数 - 必需参数且在有效范围内
	if params.Timeout <= 0 || params.Timeout > 600000 {
		return nil, BashResult{}, fmt.Errorf("timeout parameter is required and must be between 1 and 600000 milliseconds")
	}

	// 设置超时时间（todo.md标准使用毫秒，最大600000毫秒=600秒）
	var timeout time.Duration
	if params.Timeout > 600000 {
		timeout = 600000 * time.Millisecond // 最大600秒
	} else {
		timeout = time.Duration(params.Timeout) * time.Millisecond
	}

	startTime := time.Now()

	// 根据操作系统获取正确的shell命令
	shell, args := getShellCommand(params.Command)
	cmd := exec.Command(shell, args...)

	// 检查是否后台执行
	if params.RunInBackground {
		// 创建管道来捕获输出
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, BashResult{}, fmt.Errorf("Failed to create stdout pipe: %w", err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, BashResult{}, fmt.Errorf("Failed to create stderr pipe: %w", err)
		}

		// 创建进程信息对象，使用带限制的缓冲区
		processInfo := &ProcessInfo{
			Cmd:       cmd,
			Stdout:    NewLimitedBuffer(1024 * 100), // 100KB 限制
			Stderr:    NewLimitedBuffer(1024 * 100), // 100KB 限制
			StartTime: startTime,
			Done:      make(chan struct{}),
		}
		processInfo.setStatus("running") // 初始化状态

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
			defer stdout.Close() // 确保管道被关闭
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text() + "\n"
				processInfo.Stdout.Write([]byte(line))
			}
		}()

		go func() {
			defer stderr.Close() // 确保管道被关闭
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text() + "\n"
				processInfo.Stderr.Write([]byte(line))
			}
		}()

		// 异步等待进程结束或超时，完成后通知
		go func() {
			// 创建带超时的context
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()
			
			select {
			case err := <-done:
				// 进程正常结束
				if err != nil {
					processInfo.setStatus("failed")
				} else {
					processInfo.setStatus("completed")
				}
			case <-ctx.Done():
				// 超时，终止进程
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				processInfo.setStatus("failed") // 超时终止视为失败
			}
			
			// 进程结束后，将进程信息标记为完成
			close(processInfo.Done)
			
			// 从跟踪列表中删除进程，这个操作需要在锁的保护下完成
			// 使用原子状态检查避免竞态条件
			processMutex.Lock()
			// 再次检查进程是否仍在映射中（可能已被其他操作删除）
			if _, exists := backgroundProcesses[cmd.Process.Pid]; exists {
				delete(backgroundProcesses, cmd.Process.Pid)
			}
			processMutex.Unlock()
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

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, BashResult{}, fmt.Errorf("Failed to start command: %w", err)
	}

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
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
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

	// 获取进程信息，使用锁保护
	processMutex.Lock()
	processInfo, exists := backgroundProcesses[pid]
	if !exists {
		processMutex.Unlock()
		return nil, BashOutputResult{}, fmt.Errorf("Background process with bash_id %s not found", params.BashID)
	}

	// 读取当前输出（使用线程安全的方法）
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

	// 检查进程状态 - 使用原子状态避免竞态条件
	status := processInfo.getStatus()
	var exitCode int

	// 如果状态不是running，说明进程已经结束
	if status != "running" {
		// 进程已经结束，获取退出码
		if processInfo.Cmd.ProcessState != nil {
			exitCode = processInfo.Cmd.ProcessState.ExitCode()
		} else {
			exitCode = 1 // 假设非正常退出
		}
	} else {
		// 进程仍在运行，使用非阻塞方式检查是否结束
		select {
		case <-processInfo.Done:
			// 进程刚刚结束，更新状态
			status = processInfo.getStatus()
			if processInfo.Cmd.ProcessState != nil {
				exitCode = processInfo.Cmd.ProcessState.ExitCode()
			} else {
				exitCode = 1
			}
		default:
			// 进程仍在运行
			status = "running"
		}
	}

	processMutex.Unlock()

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

	// 检查进程是否仍在运行
	if processInfo.Cmd.Process == nil {
		processMutex.Unlock()
		return nil, KillShellResult{}, fmt.Errorf("Process with shell_id %s is not running or already terminated", params.ShellID)
	}

	// 尝试终止进程 (先尝试优雅终止，失败则强制终止)
	var killErr error
	if err := processInfo.Cmd.Process.Signal(os.Interrupt); err != nil {
		// 如果中断信号失败，强制终止
		if err := processInfo.Cmd.Process.Kill(); err != nil {
			killErr = err
		}
	}

	// 等待进程结束，但不阻塞太久
	done := make(chan error, 1)
	go func() {
		done <- processInfo.Cmd.Wait()
	}()

	select {
	case <-done:
		// 进程已终止
	case <-time.After(5 * time.Second):
		// 超时，强制终止进程
		if processInfo.Cmd.Process != nil {
			_ = processInfo.Cmd.Process.Kill()
		}
	}

	// 从跟踪列表中删除，如果进程还未结束则会由goroutine处理
	delete(backgroundProcesses, pid)
	processMutex.Unlock()

	// 如果有终止错误，返回错误信息
	if killErr != nil {
		return nil, KillShellResult{
			Message: fmt.Sprintf("Process with shell_id %s terminated with warnings: %v", params.ShellID, killErr),
			ShellID: params.ShellID,
		}, nil
	}

	return nil, KillShellResult{
		Message: fmt.Sprintf("Successfully killed background process with shell_id: %s", params.ShellID),
		ShellID: params.ShellID,
	}, nil
}
