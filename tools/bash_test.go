package tools

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
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
	assert.Equal(t, -1, result.ExitCode, "Auto-backgrounded command has not finished, exit code should be -1")
	assert.NotEmpty(t, result.ShellID, "Should return a background shell ID")
	assert.Contains(t, result.Output, "automatically converted to background", "Output should mention auto-background conversion")

	// 清理转为后台的进程，避免测试间副作用
	_, _, _ = killShellHandler(context.Background(), req, KillShellParams{ShellID: result.ShellID})

	t.Logf("Timeout test - ShellID: %s", result.ShellID)
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
	assert.Equal(t, 0, cap(buf.data), "Buffer should not preallocate memory")
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
	shell, args := getShellCommand("echo test")
	assert.NotEmpty(t, shell, "Should return a shell command")
	assert.NotEmpty(t, args, "Should return shell arguments")

	assert.Equal(t, "bash", shell, "Should use bash")
	assert.Contains(t, args, "-c", "Should use -c flag for command")
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
	// 修复后：已完成进程保留在映射中，bash_output 应可靠返回 completed 状态与最终输出
	_, outputResult2, err2 := bashOutputHandler(context.Background(), outputReq, outputParams)
	assert.NoError(t, err2, "bashOutputHandler should not return error for completed process")
	assert.NotNil(t, outputResult2, "BashOutputResult should not be nil")
	assert.Equal(t, "completed", outputResult2.Status, "Process should be completed")
	assert.Equal(t, 0, outputResult2.ExitCode, "Exit code should be 0 for success")
	assert.Contains(t, outputResult2.Output, "Task completed", "Output should contain result")

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
		{
			name:     "With cursor control codes",
			input:    "\x1b[2K\x1b[1Gprogress 50%\x1b[?25l",
			expected: "progress 50%",
		},
		{
			name:     "With save/restore cursor and charset escapes",
			input:    "\x1b7saved\x1b8 \x1b(Bplain",
			expected: "saved plain",
		},
		{
			name:     "With OSC title sequences",
			input:    "\x1b]0;my title\x07text\x1b]2;t\x1b\\end",
			expected: "textend",
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

	// 使用无效的正则表达式，应返回错误而不是静默返回全部输出
	outputReq := &mcp.CallToolRequest{}
	outputParams := BashOutputParams{
		BashID: result.ShellID,
		Filter: "[invalid regex", // 无效的正则
	}

	_, _, err = bashOutputHandler(context.Background(), outputReq, outputParams)
	assert.Error(t, err, "bashOutputHandler should reject invalid regex")
	assert.Contains(t, err.Error(), "invalid filter regex")

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
	assert.NoError(t, err, "Killing a completed process should not error")
	assert.Contains(t, killResult.Message, "already completed")

	// 已完成的进程不应被移除，仍可查询最终输出
	_, outputResult, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: result.ShellID})
	assert.NoError(t, err)
	assert.Equal(t, "completed", outputResult.Status)
	assert.Contains(t, outputResult.Output, "done")

	// 清理
	cleanupBackgroundProcesses()
}

// TestBashOutputNotTruncated 回归测试：前台命令的输出必须完整，不能因 Wait 与读取并发而丢失
func TestBashOutputNotTruncated(t *testing.T) {
	for i := 0; i < 50; i++ {
		_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{Command: "seq 1 2000", Timeout: 10000})
		assert.NoError(t, err)
		if !assert.True(t, strings.HasSuffix(result.Output, "\n2000\n"), "run %d: output truncated, tail=%q", i, result.Output[max(0, len(result.Output)-20):]) {
			return
		}
	}
}

// TestBashLongLine 回归测试：超过 64KB 的单行输出不能导致命令被 SIGPIPE 杀死
func TestBashLongLine(t *testing.T) {
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command: "head -c 200000 /dev/zero | tr '\\0' a; echo; echo DONE",
		Timeout: 10000,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.True(t, strings.HasSuffix(result.Output, "\nDONE\n"), "output should end with DONE")
	assert.True(t, strings.HasPrefix(result.Output, "[... 97606 bytes of earlier output truncated ...]\n"), "truncation should be reported")
}

