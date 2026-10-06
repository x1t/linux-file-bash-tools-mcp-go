package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LimitedBuffer 实现一个带限制的缓冲区，防止无限内存增长
type LimitedBuffer struct {
	data    []byte
	maxSize int
	total   int // 累计写入的字节数（含已丢弃部分），用作增量读取的偏移
	mutex   sync.Mutex
}

// BashParams 定义bash命令参数 (完全符合todo.md标准)
type BashParams struct {
	Command         string `json:"command" jsonschema:"Shell command to execute"`
	Description     string `json:"description,omitempty" jsonschema:"5-10 word brief description of command functionality"`
	Timeout         int    `json:"timeout" jsonschema:"Required timeout in milliseconds (max 600000)"`
	RunInBackground bool   `json:"run_in_background,omitempty" jsonschema:"Set to true to run command in background"`
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
	Output   string `json:"output"`            // Combined stdout and stderr
	ExitCode int    `json:"exitCode"`          // Command exit code
	ShellID  string `json:"shellId,omitempty"` // Shell ID for background processes
}

// BashOutputResult 定义获取输出的结果 (完全符合todo.md标准)
type BashOutputResult struct {
	Output   string `json:"output"`             // New output since last check
	Status   string `json:"status"`             // 'running' | 'completed' | 'failed' | 'timed_out'
	ExitCode int    `json:"exitCode,omitempty"` // Exit code when completed
}

// KillShellResult 定义终止进程结果 (完全符合todo.md标准)
type KillShellResult struct {
	Message string `json:"message"`  // Success message
	ShellID string `json:"shell_id"` // Shell ID of the killed process
}

// NewLimitedBuffer 创建一个新的带限制的缓冲区
func NewLimitedBuffer(maxSize int) *LimitedBuffer {
	// 不预分配：多数命令输出很少，按需增长
	return &LimitedBuffer{maxSize: maxSize}
}

// Write 向缓冲区写入数据，如果超过最大尺寸则丢弃旧数据
func (lb *LimitedBuffer) Write(p []byte) (n int, err error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.total += len(p)
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

// Since 返回累计偏移 offset 之后写入且仍保留在缓冲区中的内容、新的累计偏移，
// 以及期间因超出上限而丢弃的字节数。ready 非 nil 时只返回其给出的前缀长度，其余留待下次读取
func (lb *LimitedBuffer) Since(offset int, ready func([]byte) int) (string, int, int) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	base := lb.total - len(lb.data)
	start, dropped := offset-base, 0
	if start < 0 {
		dropped, start = -start, 0
		// 丢弃旧数据时可能切断了多字节字符，跳过开头残缺的字节
		for start < len(lb.data) && start < utf8.UTFMax-1 && !utf8.RuneStart(lb.data[start]) {
			start++
			dropped++
		}
	}

	end := len(lb.data)
	if ready != nil {
		end = start + ready(lb.data[start:])
	}
	return string(lb.data[start:end]), base + end, dropped
}

// readyEnd 返回运行中进程的输出 b 可以安全返回的前缀长度：wholeLines 为 true 时只到最后一个换行符（未换行部分过长时除外）；
// 否则去掉末尾不完整的 UTF-8 字符与 ANSI 序列，避免被拆到两次查询中无法识别
func readyEnd(b []byte, wholeLines bool) int {
	if wholeLines {
		// 超长未换行的输出（如 \r 进度条）不再等待换行，避免在返回前就因超出缓冲区上限被丢弃
		if end := bytes.LastIndexByte(b, '\n') + 1; len(b)-end < maxPendingLine {
			return end
		}
	}
	end := completeRunesEnd(b)
	esc := bytes.LastIndexByte(b[:end], 0x1b)
	// 只扣住可能被后续数据补全的序列前缀；过长仍未结束的序列（如异常 OSC）不再等待
	if esc >= 0 && end-esc < maxPendingANSI && ansiPartialRegex.Match(b[esc:end]) {
		return esc
	}
	return end
}

