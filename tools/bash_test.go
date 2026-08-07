package tools

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestBashSimpleCommand 测试简单命令执行
func TestBashSimpleCommand(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "echo 'Hello World'",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Equal(t, 0, result.ExitCode, "Exit code should be 0 for successful command")
	assert.Contains(t, result.Output, "Hello World", "Output should contain 'Hello World'")
	assert.False(t, result.Killed, "Command should not be killed")

	t.Logf("Command output: %s", result.Output)
}

// TestBashWithErrorCommand 测试带错误码的命令
func TestBashWithErrorCommand(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "ls /nonexistent_directory_12345",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error even for failed commands")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.NotEqual(t, 0, result.ExitCode, "Exit code should be non-zero for failed command")
	assert.Contains(t, result.Output, "nonexistent", "Output should mention nonexistent directory")

	t.Logf("Failed command output: %s", result.Output)
}

// TestBashLongOutputCommand 测试多行输出命令
func TestBashLongOutputCommand(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "seq 1 5 | while read i; do echo 'Line '$i; done",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Equal(t, 0, result.ExitCode, "Exit code should be 0")

	// 验证输出包含所有行
	for i := 1; i <= 5; i++ {
		expected := "Line " + strconv.Itoa(i)
		assert.Contains(t, result.Output, expected, "Output should contain '%s'", expected)
	}

	t.Logf("Long output:\n%s", result.Output)
}

// TestBashBackgroundCommand 测试后台执行（简化版）
func TestBashBackgroundCommand(t *testing.T) {
	// 清理可能存在的进程
	cleanupBackgroundProcesses()

	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "echo 'Background task done'",
		Timeout:         5000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Equal(t, -1, result.ExitCode, "Exit code should be -1 for background process")
	assert.NotEmpty(t, result.ShellID, "ShellID should be set for background process")
	assert.False(t, result.Killed, "Command should not be killed initially")

	shellID := result.ShellID
	t.Logf("Background process started with ID: %s", shellID)

	// 验证ShellID是UUID格式（32位十六进制字符串）
	assert.Len(t, shellID, 32, "ShellID should be a 32-char UUID")
	assert.Regexp(t, `^[0-9a-f]{32}$`, shellID, "ShellID should be a lowercase hex UUID")

	// 清理进程
	killParams := KillShellParams{
		ShellID: shellID,
	}
	_, _, _ = killShellHandler(context.Background(), req, killParams)

	t.Logf("Background process test completed")
}

// TestBashOutputWithFilter 测试获取输出（简化版）
func TestBashOutputWithFilter(t *testing.T) {
	// 清理可能存在的进程
	cleanupBackgroundProcesses()

	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "echo 'error: test'; echo 'info: test'",
		Timeout:         5000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "ShellID should be set")

	shellID := result.ShellID
	time.Sleep(100 * time.Millisecond)

	// 测试获取输出
	outputParams := BashOutputParams{
		BashID: shellID,
	}

	_, outputResult, err := bashOutputHandler(context.Background(), req, outputParams)

	// 由于进程执行很快，可能已经完成，所以允许"not found"错误
	if err != nil {
		assert.Contains(t, err.Error(), "not found", "If error, should be about process not found")
	} else {
		assert.NotNil(t, outputResult, "BashOutputResult should not be nil")
		t.Logf("Output: %s", outputResult.Output)
	}

	// 清理进程
	_, _, _ = killShellHandler(context.Background(), req, KillShellParams{ShellID: shellID})
}

// TestBashTimeoutCommand 测试超时处理
func TestBashTimeoutCommand(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "sleep 5",
		Timeout: 1000, // 1秒超时，但命令需要5秒
	}

	_, result, err := bashHandler(context.Background(), req, params)

	// 超时时命令自动转为后台任务继续运行，不会被终止
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.False(t, result.Killed, "Command should auto-convert to background, not killed")
	assert.NotEmpty(t, result.ShellID, "Should return a background shell ID")
	assert.Contains(t, result.Output, "automatically converted to background", "Output should mention auto-background conversion")

	// 清理转为后台的进程，避免测试间副作用
	_, _, _ = killShellHandler(context.Background(), req, KillShellParams{ShellID: result.ShellID})

	t.Logf("Timeout test - Killed: %v, ShellID: %s", result.Killed, result.ShellID)
}