// TestBashInterleavedOutput stdout 与 stderr 应按实际写入顺序交错，且无多余空行
func TestBashInterleavedOutput(t *testing.T) {
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command: "echo out1; echo err >&2; echo out2",
		Timeout: 5000,
	})
	assert.NoError(t, err)
	assert.Equal(t, "out1\nerr\nout2\n", result.Output)
}

// TestBashOrphanChildDoesNotBlock 后台孙进程占住输出管道时，前台命令不能一直阻塞
func TestBashOrphanChildDoesNotBlock(t *testing.T) {
	start := time.Now()
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command: "sleep 1000 & echo $!",
		Timeout: 10000,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Less(t, time.Since(start), 5*time.Second, "should return shortly after shell exits")

	// 留下了后台子进程：应返回 ShellID 并提示，且可通过 kill_shell 终止
	assert.NotEmpty(t, result.ShellID, "leftover processes should keep a shell ID")
	assert.Contains(t, result.Output, "still running")
	orphan, err := strconv.Atoi(strings.SplitN(result.Output, "\n", 2)[0])
	if !assert.NoError(t, err) {
		return
	}
	_, output, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: result.ShellID})
	assert.NoError(t, err)
	assert.Equal(t, "", output.Output, "output already returned by bash must not be repeated")
	_, killResult, err := killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: result.ShellID})
	assert.NoError(t, err)
	assert.Contains(t, killResult.Message, "Successfully killed")
	assert.True(t, processGone(orphan), "orphan should be killed by kill_shell")
}

// TestKillShellDoesNotBlockOthers kill_shell 等待进程退出期间不能持有全局锁阻塞其他调用
func TestKillShellDoesNotBlockOthers(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "trap '' TERM; sleep 30",
		Timeout:         60000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	killDone := make(chan string, 1)
	go func() {
		_, res, _ := killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: bg.ShellID})
		killDone <- res.Message
	}()
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{Command: "echo ok", Timeout: 10000})
	assert.NoError(t, err)
	assert.Equal(t, "ok\n", result.Output)
	assert.Less(t, time.Since(start), 1*time.Second, "unrelated command must not wait for kill_shell")

	// SIGTERM 被忽略，最终应由 SIGKILL 终止
	assert.Contains(t, <-killDone, "Successfully killed")
	_, ok := lookupProcess(bg.ShellID)
	assert.False(t, ok, "killed process should be removed")
}

// TestLimitedBufferSince 增量读取：只返回偏移之后的数据，并报告被丢弃的字节数
func TestLimitedBufferSince(t *testing.T) {
	buf := NewLimitedBuffer(4)
	buf.Write([]byte("ab"))
	chunk, offset, dropped := buf.Since(0, nil)
	assert.Equal(t, "ab", chunk)
	assert.Equal(t, 2, offset)
	assert.Equal(t, 0, dropped)

	buf.Write([]byte("c"))
	chunk, offset, _ = buf.Since(offset, nil)
	assert.Equal(t, "c", chunk)
	assert.Equal(t, 3, offset)

	chunk, offset, _ = buf.Since(offset, nil)
	assert.Equal(t, "", chunk, "no new data")
	assert.Equal(t, 3, offset)

	// 超出上限后缓冲区只剩 "cdef"：从 0 读取时 "ab" 已被丢弃
	buf.Write([]byte("def"))
	chunk, offset, dropped = buf.Since(0, nil)
	assert.Equal(t, "cdef", chunk)
	assert.Equal(t, 6, offset)
	assert.Equal(t, 2, dropped)
	chunk, _, dropped = buf.Since(5, nil)
	assert.Equal(t, "f", chunk)
	assert.Equal(t, 0, dropped)
}

