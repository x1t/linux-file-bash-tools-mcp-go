package tools

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolvePathAbsolute 测试绝对路径解析
func TestResolvePathAbsolute(t *testing.T) {
	absolutePath := "/tmp/test.txt"
	
	result, err := resolvePath(absolutePath, "")
	
	assert.NoError(t, err, "Should not return error for absolute path")
	assert.Equal(t, filepath.Clean(absolutePath), result, "Should return cleaned absolute path")
}

// TestResolvePathWithBasePath 测试使用基准路径解析相对路径
func TestResolvePathWithBasePath(t *testing.T) {
	relativePath := "test.txt"
	basePath := "/tmp"
	expectedPath := filepath.Join(basePath, relativePath)
	
	result, err := resolvePath(relativePath, basePath)
	
	assert.NoError(t, err, "Should not return error when base path provided")
	// 使用filepath.Abs获取绝对路径进行比较
	expectedAbs, _ := filepath.Abs(expectedPath)
	resultAbs, _ := filepath.Abs(result)
	assert.Equal(t, expectedAbs, resultAbs, "Should join base path and relative path")
}

// TestResolvePathWithoutBasePath 测试没有基准路径时解析相对路径
func TestResolvePathWithoutBasePath(t *testing.T) {
	relativePath := "test.txt"
	
	result, err := resolvePath(relativePath, "")
	
	assert.NoError(t, err, "Should not return error for relative path without base")
	// 结果应该是相对于当前工作目录的绝对路径
	expected, _ := filepath.Abs(relativePath)
	assert.Equal(t, expected, result, "Should return absolute path relative to current directory")
}

// TestResolvePathEmptyPath 测试空路径
func TestResolvePathEmptyPath(t *testing.T) {
	result, err := resolvePath("", "")
	
	assert.NoError(t, err, "Should not return error for empty path")
	expected, _ := filepath.Abs("")
	assert.Equal(t, expected, result, "Should return current directory")
}

// TestResolvePathAbsoluteWithBase 测试绝对路径与基准路径（基准应被忽略）
func TestResolvePathAbsoluteWithBase(t *testing.T) {
	absolutePath := "/tmp/test.txt"
	basePath := "/some/other/path"
	
	result, err := resolvePath(absolutePath, basePath)
	
	assert.NoError(t, err, "Should not return error for absolute path")
	assert.Equal(t, filepath.Clean(absolutePath), result, "Should return absolute path ignoring base path")
}

// TestResolvePathNestedRelativeWithBase 测试嵌套相对路径与基准路径
func TestResolvePathNestedRelativeWithBase(t *testing.T) {
	relativePath := "subdir/test.txt"
	basePath := "/tmp"
	expectedPath := filepath.Join(basePath, relativePath)
	
	result, err := resolvePath(relativePath, basePath)
	
	assert.NoError(t, err, "Should not return error for nested relative path")
	expectedAbs, _ := filepath.Abs(expectedPath)
	resultAbs, _ := filepath.Abs(result)
	assert.Equal(t, expectedAbs, resultAbs, "Should correctly join nested relative path with base path")
}