// completeRunesEnd 返回 b 去掉末尾不完整 UTF-8 字符后的长度
func completeRunesEnd(b []byte) int {
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				return i
			}
			break
		}
	}
	return len(b)
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

	completedIds := []string{}
	for id, processInfo := range backgroundProcesses {
		select {
		case <-processInfo.Done:
			// 进程组中仍有后台子进程时保留记录，确保仍可通过 kill_shell / 服务退出终止；
			// 每轮探测也让"进程组已空"被尽早记录，缩小进程组 ID 被复用后误杀的窗口
			if !processInfo.groupAlive() && time.Since(processInfo.completedAt) > completedRetention {
				completedIds = append(completedIds, id)
			}
		default:
			// 进程仍在运行
		}
	}

	// 删除已完成的进程
	for _, id := range completedIds {
		backgroundProcesses[id].release()
		delete(backgroundProcesses, id)
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
	Output    *LimitedBuffer // stdout 与 stderr 共用，保留输出原始顺序
	StartTime time.Time
	Mutex     sync.Mutex    // 保护 readOffset
	Done      chan struct{} // 用于通知进程结束
	status    atomic.Value  // 原子状态："running", "completed", "failed", "timed_out"
	timedOut  atomic.Bool   // 是否因超过最长运行时间被终止
	// groupExited 是否已观察到进程组为空；之后进程组 ID 可能被系统复用，不能再发信号
	groupExited atomic.Bool
	// outputReader 输出管道读端；drained 在读到 EOF（所有写端都已关闭）后关闭
	outputReader *os.File
	drained      <-chan struct{}
	// readOffset bash_output 上次读取到的累计偏移
	readOffset int
	// completedAt 进程结束时间，在 Done 关闭前写入
	completedAt time.Time
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

// ansiPattern ANSI 转义序列：CSI 序列（颜色、光标移动、清屏等）、OSC 序列（如终端标题），
// 以及其他 ESC 序列（如保存/恢复光标 "\x1b7"、字符集切换 "\x1b(B"）
const ansiPattern = `\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[ -/]*[0-Z\\^-~]`

// ansiPartialPattern 尚未写完、后续可能补全的 ANSI 序列前缀
const ansiPartialPattern = `^\x1b(?:[ -/]*|\[[0-?]*[ -/]*|\][^\x07\x1b]*)$`

// maxPendingANSI 等待补全的 ANSI 序列最大长度
const maxPendingANSI = 64

// maxPendingLine 按行过滤时等待换行的最大长度，需小于输出缓冲区上限（100KB）
const maxPendingLine = 64 * 1024

// 全局变量来跟踪后台进程
var (
	backgroundProcesses = make(map[string]*ProcessInfo)
	processMutex        sync.Mutex
	ansiRegex           = regexp.MustCompile(ansiPattern)
	ansiPartialRegex    = regexp.MustCompile(ansiPartialPattern)
	// 进程退出后等待输出读完的上限，避免后台孙进程（如 "cmd &"）持有管道时一直无法标记完成
	pipeWaitDelay = 1 * time.Second
	// 已完成的后台进程保留时长，期间可通过 bash_output 查询最终状态与输出
	completedRetention = 10 * time.Minute
	// 前台命令超时自动转后台后，从启动算起的最长运行时间
	maxAutoBackgroundRuntime = 10 * time.Minute
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

// generateUUID 生成一个简单的UUID
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// AddBashTools 注册所有bash工具
func AddBashTools(server *mcp.Server) {
	// 启动定期清理协程
	startCleanupRoutine()

	// Bash工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bash",
		Description: "执行shell命令 - file-bash-tools.bash (MCP)(command: \"ls -la\", timeout: 30000, run_in_background: false) - timeout是必需参数（毫秒）；前台超时自动转后台，从启动算起最长运行10分钟；后台模式下timeout即最长运行时间；命令正常结束但用&留下的后台子进程不限时长，会返回shell ID供kill_shell停止",
	}, bashHandler)

	// BashOutput工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bash_output",
		Description: "获取后台进程输出 - file-bash-tools.bash_output (MCP)(bash_id: \"12345\", filter: \"error\") - 仅返回自上次查询以来的新输出，支持正则过滤；status: running/completed/failed/timed_out；已完成进程保留10分钟（仍有后台子进程时保留至其退出）",
	}, bashOutputHandler)

	// KillShell工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "kill_shell",
		Description: "终止后台进程 - file-bash-tools.kill_shell (MCP)(shell_id: \"12345\") - 优雅终止整个进程组（含命令启动的后台子进程）",
	}, killShellHandler)
}

// StopBashTools 停止bash工具相关资源
func StopBashTools() {
	stopCleanupRoutine()

	// 清理所有后台进程
	processMutex.Lock()
	defer processMutex.Unlock()

	// shell 已结束的记录也要终止：其进程组中可能还有后台子进程
	for id, processInfo := range backgroundProcesses {
		if processInfo.groupAlive() {
			_ = syscall.Kill(-processInfo.Cmd.Process.Pid, syscall.SIGKILL)
		}
		processInfo.release()
		delete(backgroundProcesses, id)
	}
}