// TestLimitedBufferSinceUTF8 增量读取不能切断多字节字符
func TestLimitedBufferSinceUTF8(t *testing.T) {
	t.Run("HoldIncompleteTrailingRune", func(t *testing.T) {
		buf := NewLimitedBuffer(100)
		buf.Write([]byte("a\xe4\xbd")) // "你" = e4 bd a0，只写入前两个字节
		chunk, offset, _ := buf.Since(0, completeRunesEnd)
		assert.Equal(t, "a", chunk, "incomplete rune should be held back")
		assert.Equal(t, 1, offset)

		buf.Write([]byte("\xa0b"))
		chunk, offset, _ = buf.Since(offset, completeRunesEnd)
		assert.Equal(t, "你b", chunk)
		assert.Equal(t, 5, offset)
	})

	t.Run("FinalReturnsEverything", func(t *testing.T) {
		buf := NewLimitedBuffer(100)
		buf.Write([]byte("a\xe4\xbd"))
		chunk, offset, _ := buf.Since(0, nil)
		assert.Equal(t, "a\xe4\xbd", chunk, "final read must not hold back bytes forever")
		assert.Equal(t, 3, offset)
	})

	t.Run("SkipBrokenLeadingRuneAfterDrop", func(t *testing.T) {
		buf := NewLimitedBuffer(4)
		buf.Write([]byte("你好")) // e4 bd a0 e5 a5 bd，缓冲区只保留后 4 字节：a0 + "好"
		chunk, offset, dropped := buf.Since(0, completeRunesEnd)
		assert.Equal(t, "好", chunk)
		assert.Equal(t, 6, offset)
		assert.Equal(t, 3, dropped, "2 dropped bytes plus 1 skipped broken byte")
	})
}

// TestBashOutputIncremental bash_output 只返回自上次查询以来的新输出
func TestBashOutputIncremental(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "echo one; sleep 0.6; echo two",
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	params := BashOutputParams{BashID: bg.ShellID}

	time.Sleep(300 * time.Millisecond)
	_, first, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "running", first.Status)
	assert.Equal(t, "one\n", first.Output)

	time.Sleep(800 * time.Millisecond)
	_, second, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "completed", second.Status)
	assert.Equal(t, "two\n", second.Output, "already-read output must not be returned again")

	_, third, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "completed", third.Status)
	assert.Equal(t, "", third.Output)
}

// TestCleanupRetention 已完成进程在保留期内不被清理，超过保留期才移除
func TestCleanupRetention(t *testing.T) {
	newDone := func(completedAt time.Time) *ProcessInfo {
		processInfo := &ProcessInfo{Done: make(chan struct{}), completedAt: completedAt}
		processInfo.groupExited.Store(true) // 进程组已空
		close(processInfo.Done)
		return processInfo
	}
	processMutex.Lock()
	backgroundProcesses["retention-expired"] = newDone(time.Now().Add(-completedRetention - time.Minute))
	backgroundProcesses["retention-recent"] = newDone(time.Now().Add(-time.Minute))
	processMutex.Unlock()

	cleanupCompletedProcesses()

	_, expired := lookupProcess("retention-expired")
	_, recent := lookupProcess("retention-recent")
	assert.False(t, expired, "process past retention should be removed")
	assert.True(t, recent, "recently completed process should be kept")
	removeProcess("retention-recent")
}

// TestBashForegroundNotRetained 前台命令完成后直接返回输出，不再占用进程表
func TestBashForegroundNotRetained(t *testing.T) {
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{Command: "echo hi", Timeout: 5000})
	assert.NoError(t, err)
	assert.Equal(t, "hi\n", result.Output)
	assert.Empty(t, result.ShellID, "completed foreground command should not expose a shell ID")
}

// TestBashCancelKillsProcess 客户端通过 MCP 取消请求时，前台命令的整个进程组应被终止
func TestBashCancelKillsProcess(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "bash"}, bashHandler)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer session.Close()

	pidFile := t.TempDir() + "/pid"
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "bash",
		Arguments: map[string]any{"command": "sleep 30 & echo $! > " + pidFile + "; wait", "timeout": 60000},
	})
	assert.Error(t, err, "cancelled call should return an error")

	data, err := os.ReadFile(pidFile)
	if !assert.NoError(t, err) {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if !assert.NoError(t, err) {
		return
	}
	assert.Eventually(t, func() bool {
		return processGone(pid)
	}, 3*time.Second, 50*time.Millisecond, "grandchild process should be killed on cancel")
}

