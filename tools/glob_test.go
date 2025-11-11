package tools
import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestGlobSimplePattern 测试简单模式匹配 *_data.go
func TestGlobSimplePattern(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "glob")
	
	req := &mcp.CallToolRequest{}
	params := GlobParams{
		Pattern:   "*_data.go",
		Path:      testDir,
		HeadLimit: 10,
	}

	_, globResult, err := globHandler(context.Background(), req, params)

	assert.NoError(t, err, "globHandler should not return error")
	assert.NotNil(t, globResult, "GlobResult should not be nil")

	files := globResult.Files
	assert.True(t, len(files) > 0, "Should find at least one _data.go file")

	for _, file := range files {
		assert.True(t, strings.HasSuffix(file, "_data.go"), "Found file should have _data.go extension")
	}

	t.Logf("Found %d _data.go files in pattern *_data.go", len(files))
	for _, file := range files {
		t.Logf("  - %s", file)
	}
}

// TestGlobNestedPattern 测试嵌套目录模式 **/*_data.go
func TestGlobNestedPattern(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "glob")

	req := &mcp.CallToolRequest{}
	params := GlobParams{
		Pattern:   "**/*_data.go",
		Path:      testDir,
		HeadLimit: 10,
	}

	_, globResult, err := globHandler(context.Background(), req, params)

	assert.NoError(t, err, "globHandler should not return error")
	assert.NotNil(t, globResult, "GlobResult should not be nil")

	files := globResult.Files
	assert.True(t, len(files) >= 2, "Should find at least 2 _data.go files")

	hasNestedFile := false
	for _, file := range files {
		if filepath.Base(filepath.Dir(file)) == "subdir" {
			hasNestedFile = true
			break
		}
	}
	assert.True(t, hasNestedFile, "Should find at least one file in subdir")

	t.Logf("Found %d _data.go files with pattern **/*_data.go", len(files))
	for _, file := range files {
		t.Logf("  - %s", file)
	}
}

// TestGlobTxtPattern 测试.txt文件匹配
func TestGlobTxtPattern(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "glob")

	req := &mcp.CallToolRequest{}
	params := GlobParams{
		Pattern:   "*.txt",
		Path:      testDir,
		HeadLimit: 10,
	}

	_, globResult, err := globHandler(context.Background(), req, params)

	assert.NoError(t, err, "globHandler should not return error")

	files := globResult.Files
	assert.Equal(t, 1, len(files), "Should find exactly 1 .txt file")
	assert.Equal(t, "test.txt", filepath.Base(files[0]))

	t.Logf("Found .txt file: %s", files[0])
}

// TestGlobNoMatches 测试无匹配的情况
func TestGlobNoMatches(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "glob")

	req := &mcp.CallToolRequest{}
	params := GlobParams{
		Pattern:   "*.xyz",
		Path:      testDir,
		HeadLimit: 10,
	}

	_, globResult, err := globHandler(context.Background(), req, params)

	assert.NoError(t, err, "globHandler should not return error")

	files := globResult.Files
	assert.Equal(t, 0, len(files), "Should find no files with pattern *.xyz")

	t.Log("Pattern *.xyz correctly returned no matches")
}

// getTestProjectRoot 获取项目根目录
func getTestProjectRoot() string {
	wd, _ := os.Getwd()
	return wd
}