// getShellCommand 返回执行命令所用的shell及参数
func getShellCommand(command string) (string, []string) {
	return "bash", []string{"-c", command}
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

	// 设置超时时间（已在上面校验为 1~600000ms 的有效值）
	timeout := time.Duration(params.Timeout) * time.Millisecond

	startTime := time.Now()

	// 获取shell命令
	shell, args := getShellCommand(params.Command)
	cmd := exec.Command(shell, args...)
	// 独立进程组，便于连同子进程一起终止
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// 创建进程信息的UUID
	shellID := generateUUID()

	// 创建进程信息对象
	processInfo := &ProcessInfo{
		Cmd:       cmd,
		Output:    NewLimitedBuffer(1024 * 100), // 100KB 限制
		StartTime: startTime,
		Done:      make(chan struct{}),
	}
	processInfo.setStatus("running")

	// 启动命令
	outputReader, drained, err := startWithOutputPipe(cmd, processInfo.Output)
	if err != nil {
		return nil, BashResult{}, fmt.Errorf("Failed to start command: %w", err)
	}
	processInfo.outputReader = outputReader
	processInfo.drained = drained

	// 立即注册到全局映射 (统一管理)
	processMutex.Lock()
	backgroundProcesses[shellID] = processInfo
	processMutex.Unlock()

	// 启动监管协程 (Supervisor)
	// 负责等待进程结束、更新状态、通知完成；
	// 已完成进程保留在映射中供 bash_output 查询最终状态/输出，由定期清理协程移除
	go func() {
		_ = cmd.Wait()
		// shell 退出前写入的输出都已在管道中，等读完；后台孙进程仍持有管道时最多等 pipeWaitDelay
		select {
		case <-drained:
		case <-time.After(pipeWaitDelay):
		}

		switch {
		case processInfo.timedOut.Load():
			processInfo.setStatus("timed_out")
		case cmd.ProcessState != nil && cmd.ProcessState.Success():
			processInfo.setStatus("completed")
		default:
			processInfo.setStatus("failed")
		}

		// 通知完成；顺带探测一次进程组，没有残留子进程时立即记录进程组已空
		processInfo.completedAt = time.Now()
		processInfo.groupAlive()
		close(processInfo.Done)
	}()

	// 根据模式处理等待逻辑
	if params.RunInBackground {
		// 模式 A: 明确后台运行
		// 显式后台模式下 timeout 即最长运行时间
		startTimeoutKiller(processInfo, timeout)

		// 立即返回
		return nil, BashResult{
			Output:   "",
			ExitCode: -1,
			ShellID:  shellID,
		}, nil
	}

	// 模式 B: 前台运行 (支持超时自动转后台)
	select {
	case <-processInfo.Done:
		// 情况 1: 在超时前完成
		chunk, offset, dropped := processInfo.Output.Since(0, nil)
		result := BashResult{
			Output:   withTruncationNote(cleanANSI(chunk), dropped),
			ExitCode: cmd.ProcessState.ExitCode(),
		}
		if !processInfo.groupAlive() {
			// 没有残留进程，输出已直接返回，无需保留
			removeProcess(shellID)
			return nil, result, nil
		}
		// 命令留下了后台子进程（如 "server &"），保留记录并返回 ID，供 bash_output / kill_shell 使用；
		// 已返回的输出不再由 bash_output 重复返回
		processInfo.Mutex.Lock()
		processInfo.readOffset = offset
		processInfo.Mutex.Unlock()
		result.ShellID = shellID
		result.Output += fmt.Sprintf("\n[Background processes started by this command are still running. Shell ID: %s (use bash_output to read their output, kill_shell to stop them)]", shellID)
		return nil, result, nil

	case <-ctx.Done():
		// 情况 2: 客户端取消请求 -> 终止整个进程组
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		removeProcess(shellID)
		return nil, BashResult{}, fmt.Errorf("command cancelled: %w", ctx.Err())

	case <-time.After(timeout):
		// 情况 3: 超时 -> 自动转为后台任务继续运行，从启动算起最长运行 maxAutoBackgroundRuntime；
		// 命令尚未结束，退出码为 -1
		remaining := time.Until(startTime.Add(maxAutoBackgroundRuntime))
		if remaining <= 0 {
			// 已无剩余运行时间：直接终止，如实返回已有输出
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			removeProcess(shellID)
			chunk, _, dropped := processInfo.Output.Since(0, nil)
			return nil, BashResult{
				Output:   fmt.Sprintf("⏱️ Command exceeded maximum runtime (%v) and was terminated.\n\n", maxAutoBackgroundRuntime) + withTruncationNote(cleanANSI(chunk), dropped),
				ExitCode: -1,
			}, nil
		}
		startTimeoutKiller(processInfo, remaining)
		msg := fmt.Sprintf("⏱️ Command exceeded timeout (%dms), automatically converted to background task (terminated if still running %v after start).\n\n✅ Task ID: %s", params.Timeout, maxAutoBackgroundRuntime, shellID)

		return nil, BashResult{
			Output:   msg,
			ExitCode: -1,
			ShellID:  shellID,
		}, nil
	}
}