// TestBashOutputUTF8Split 多字节字符跨两次查询写入时，不能被切成乱码
func TestBashOutputUTF8Split(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         `printf 'x\xe4\xbd'; sleep 0.6; printf '\xa0\n'`,
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	params := BashOutputParams{BashID: bg.ShellID}

	time.Sleep(300 * time.Millisecond)
	_, first, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "x", first.Output)

	time.Sleep(800 * time.Millisecond)
	_, second, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "completed", second.Status)
	assert.Equal(t, "你\n", second.Output)
}

// TestBashOutputTruncationNote 增量读取期间有输出被丢弃时，应提示丢弃的字节数
func TestBashOutputTruncationNote(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "head -c 150000 /dev/zero | tr '\\0' a",
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	params := BashOutputParams{BashID: bg.ShellID}
	processInfo, _ := lookupProcess(bg.ShellID)
	select {
	case <-processInfo.Done:
	case <-time.After(5 * time.Second):
		t.Fatal("command did not finish")
	}

	_, result, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(result.Output, "[... 47600 bytes of earlier output truncated ...]\n"))
	assert.Equal(t, 1024*100, len(result.Output)-len("[... 47600 bytes of earlier output truncated ...]\n"))
}

// TestBashBackgroundTimeoutStatus 显式后台任务超时被终止时，状态为 timed_out
func TestBashBackgroundTimeoutStatus(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 30",
		Timeout:         300,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	assertEventuallyStatus(t, bg.ShellID, "timed_out")
}

// TestBashAutoBackgroundMaxRuntime 超时自动转后台的命令，从启动算起超过最长运行时间后被终止
func TestBashAutoBackgroundMaxRuntime(t *testing.T) {
	original := maxAutoBackgroundRuntime
	maxAutoBackgroundRuntime = 1 * time.Second
	defer func() { maxAutoBackgroundRuntime = original }()

	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{Command: "sleep 30", Timeout: 300})
	assert.NoError(t, err)
	assert.Contains(t, result.Output, "automatically converted to background")

	_, output, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: result.ShellID})
	assert.NoError(t, err)
	assert.Equal(t, "running", output.Status, "should keep running before max runtime")
	assertEventuallyStatus(t, result.ShellID, "timed_out")
}

// TestStopBashToolsKillsOrphans 服务退出时，shell 已结束但留下的后台子进程也要被终止
func TestStopBashToolsKillsOrphans(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 1000 & echo $!",
		Timeout:         60000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	orphan := orphanPID(t, bg.ShellID)

	StopBashTools()

	_, exists := lookupProcess(bg.ShellID)
	assert.False(t, exists, "StopBashTools should clear all entries")
	assert.Eventually(t, func() bool { return processGone(orphan) }, 3*time.Second, 50*time.Millisecond, "orphan should be killed on shutdown")
}

// assertEventuallyStatus 等待后台进程进入指定状态
func assertEventuallyStatus(t *testing.T, shellID, want string) {
	t.Helper()
	assert.Eventually(t, func() bool {
		_, r, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: shellID})
		return err == nil && r.Status == want
	}, 5*time.Second, 100*time.Millisecond, "status should become %s", want)
}

// processGone 进程不存在或已成为僵尸（被杀后尚未被父进程回收）
func processGone(pid int) bool {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	return err != nil || strings.Contains(string(stat), ") Z ")
}

// TestReadyEnd 运行中进程的输出只返回可安全处理的前缀
func TestReadyEnd(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wholeLines bool
		expected   int
	}{
		{"complete text", "abc", false, 3},
		{"incomplete rune", "a\xe4\xbd", false, 1},
		{"incomplete CSI", "abc\x1b[3", false, 3},
		{"lone trailing ESC", "abc\x1b", false, 3},
		{"complete CSI", "abc\x1b[31m", false, 8},
		{"incomplete OSC", "a\x1b]0;title", false, 1},
		{"non-sequence ESC is released", "\x1b" + strings.Repeat("x", 10), false, 11},
		{"save cursor is not held", "50%\x1b7", false, 5},
		{"charset escape is not held", "a\x1b(B", false, 4},
		{"incomplete charset escape", "a\x1b(", false, 1},
		{"long unterminated OSC is released", "\x1b]" + strings.Repeat("x", maxPendingANSI), false, maxPendingANSI + 2},
		{"whole lines", "line1\nline2\npart", true, 12},
		{"no complete line", "partial", true, 0},
		{"overlong partial line is released", "a\n" + strings.Repeat("x", maxPendingLine), true, maxPendingLine + 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, readyEnd([]byte(tt.input), tt.wholeLines))
		})
	}
}

