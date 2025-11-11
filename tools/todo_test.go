package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestGetStringFromInterface 测试从interface获取字符串
func TestGetStringFromInterface(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "",
		},
		{
			name:     "string value",
			input:    "test string",
			expected: "test string",
		},
		{
			name:     "integer value",
			input:    42,
			expected: "42",
		},
		{
			name:     "float value",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "boolean value",
			input:    true,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringFromInterface(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCalculateStats 测试统计计算
func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name     string
		todos    []TodoItem
		expected TodoStats
	}{
		{
			name:     "empty todos",
			todos:    []TodoItem{},
			expected: TodoStats{Total: 0, Pending: 0, InProgress: 0, Completed: 0},
		},
		{
			name: "all pending",
			todos: []TodoItem{
				{Content: "Task 1", Status: "pending", ActiveForm: "Doing task 1"},
				{Content: "Task 2", Status: "pending", ActiveForm: "Doing task 2"},
			},
			expected: TodoStats{Total: 2, Pending: 2, InProgress: 0, Completed: 0},
		},
		{
			name: "all in progress",
			todos: []TodoItem{
				{Content: "Task 1", Status: "in_progress", ActiveForm: "Doing task 1"},
				{Content: "Task 2", Status: "in_progress", ActiveForm: "Doing task 2"},
			},
			expected: TodoStats{Total: 2, Pending: 0, InProgress: 2, Completed: 0},
		},
		{
			name: "all completed",
			todos: []TodoItem{
				{Content: "Task 1", Status: "completed", ActiveForm: "Did task 1"},
				{Content: "Task 2", Status: "completed", ActiveForm: "Did task 2"},
			},
			expected: TodoStats{Total: 2, Pending: 0, InProgress: 0, Completed: 2},
		},
		{
			name: "mixed statuses",
			todos: []TodoItem{
				{Content: "Task 1", Status: "pending", ActiveForm: "Doing task 1"},
				{Content: "Task 2", Status: "in_progress", ActiveForm: "Doing task 2"},
				{Content: "Task 3", Status: "completed", ActiveForm: "Did task 3"},
				{Content: "Task 4", Status: "pending", ActiveForm: "Doing task 4"},
			},
			expected: TodoStats{Total: 4, Pending: 2, InProgress: 1, Completed: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStats(tt.todos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTodoWriteHandlerWithArray 测试数组格式的todos参数
func TestTodoWriteHandlerWithArray(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: []interface{}{
			map[string]interface{}{
				"content":    "Task 1",
				"status":     "pending",
				"activeForm": "Doing task 1",
			},
			map[string]interface{}{
				"content":    "Task 2",
				"status":     "completed",
				"activeForm": "Did task 2",
			},
		},
	}

	// 创建模拟请求
	req := &mcp.CallToolRequest{}

	result, todoResult, err := todoWriteHandler(context.Background(), req, params)
	assert.NoError(t, err, "todoWriteHandler should not return error")
	// CallToolResult is nil on success
	assert.Nil(t, result, "CallToolResult should be nil on success")
	assert.Equal(t, 2, todoResult.Stats.Total)
	assert.Equal(t, 1, todoResult.Stats.Pending)
	assert.Equal(t, 0, todoResult.Stats.InProgress)
	assert.Equal(t, 1, todoResult.Stats.Completed)
	assert.Contains(t, todoResult.Message, "2 items")
}

// TestTodoWriteHandlerWithJSONString 测试JSON字符串格式的todos参数
func TestTodoWriteHandlerWithJSONString(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	jsonString := `[{"content":"Task 1","status":"in_progress","activeForm":"Doing task 1"},{"content":"Task 2","status":"pending","activeForm":"Doing task 2"}]`
	params := TodoWriteParams{
		Todos: jsonString,
	}

	req := &mcp.CallToolRequest{}

	result, todoResult, err := todoWriteHandler(context.Background(), req, params)
	assert.NoError(t, err, "todoWriteHandler should not return error for JSON string")
	assert.Nil(t, result, "CallToolResult should be nil on success")
	assert.Equal(t, 2, todoResult.Stats.Total)
	assert.Equal(t, 1, todoResult.Stats.InProgress)
	assert.Equal(t, 1, todoResult.Stats.Pending)
}

// TestTodoWriteHandlerWithDirectArray 测试直接TodoItem数组格式
func TestTodoWriteHandlerWithDirectArray(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	todos := []TodoItem{
		{Content: "Task 1", Status: "completed", ActiveForm: "Did task 1"},
		{Content: "Task 2", Status: "completed", ActiveForm: "Did task 2"},
		{Content: "Task 3", Status: "pending", ActiveForm: "Doing task 3"},
	}

	params := TodoWriteParams{
		Todos: todos,
	}

	req := &mcp.CallToolRequest{}

	result, todoResult, err := todoWriteHandler(context.Background(), req, params)
	assert.NoError(t, err, "todoWriteHandler should not return error for direct array")
	assert.Nil(t, result, "CallToolResult should be nil on success")
	assert.Equal(t, 3, todoResult.Stats.Total)
	assert.Equal(t, 1, todoResult.Stats.Pending)
	assert.Equal(t, 2, todoResult.Stats.Completed)
}

// TestTodoWriteHandlerInvalidStatus 测试无效状态值
func TestTodoWriteHandlerInvalidStatus(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: []interface{}{
			map[string]interface{}{
				"content":    "Task 1",
				"status":     "invalid_status",
				"activeForm": "Doing task 1",
			},
		},
	}

	req := &mcp.CallToolRequest{}

	_, _, err := todoWriteHandler(context.Background(), req, params)
	assert.Error(t, err, "todoWriteHandler should return error for invalid status")
	assert.Contains(t, err.Error(), "Invalid status value")
}

// TestTodoWriteHandlerEmptyContent 测试空content字段
func TestTodoWriteHandlerEmptyContent(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: []interface{}{
			map[string]interface{}{
				"content":    "",
				"status":     "pending",
				"activeForm": "Doing task 1",
			},
		},
	}

	req := &mcp.CallToolRequest{}

	_, _, err := todoWriteHandler(context.Background(), req, params)
	assert.Error(t, err, "todoWriteHandler should return error for empty content")
	assert.Contains(t, err.Error(), "content field cannot be empty")
}

// TestTodoWriteHandlerEmptyActiveForm 测试空activeForm字段
func TestTodoWriteHandlerEmptyActiveForm(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: []interface{}{
			map[string]interface{}{
				"content":    "Task 1",
				"status":     "pending",
				"activeForm": "",
			},
		},
	}

	req := &mcp.CallToolRequest{}

	_, _, err := todoWriteHandler(context.Background(), req, params)
	assert.Error(t, err, "todoWriteHandler should return error for empty activeForm")
	assert.Contains(t, err.Error(), "activeForm field cannot be empty")
}

// TestTodoWriteHandlerInvalidJSONString 测试无效JSON字符串
func TestTodoWriteHandlerInvalidJSONString(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: `{"invalid": json}`,
	}

	req := &mcp.CallToolRequest{}

	_, _, err := todoWriteHandler(context.Background(), req, params)
	assert.Error(t, err, "todoWriteHandler should return error for invalid JSON string")
	assert.Contains(t, err.Error(), "Failed to parse todos JSON string")
}