// startWithOutputPipe 启动命令，stdout 与 stderr 共用一个管道写入 buf（保持输出交错顺序）。
// 读端由协程持续读取直到所有写端关闭：后台孙进程（如 "npm run dev &"）在命令结束后仍可继续输出，
// 不会因管道被关闭而在下次写入时被 SIGPIPE 杀死。返回管道读端，以及读取结束后关闭的通道
func startWithOutputPipe(cmd *exec.Cmd, buf io.Writer) (*os.File, <-chan struct{}, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	err = cmd.Start()
	pw.Close() // 子进程已继承写端，关闭本进程的副本，才能在所有子进程退出后读到 EOF
	if err != nil {
		pr.Close()
		return nil, nil, err
	}

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		defer pr.Close()
		_, _ = io.Copy(buf, pr)
	}()
	return pr, drained, nil
}

// startTimeoutKiller 在 d 之后若进程组仍有进程，终止整个进程组；shell 仍在运行时标记超时（状态由 Supervisor 更新）
func startTimeoutKiller(processInfo *ProcessInfo, d time.Duration) {
	go func() {
		// shell 提前结束也要等到时限：它启动的后台子进程同样受最长运行时间约束
		time.Sleep(d)
		pid := processInfo.Cmd.Process.Pid
		if !processInfo.groupAlive() {
			return
		}
		// shell 已退出时不算超时（状态以 shell 为准），只清理残留子进程
		if err := processInfo.Cmd.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
			processInfo.timedOut.Store(true)
		}
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}()
}

// withTruncationNote 有输出因超出缓冲区上限被丢弃时，在开头加上提示
func withTruncationNote(output string, dropped int) string {
	if dropped == 0 {
		return output
	}
	return fmt.Sprintf("[... %d bytes of earlier output truncated ...]\n", dropped) + output
}

// bashOutputHandler 处理获取后台进程输出 (完全符合todo.md标准)
func bashOutputHandler(ctx context.Context, req *mcp.CallToolRequest, params BashOutputParams) (*mcp.CallToolResult, BashOutputResult, error) {
	// 验证必需参数
	if params.BashID == "" {
		return nil, BashOutputResult{}, fmt.Errorf("bash_id parameter is required")
	}

	var filter *regexp.Regexp
	if params.Filter != "" {
		re, err := regexp.Compile(params.Filter)
		if err != nil {
			return nil, BashOutputResult{}, fmt.Errorf("invalid filter regex: %w", err)
		}
		filter = re
	}

	processInfo, exists := lookupProcess(params.BashID)
	if !exists {
		return nil, BashOutputResult{}, fmt.Errorf("Background process with bash_id %s not found", params.BashID)
	}

	// 先确定状态再读输出：Done 关闭时 Wait 已返回，输出已完整、ProcessState 已写入
	status := "running"
	exitCode := 0
	select {
	case <-processInfo.Done:
		status = processInfo.getStatus()
		exitCode = processInfo.Cmd.ProcessState.ExitCode()
	default:
	}

	processInfo.Mutex.Lock()
	// 输出是否已全部读完要看管道：shell 退出后，它留下的后台子进程仍可能继续写入
	var ready func([]byte) int
	select {
	case <-processInfo.drained:
	default:
		// filter 按行匹配，仍有写入方时只返回完整的行；不过滤时保留进度条等未换行输出的实时性
		ready = func(b []byte) int { return readyEnd(b, filter != nil) }
	}
	chunk, offset, dropped := processInfo.Output.Since(processInfo.readOffset, ready)
	processInfo.readOffset = offset
	processInfo.Mutex.Unlock()

	output := cleanANSI(chunk)
	if filter != nil {
		var filteredLines []string
		for _, line := range strings.Split(output, "\n") {
			if filter.MatchString(line) {
				filteredLines = append(filteredLines, line)
			}
		}
		output = strings.Join(filteredLines, "\n")
	}

	return nil, BashOutputResult{
		Output:   withTruncationNote(output, dropped),
		Status:   status,
		ExitCode: exitCode,
	}, nil
}

