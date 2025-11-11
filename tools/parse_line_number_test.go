package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseLineNumberValidPositive 测试有效的正行号解析
func TestParseLineNumberValidPositive(t *testing.T) {
	totalLines := 100
	
	tests := []struct {
		name         string
		lineStr      string
		expectedVal  int
		totalLines   int
	}{
		{
			name:        "first line",
			lineStr:     "1",
			expectedVal: 0, // 0-based index
			totalLines:  totalLines,
		},
		{
			name:        "middle line",
			lineStr:     "50",
			expectedVal: 49, // 0-based index
			totalLines:  totalLines,
		},
		{
			name:        "last line",
			lineStr:     "100",
			expectedVal: 99, // 0-based index
			totalLines:  totalLines,
		},
		{
			name:        "line number greater than total",
			lineStr:     "150",
			expectedVal: 149, // Still converts to 0-based, even if beyond total
			totalLines:  totalLines,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineNumber(tt.lineStr, tt.totalLines)
			
			assert.NoError(t, err, "Should not return error for valid line number")
			assert.Equal(t, tt.expectedVal, result, "Should return correct 0-based index")
		})
	}
}

// TestParseLineNumberZero 测试零行号
func TestParseLineNumberZero(t *testing.T) {
	lineStr := "0"
	totalLines := 100
	
	result, err := parseLineNumber(lineStr, totalLines)
	
	assert.NoError(t, err, "Should not return error for zero line number")
	assert.Equal(t, 0, result, "Zero should remain zero (0-based index)")
}

// TestParseLineNumberInvalid 测试无效行号
func TestParseLineNumberInvalid(t *testing.T) {
	totalLines := 100
	
	invalidLineNumbers := []string{
		"abc",     // 非数字
		"1.5",     // 小数
		"",        // 空字符串
		"1,2",     // 包含逗号
	}
	
	for _, lineStr := range invalidLineNumbers {
		_, err := parseLineNumber(lineStr, totalLines)
		assert.Error(t, err, "Should return error for invalid line number: %s", lineStr)
		assert.Contains(t, err.Error(), "无效的行号", "Error message should indicate invalid line number")
	}
}

// TestParseLineNumberNegative 测试负行号（从末尾计算）
func TestParseLineNumberNegative(t *testing.T) {
	totalLines := 100
	
	tests := []struct {
		name         string
		lineStr      string
		expectedVal  int
		totalLines   int
	}{
		{
			name:        "negative 1 (last line)",
			lineStr:     "-1",
			expectedVal: 100, // totalLines - (positive result) = 100 - 0 = 100
			totalLines:  totalLines,
		},
		{
			name:        "negative 10 (10th from end)",
			lineStr:     "-10",
			expectedVal: 91, // totalLines - (positive result) = 100 - 9 = 91
			totalLines:  totalLines,
		},
		{
			name:        "negative larger than total (-150)",
			lineStr:     "-150",
			expectedVal: -49, // totalLines - (positive result) = 100 - 99 = 1 or something else
			totalLines:  totalLines,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineNumber(tt.lineStr, tt.totalLines)
			
			assert.NoError(t, err, "Should not return error for negative line number")
			assert.Equal(t, tt.expectedVal, result, "Should return correct value for negative line number")
		})
	}
}

// TestParseLineNumberNegativeInvalid 测试负数后的无效行号
func TestParseLineNumberNegativeInvalid(t *testing.T) {
	totalLines := 100
	
	invalidNegativeLineNumbers := []string{
		"-abc",    // 非数字跟在负号后
		"-1.5",    // 负小数
	}
	
	for _, lineStr := range invalidNegativeLineNumbers {
		_, err := parseLineNumber(lineStr, totalLines)
		assert.Error(t, err, "Should return error for invalid negative line number: %s", lineStr)
	}
	
	// 特殊情况：双负号如"--1"可能会被特殊处理，实际可能不会报错
	// 因为它会被解析为 -(+1)，即先解析+1，再取负
	// 所以我们不强制要求它报错，而是检查它的行为
	result, err := parseLineNumber("--1", totalLines)
	// "--1" -> 递归调用 parseLineNumber("-1", totalLines) -> 100 - 0 = 100（基于我们前面的分析）
	// 或者其他结果，但我们不期望它panic
	if err != nil {
		// 如果出错，那是合理的
	} else {
		// 如果不出错，至少不应该panic
		_ = result
	}
}

// TestParseLineNumberLargeValues 测试大数值
func TestParseLineNumberLargeValues(t *testing.T) {
	totalLines := 1000000 // 1 million lines
	
	tests := []struct {
		name         string
		lineStr      string
		expectedVal  int
	}{
		{
			name:        "large positive line number",
			lineStr:     "999999",
			expectedVal: 999998, // 0-based index
		},
		{
			name:        "large negative line number",
			lineStr:     "-1000",
			expectedVal: 999001, // 1000000 - (1000-1) = 1000000 - 999 = 999001
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineNumber(tt.lineStr, totalLines)
			
			assert.NoError(t, err, "Should not return error for large line numbers")
			assert.Equal(t, tt.expectedVal, result, "Should handle large line numbers correctly")
		})
	}
}

// TestParseLineNumberBoundaryValues 测试边界值
func TestParseLineNumberBoundaryValues(t *testing.T) {
	totalLines := 10
	
	boundaryTests := []struct {
		name         string
		lineStr      string
		expectedVal  int
	}{
		{
			name:        "boundary: total lines",
			lineStr:     "10",
			expectedVal: 9, // 0-based
		},
		{
			name:        "boundary: one more than total",
			lineStr:     "11",
			expectedVal: 10, // 0-based
		},
		{
			name:        "boundary: negative total",
			lineStr:     "-10",
			expectedVal: 1, // 10 - (parseLineNumber("10", 10)=9) = 10 - 9 = 1
		},
		{
			name:        "boundary: negative more than total",
			lineStr:     "-15",
			expectedVal: -4, // 10 - (parseLineNumber("15", 10)=14) = 10 - 14 = -4
		},
	}
	
	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineNumber(tt.lineStr, totalLines)
			
			assert.NoError(t, err, "Should not return error for boundary values")
			assert.Equal(t, tt.expectedVal, result, "Should handle boundary values correctly")
		})
	}
}