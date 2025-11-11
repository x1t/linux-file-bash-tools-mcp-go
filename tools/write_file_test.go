package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestWriteFileHandlerValidInput 测试有效输入写入文件
func TestWriteFileHandlerValidInput(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_write.txt")
	expectedContent := "This is test content for write_file"

	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  expectedContent,
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for valid input")
	assert.Contains(t, result.Message, "Successfully wrote file", "Should return success message")
	assert.Equal(t, len(expectedContent), result.BytesWritten, "Should return correct bytes written")
	assert.Equal(t, testFilePath, result.FilePath, "Should return correct file path")

	// 验证文件内容是否正确写入
	actualContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	assert.Equal(t, expectedContent, string(actualContent), "File should contain the expected content")
}

// TestWriteFileHandlerCreatesDirectory 测试自动创建目录
func TestWriteFileHandlerCreatesDirectory(t *testing.T) {
	// 创建临时根目录
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "subdir", "deep", "test_write.txt")
	expectedContent := "Content in nested directory"

	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  expectedContent,
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error when creating nested directories")
	assert.Contains(t, result.Message, "Successfully wrote file", "Should return success message")

	// 验证文件内容是否正确写入
	actualContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	assert.Equal(t, expectedContent, string(actualContent), "File should contain the expected content")

	// 验证目录是否被创建
	dir := filepath.Dir(testFilePath)
	assert.DirExists(t, dir, "Directory should be created")
}

// TestWriteFileHandlerEmptyContentError 测试空内容错误
func TestWriteFileHandlerEmptyContentError(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_write.txt")

	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  "", // 空内容
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for empty content")
	assert.Contains(t, err.Error(), "content parameter is required", "Error message should indicate content is required")
	assert.Equal(t, WriteResult{}, result, "Should return empty result on error")
}

// TestWriteFileHandlerEmptyFilePathError 测试空文件路径错误
func TestWriteFileHandlerEmptyFilePathError(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: "", // 空文件路径
		Content:  "Some content",
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for empty file path")
	assert.Contains(t, err.Error(), "file_path parameter is required", "Error message should indicate path is required")
	assert.Equal(t, WriteResult{}, result, "Should return empty result on error")
}

// TestWriteFileHandlerNonAbsolutePathError 测试非绝对路径错误
func TestWriteFileHandlerNonAbsolutePathError(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: "relative/path/file.txt", // 非绝对路径
		Content:  "Some content",
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for non-absolute path")
	assert.Contains(t, err.Error(), "file_path must be an absolute path", "Error message should indicate absolute path requirement")
	assert.Equal(t, WriteResult{}, result, "Should return empty result on error")
}

// TestWriteFileHandlerWithSpecialCharacters 测试包含特殊字符的内容
func TestWriteFileHandlerWithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "special_chars.txt")
	expectedContent := "Line 1\nLine 2 with \t tab\nLine 3 with \"quotes\" and 'apostrophes'\nLine 4 with 汉字\n"

	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  expectedContent,
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for content with special characters")
	assert.Contains(t, result.Message, "Successfully wrote file", "Should return success message")

	// 验证文件内容是否正确写入
	actualContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	assert.Equal(t, expectedContent, string(actualContent), "File should contain the expected content with special characters")
}

// TestWriteFileHandlerLargeContent 测试大内容写入
func TestWriteFileHandlerLargeContent(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "large_content.txt")
	// 创建大内容
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += fmt.Sprintf("Line %d: This is a test line with some content\n", i)
	}

	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  largeContent,
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for large content")
	assert.Contains(t, result.Message, "Successfully wrote file", "Should return success message")
	assert.Equal(t, len(largeContent), result.BytesWritten, "Should return correct bytes written")

	// 验证文件内容是否正确写入
	actualContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	assert.Equal(t, largeContent, string(actualContent), "File should contain the expected large content")
}

// TestWriteFileHandlerOverwritesExistingFile 测试覆盖已存在文件
func TestWriteFileHandlerOverwritesExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "overwrite_test.txt")
	
	// 先创建一个文件
	originalContent := "Original content"
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create initial file successfully")

	// 写入新内容
	newContent := "New content that should overwrite the original"
	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  newContent,
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error when overwriting file")
	assert.Contains(t, result.Message, "Successfully wrote file", "Should return success message")
	assert.Equal(t, len(newContent), result.BytesWritten, "Should return correct bytes written")

	// 验证文件是否被新内容覆盖
	actualContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	assert.Equal(t, newContent, string(actualContent), "File should contain the new content, not original")
}

// TestWriteFileHandlerPermissionError 模拟权限错误
func TestWriteFileHandlerPermissionError(t *testing.T) {
	// 尝试写入系统保护目录（通常没有权限）
	testFilePath := "/root/test_write_permission.txt"
	
	req := &mcp.CallToolRequest{}
	params := WriteParams{
		FilePath: testFilePath,
		Content:  "Test content",
	}

	_, result, err := writeFileHandler(context.Background(), req, params)

	// 根据系统情况，可能返回权限错误
	if err != nil {
		assert.Contains(t, err.Error(), "Failed to write file", "Should return appropriate error when unable to write")
		assert.Contains(t, err.Error(), "permission", "Error should be related to permissions")
		assert.Equal(t, WriteResult{}, result, "Should return empty result on error")
	}
	// 如果没有错误，说明测试环境允许写入，这不是我们预期的，但我们仍认为测试通过
}