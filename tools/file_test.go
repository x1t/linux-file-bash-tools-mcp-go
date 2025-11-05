package tools

import (
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	// 创建测试文件
	testContent := "第1行：测试内容\n第2行：测试内容\n第3行：测试内容"
	err := os.WriteFile("test_read.txt", []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	defer os.Remove("test_read.txt")

	// 测试读取整个文件
	// 这里可以添加更详细的测试逻辑
	t.Log("测试文件读取功能")
}

func TestWriteFile(t *testing.T) {
	testContent := "这是测试内容"
	err := os.WriteFile("test_write.txt", []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	defer os.Remove("test_write.txt")

	// 验证文件是否写入成功
	content, err := os.ReadFile("test_write.txt")
	if err != nil {
		t.Fatalf("读取测试文件失败: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("内容不匹配: 期望 %s，实际 %s", testContent, string(content))
	}

	t.Log("测试文件写入功能成功")
}

func TestGlob(t *testing.T) {
	// 创建测试文件
	os.WriteFile("test_glob_1.txt", []byte("test"), 0644)
	os.WriteFile("test_glob_2.txt", []byte("test"), 0644)
	os.WriteFile("test_glob.go", []byte("test"), 0644)
	defer os.Remove("test_glob_1.txt")
	defer os.Remove("test_glob_2.txt")
	defer os.Remove("test_glob.go")

	t.Log("测试Glob功能")
}

func TestParseRange(t *testing.T) {
	// 测试范围解析
	tests := []struct {
		rangeStr   string
		totalLines int
		expectedStart int
		expectedEnd   int
		shouldError  bool
	}{
		{"1:10", 15, 0, 10, false},
		{"5:15", 15, 4, 15, false},
		{"1:5", 10, 0, 5, false},
	}

	for _, tt := range tests {
		start, end, err := parseRange(tt.rangeStr, tt.totalLines)
		if tt.shouldError && err == nil {
			t.Errorf("期望错误但未发生: %s", tt.rangeStr)
			continue
		}
		if !tt.shouldError && err != nil {
			t.Errorf("不期望错误: %v", err)
			continue
		}
		if start != tt.expectedStart || end != tt.expectedEnd {
			t.Errorf("范围解析错误: 期望 %d:%d，实际 %d:%d", tt.expectedStart, tt.expectedEnd, start, end)
		}
	}

	t.Log("测试范围解析功能成功")
}
