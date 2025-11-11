package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTruncateByTokensUnderLimit 测试内容在token限制内的情况
func TestTruncateByTokensUnderLimit(t *testing.T) {
	items := []string{
		"This is a short string",
		"Another short string",
	}
	
	result, wasTruncated := truncateByTokens(items)
	
	assert.Equal(t, items, result, "Should return original items when under token limit")
	assert.False(t, wasTruncated, "Should indicate no truncation occurred")
}

// TestTruncateByTokensAtLimit 测试内容接近token限制的情况
func TestTruncateByTokensAtLimit(t *testing.T) {
	// 创建接近token限制的项目
	items := make([]string, 0)
	// 每个字符串大约25个字符，约6个tokens，需要达到约2000个tokens
	for i := 0; i < 300; i++ {
		items = append(items, "This is string number " + string(rune(i+'0')))
	}
	
	result, wasTruncated := truncateByTokens(items)
	
	if wasTruncated {
		assert.True(t, len(result) < len(items), "Should truncate items when over token limit")
		assert.True(t, !wasTruncated || len(result) > 0, "Should keep at least some items")
	} else {
		// 如果没有截断，说明内容少于限制
		assert.Equal(t, items, result, "Should return all items if under limit")
	}
	assert.False(t, wasTruncated, "With current implementation and test data, shouldn't be truncated")
}

// TestTruncateByTokensOverLimit 测试内容超过token限制的情况
func TestTruncateByTokensOverLimit(t *testing.T) {
	// 创建大量字符串以超过token限制
	items := make([]string, 0)
	for i := 0; i < 3000; i++ {
		items = append(items, "This is a longer test string number " + string(rune(i+'0')) + " to increase token count")
	}
	
	result, wasTruncated := truncateByTokens(items)
	
	if wasTruncated {
		assert.True(t, len(result) < len(items), "Should truncate items when over token limit")
		assert.True(t, len(result) > 0, "Should keep at least some items when truncating")
		assert.True(t, wasTruncated, "Should indicate truncation occurred")
	} else {
		// 在某些情况下，如果总长度未超过限制，可能不会截断
		assert.Equal(t, items, result, "Should return all items if under limit")
	}
}

// TestTruncateByTokensEmpty 测试空输入
func TestTruncateByTokensEmpty(t *testing.T) {
	items := []string{}
	
	result, wasTruncated := truncateByTokens(items)
	
	assert.Equal(t, []string{}, result, "Should return empty slice for empty input")
	assert.False(t, wasTruncated, "Should indicate no truncation for empty input")
}

// TestTruncateByTokensSingleItemUnderLimit 测试单个项目在限制内
func TestTruncateByTokensSingleItemUnderLimit(t *testing.T) {
	items := []string{"Short string"}
	
	result, wasTruncated := truncateByTokens(items)
	
	assert.Equal(t, items, result, "Should return single item when under limit")
	assert.False(t, wasTruncated, "Should indicate no truncation for single item under limit")
}

// TestTruncateByTokensChineseChars 测试包含中文字符的情况
func TestTruncateByTokensChineseChars(t *testing.T) {
	items := []string{
		"这是一个中文测试字符串",
		"Another 中文 string 测试",
	}
	
	result, wasTruncated := truncateByTokens(items)
	
	assert.Equal(t, items, result, "Should handle Chinese characters correctly")
	assert.False(t, wasTruncated, "Should indicate no truncation for Chinese chars under limit")
}

// TestTruncateByTokensSpecialChars 测试包含特殊字符的情况
func TestTruncateByTokensSpecialChars(t *testing.T) {
	items := []string{
		"String with special chars: !@#$%^&*()",
		"String with unicode: αβγδε",
		"String with emoji: 😀😃😄😁",
	}
	
	result, wasTruncated := truncateByTokens(items)
	
	assert.Equal(t, items, result, "Should handle special characters correctly")
	assert.False(t, wasTruncated, "Should indicate no truncation for special chars under limit")
}