// TestTodoWriteHandlerInvalidItemFormat 测试无效的todo item格式
func TestTodoWriteHandlerInvalidItemFormat(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: []interface{}{
			"not a map", // 无效格式
		},
	}

	req := &mcp.CallToolRequest{}

	_, _, err := todoWriteHandler(context.Background(), req, params)
	assert.Error(t, err, "todoWriteHandler should return error for invalid item format")
	assert.Contains(t, err.Error(), "Invalid todo item format")
}

// TestTodoWriteHandlerInvalidParameterType 测试无效的参数类型
func TestTodoWriteHandlerInvalidParameterType(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	params := TodoWriteParams{
		Todos: 123, // 无效类型
	}

	req := &mcp.CallToolRequest{}

	_, _, err := todoWriteHandler(context.Background(), req, params)
	assert.Error(t, err, "todoWriteHandler should return error for invalid parameter type")
	assert.Contains(t, err.Error(), "Invalid todos parameter type")
}

// TestTodoWriteHandlerMultipleTodos 测试多个待办事项
func TestTodoWriteHandlerMultipleTodos(t *testing.T) {
	// 清理全局状态
	currentTodos = nil

	todos := []TodoItem{
		{Content: "Task 1", Status: "pending", ActiveForm: "Doing task 1"},
		{Content: "Task 2", Status: "in_progress", ActiveForm: "Doing task 2"},
		{Content: "Task 3", Status: "completed", ActiveForm: "Did task 3"},
		{Content: "Task 4", Status: "pending", ActiveForm: "Doing task 4"},
		{Content: "Task 5", Status: "in_progress", ActiveForm: "Doing task 5"},
	}

	params := TodoWriteParams{
		Todos: todos,
	}

	req := &mcp.CallToolRequest{}

	result, todoResult, err := todoWriteHandler(context.Background(), req, params)
	assert.NoError(t, err, "todoWriteHandler should not return error for multiple todos")
	assert.Nil(t, result, "CallToolResult should be nil on success")
	assert.Equal(t, 5, todoResult.Stats.Total)
	assert.Equal(t, 2, todoResult.Stats.Pending)
	assert.Equal(t, 2, todoResult.Stats.InProgress)
	assert.Equal(t, 1, todoResult.Stats.Completed)
}