// TestBashOutputFilterSplitLine 一行被拆在两次查询之间时，filter 仍应匹配完整的行
func TestBashOutputFilterSplitLine(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "printf 'build err'; sleep 0.6; printf 'or: x\\nok\\n'",
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	params := BashOutputParams{BashID: bg.ShellID, Filter: "error"}

	time.Sleep(300 * time.Millisecond)
	_, first, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "running", first.Status)
	assert.Equal(t, "", first.Output, "incomplete line should be held back when filtering")

	time.Sleep(800 * time.Millisecond)
	_, second, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "build error: x", second.Output)
}

// TestBashOutputPartialLineWithoutFilter 不过滤时，未换行的输出（如进度条）应实时返回
func TestBashOutputPartialLineWithoutFilter(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "printf 'progress 50%%'; sleep 30",
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	defer killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: bg.ShellID})

	time.Sleep(300 * time.Millisecond)
	_, result, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: bg.ShellID})
	assert.NoError(t, err)
	assert.Equal(t, "progress 50%", result.Output)
}

// TestBashOutputANSISplit 被拆在两次查询之间的颜色码仍应被清理
func TestBashOutputANSISplit(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         `printf 'a\033[3'; sleep 0.6; printf '1mb\n'`,
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	params := BashOutputParams{BashID: bg.ShellID}

	time.Sleep(300 * time.Millisecond)
	_, first, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "a", first.Output)

	time.Sleep(800 * time.Millisecond)
	_, second, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "b\n", second.Output)
}

// TestBashAutoBackgroundNoRemainingRuntime 超时时已无剩余运行时间，应直接终止并如实返回
func TestBashAutoBackgroundNoRemainingRuntime(t *testing.T) {
	original := maxAutoBackgroundRuntime
	maxAutoBackgroundRuntime = 300 * time.Millisecond
	defer func() { maxAutoBackgroundRuntime = original }()

	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{Command: "echo $$; sleep 30", Timeout: 300})
	assert.NoError(t, err)
	assert.Equal(t, -1, result.ExitCode)
	assert.Empty(t, result.ShellID, "terminated command should not expose a shell ID")
	assert.Contains(t, result.Output, "exceeded maximum runtime")
	assert.NotContains(t, result.Output, "converted to background")

	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	pid, err := strconv.Atoi(lines[len(lines)-1])
	if assert.NoError(t, err, "output should include the command's own output") {
		assert.Eventually(t, func() bool { return processGone(pid) }, 3*time.Second, 50*time.Millisecond)
	}
}

// TestBashBackgroundTimeoutAfterShellExit shell 已正常退出、仅残留子进程占用管道时触发超时，不应标记为 timed_out
func TestBashBackgroundTimeoutAfterShellExit(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 30 & exit 0",
		Timeout:         300,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	assertEventuallyStatus(t, bg.ShellID, "completed")
}

// TestBashOrphanChildSurvives 命令结束后，后台孙进程（如 "npm run dev &"）继续输出时不能被 SIGPIPE 杀死，
// 其后续输出仍可通过 bash_output 获取
func TestBashOrphanChildSurvives(t *testing.T) {
	aliveFile := t.TempDir() + "/alive"
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "(sleep 1.5; echo late; echo alive > " + aliveFile + "; sleep 30) & echo started",
		Timeout:         10000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	assertEventuallyStatus(t, bg.ShellID, "completed")

	time.Sleep(2 * time.Second)
	_, statErr := os.Stat(aliveFile)
	assert.NoError(t, statErr, "orphan child should survive writing to stdout")
	_, result, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: bg.ShellID})
	assert.NoError(t, err)
	assert.Equal(t, "late\n", result.Output, "orphan output should still be captured")

	// 清理残留的孙进程
	_, _, _ = killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: bg.ShellID})
}