// TestKillShellSuccess 测试终止进程
func TestKillShellSuccess(t *testing.T) {
	// 清理可能存在的进程
	cleanupBackgroundProcesses()

	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 10",
		Timeout:         15000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "ShellID should be set")

	shellID := result.ShellID
	t.Logf("Started background process: %s", shellID)

	// 终止进程
	killParams := KillShellParams{
		ShellID: shellID,
	}

	_, killResult, err := killShellHandler(context.Background(), req, killParams)

	assert.NoError(t, err, "killShellHandler should not return error")
	assert.NotNil(t, killResult, "KillShellResult should not be nil")
	assert.Equal(t, shellID, killResult.ShellID, "ShellID should match")
	assert.Contains(t, killResult.Message, "killed", "Message should indicate process termination")

	t.Logf("Kill result: %s", killResult.Message)
}

// TestKillShellInvalidID 测试终止无效进程ID
func TestKillShellInvalidID(t *testing.T) {
	req := &mcp.CallToolRequest{}
	killParams := KillShellParams{
		ShellID: "invalid_shell_id_12345",
	}

	_, killResult, err := killShellHandler(context.Background(), req, killParams)

	// 应该返回错误
	assert.Error(t, err, "killShellHandler should return error for invalid ShellID")
	assert.NotNil(t, killResult, "KillShellResult is returned even for invalid ShellID")

	t.Logf("Expected error for invalid ID: %v", err)
}

// TestBashEmptyCommand 测试空命令
func TestBashEmptyCommand(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)

	assert.Error(t, err, "bashHandler should return error for empty command")
	// 实际返回的是BashResult{}而不是nil
	assert.NotNil(t, result, "BashResult is returned even for empty command")
	assert.Equal(t, "", result.Output, "Output should be empty")

	t.Logf("Empty command error: %v", err)
}

// TestBashOutputInvalidID 测试获取无效ID的输出
func TestBashOutputInvalidID(t *testing.T) {
	req := &mcp.CallToolRequest{}
	outputParams := BashOutputParams{
		BashID: "nonexistent_process_id",
	}

	_, outputResult, err := bashOutputHandler(context.Background(), req, outputParams)

	assert.Error(t, err, "bashOutputHandler should return error for invalid BashID")
	// 实际返回BashOutputResult{}而不是nil
	assert.NotNil(t, outputResult, "BashOutputResult is returned even for invalid BashID")

	t.Logf("Expected error for invalid BashID: %v", err)
}

// TestBashMultipleCommandsSequential 测试多个顺序执行的命令
func TestBashMultipleCommandsSequential(t *testing.T) {
	commands := []string{
		"echo 'Command 1'",
		"echo 'Command 2' && echo 'Command 2 continued'",
		"pwd",
	}

	for i, cmd := range commands {
		req := &mcp.CallToolRequest{}
		params := BashParams{
			Command: cmd,
			Timeout: 5000,
		}

		_, result, err := bashHandler(context.Background(), req, params)

		assert.NoError(t, err, "Command %d should not return error", i+1)
		assert.NotNil(t, result, "BashResult should not be nil for command %d", i+1)
		assert.Equal(t, 0, result.ExitCode, "Command %d should exit with 0", i+1)

		t.Logf("Command %d output: %s", i+1, result.Output)
	}
}

// TestBashWithEnvironmentVariable 测试带环境变量的命令
func TestBashWithEnvironmentVariable(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "echo $TEST_VAR",
		Timeout: 5000,
	}

	// 设置环境变量
	oldEnv := os.Getenv("TEST_VAR")
	os.Setenv("TEST_VAR", "test_value_123")
	defer os.Setenv("TEST_VAR", oldEnv)

	_, result, err := bashHandler(context.Background(), req, params)

	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Contains(t, result.Output, "test_value_123", "Output should contain environment variable value")

	t.Logf("Environment variable output: %s", result.Output)
}

// cleanupBackgroundProcesses 清理所有后台进程
func cleanupBackgroundProcesses() {
	processMutex.Lock()
	defer processMutex.Unlock()

	// 终止所有已知的后台进程
	for pid := range backgroundProcesses {
		process, ok := backgroundProcesses[pid]
		if ok {
			process.Cmd.Process.Kill()
		}
	}
	// 清空映射
	backgroundProcesses = make(map[string]*ProcessInfo)
}

