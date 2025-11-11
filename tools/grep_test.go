package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestGrepSimpleString 测试简单字符串搜索
func TestGrepSimpleString(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:       "Hello",
		Path:          testDir,
		OutputMode:    "content",
		ShowLineNum:   true,
		HeadLimit:     10,
		CaseSensitive: false,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error")

	matches := grepResult.Matches
	assert.True(t, len(matches) > 0, "Should find at least one match for 'Hello'")

	foundHello := false
	for _, match := range matches {
		if strings.Contains(match, "Hello") {
			foundHello = true
			t.Logf("Found match: %s", match)
			break
		}
	}
	assert.True(t, foundHello, "Should find 'Hello' in the matches")
}

// TestGrepWithRegex 测试正则表达式搜索
func TestGrepWithRegex(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:     "func\\s+\\w+",
		Path:        testDir,
		Regex:       true,
		OutputMode:  "content",
		ShowLineNum: true,
		HeadLimit:   10,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error")

	matches := grepResult.Matches
	assert.True(t, len(matches) > 0, "Should find at least one function definition")

	t.Logf("Found %d function definitions", len(matches))
	for _, match := range matches {
		t.Logf("  %s", match)
	}
}

// TestGrepCaseSensitive 测试大小写敏感搜索
func TestGrepCaseSensitive(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}

	paramsInsensitive := GrepParams{
		Pattern:       "hello",
		Path:          testDir,
		CaseSensitive: false,
		OutputMode:    "content",
		HeadLimit:     10,
	}
	_, resultInsensitive, _ := grepHandler(context.Background(), req, paramsInsensitive)

	paramsSensitive := GrepParams{
		Pattern:       "Hello",
		Path:          testDir,
		CaseSensitive: true,
		OutputMode:    "content",
		HeadLimit:     10,
	}
	_, resultSensitive, _ := grepHandler(context.Background(), req, paramsSensitive)

	assert.True(t, len(resultInsensitive.Matches) >= len(resultSensitive.Matches),
		"Case insensitive search should find at least as many matches as case sensitive")

	t.Logf("Case insensitive matches: %d, Case sensitive matches: %d",
		len(resultInsensitive.Matches), len(resultSensitive.Matches))
}

// TestGrepOutputFilesWithMatches 测试只返回匹配的文件
func TestGrepOutputFilesWithMatches(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:     "package",
		Path:        testDir,
		OutputMode:  "files_with_matches",
		HeadLimit:   10,
		CaseSensitive: false,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error")

	matches := grepResult.Matches
	assert.True(t, len(matches) > 0, "Should find at least one file with 'package'")

	t.Logf("Found %d files containing 'package':", len(matches))
	for _, file := range matches {
		t.Logf("  - %s", file)
	}
}

// TestGrepSingleFile 测试搜索单个文件
func TestGrepSingleFile(t *testing.T) {
	testFile := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep", "file1_data.go")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:       "main",
		FilePath:      testFile,
		OutputMode:    "content",
		ShowLineNum:   true,
		HeadLimit:     10,
		CaseSensitive: false,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error")

	matches := grepResult.Matches
	assert.True(t, len(matches) > 0, "Should find 'main' in file1_data.go")

	for _, match := range matches {
		assert.Contains(t, match, "file1_data.go", "Match should be from file1_data.go")
	}

	t.Logf("Found %d matches in %s", len(matches), testFile)
	for _, match := range matches {
		t.Logf("  %s", match)
	}
}

// TestGrepNoMatch 测试无匹配的情况
func TestGrepNoMatch(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:     "THIS_SHOULD_NOT_EXIST",
		Path:        testDir,
		OutputMode:  "content",
		HeadLimit:   10,
		CaseSensitive: false,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error for no matches")

	matches := grepResult.Matches
	assert.Equal(t, 0, len(matches), "Should find no matches for non-existent pattern")

	t.Log("Correctly found no matches for non-existent pattern")
}

// TestGrepKeywordTest 测试搜索关键词 "test"
func TestGrepKeywordTest(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:       "test",
		Path:          testDir,
		OutputMode:    "content",
		ShowLineNum:   true,
		HeadLimit:     10,
		CaseSensitive: false,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error")

	matches := grepResult.Matches
	assert.True(t, len(matches) > 0, "Should find at least one match for 'test'")

	t.Logf("Found %d matches for 'test':", len(matches))
	for _, match := range matches {
		t.Logf("  %s", match)
	}
}

// TestGrepCountMode 测试计数模式
func TestGrepCountMode(t *testing.T) {
	testDir := filepath.Join(getTestProjectRoot(), "..", "testdata", "grep")

	req := &mcp.CallToolRequest{}
	params := GrepParams{
		Pattern:     "package",
		Path:        testDir,
		OutputMode:  "count",
		HeadLimit:   10,
		CaseSensitive: false,
	}

	_, grepResult, err := grepHandler(context.Background(), req, params)

	assert.NoError(t, err, "grepHandler should not return error")

	assert.True(t, grepResult.Count > 0, "Count should be greater than 0")

	t.Logf("Count mode: found %d occurrences of 'package'", grepResult.Count)
}