// TestKillShellKillsOrphans shell 已结束、但留下后台子进程时，kill_shell 仍应终止整个进程组
func TestKillShellKillsOrphans(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 1000 & echo $!",
		Timeout:         60000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	orphan := orphanPID(t, bg.ShellID)

	_, killResult, err := killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: bg.ShellID})
	assert.NoError(t, err)
	assert.Contains(t, killResult.Message, "Successfully killed")
	assert.True(t, processGone(orphan), "orphan should be killed")
	_, exists := lookupProcess(bg.ShellID)
	assert.False(t, exists)
}

// TestBashBackgroundTimeoutKillsOrphans 后台模式的 timeout 同样约束 shell 留下的后台子进程
func TestBashBackgroundTimeoutKillsOrphans(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 1000 & echo $!",
		Timeout:         1500,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	orphan := orphanPID(t, bg.ShellID)
	assert.False(t, processGone(orphan), "orphan should still run before timeout")

	assert.Eventually(t, func() bool { return processGone(orphan) }, 3*time.Second, 50*time.Millisecond, "orphan should be killed at timeout")
	_, result, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: bg.ShellID})
	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status, "status reflects the shell, which exited normally")
}

// orphanPID 等待后台命令（输出 $! 后即退出的 shell）完成，返回其留下的后台子进程 PID
func orphanPID(t *testing.T, shellID string) int {
	t.Helper()
	var output string
	assert.Eventually(t, func() bool {
		_, r, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, BashOutputParams{BashID: shellID})
		output += r.Output
		return err == nil && r.Status == "completed"
	}, 5*time.Second, 50*time.Millisecond)
	pid, err := strconv.Atoi(strings.TrimSpace(output))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return pid
}

// TestGroupExitedIsSticky 观察到进程组为空后永久记录，此后即使进程组 ID 被复用也不再发信号
func TestGroupExitedIsSticky(t *testing.T) {
	// 没有残留子进程的命令：完成时即记录进程组已空
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{Command: "true", Timeout: 600000, RunInBackground: true})
	assert.NoError(t, err)
	assertEventuallyStatus(t, bg.ShellID, "completed")
	processInfo, ok := lookupProcess(bg.ShellID)
	if assert.True(t, ok) {
		assert.True(t, processInfo.groupExited.Load(), "empty group should be recorded on completion")
	}

	// 模拟进程组 ID 被复用：victim 是一个无关进程组，记录已标记进程组为空
	victim := exec.Command("sleep", "30")
	victim.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if !assert.NoError(t, victim.Start()) {
		return
	}
	defer func() {
		_ = victim.Process.Kill()
		_ = victim.Wait()
	}()
	reused := &ProcessInfo{Cmd: &exec.Cmd{Process: victim.Process}, Done: make(chan struct{})}
	reused.groupExited.Store(true)
	close(reused.Done)
	processMutex.Lock()
	backgroundProcesses["reused-pgid"] = reused
	processMutex.Unlock()

	_, killResult, err := killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: "reused-pgid"})
	assert.NoError(t, err)
	assert.Contains(t, killResult.Message, "already completed")
	StopBashTools()
	time.Sleep(100 * time.Millisecond)
	assert.False(t, processGone(victim.Process.Pid), "unrelated process group must not be signalled")
}

// TestCleanupKeepsRecordsWithLiveGroup 进程组中仍有后台子进程时，超过保留期也不清理记录
func TestCleanupKeepsRecordsWithLiveGroup(t *testing.T) {
	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 1000 & echo $!",
		Timeout:         60000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	orphan := orphanPID(t, bg.ShellID)
	processInfo, ok := lookupProcess(bg.ShellID)
	if !assert.True(t, ok) {
		return
	}
	<-processInfo.Done
	processInfo.completedAt = time.Now().Add(-completedRetention - time.Minute)

	cleanupCompletedProcesses()
	_, exists := lookupProcess(bg.ShellID)
	assert.True(t, exists, "record with live process group must be kept")

	_, killResult, err := killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: bg.ShellID})
	assert.NoError(t, err)
	assert.Contains(t, killResult.Message, "Successfully killed")
	assert.True(t, processGone(orphan))
}