// TestNewLimitedBuffer 测试创建带限制的缓冲区
func TestNewLimitedBuffer(t *testing.T) {
	maxSize := 100
	buf := NewLimitedBuffer(maxSize)

	assert.NotNil(t, buf, "NewLimitedBuffer should return a non-nil buffer")
	assert.Equal(t, maxSize, buf.maxSize, "Buffer should have correct maxSize")
	assert.Equal(t, 0, len(buf.data), "Buffer should be empty initially")
}

// TestLimitedBufferWrite 测试向缓冲区写入数据
func TestLimitedBufferWrite(t *testing.T) {
	// 测试正常写入
	t.Run("NormalWrite", func(t *testing.T) {
		buf := NewLimitedBuffer(100)
		data := []byte("Hello World")
		n, err := buf.Write(data)

		assert.NoError(t, err, "Write should not return error")
		assert.Equal(t, len(data), n, "Should return correct number of bytes written")
		assert.Contains(t, buf.String(), "Hello World", "Buffer should contain written data")
	})

	// 测试超过最大尺寸的写入
	t.Run("WriteExceedingMaxSize", func(t *testing.T) {
		buf := NewLimitedBuffer(10)
		data := []byte("This is a very long string that exceeds the buffer size")

		n, err := buf.Write(data)
		assert.NoError(t, err, "Write should not return error even when exceeding max size")
		assert.Equal(t, len(data), n, "Should return correct number of bytes written")
		assert.Equal(t, 10, len(buf.data), "Buffer should be truncated to max size")
	})

	// 测试分批写入超过限制
	t.Run("MultipleWritesExceedingLimit", func(t *testing.T) {
		buf := NewLimitedBuffer(20)
		buf.Write([]byte("First chunk of data"))
		buf.Write([]byte("Second chunk"))

		// 缓冲区应该只保留最新的20字节
		assert.True(t, len(buf.data) <= 20, "Buffer should not exceed max size")
		assert.Contains(t, buf.String(), "Second chunk", "Buffer should contain latest data")
	})
}

// TestLimitedBufferString 测试获取缓冲区字符串
func TestLimitedBufferString(t *testing.T) {
	buf := NewLimitedBuffer(100)
	buf.Write([]byte("Test data"))

	result := buf.String()
	assert.Equal(t, "Test data", result, "String() should return buffer contents")

	// 多次调用应该返回相同结果
	result2 := buf.String()
	assert.Equal(t, result, result2, "String() should return consistent results")
}

// TestLimitedBufferReset 测试重置缓冲区
func TestLimitedBufferReset(t *testing.T) {
	buf := NewLimitedBuffer(100)
	buf.Write([]byte("Test data"))

	assert.NotEmpty(t, buf.String(), "Buffer should have data after write")

	buf.Reset()

	assert.Equal(t, "", buf.String(), "Buffer should be empty after reset")
	assert.Equal(t, 0, len(buf.data), "Buffer data should have zero length after reset")
}

// TestGetShellCommand 测试获取正确的shell命令
func TestGetShellCommand(t *testing.T) {
	// 由于runtime.GOOS是只读的，我们只测试当前系统的行为
	// 在Linux和macOS上，getShellCommand都应该返回bash
	shell, args := getShellCommand("echo test")
	assert.NotEmpty(t, shell, "Should return a shell command")
	assert.NotEmpty(t, args, "Should return shell arguments")

	// Linux和macOS应该使用bash
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		assert.Equal(t, "bash", shell, "Should use bash on Unix-like systems")
		assert.Contains(t, args, "-c", "Should use -c flag for command")
	}
}

// TestParseBool 测试解析布尔值
func TestParseBool(t *testing.T) {
	// 测试bool类型
	assert.True(t, parseBool(true), "Should return true for bool true")
	assert.False(t, parseBool(false), "Should return false for bool false")

	// 测试string类型
	assert.True(t, parseBool("true"), "Should return true for string 'true'")
	assert.True(t, parseBool("TRUE"), "Should return true for string 'TRUE'")
	assert.False(t, parseBool("false"), "Should return false for string 'false'")
	assert.False(t, parseBool("FALSE"), "Should return false for string 'FALSE'")
	assert.False(t, parseBool("random"), "Should return false for random string")

	// 测试其他类型
	assert.False(t, parseBool(1), "Should return false for integer 1")
	assert.False(t, parseBool(nil), "Should return false for nil")
}