// lookupProcess 在全局映射中查找进程（仅在查找期间持锁）
func lookupProcess(id string) (*ProcessInfo, bool) {
	processMutex.Lock()
	defer processMutex.Unlock()
	processInfo, exists := backgroundProcesses[id]
	return processInfo, exists
}

// removeProcess 从全局映射中移除进程
func removeProcess(id string) {
	processMutex.Lock()
	defer processMutex.Unlock()
	if processInfo, exists := backgroundProcesses[id]; exists {
		processInfo.release()
		delete(backgroundProcesses, id)
	}
}

// release 关闭输出管道读端，结束读取协程。仍持有写端的进程（如用 setsid 脱离进程组的后台进程）
// 下次写入时会收到 SIGPIPE，避免读取协程与缓冲区随其生命周期一直泄漏
func (pi *ProcessInfo) release() {
	_ = pi.outputReader.Close()
}

// groupAlive 进程组中是否还有存活（非僵尸）进程。一旦观察到进程组为空就永久记录、
// 不再探测：此后进程组 ID 可能被系统复用，继续探测会把无关进程组误判为存活并误杀
func (pi *ProcessInfo) groupAlive() bool {
	if pi.groupExited.Load() {
		return false
	}
	pgid := pi.Cmd.Process.Pid
	// kill 探测开销小但会把僵尸算作存活（如服务以 PID 1 运行时，被收养的孤儿进程退出后无人回收）
	if syscall.Kill(-pgid, 0) == nil && groupHasLiveProcess(pgid) {
		return true
	}
	pi.groupExited.Store(true)
	return false
}

// groupHasLiveProcess 扫描 /proc，判断进程组中是否有状态不是僵尸（Z）的进程
func groupHasLiveProcess(pgid int) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return true // 无法判断时按存活处理，宁可多等也不提前判定为空
	}
	want := strconv.Itoa(pgid)
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		stat, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if err != nil {
			continue // 进程已退出
		}
		// 格式为 "pid (comm) state ppid pgrp ..."，comm 可能含空格或括号，从最后一个 ')' 之后解析
		fields := strings.Fields(string(stat[bytes.LastIndexByte(stat, ')')+1:]))
		if len(fields) > 2 && fields[2] == want && fields[0] != "Z" {
			return true
		}
	}
	return false
}

// waitGroupExit 等待进程组中的进程全部退出，超时返回 false
func waitGroupExit(processInfo *ProcessInfo, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for processInfo.groupAlive() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
	return true
}

// killShellHandler 处理终止进程 (完全符合todo.md标准)
func killShellHandler(ctx context.Context, req *mcp.CallToolRequest, params KillShellParams) (*mcp.CallToolResult, KillShellResult, error) {
	// 验证必需参数
	if params.ShellID == "" {
		return nil, KillShellResult{}, fmt.Errorf("shell_id parameter is required")
	}

	processInfo, exists := lookupProcess(params.ShellID)
	if !exists {
		return nil, KillShellResult{}, fmt.Errorf("Background process with shell_id %s not found", params.ShellID)
	}

	// shell 退出后，它启动的后台子进程（如 "npm run dev &"）仍在同一进程组中，
	// 因此以进程组是否还有进程为准，而不是 shell 是否结束
	pid := processInfo.Cmd.Process.Pid
	if !processInfo.groupAlive() {
		return nil, KillShellResult{
			Message: fmt.Sprintf("Process with shell_id %s has already completed", params.ShellID),
			ShellID: params.ShellID,
		}, nil
	}

	// 先 SIGTERM 整个进程组，5秒内未全部退出再 SIGKILL（等待期间不持全局锁）
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return nil, KillShellResult{}, fmt.Errorf("Failed to kill process with shell_id %s: %w", params.ShellID, err)
	}
	if !waitGroupExit(processInfo, 5*time.Second) {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		waitGroupExit(processInfo, 2*time.Second)
	}

	removeProcess(params.ShellID)

	return nil, KillShellResult{
		Message: fmt.Sprintf("Successfully killed background process with shell_id: %s", params.ShellID),
		ShellID: params.ShellID,
	}, nil
}
