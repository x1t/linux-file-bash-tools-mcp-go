package tools

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSearchLinesBasicStringMatch 测试基本字符串匹配
func TestSearchLinesBasicStringMatch(t *testing.T) {
	lines := []string{"Hello world", "Go programming", "Test string"}
	pattern := "Go"
	re := (*regexp.Regexp)(nil) // nil表示不使用正则
	caseSensitive := true
	ignoreCase := false
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Equal(t, 1, len(matches), "Should find 1 match")
	assert.Contains(t, matches[0], "Go programming", "Should match the correct line")
}

// TestSearchLinesCaseInsensitive 测试大小写不敏感匹配
func TestSearchLinesCaseInsensitive(t *testing.T) {
	lines := []string{"Hello World", "Go Programming", "test string"}
	pattern := "WORLD"
	re := (*regexp.Regexp)(nil)
	caseSensitive := false
	ignoreCase := true
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Equal(t, 1, len(matches), "Should find 1 match")
	assert.Contains(t, matches[0], "Hello World", "Should match case insensitive")
}

// TestSearchLinesRegexMatch 测试正则表达式匹配
func TestSearchLinesRegexMatch(t *testing.T) {
	lines := []string{"Hello 123", "Go Programming", "test 456"}
	pattern := "\\d+"
	re, _ := regexp.Compile(pattern)
	caseSensitive := false
	ignoreCase := false
	regex := true
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Greater(t, len(matches), 0, "Should find matches")
	assert.Contains(t, matches[0], "Hello 123", "Should match regex pattern")
}

// TestSearchLinesWithContext 测试上下文行
func TestSearchLinesWithContext(t *testing.T) {
	lines := []string{"Line 1", "Line 2", "Target line", "Line 4", "Line 5"}
	pattern := "Target"
	re := (*regexp.Regexp)(nil)
	caseSensitive := true
	ignoreCase := false
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 1
	contextAfter := 1
	context := 0 // 不使用统一上下文，使用独立的before/after
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	// 应该匹配包含"Target"的行以及其前后各1行，总共3行
	assert.Equal(t, 3, len(matches), "Should return 3 lines with context")
	assert.Contains(t, matches[1], "Target line", "Matched line should be in the middle")
	assert.Contains(t, matches[0], "Line 2", "Should include before context")
	assert.Contains(t, matches[2], "Line 4", "Should include after context")
}

// TestSearchLinesWithContextUnified 测试统一上下文
func TestSearchLinesWithContextUnified(t *testing.T) {
	lines := []string{"Line 1", "Line 2", "Line 3", "Target line", "Line 5", "Line 6", "Line 7"}
	pattern := "Target"
	re := (*regexp.Regexp)(nil)
	caseSensitive := true
	ignoreCase := false
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 0 // 不使用独立的before
	contextAfter := 0  // 不使用独立的after
	context := 2       // 使用统一上下文：前后各2行
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	// 应该返回匹配行及前后各2行，总共5行
	assert.Equal(t, 5, len(matches), "Should return 5 lines with unified context")
	assert.Contains(t, matches[2], "Target line", "Matched line should be in the middle")
}

// TestSearchLinesWithLineNumbers 测试显示行号
func TestSearchLinesWithLineNumbers(t *testing.T) {
	lines := []string{"Line 1", "Target here", "Line 3"}
	pattern := "Target"
	re := (*regexp.Regexp)(nil)
	caseSensitive := true
	ignoreCase := false
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := true  // 显示行号
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Equal(t, 1, len(matches), "Should find 1 match")
	assert.Contains(t, matches[0], "/test/file.txt:2: ", "Should include file path and line number")
	assert.Contains(t, matches[0], "Target here", "Should include matched content")
}

// TestSearchLinesFilesWithMatches 测试文件匹配模式
func TestSearchLinesFilesWithMatches(t *testing.T) {
	lines := []string{"Hello world", "Go programming", "Test string"}
	pattern := "Go"
	re := (*regexp.Regexp)(nil)
	caseSensitive := true
	ignoreCase := false
	regex := false
	filePath := "/test/file.txt"
	outputMode := "files_with_matches"  // 特殊模式
	showLineNum := false
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Equal(t, 0, len(matches), "Should not populate matches in files_with_matches mode")
	assert.Contains(t, matchedFiles, "/test/file.txt", "Should record file in matchedFiles")
	assert.Equal(t, true, matchedFiles["/test/file.txt"], "Should mark file as matched")
}

// TestSearchLinesNoMatch 测试无匹配情况
func TestSearchLinesNoMatch(t *testing.T) {
	lines := []string{"Hello world", "Go programming", "Test string"}
	pattern := "nonexistent"
	re := (*regexp.Regexp)(nil)
	caseSensitive := true
	ignoreCase := false
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Equal(t, 0, len(matches), "Should find no matches")
	assert.Equal(t, 0, len(matchedFiles), "Should not add file to matchedFiles")
}

// TestSearchLinesMultipleMatches 测试多个匹配
func TestSearchLinesMultipleMatches(t *testing.T) {
	lines := []string{"Test line", "Another test", "Test again", "No match here", "Final test"}
	pattern := "test"
	re := (*regexp.Regexp)(nil)
	caseSensitive := false
	ignoreCase := true
	regex := false
	filePath := "/test/file.txt"
	outputMode := "content"
	showLineNum := false
	contextBefore := 0
	contextAfter := 0
	context := 0
	
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	
	searchLines(lines, pattern, re, caseSensitive, ignoreCase, regex, filePath, outputMode, showLineNum, 
		contextBefore, contextAfter, context, &matches, &matchedFiles)
	
	assert.Equal(t, 4, len(matches), "Should find 4 matches (case insensitive finds 'Test', 'test', 'Test', 'test')")
}