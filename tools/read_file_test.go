package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestReadFileHandlerValidFile 测试读取有效文件
func TestReadFileHandlerValidFile(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for valid file")
	assert.Equal(t, "1: Line 1\n2: Line 2\n3: Line 3\n4: ", result.Content, "Should return correct file content with line numbers")
	assert.Equal(t, 4, result.TotalLines, "Should return correct total lines count (including trailing newline)")
	assert.Equal(t, 4, result.LinesReturned, "Should return correct lines returned count")
}

// TestReadFileHandlerWithOffsetAndLimit 测试使用offset和limit参数
func TestReadFileHandlerWithOffsetAndLimit(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
		Offset:   2, // 开始于第2行
		Limit:    2, // 读取2行
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Contains(t, result.Content, "2: Line 2", "Should start from line 2")
	assert.Contains(t, result.Content, "3: Line 3", "Should include line 3")
	assert.Equal(t, 6, result.TotalLines, "Should return correct total lines (including trailing newline)")
	assert.Equal(t, 2, result.LinesReturned, "Should return correct lines returned count")
}

// TestReadFileHandlerWithoutLineNumbers 测试不显示行号的情况
func TestReadFileHandlerWithoutLineNumbers(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath:       testFilePath,
		ShowLineNumbers: false,
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Equal(t, testContent, result.Content, "Should return content without line numbers")
	assert.Equal(t, 4, result.TotalLines, "Should return correct total lines (including trailing newline)")
	assert.Equal(t, 4, result.LinesReturned, "Should return correct lines returned count")
}

// TestReadFileHandlerWithLineNumbersExplicitTrue 测试显式设置显示行号为true
func TestReadFileHandlerWithLineNumbersExplicitTrue(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath:       testFilePath,
		ShowLineNumbers: true,
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Contains(t, result.Content, "1: Line 1", "Should include line numbers")
	assert.Equal(t, 4, result.TotalLines, "Should return correct total lines (including trailing newline)")
	assert.Equal(t, 4, result.LinesReturned, "Should return correct lines returned count")
}

// TestReadFileHandlerWithLineNumbersDefault 测试默认显示行号（参数为nil）
func TestReadFileHandlerWithLineNumbersDefault(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
		// 不设置ShowLineNumbers，应该默认为true
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Contains(t, result.Content, "1: Line 1", "Should include line numbers by default")
	assert.Equal(t, 4, result.TotalLines, "Should return correct total lines (including trailing newline)")
	assert.Equal(t, 4, result.LinesReturned, "Should return correct lines returned count")
}

// TestReadFileHandlerEmptyFile 测试读取空文件
func TestReadFileHandlerEmptyFile(t *testing.T) {
	// 创建临时空文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "empty.txt")
	err := os.WriteFile(testFilePath, []byte(""), 0644)
	assert.NoError(t, err, "Should create empty test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for empty file")
	assert.Equal(t, "1: ", result.Content, "Should return single line with line number for empty file")
	assert.Equal(t, 1, result.TotalLines, "Should return 1 for empty file")
	assert.Equal(t, 1, result.LinesReturned, "Should return 1 for empty file")
}

// TestReadFileHandlerNonExistentFile 测试读取不存在的文件
func TestReadFileHandlerNonExistentFile(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: "/non/existent/file.txt",
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for non-existent file")
	assert.Contains(t, err.Error(), "Failed to read file", "Error message should indicate file read failure")
	assert.Equal(t, ReadResult{}, result, "Should return empty result on error")
}

// TestReadFileHandlerNonAbsolutePath 测试非绝对路径参数
func TestReadFileHandlerNonAbsolutePath(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: "relative/path/file.txt", // 非绝对路径
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for non-absolute path")
	assert.Contains(t, err.Error(), "file_path must be an absolute path", "Error message should indicate path requirement")
	assert.Equal(t, ReadResult{}, result, "Should return empty result on error")
}

// TestReadFileHandlerEmptyFilePath 测试空文件路径参数
func TestReadFileHandlerEmptyFilePath(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: "", // 空文件路径
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for empty file path")
	assert.Contains(t, err.Error(), "file_path parameter is required", "Error message should indicate required parameter")
	assert.Equal(t, ReadResult{}, result, "Should return empty result on error")
}

// TestReadFileHandlerWithLargeOffset 测试大offset值（超出文件行数）
func TestReadFileHandlerWithLargeOffset(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
		Offset:   10, // 大于文件总行数
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error even with large offset")
	assert.Equal(t, "4: ", result.Content, "Should return empty line with line number when offset is too large")
	assert.Equal(t, 3, result.TotalLines, "Should return correct total lines")
	assert.Equal(t, 0, result.LinesReturned, "Should return 0 lines returned when offset is beyond file")
}

// TestReadFileHandlerWithLimitOnly 测试仅使用limit参数
func TestReadFileHandlerWithLimitOnly(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
		Limit:    3, // 仅限制返回3行，从第1行开始
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Contains(t, result.Content, "1: Line 1", "Should start from line 1")
	assert.Contains(t, result.Content, "2: Line 2", "Should include line 2")
	assert.Contains(t, result.Content, "3: Line 3", "Should include line 3")
	assert.NotContains(t, result.Content, "4: Line 4", "Should not include line 4 due to limit")
	assert.Equal(t, 6, result.TotalLines, "Should return correct total lines (including trailing newline)")
	assert.Equal(t, 3, result.LinesReturned, "Should return correct lines returned count")
}

// TestReadFileHandlerWithOffsetBeyondTotalLines 测试offset超出总行数
func TestReadFileHandlerWithOffsetBeyondTotalLines(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\n"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := ReadParams{
		FilePath: testFilePath,
		Offset:   5, // 远大于文件总行数
		Limit:    2,
	}

	_, result, err := readFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Equal(t, "4: ", result.Content, "Should return empty content when offset is beyond total lines")
	assert.Equal(t, 3, result.TotalLines, "Should return correct total lines (including trailing newline)")
	assert.Equal(t, 0, result.LinesReturned, "Should return 0 lines when offset is beyond total lines")
}