// TestBashOutputSplitAfterShellExit shell 已退出但留下的后台子进程仍在写入时，多字节字符仍不能被切断
func TestBashOutputSplitAfterShellExit(t *testing.T) {
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command: `(sleep 1.5; printf 'x\xe4\xbd'; sleep 0.6; printf '\xa0\n'; sleep 1000) &`,
		Timeout: 10000,
	})
	assert.NoError(t, err)
	if !assert.NotEmpty(t, result.ShellID) {
		return
	}
	defer killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: result.ShellID})
	params := BashOutputParams{BashID: result.ShellID}

	time.Sleep(1 * time.Second) // 前台约在 1s 返回，此时已到第 ~2s
	_, first, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "completed", first.Status, "status reflects the shell")
	assert.Equal(t, "x", first.Output, "incomplete rune must be held while children still write")

	time.Sleep(800 * time.Millisecond)
	_, second, err := bashOutputHandler(context.Background(), &mcp.CallToolRequest{}, params)
	assert.NoError(t, err)
	assert.Equal(t, "你\n", second.Output)
}

// TestGroupAliveIgnoresZombies 进程组中只剩僵尸进程时视为已空（如服务以 PID 1 运行、孤儿进程无人回收）
func TestGroupAliveIgnoresZombies(t *testing.T) {
	// 设为子进程收割者：shell 留下的孤儿进程会被本进程收养，退出后成为无人回收的僵尸，模拟 PID 1 场景
	const prSetChildSubreaper = 36
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prSetChildSubreaper, 1, 0); errno != 0 {
		t.Fatalf("prctl: %v", errno)
	}
	defer syscall.RawSyscall(syscall.SYS_PRCTL, prSetChildSubreaper, 0, 0)

	_, bg, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command:         "sleep 0.3 & echo $!",
		Timeout:         60000,
		RunInBackground: true,
	})
	assert.NoError(t, err)
	orphan := orphanPID(t, bg.ShellID)
	defer syscall.Wait4(orphan, nil, 0, nil) // 回收被本进程收养的僵尸
	assert.Eventually(t, func() bool { return processGone(orphan) }, 3*time.Second, 50*time.Millisecond, "orphan should become a zombie")
	processInfo, ok := lookupProcess(bg.ShellID)
	if !assert.True(t, ok) {
		return
	}
	assert.NoError(t, syscall.Kill(-processInfo.Cmd.Process.Pid, 0), "zombie still makes kill() see the group")

	start := time.Now()
	_, killResult, err := killShellHandler(context.Background(), &mcp.CallToolRequest{}, KillShellParams{ShellID: bg.ShellID})
	assert.NoError(t, err)
	assert.Contains(t, killResult.Message, "already completed", "group with only zombies is empty")
	assert.Less(t, time.Since(start), 1*time.Second)
}

// TestBashDetachedProcessReleasesPipe 用 setsid 脱离进程组的进程不能让读取协程永久泄漏：记录移除后管道读端关闭
func TestBashDetachedProcessReleasesPipe(t *testing.T) {
	_, result, err := bashHandler(context.Background(), &mcp.CallToolRequest{}, BashParams{
		Command: `setsid sh -c 'echo $$; sleep 1.5; echo late; sleep 1000' & sleep 0.2`,
		Timeout: 10000,
	})
	assert.NoError(t, err)
	assert.Empty(t, result.ShellID, "detached process is outside the group, record should be removed")
	detached, err := strconv.Atoi(strings.TrimSpace(result.Output))
	if !assert.NoError(t, err) {
		return
	}
	defer syscall.Kill(-detached, syscall.SIGKILL)

	// 读端已关闭：脱离的进程下次写入时被 SIGPIPE 终止，而不是一直占着管道和读取协程
	assert.Eventually(t, func() bool { return processGone(detached) }, 4*time.Second, 50*time.Millisecond)
}