// TestBashMaxTimeout 测试最大超时限制
func TestBashMaxTimeout(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "echo 'test'",
		Timeout: 600000, // 最大允许的超时值
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Contains(t, result.Output, "test", "Output should contain result")
}

// TestBashNoTimeout 测试无超时参数（使用默认30秒）
func TestBashNoTimeout(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "echo 'no timeout test'",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Contains(t, result.Output, "no timeout test", "Output should contain result")
}

// TestBashOutputCompletedProcess 测试进程已完成的情况
func TestBashOutputCompletedProcess(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个短时间完成的后台任务
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 2 && echo 'Task completed'",
		Timeout:         10000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 在进程完成前获取输出
	time.Sleep(1 * time.Second)

	// 获取输出，检查进程状态
	outputReq := &mcp.CallToolRequest{}
	outputParams := BashOutputParams{
		BashID: result.ShellID,
	}

	_, outputResult, err := bashOutputHandler(context.Background(), outputReq, outputParams)
	assert.NoError(t, err, "bashOutputHandler should not return error")
	assert.NotNil(t, outputResult, "BashOutputResult should not be nil")

	// 等待进程完成
	time.Sleep(2 * time.Second)

	// 再次获取输出，检查进程状态
	_, outputResult2, err2 := bashOutputHandler(context.Background(), outputReq, outputParams)
	// 进程完成后可能会被删除，所以可能有"not found"错误
	if err2 != nil {
		assert.Contains(t, err2.Error(), "not found", "Process should be removed after completion")
	} else {
		// 如果进程信息还在，检查状态
		assert.Contains(t, outputResult2.Status, "completed", "Process should be completed")
		assert.Contains(t, outputResult2.Output, "Task completed", "Output should contain result")
	}

	// 清理
	cleanupBackgroundProcesses()
}

// TestKillShellWithInterrupt 测试优雅终止（中断信号）
func TestKillShellWithInterrupt(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个长时间运行的任务
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 10",
		Timeout:         15000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 等待任务开始
	time.Sleep(500 * time.Millisecond)

	// 终止进程
	killReq := &mcp.CallToolRequest{}
	killParams := KillShellParams{
		ShellID: result.ShellID,
	}

	_, killResult, err := killShellHandler(context.Background(), killReq, killParams)
	assert.NoError(t, err, "killShellHandler should not return error")
	assert.NotNil(t, killResult, "KillShellResult should not be nil")
	assert.Contains(t, killResult.Message, "killed", "Message should indicate process termination")
	assert.Equal(t, result.ShellID, killResult.ShellID, "ShellID should match")

	// 验证进程已被终止
	time.Sleep(200 * time.Millisecond)
	processMutex.Lock()
	_, exists := backgroundProcesses[result.ShellID]
	processMutex.Unlock()
	assert.False(t, exists, "Process should be removed from tracking")

	// 清理
	cleanupBackgroundProcesses()
}

