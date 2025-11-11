package tools

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetShellCommandBasedOnOS 测试根据操作系统获取shell命令
func TestGetShellCommandBasedOnOS(t *testing.T) {
	command := "echo test"
	
	shell, args := getShellCommand(command)
	
	// 根据当前操作系统验证返回的shell和参数
	switch runtime.GOOS {
	case "linux", "darwin": // Unix-like systems
		assert.Equal(t, "bash", shell, "Should return bash for Unix-like systems")
		assert.Contains(t, args, "-c", "Should use -c flag for bash")
		assert.Contains(t, args, command, "Should include the command in arguments")
	case "windows":
		// For Windows, it could be one of several shells
		assert.NotEmpty(t, shell, "Should return a shell for Windows")
		assert.Contains(t, shell, ".exe", "Windows shell should be an executable")
		assert.NotEmpty(t, args, "Should return arguments for Windows")
	default:
		// For other Unix-like systems, should default to bash
		assert.Equal(t, "bash", shell, "Should default to bash for other systems")
		assert.Contains(t, args, "-c", "Should use -c flag for other systems")
		assert.Contains(t, args, command, "Should include the command in arguments")
	}
}

// TestGetShellCommandEmpty 测试空命令
func TestGetShellCommandEmpty(t *testing.T) {
	command := ""
	
	shell, args := getShellCommand(command)
	
	// 根据当前操作系统验证返回的shell和参数
	switch runtime.GOOS {
	case "linux", "darwin": // Unix-like systems
		assert.Equal(t, "bash", shell, "Should return bash for Unix-like systems")
		assert.Contains(t, args, "-c", "Should use -c flag for bash")
		assert.Contains(t, args, command, "Should include the empty command in arguments")
	case "windows":
		// For Windows, it could be one of several shells
		assert.NotEmpty(t, shell, "Should return a shell for Windows")
		assert.Contains(t, shell, ".exe", "Windows shell should be an executable")
		assert.NotEmpty(t, args, "Should return arguments for Windows")
	default:
		// For other Unix-like systems, should default to bash
		assert.Equal(t, "bash", shell, "Should default to bash for other systems")
		assert.Contains(t, args, "-c", "Should use -c flag for other systems")
		assert.Contains(t, args, command, "Should include the command in arguments")
	}
}

// TestGetShellCommandWithSpecialChars 测试包含特殊字符的命令
func TestGetShellCommandWithSpecialChars(t *testing.T) {
	command := "echo 'Hello World' && ls -la | grep test"
	
	shell, args := getShellCommand(command)
	
	// 根据当前操作系统验证返回的shell和参数
	switch runtime.GOOS {
	case "linux", "darwin": // Unix-like systems
		assert.Equal(t, "bash", shell, "Should return bash for Unix-like systems")
		assert.Contains(t, args, "-c", "Should use -c flag for bash")
		assert.Contains(t, args, command, "Should include the full command in arguments")
	case "windows":
		// For Windows, it could be one of several shells
		assert.NotEmpty(t, shell, "Should return a shell for Windows")
		assert.Contains(t, shell, ".exe", "Windows shell should be an executable")
		assert.NotEmpty(t, args, "Should return arguments for Windows")
	default:
		// For other Unix-like systems, should default to bash
		assert.Equal(t, "bash", shell, "Should default to bash for other systems")
		assert.Contains(t, args, "-c", "Should use -c flag for other systems")
		assert.Contains(t, args, command, "Should include the command in arguments")
	}
}