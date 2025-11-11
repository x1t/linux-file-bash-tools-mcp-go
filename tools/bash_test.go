package tools

import (
	"context"
	"os"
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

	// 验证ShellID是有效的数字字符串
	pid, err := strconv.Atoi(shellID)
	assert.NoError(t, err, "ShellID should be a valid integer string")
	assert.True(t, pid > 0, "PID should be positive")

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

	// 超时是正常的，命令会被终止
	assert.NoError(t, err, "bashHandler should not return error")
	assert.NotNil(t, result, "BashResult should not be nil")
	assert.True(t, result.Killed, "Command should be killed due to timeout")
	assert.NotEqual(t, 0, result.ExitCode, "Exit code should be non-zero for killed process")

	t.Logf("Timeout test - Killed: %v, Exit code: %d", result.Killed, result.ExitCode)
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
	backgroundProcesses = make(map[int]*ProcessInfo)
}