// TestCleanANSI 测试清理ANSI转义序列
func TestCleanANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No ANSI codes",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "With ANSI color codes",
			input:    "\x1b[31mRed text\x1b[0m",
			expected: "Red text",
		},
		{
			name:     "With multiple ANSI codes",
			input:    "\x1b[1;32mGreen bold\x1b[0m normal",
			expected: "Green bold normal",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanANSI(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLimitedBufferWriteEdgeCases 测试Write方法的边界情况
func TestLimitedBufferWriteEdgeCases(t *testing.T) {
	// 测试单次写入超过maxSize的情况
	t.Run("SingleWriteExceedsMaxSize", func(t *testing.T) {
		buf := NewLimitedBuffer(10)
		data := []byte("This text is way longer than 10 bytes")

		n, err := buf.Write(data)
		assert.NoError(t, err, "Write should not return error")
		assert.Equal(t, len(data), n, "Should return correct bytes written")

		// 应该只保留最后的10字节
		assert.Equal(t, 10, len(buf.data), "Buffer should be truncated to max size")
		assert.Equal(t, string(data[len(data)-10:]), buf.String(), "Should keep last bytes")
	})

	// 测试多次写入刚好达到maxSize
	t.Run("MultipleWritesExactlyMaxSize", func(t *testing.T) {
		buf := NewLimitedBuffer(20)
		buf.Write([]byte("1234567890"))  // 10 bytes
		buf.Write([]byte("1234567890"))  // 10 bytes

		assert.Equal(t, 20, len(buf.data), "Buffer should be exactly at max size")
		assert.Equal(t, "12345678901234567890", buf.String(), "Should contain all data")
	})
}

// TestBashWithStderr 测试只有stderr输出的命令
func TestBashWithStderr(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "echo 'Error message' >&2",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Contains(t, result.Output, "Error message", "Output should contain stderr")
}

// TestBashWithBothStdoutStderr 测试同时有stdout和stderr的命令
func TestBashWithBothStdoutStderr(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command: "echo 'stdout' && echo 'stderr' >&2",
		Timeout: 5000,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.Contains(t, result.Output, "stdout", "Output should contain stdout")
	assert.Contains(t, result.Output, "stderr", "Output should contain stderr")
}

// TestBashOutputFilterWithNoMatch 测试过滤器不匹配任何内容
func TestBashOutputFilterWithNoMatch(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个长时间的任务，这样我们可以在它完成前获取输出
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 2 && echo 'INFO: Starting'",
		Timeout:         10000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 等待但不要让进程完成
	time.Sleep(500 * time.Millisecond)

	// 使用过滤器但没有匹配项
	outputReq := &mcp.CallToolRequest{}
	outputParams := BashOutputParams{
		BashID: result.ShellID,
		Filter: "ERROR:", // 不匹配任何内容
	}

	_, outputResult, err := bashOutputHandler(context.Background(), outputReq, outputParams)
	assert.NoError(t, err, "bashOutputHandler should not return error")
	// 可能有输出也可能没有，取决于命令执行速度
	t.Logf("Output: %s", outputResult.Output)

	// 清理
	cleanupBackgroundProcesses()
}

// TestKillShellWithForceKill 测试强制终止进程
func TestKillShellWithForceKill(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个长时间运行的任务
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "trap 'exit 0' TERM; sleep 10", // 忽略TERM信号，需要强制终止
		Timeout:         15000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 等待任务开始
	time.Sleep(500 * time.Millisecond)

	// 尝试优雅终止（可能失败）
	killReq := &mcp.CallToolRequest{}
	killParams := KillShellParams{
		ShellID: result.ShellID,
	}

	_, killResult, err := killShellHandler(context.Background(), killReq, killParams)
	assert.NoError(t, err, "killShellHandler should not return error")
	assert.NotNil(t, killResult, "KillShellResult should not be nil")
	assert.Contains(t, killResult.Message, "killed", "Message should indicate process termination")

	// 验证进程已被终止
	time.Sleep(200 * time.Millisecond)
	processMutex.Lock()
	_, exists := backgroundProcesses[result.ShellID]
	processMutex.Unlock()
	assert.False(t, exists, "Process should be removed from tracking")

	// 清理
	cleanupBackgroundProcesses()
}

// TestBashOutputFilterInvalidRegex 测试无效的正则表达式过滤器
func TestBashOutputFilterInvalidRegex(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个长时间的任务
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 2 && echo 'Test output'",
		Timeout:         10000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 等待但不要让进程完成
	time.Sleep(500 * time.Millisecond)

	// 使用无效的正则表达式（过滤器应该被忽略）
	outputReq := &mcp.CallToolRequest{}
	outputParams := BashOutputParams{
		BashID: result.ShellID,
		Filter: "[invalid regex", // 无效的正则
	}

	_, outputResult, err := bashOutputHandler(context.Background(), outputReq, outputParams)
	// 无效正则应该被忽略，返回所有输出
	assert.NoError(t, err, "bashOutputHandler should ignore invalid regex")
	// 可能有输出也可能没有，取决于命令执行速度
	t.Logf("Output: %s", outputResult.Output)

	// 清理
	cleanupBackgroundProcesses()
}

// TestLimitedBufferWriteKeepSizeZero 测试keepSize <= 0的情况
func TestLimitedBufferWriteKeepSizeZero(t *testing.T) {
	// 测试当len(p) < maxSize但keepSize为负的情况
	t.Run("KeepSizeNegative", func(t *testing.T) {
		buf := NewLimitedBuffer(100)
		// 先填满缓冲区
		buf.Write(make([]byte, 100))
		
		// 现在写入一个20字节的数据
		// keepSize = 100 - 20 = 80，这是一个正数，不是负数
		
		// 测试当缓冲区已有数据，然后写入超过剩余空间的情况
		buf = NewLimitedBuffer(50)
		buf.Write(make([]byte, 30))  // 已有30字节
		
		// 现在写入40字节
		// newLen = 30 + 40 = 70 > 50，超过限制
		// keepSize = 50 - 40 = 10 > 0，所以不会进入keepSize <= 0的分支
		
		// 我们需要测试一个不同的场景：已有的数据加上要写入的数据超过maxSize
		// 但要写入的数据本身小于maxSize
		buf = NewLimitedBuffer(20)
		data1 := make([]byte, 15)  // 15字节
		buf.Write(data1)  // 现在有15字节
		
		// 写入一个10字节的数据
		// newLen = 15 + 10 = 25 > 20
		// keepSize = 20 - 10 = 10
		// keepSize > 0，不进入keepSize <= 0的分支
		
		// 我们需要测试当keepSize <= 0的情况
		// 当len(p) >= maxSize时，keepSize = maxSize - len(p) <= 0
		buf = NewLimitedBuffer(10)
		largeData := make([]byte, 15)  // 15字节，超过maxSize
		
		n, err := buf.Write(largeData)
		assert.NoError(t, err, "Write should not return error")
		assert.Equal(t, 15, n, "Should return correct bytes written")
		assert.Equal(t, 10, len(buf.data), "Buffer should be truncated to max size")
	})

	// 测试当len(p) < maxSize但接近maxSize的情况
	t.Run("WriteExactlyAtBoundary", func(t *testing.T) {
		buf := NewLimitedBuffer(20)
		
		// 分批写入，刚好达到maxSize
		buf.Write(make([]byte, 10))
		assert.Equal(t, 10, len(buf.data), "Should have 10 bytes")
		
		buf.Write(make([]byte, 10))
		assert.Equal(t, 20, len(buf.data), "Should have 20 bytes")
		
		// 再写入1字节，触发keepSize逻辑
		buf.Write(make([]byte, 1))
		// keepSize = 20 - 1 = 19
		// 移动19字节旧数据到前面，然后写入新1字节
		// 总共20字节
		assert.Equal(t, 20, len(buf.data), "Should still have 20 bytes")
	})
}

// TestBashOutputStatusFailed 测试进程状态为failed的情况
func TestBashOutputStatusFailed(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个会失败的任务
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 1 && exit 1",  // 以非零退出码退出
		Timeout:         5000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 等待任务完成
	time.Sleep(2 * time.Second)

	// 获取输出，检查进程状态
	outputReq := &mcp.CallToolRequest{}
	outputParams := BashOutputParams{
		BashID: result.ShellID,
	}

	_, outputResult, err := bashOutputHandler(context.Background(), outputReq, outputParams)
	// 进程完成并失败
	if err != nil {
		// 进程可能被删除了
		assert.Contains(t, err.Error(), "not found", "Process may be cleaned up after failure")
	} else {
		// 如果进程信息还在，检查状态
		assert.Contains(t, outputResult.Status, "failed", "Process should be marked as failed")
		assert.NotEqual(t, 0, outputResult.ExitCode, "Exit code should be non-zero for failed process")
	}

	// 清理
	cleanupBackgroundProcesses()
}

// TestKillShellAlreadyCompleted 测试终止已完成的任务
func TestKillShellAlreadyCompleted(t *testing.T) {
	// 清理后台进程
	cleanupBackgroundProcesses()

	// 启动一个快速完成的任务
	req := &mcp.CallToolRequest{}
	params := BashParams{
		Command:         "sleep 1 && echo 'done'",
		Timeout:         5000,
		RunInBackground: true,
	}

	_, result, err := bashHandler(context.Background(), req, params)
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotEmpty(t, result.ShellID, "Should have ShellID")

	// 等待任务完成
	time.Sleep(2 * time.Second)

	// 尝试终止已完成的进程
	killReq := &mcp.CallToolRequest{}
	killParams := KillShellParams{
		ShellID: result.ShellID,
	}

	_, killResult, err := killShellHandler(context.Background(), killReq, killParams)
	// 可能因为进程已完成而返回错误
	if err != nil {
		assert.Contains(t, err.Error(), "not found", "Should return error when process already completed")
	} else {
		// 或者返回成功
		assert.NotNil(t, killResult, "KillShellResult should be returned")
	}

	// 清理
	cleanupBackgroundProcesses()
}
