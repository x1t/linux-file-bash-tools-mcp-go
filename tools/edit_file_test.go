package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestEditFileHandlerValidReplacement 测试有效替换
func TestEditFileHandlerValidReplacement(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit.txt")
	originalContent := "This is the original content.\nIt has multiple lines.\nThe original word appears here."
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "original",
		NewString: "modified",
		ReplaceAll: true, // 替换所有匹配项
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for valid replacement")
	assert.Contains(t, result.Message, "Successfully edited file", "Should return success message")
	assert.Equal(t, 2, result.Replacements, "Should return correct number of replacements")
	assert.Equal(t, testFilePath, result.FilePath, "Should return correct file path")

	// 验证文件内容是否被正确修改
	modifiedContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	expectedContent := "This is the modified content.\nIt has multiple lines.\nThe modified word appears here."
	assert.Equal(t, expectedContent, string(modifiedContent), "File should contain modified content")
}

// TestEditFileHandlerReplaceAll 测试替换所有匹配项
func TestEditFileHandlerReplaceAll(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_all.txt")
	originalContent := "original and original and another original"
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "original",
		NewString: "modified",
		ReplaceAll: true,
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Equal(t, 3, result.Replacements, "Should replace all occurrences")
	
	// 验证文件内容
	modifiedContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	expectedContent := "modified and modified and another modified"
	assert.Equal(t, expectedContent, string(modifiedContent), "Should replace all occurrences")
}

// TestEditFileHandlerReplaceFirstOnly 测试仅替换第一个匹配项
func TestEditFileHandlerReplaceFirstOnly(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_first.txt")
	originalContent := "original and original and another original"
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "original",
		NewString: "modified",
		ReplaceAll: false, // 默认行为，只替换第一个
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error")
	assert.Equal(t, 1, result.Replacements, "Should replace only the first occurrence")
	
	// 验证文件内容
	modifiedContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	expectedContent := "modified and original and another original"
	assert.Equal(t, expectedContent, string(modifiedContent), "Should replace only the first occurrence")
}

// TestEditFileHandlerNotFound 测试未找到要替换的字符串
func TestEditFileHandlerNotFound(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_not_found.txt")
	originalContent := "This is the original content."
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "nonexistent",
		NewString: "replacement",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error when old string is not found")
	assert.Contains(t, err.Error(), "not found in file", "Error message should indicate string not found")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
	
	// 验证文件内容未被修改
	unmodifiedContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	assert.Equal(t, originalContent, string(unmodifiedContent), "File should remain unmodified")
}

// TestEditFileHandlerEmptyFilePath 测试空文件路径
func TestEditFileHandlerEmptyFilePath(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  "", // 空文件路径
		OldString: "original",
		NewString: "replacement",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for empty file path")
	assert.Contains(t, err.Error(), "file_path parameter is required", "Error message should indicate required parameter")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
}

// TestEditFileHandlerEmptyOldString 测试空old_string参数
func TestEditFileHandlerEmptyOldString(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_empty_old.txt")
	originalContent := "This is content."
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "", // 空old_string
		NewString: "replacement",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for empty old_string")
	assert.Contains(t, err.Error(), "old_string and new_string parameters are required", "Error message should indicate required parameter")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
}

// TestEditFileHandlerEmptyNewString 测试空new_string参数
func TestEditFileHandlerEmptyNewString(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_empty_new.txt")
	originalContent := "This is content."
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "content",
		NewString: "", // 空new_string
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for empty new_string")
	assert.Contains(t, err.Error(), "old_string and new_string parameters are required", "Error message should indicate required parameter")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
}

// TestEditFileHandlerSameOldAndNew 测试old_string和new_string相同
func TestEditFileHandlerSameOldAndNew(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_same.txt")
	originalContent := "This is content."
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "content",
		NewString: "content", // 与old_string相同
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error when old_string and new_string are the same")
	assert.Contains(t, err.Error(), "old_string and new_string must be different", "Error message should indicate strings must be different")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
}

// TestEditFileHandlerNonExistentFile 测试不存在的文件
func TestEditFileHandlerNonExistentFile(t *testing.T) {
	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  "/non/existent/file.txt",
		OldString: "original",
		NewString: "replacement",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error for non-existent file")
	assert.Contains(t, err.Error(), "Failed to read file", "Error message should indicate file read failure")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
}

// TestEditFileHandlerWithSpecialCharacters 测试包含特殊字符的替换
func TestEditFileHandlerWithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_special.txt")
	originalContent := "Line with \"quotes\" and 'apostrophes'\nAnd 汉字 characters\nAnd emoji 😀"
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "汉字",
		NewString: " chinese",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for content with special characters")
	assert.Equal(t, 1, result.Replacements, "Should return correct number of replacements")
	
	// 验证文件内容
	modifiedContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	expectedContent := "Line with \"quotes\" and 'apostrophes'\nAnd  chinese characters\nAnd emoji 😀"
	assert.Equal(t, expectedContent, string(modifiedContent), "Should handle special characters correctly")
}

// TestEditFileHandlerNewlineReplacement 测试换行符相关的替换
func TestEditFileHandlerNewlineReplacement(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_newline.txt")
	originalContent := "Line 1\nLine 2\nLine 3\n"
	err := os.WriteFile(testFilePath, []byte(originalContent), 0644)
	assert.NoError(t, err, "Should create test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "Line 2\n",
		NewString: "Modified Line 2\n",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.NoError(t, err, "Should not return error for newline replacement")
	assert.Equal(t, 1, result.Replacements, "Should return correct number of replacements")
	
	// 验证文件内容
	modifiedContent, err := os.ReadFile(testFilePath)
	assert.NoError(t, err, "Should read file successfully")
	expectedContent := "Line 1\nModified Line 2\nLine 3\n"
	assert.Equal(t, expectedContent, string(modifiedContent), "Should handle newline replacement correctly")
}

// TestEditFileHandlerEmptyFile 测试空文件
func TestEditFileHandlerEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_edit_empty.txt")
	err := os.WriteFile(testFilePath, []byte(""), 0644)
	assert.NoError(t, err, "Should create empty test file successfully")

	req := &mcp.CallToolRequest{}
	params := EditParams{
		FilePath:  testFilePath,
		OldString: "original",
		NewString: "replacement",
	}

	_, result, err := editFileHandler(context.Background(), req, params)

	assert.Error(t, err, "Should return error when trying to replace in empty file without matches")
	assert.Contains(t, err.Error(), "not found in file", "Error message should indicate string not found")
	assert.Equal(t, EditResult{}, result, "Should return empty result on error")
}