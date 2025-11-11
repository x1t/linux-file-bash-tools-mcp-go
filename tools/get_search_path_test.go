package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetSearchPathFirstNonEmpty 测试返回第一个非空参数
func TestGetSearchPathFirstNonEmpty(t *testing.T) {
	result := getSearchPath("first", "second", "third")
	assert.Equal(t, "first", result, "Should return first non-empty parameter")
}

// TestGetSearchPathWithEmptyFirst 测试第一个参数为空时返回第二个
func TestGetSearchPathWithEmptyFirst(t *testing.T) {
	result := getSearchPath("", "second", "third")
	assert.Equal(t, "second", result, "Should return first non-empty parameter after empty")
}

// TestGetSearchPathWithMultipleEmpty 测试多个连续空参数后返回非空
func TestGetSearchPathWithMultipleEmpty(t *testing.T) {
	result := getSearchPath("", "", "third", "fourth")
	assert.Equal(t, "third", result, "Should return first non-empty parameter after multiple empty")
}

// TestGetSearchPathAllEmpty 测试所有参数都为空
func TestGetSearchPathAllEmpty(t *testing.T) {
	result := getSearchPath("", "", "")
	assert.Equal(t, "", result, "Should return empty string when all parameters are empty")
}

// TestGetSearchPathSingleEmpty 测试单个空参数
func TestGetSearchPathSingleEmpty(t *testing.T) {
	result := getSearchPath("")
	assert.Equal(t, "", result, "Should return empty string for single empty parameter")
}

// TestGetSearchPathSingleNonEmpty 测试单个非空参数
func TestGetSearchPathSingleNonEmpty(t *testing.T) {
	result := getSearchPath("only")
	assert.Equal(t, "only", result, "Should return single non-empty parameter")
}

// TestGetSearchPathWithEmptyAndNonEmptyComplex 测试复杂的参数组合
func TestGetSearchPathWithEmptyAndNonEmptyComplex(t *testing.T) {
	result := getSearchPath("", "", "", "found", "notChecked")
	assert.Equal(t, "found", result, "Should return first non-empty parameter regardless of following values")
}

// TestGetSearchPathNilInput 测试函数可变参数的边界情况
func TestGetSearchPathNilInput(t *testing.T) {
	result := getSearchPath() // 无参数
	assert.Equal(t, "", result, "Should return empty string when no parameters provided")
}