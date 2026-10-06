package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetShellCommandBasedOnOS 测试根据操作系统获取shell命令
func TestGetShellCommandBasedOnOS(t *testing.T) {
	command := "echo test"
	
	shell, args := getShellCommand(command)
	
	assert.Equal(t, "bash", shell, "Should return bash")
	assert.Contains(t, args, "-c", "Should use -c flag")
	assert.Contains(t, args, command, "Should include the command in arguments")
}

// TestGetShellCommandEmpty 测试空命令
func TestGetShellCommandEmpty(t *testing.T) {
	command := ""
	
	shell, args := getShellCommand(command)
	
	assert.Equal(t, "bash", shell, "Should return bash")
	assert.Contains(t, args, "-c", "Should use -c flag")
	assert.Contains(t, args, command, "Should include the empty command in arguments")
}

// TestGetShellCommandWithSpecialChars 测试包含特殊字符的命令
func TestGetShellCommandWithSpecialChars(t *testing.T) {
	command := "echo 'Hello World' && ls -la | grep test"
	
	shell, args := getShellCommand(command)
	
	assert.Equal(t, "bash", shell, "Should return bash")
	assert.Contains(t, args, "-c", "Should use -c flag")
	assert.Contains(t, args, command, "Should include the full command in arguments")
}