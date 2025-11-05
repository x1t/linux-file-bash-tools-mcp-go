package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TodoItem 定义单个待办事项
type TodoItem struct {
	Content   string `json:"content" jsonschema:"任务描述" jsonschema:"required"`
	Status    string `json:"status" jsonschema:"任务状态 (pending|in_progress|completed)" jsonschema:"required"`
	ActiveForm string `json:"activeForm" jsonschema:"任务描述的主动形式" jsonschema:"required"`
}

// TodoWriteParams 定义TodoWrite工具参数
type TodoWriteParams struct {
	Todos interface{} `json:"todos" jsonschema:"更新后的待办事项列表" jsonschema:"required"`
}

// TodoStats 定义待办事项统计
type TodoStats struct {
	Total       int `json:"total"`
	Pending     int `json:"pending"`
	InProgress  int `json:"in_progress"`
	Completed   int `json:"completed"`
}

// TodoWriteResult 定义TodoWrite工具结果
type TodoWriteResult struct {
	Message string   `json:"message"`
	Stats   TodoStats `json:"stats"`
}

// 全局变量存储当前待办事项
var (
	currentTodos []TodoItem
	todoMutex    sync.Mutex
)

// AddTodoTools 注册TodoWrite工具
func AddTodoTools(server *mcp.Server) {
	// TodoWrite工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "todo_write",
		Description: "更新待办事项列表 - file-bash-tools.todo_write (MCP)(todos: [{content: \"任务描述\", status: \"pending\", activeForm: \"执行中任务\"}]) - 管理任务状态",
	}, todoWriteHandler)
}

// todoWriteHandler 处理TodoWrite命令
func todoWriteHandler(ctx context.Context, req *mcp.CallToolRequest, params TodoWriteParams) (*mcp.CallToolResult, TodoWriteResult, error) {
	todoMutex.Lock()
	defer todoMutex.Unlock()

	// 处理不同格式的todos参数
	var todosArray []TodoItem

	switch v := params.Todos.(type) {
	case string:
		// 如果todos是JSON字符串，解析它
		if err := json.Unmarshal([]byte(v), &todosArray); err != nil {
			return nil, TodoWriteResult{}, fmt.Errorf("Failed to parse todos JSON string: %w", err)
		}
	case []interface{}:
		// 如果todos是interface数组，转换为TodoItem数组
		todosArray = make([]TodoItem, len(v))
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				todosArray[i] = TodoItem{
					Content:   getStringFromInterface(itemMap["content"]),
					Status:    getStringFromInterface(itemMap["status"]),
					ActiveForm: getStringFromInterface(itemMap["activeForm"]),
				}
			} else {
				return nil, TodoWriteResult{}, fmt.Errorf("Invalid todo item format at index %d", i)
			}
		}
	case []TodoItem:
		// 如果已经是TodoItem数组，直接使用
		todosArray = v
	default:
		return nil, TodoWriteResult{}, fmt.Errorf("Invalid todos parameter type: expected string or array, got %T", params.Todos)
	}

	// 验证输入数据
	for i, todo := range todosArray {
		// 验证status字段
		if todo.Status != "pending" && todo.Status != "in_progress" && todo.Status != "completed" {
			return nil, TodoWriteResult{}, fmt.Errorf("Invalid status value: %s (must be pending, in_progress or completed)", todo.Status)
		}

		// 验证必填字段
		if todo.Content == "" {
			return nil, TodoWriteResult{}, fmt.Errorf("Todo[%d] content field cannot be empty", i)
		}
		if todo.ActiveForm == "" {
			return nil, TodoWriteResult{}, fmt.Errorf("Todo[%d] activeForm field cannot be empty", i)
		}
	}

	// 更新全局待办事项
	currentTodos = make([]TodoItem, len(todosArray))
	copy(currentTodos, todosArray)

	// 计算统计信息
	stats := calculateStats(currentTodos)

	return nil, TodoWriteResult{
		Message: fmt.Sprintf("Successfully updated todo list with %d items", stats.Total),
		Stats:   stats,
	}, nil
}

// getStringFromInterface 从interface{}安全地获取字符串值
func getStringFromInterface(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", v)
}

// calculateStats 计算待办事项统计信息
func calculateStats(todos []TodoItem) TodoStats {
	stats := TodoStats{}

	for _, todo := range todos {
		stats.Total++
		switch todo.Status {
		case "pending":
			stats.Pending++
		case "in_progress":
			stats.InProgress++
		case "completed":
			stats.Completed++
		}
	}

	return stats
}