package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadParams 定义读取文件参数 (完全符合todo.md标准)
type ReadParams struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path of the file to read"`
	Offset   int    `json:"offset,omitempty" jsonschema:"Starting line number (default: 1)"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Number of lines to read"`
	ShowLineNumbers interface{} `json:"show_line_numbers,omitempty" jsonschema:"Whether to show line numbers (default: true)"`
}

// WriteParams 定义写入文件参数 (完全符合todo.md标准)
type WriteParams struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path of the file to write"`
	Content  string `json:"content" jsonschema:"Content to write to the file"`
}

// EditParams 定义编辑文件参数 (完全符合todo.md标准)
type EditParams struct {
	FilePath  string `json:"file_path" jsonschema:"Path to the file to edit"`
	// todo.md标准参数
	OldString  string `json:"old_string" jsonschema:"String to be replaced"`
	NewString  string `json:"new_string" jsonschema:"String to replace with"`
	ReplaceAll bool `json:"replace_all,omitempty" jsonschema:"Replace all occurrences (default: false)"`
	// BasePath 基准目录，用于解析相对路径
	BasePath string `json:"basePath,omitempty" jsonschema:"Base directory for resolving relative paths"`
}

// GlobParams 定义glob参数
type GlobParams struct {
	Pattern string `json:"pattern" jsonschema:"File pattern to match (e.g., *.go, **/*.txt)"`
	// Path 搜索路径，默认为当前目录
	Path     string `json:"path,omitempty" jsonschema:"Directory to search in (default: current directory)"`
	Path1    string `json:"Path,omitempty" jsonschema:"Directory to search in (alternative naming)"`
	HeadLimit int   `json:"head_limit,omitempty" jsonschema:"Maximum number of results to return"`
	Offset    int   `json:"offset,omitempty" jsonschema:"Number of results to skip (for pagination)"`
	// BasePath 基准目录，用于解析相对路径
	BasePath string `json:"basePath,omitempty" jsonschema:"Base directory for resolving relative paths"`
}

// GrepParams 定义搜索参数
type GrepParams struct {
	Pattern       string      `json:"pattern" jsonschema:"Text or regex pattern to search for"`
	FilePath      string      `json:"file_path,omitempty" jsonschema:"Specific file to search in"`
	Path          string      `json:"path,omitempty" jsonschema:"Directory to search in"`
	GlobPattern   string      `json:"glob,omitempty" jsonschema:"Glob pattern to filter files (e.g., '*.js')"`
	FileType      string      `json:"type,omitempty" jsonschema:"File type to search (e.g., 'js', 'py', 'rust')"`
	OutputMode    string      `json:"output_mode,omitempty" jsonschema:"Output mode: 'content' | 'files_with_matches' | 'count'"`
	ShowLineNum   interface{} `json:"-n,omitempty" jsonschema:"Show line numbers in output"`
	CaseSensitive interface{} `json:"case_sensitive,omitempty" jsonschema:"Whether to perform case-sensitive search"`
	IgnoreCase    interface{} `json:"-i,omitempty" jsonschema:"Case insensitive search (alternative to caseSensitive)"`
	Regex         interface{} `json:"regex,omitempty" jsonschema:"Whether to use regex pattern"`
	Multiline     interface{} `json:"multiline,omitempty" jsonschema:"Enable multiline mode"`
	ContextBefore int         `json:"-B,omitempty" jsonschema:"Number of lines to show before match (context before)"`
	ContextAfter  int         `json:"-A,omitempty" jsonschema:"Number of lines to show after match (context after)"`
	Context       int         `json:"-C,omitempty" jsonschema:"Number of lines to show before and after match (context)"`
	HeadLimit     int         `json:"head_limit,omitempty" jsonschema:"Maximum number of results to return"`
	Offset        int         `json:"offset,omitempty" jsonschema:"Number of results to skip (for pagination)"`
	// BasePath 基准目录，用于解析相对路径
	BasePath      string      `json:"basePath,omitempty" jsonschema:"Base directory for resolving relative paths"`
}

// ReadResult 定义读取结果 (完全符合todo.md标准)
type ReadResult struct {
	Content      string `json:"content"`        // Text content with line numbers
	TotalLines   int    `json:"total_lines"`   // Total number of lines in file
	LinesReturned int   `json:"lines_returned"` // Number of lines actually returned
}

// WriteResult 定义写入结果 (完全符合todo.md标准)
type WriteResult struct {
	Message      string `json:"message"`       // Success message
	BytesWritten int    `json:"bytes_written"` // Number of bytes written
	FilePath     string `json:"file_path"`     // Absolute path of written file
}

// EditResult 定义编辑结果 (完全符合todo.md标准)
type EditResult struct {
	Message     string `json:"message"`
	Replacements int   `json:"replacements"`
	FilePath    string `json:"file_path"`
}

// GlobResult 定义glob结果
type GlobResult struct {
	Files []string `json:"files"`
	Count int      `json:"count"`
	Truncated bool   `json:"truncated,omitempty"`
}

// GrepResult 定义搜索结果
type GrepResult struct {
	Matches []string            `json:"matches"`
	Count   int                 `json:"count"`
	Truncated bool               `json:"truncated,omitempty"`
	Counts  []map[string]interface{} `json:"counts,omitempty"`  // For count mode
	Files   []string            `json:"files,omitempty"`    // For files_with_matches mode
}

// 常量定义
const (
	MAX_TOKENS = 8000
	// 估算token数量：大致按照字符数/4来估算（1个token约等于4个英文字符或2个中文字符）
	TOKEN_ESTIMATE_RATIO = 4
	// 最大文件大小（字节），防止读取过大的文件导致内存问题
	MAX_FILE_SIZE = 10 * 1024 * 1024 // 10MB
)

// AddFileTools 注册所有文件操作工具
func AddFileTools(server *mcp.Server) {
	// Read工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_file",
		Description: "读取文件内容，支持指定行范围 - file-bash-tools.read_file (MCP)(file_path: \"/root/XRSS/README.md\")",
	}, readFileHandler)

	// Write工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "写入内容到文件 - file-bash-tools.write_file (MCP)(file_path: \"/path/to/file.txt\", content: \"文件内容\") - 完全覆盖写入",
	}, writeFileHandler)

	// Edit工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "edit_file",
		Description: "编辑文件内容 - file-bash-tools.edit_file (MCP)(file_path: \"/path/to/file.txt\", old_string: \"原文本\", new_string: \"新文本\") - 完全替换指定文本",
	}, editFileHandler)

	// Glob工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "glob",
		Description: "根据模式匹配文件路径 - file-bash-tools.glob (MCP)(pattern: \"*.py\") ⎿  ⚠ Large MCP response (~23.5k tokens), this can fill up context quickly - 最多8000token，超出截断",
	}, globHandler)

	// Grep工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "grep",
		Description: "在文件或目录中搜索文本内容 - file-bash-tools.grep (MCP)(pattern: \"def\", path: \"/root/XRSS\", output_mode:\"files_with_matches\") - 最多8000token，超出截断",
	}, grepHandler)
}

// readFileHandler 处理文件读取请求 (完全符合todo.md标准)
func readFileHandler(ctx context.Context, req *mcp.CallToolRequest, params ReadParams) (*mcp.CallToolResult, ReadResult, error) {
	// 验证必需参数
	if params.FilePath == "" {
		return nil, ReadResult{}, fmt.Errorf("file_path parameter is required")
	}

	// 验证文件路径是绝对路径
	if !filepath.IsAbs(params.FilePath) {
		return nil, ReadResult{}, fmt.Errorf("file_path must be an absolute path")
	}

	// 执行安全路径检查
	if err := isPathInSafeZone(params.FilePath); err != nil {
		return nil, ReadResult{}, fmt.Errorf("安全路径检查失败: %w", err)
	}

	// 检查文件大小，防止读取过大的文件
	fileInfo, err := os.Stat(params.FilePath)
	if err != nil {
		// 如果文件不存在，保留原来的错误消息
		return nil, ReadResult{}, fmt.Errorf("Failed to read file: %w", err)
	}

	if fileInfo.Size() > MAX_FILE_SIZE {
		return nil, ReadResult{}, fmt.Errorf("文件过大 (%d bytes)，超过最大限制 %d bytes", fileInfo.Size(), MAX_FILE_SIZE)
	}

	// 读取文件
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return nil, ReadResult{}, fmt.Errorf("Failed to read file: %w", err)
	}

	text := string(content)
	allLines := strings.Split(text, "\n")
	totalLines := len(allLines)

	// 如果文件完全为空（0字节），则总行数为1
	if len(content) == 0 {
		totalLines = 1
		allLines = []string{""}
	}

	// 处理offset和limit参数 (符合todo.md标准)
	start := 0
	end := totalLines

	// 处理offset参数，确保在有效范围内
	if params.Offset > 0 {
		start = params.Offset - 1 // 转换为0基索引
		if start < 0 {
			start = 0
		}
		if start >= totalLines {
			start = totalLines
		}
	} else if params.Offset < 0 {
		// offset为负数时，从末尾开始计算
		start = totalLines + params.Offset
		if start < 0 {
			start = 0
		}
	}

	// 处理limit参数，确保在有效范围内
	if params.Limit > 0 {
		end = start + params.Limit
		if end > totalLines {
			end = totalLines
		}
	} else if params.Limit < 0 {
		// limit为负数时，忽略该参数
		end = totalLines
	}

	// 提取指定范围的行
	var selectedText string
	var linesReturned int

	if start >= totalLines {
		selectedText = ""
		linesReturned = 0
	} else {
		selectedLines := allLines[start:end]
		selectedText = strings.Join(selectedLines, "\n")
		linesReturned = len(selectedLines)
	}

	var contentWithLineNumbers string
	
	// 根据参数决定是否添加行号
	if parseBool(params.ShowLineNumbers) || params.ShowLineNumbers == nil {
		// 默认显示行号或当参数为true时显示行号
		var numberedLines []string
		for i, line := range strings.Split(selectedText, "\n") {
			lineNumber := start + i + 1 // 1基行号
			numberedLines = append(numberedLines, fmt.Sprintf("%d: %s", lineNumber, line))
		}
		contentWithLineNumbers = strings.Join(numberedLines, "\n")
	} else {
		// 不显示行号
		contentWithLineNumbers = selectedText
	}

	return nil, ReadResult{
		Content:       contentWithLineNumbers,
		TotalLines:    totalLines,
		LinesReturned: linesReturned,
	}, nil
}

// writeFileHandler 处理文件写入请求 (完全符合todo.md标准)
func writeFileHandler(ctx context.Context, req *mcp.CallToolRequest, params WriteParams) (*mcp.CallToolResult, WriteResult, error) {
	// 验证必需参数
	if params.FilePath == "" {
		return nil, WriteResult{}, fmt.Errorf("file_path parameter is required")
	}

	if params.Content == "" {
		return nil, WriteResult{}, fmt.Errorf("content parameter is required")
	}

	// 解析文件路径
	actualPath, err := resolvePath(params.FilePath, "")
	if err != nil {
		return nil, WriteResult{}, fmt.Errorf("解析文件路径失败: %w", err)
	}

	// 执行安全路径检查
	if err := isPathInSafeZone(actualPath); err != nil {
		return nil, WriteResult{}, fmt.Errorf("安全路径检查失败: %w", err)
	}

	// 检查写入内容的大小，防止写入过大的文件
	contentBytes := []byte(params.Content)
	if len(contentBytes) > MAX_FILE_SIZE {
		return nil, WriteResult{}, fmt.Errorf("内容过大 (%d bytes)，超过最大限制 %d bytes", len(contentBytes), MAX_FILE_SIZE)
	}

	// 确保目录存在
	dir := filepath.Dir(actualPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, WriteResult{}, fmt.Errorf("Failed to create directory: %w", err)
	}

	// 使用原子操作写入文件，防止数据损坏
	if err := atomicWriteFile(actualPath, contentBytes, 0644); err != nil {
		return nil, WriteResult{}, fmt.Errorf("Failed to write file: %w", err)
	}

	return nil, WriteResult{
		Message:      fmt.Sprintf("Successfully wrote file: %s", actualPath),
		BytesWritten: len(contentBytes),
		FilePath:     actualPath,
	}, nil
}

// editFileHandler 处理文件编辑请求 (完全符合todo.md标准)
func editFileHandler(ctx context.Context, req *mcp.CallToolRequest, params EditParams) (*mcp.CallToolResult, EditResult, error) {
	// 使用文件路径
	filePath := params.FilePath
	if filePath == "" {
		return nil, EditResult{}, fmt.Errorf("file_path parameter is required")
	}

	// 验证必需参数 (根据todo.md标准)
	if params.OldString == "" {
		return nil, EditResult{}, fmt.Errorf("old_string parameter is required")
	}
	if params.NewString == "" {
		return nil, EditResult{}, fmt.Errorf("new_string parameter is required")
	}

	// 验证old_string和new_string不相同
	if params.OldString == params.NewString {
		return nil, EditResult{}, fmt.Errorf("old_string and new_string must be different")
	}

	// 解析文件路径
	actualPath, err := resolvePath(filePath, params.BasePath)
	if err != nil {
		return nil, EditResult{}, fmt.Errorf("Failed to resolve path: %w", err)
	}

	// 执行安全路径检查
	if err := isPathInSafeZone(actualPath); err != nil {
		return nil, EditResult{}, fmt.Errorf("安全路径检查失败: %w", err)
	}

	// 检查文件是否存在且可读
	fileInfo, err := os.Stat(actualPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, EditResult{}, fmt.Errorf("文件不存在: %s", actualPath)
		}
		return nil, EditResult{}, fmt.Errorf("无法访问文件: %w", err)
	}

	// 检查文件权限
	if fileInfo.Mode().Perm()&0444 == 0 {
		return nil, EditResult{}, fmt.Errorf("文件不可读: %s", actualPath)
	}

	// 检查文件大小，防止处理过大的文件
	if fileInfo.Size() > MAX_FILE_SIZE {
		return nil, EditResult{}, fmt.Errorf("文件过大 (%d bytes)，超过最大限制 %d bytes", fileInfo.Size(), MAX_FILE_SIZE)
	}

	// 读取原文件
	content, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, EditResult{}, fmt.Errorf("Failed to read file: %w", err)
	}

	originalText := string(content)
	var newText string
	var replacements int

	// 执行字符串替换 (符合todo.md标准)
	if params.ReplaceAll {
		// 替换所有出现的old_string
		newText = strings.ReplaceAll(originalText, params.OldString, params.NewString)
		replacements = strings.Count(originalText, params.OldString)
	} else {
		// 只替换第一个出现的old_string
		newText = strings.Replace(originalText, params.OldString, params.NewString, 1)
		replacements = 1
		if !strings.Contains(originalText, params.OldString) {
			replacements = 0
		}
	}

	// 检查是否有替换发生
	if replacements == 0 {
		return nil, EditResult{}, fmt.Errorf("old_string '%s' not found in file", params.OldString)
	}

	// 检查编辑后内容的大小，防止写入过大的文件
	newContentBytes := []byte(newText)
	if len(newContentBytes) > MAX_FILE_SIZE {
		return nil, EditResult{}, fmt.Errorf("编辑后内容过大 (%d bytes)，超过最大限制 %d bytes", len(newContentBytes), MAX_FILE_SIZE)
	}

	// 检查文件是否可写
	if fileInfo.Mode().Perm()&0222 == 0 {
		return nil, EditResult{}, fmt.Errorf("文件不可写: %s", actualPath)
	}

	// 使用原子操作写回文件
	if err := atomicWriteFile(actualPath, newContentBytes, fileInfo.Mode().Perm()); err != nil {
		return nil, EditResult{}, fmt.Errorf("Failed to write file: %w", err)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(actualPath)
	if err != nil {
		absPath = actualPath
	}

	return nil, EditResult{
		Message:     fmt.Sprintf("Successfully edited file: %s", filePath),
		Replacements: replacements,
		FilePath:    absPath,
	}, nil
}

// globHandler 处理glob匹配请求
func globHandler(ctx context.Context, req *mcp.CallToolRequest, params GlobParams) (*mcp.CallToolResult, GlobResult, error) {
	// 验证必需参数
	if params.Pattern == "" {
		return nil, GlobResult{}, fmt.Errorf("pattern参数是必需的")
	}

	// 验证模式是否为空或只包含空白字符
	if strings.TrimSpace(params.Pattern) == "" {
		return nil, GlobResult{}, fmt.Errorf("模式不能为空或只包含空白字符")
	}

	// 从多个参数名中获取搜索路径
	path := getSearchPath(params.Path, params.Path1)

	// 解析搜索路径
	searchPath, err := resolvePath(path, params.BasePath)
	if err != nil {
		return nil, GlobResult{}, fmt.Errorf("解析搜索路径失败: %w", err)
	}

	// 验证搜索路径是否存在且为目录
	info, err := os.Stat(searchPath)
	if err != nil {
		return nil, GlobResult{}, fmt.Errorf("搜索路径不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, GlobResult{}, fmt.Errorf("搜索路径必须是目录: %s", searchPath)
	}

	// 验证模式是否有效
	_, err = doublestar.Match(params.Pattern, "test")
	if err != nil {
		return nil, GlobResult{}, fmt.Errorf("无效的glob模式: %w", err)
	}

	// 确保files不为nil
	files := make([]string, 0)
	fileInfos := make(map[string]fs.FileInfo)

	// 使用Walk遍历目录
	err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 跳过无法访问的文件/目录，而不是失败整个操作
			return nil
		}

		// 跳过目录，只处理文件
		if d.IsDir() {
			return nil
		}

		// 获取相对于搜索路径的相对路径
		relPath, err := filepath.Rel(searchPath, path)
		if err != nil {
			// 如果无法获取相对路径，使用完整路径
			relPath = path
		}

		// 使用 doublestar 进行高级模式匹配，支持 ** 等复杂模式
		matched, err := doublestar.Match(params.Pattern, relPath)
		if err != nil {
			// 如果模式无效，跳过（实际上上面已经验证过模式，所以这里不应该发生）
			return nil
		}
		if matched {
			files = append(files, path)
			// 获取文件信息用于排序
			if info, err := d.Info(); err == nil {
				fileInfos[path] = info
			}
		}

		return nil
	})

	// 按修改时间排序（最新的在前）- 使用更高效的排序算法
	sort.Slice(files, func(i, j int) bool {
		info1, exists1 := fileInfos[files[i]]
		info2, exists2 := fileInfos[files[j]]
		if exists1 && exists2 {
			return info1.ModTime().After(info2.ModTime())
		}
		return exists1 // 存在修改时间信息的排在前面
	})

	// 应用分页
	var pagedFiles []string
	start := params.Offset
	if start < 0 {
		start = 0
	}

	if start >= len(files) {
		pagedFiles = []string{}
	} else {
		end := len(files)
		if params.HeadLimit > 0 {
			end = start + params.HeadLimit
			if end > len(files) {
				end = len(files)
			}
		} else if params.HeadLimit < 0 {
			// head_limit为负数时，忽略该参数
			end = len(files)
		}
		pagedFiles = files[start:end]
	}

	// 应用token截断
	truncatedFiles, wasTruncated := truncateByTokens(pagedFiles)

	return nil, GlobResult{
		Files:     truncatedFiles,
		Count:     len(truncatedFiles),
		Truncated: wasTruncated,
	}, nil
}

// grepHandler 处理搜索请求
func grepHandler(ctx context.Context, req *mcp.CallToolRequest, params GrepParams) (*mcp.CallToolResult, GrepResult, error) {
	// 验证必需参数
	if params.Pattern == "" {
		return nil, GrepResult{}, fmt.Errorf("pattern参数是必需的")
	}

	// 确保matches不为nil
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	outputMode := params.OutputMode
	if outputMode == "" {
		outputMode = "content" // 默认输出模式
	}

	// 用于count模式的计数器
	matchCount := 0

	// 预先编译正则表达式，提高性能
	var re *regexp.Regexp
	if parseBool(params.Regex) {
		// 检查正则表达式复杂度，防止ReDoS攻击
		if err := isRegexComplexitySafe(params.Pattern); err != nil {
			return nil, GrepResult{}, fmt.Errorf("正则表达式复杂度检查失败: %w", err)
		}
		
		caseInsensitive := parseBool(params.IgnoreCase) || !parseBool(params.CaseSensitive)
		var err error
		if caseInsensitive {
			re, err = regexp.Compile("(?i)" + params.Pattern)
		} else {
			re, err = regexp.Compile(params.Pattern)
		}
		if err != nil {
			// 如果正则表达式编译失败，返回错误而不是回退到字符串匹配
			return nil, GrepResult{}, fmt.Errorf("正则表达式编译失败: %w", err)
		}
	}

	// 从参数中获取文件路径
	filePath := getFilePath(params.FilePath)

	if filePath != "" {
		// 在单个文件中搜索
		actualPath, err := resolvePath(filePath, params.BasePath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("解析文件路径失败: %w", err)
		}

		// 验证文件是否存在
		info, err := os.Stat(actualPath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("文件不存在: %w", err)
		}
		if info.IsDir() {
			return nil, GrepResult{}, fmt.Errorf("指定路径是目录，不是文件: %s", actualPath)
		}

		content, err := os.ReadFile(actualPath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("读取文件失败: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		matchCount += searchLines(lines, params.Pattern, re, parseBool(params.CaseSensitive), parseBool(params.IgnoreCase), parseBool(params.Regex), actualPath, outputMode, parseBool(params.ShowLineNum), params.ContextBefore, params.ContextAfter, params.Context, &matches, &matchedFiles)
	} else {
		// 从参数中获取搜索路径
		path := getSearchPath(params.Path)

		// 在目录中搜索
		searchPath, err := resolvePath(path, params.BasePath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("解析搜索路径失败: %w", err)
		}

		// 验证搜索路径是否存在且为目录
		info, err := os.Stat(searchPath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("搜索路径不存在: %w", err)
		}
		if !info.IsDir() {
			return nil, GrepResult{}, fmt.Errorf("搜索路径必须是目录: %s", searchPath)
		}

		err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// 跳过无法访问的文件/目录，而不是失败整个操作
				return nil
			}

			// 跳过目录
			if d.IsDir() {
				return nil
			}

			// 应用文件扩展名过滤
			if params.FileType != "" {
				ext := strings.TrimPrefix(filepath.Ext(path), ".")
				if ext != params.FileType {
					return nil // 跳过不匹配的文件类型
				}
			}

			// 如果指定了glob模式，也应用glob过滤
			if params.GlobPattern != "" {
				matched, err := doublestar.Match(params.GlobPattern, path)
				if err != nil || !matched {
					return nil // 跳过不匹配的文件
				}
			}

			// 读取文件并搜索
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // 忽略读取错误
			}

			lines := strings.Split(string(content), "\n")
			matchCount += searchLines(lines, params.Pattern, re, parseBool(params.CaseSensitive), parseBool(params.IgnoreCase), parseBool(params.Regex), path, outputMode, parseBool(params.ShowLineNum), params.ContextBefore, params.ContextAfter, params.Context, &matches, &matchedFiles)

			return nil
		})

		// 错误处理已移至walk函数中忽略，这里不需要返回错误
	}

	// 如果output_mode是files_with_matches，将匹配的文件列表转换为matches
	if outputMode == "files_with_matches" {
		for filePath := range matchedFiles {
			matches = append(matches, filePath)
		}
	}

	// 应用分页
	var pagedMatches []string
	if params.Offset > 0 || params.HeadLimit > 0 {
		start := params.Offset
		if start < 0 {
			start = 0
		}
		if start >= len(matches) {
			pagedMatches = []string{}
		} else {
			end := len(matches)
			if params.HeadLimit > 0 {
				end = start + params.HeadLimit
				if end > len(matches) {
					end = len(matches)
				}
			}
			pagedMatches = matches[start:end]
		}
	} else {
		pagedMatches = matches
	}

	// 应用token截断
	truncatedMatches, wasTruncated := truncateByTokens(pagedMatches)

	// 确保matches不为nil
	if truncatedMatches == nil {
		truncatedMatches = make([]string, 0)
	}

	// 对于count模式，返回实际的匹配计数而不是结果数量
	resultCount := len(truncatedMatches)
	if outputMode == "count" {
		resultCount = matchCount
	}

	return nil, GrepResult{
		Matches:   truncatedMatches,
		Count:     resultCount,
		Truncated: wasTruncated,
	}, nil
}

// truncateByTokens 根据token限制截断字符串数组
func truncateByTokens(items []string) ([]string, bool) {
	totalChars := 0
	for _, item := range items {
		totalChars += utf8.RuneCountInString(item)
	}

	estimatedTokens := totalChars / TOKEN_ESTIMATE_RATIO

	if estimatedTokens <= MAX_TOKENS {
		return items, false
	}

	// 截断到大约MAX_TOKENS个token
	maxChars := MAX_TOKENS * TOKEN_ESTIMATE_RATIO
	var truncated []string
	currentChars := 0

	for _, item := range items {
		itemChars := utf8.RuneCountInString(item)
		if currentChars+itemChars > maxChars {
			break
		}
		truncated = append(truncated, item)
		currentChars += itemChars
	}

	return truncated, true
}

// getFilePath 从参数中获取文件路径
func getFilePath(filePath string) string {
	return filePath
}

// getSearchPath 从参数中获取搜索路径
func getSearchPath(path ...string) string {
	for _, p := range path {
		if p != "" {
			return p
		}
	}
	return ""
}

// parseRange 解析范围字符串，格式：start:end
func parseRange(r string, totalLines int) (int, int, error) {
	parts := strings.Split(r, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("范围格式错误，期望: start:end")
	}

	start := 0
	end := totalLines

	// 解析start
	if parts[0] != "" {
		var err error
		start, err = parseLineNumber(parts[0], totalLines)
		if err != nil {
			return 0, 0, err
		}
	}

	// 解析end（注意：需要加1因为切片是半开区间[ start, end )）
	if parts[1] != "" {
		var err error
		end, err = parseLineNumber(parts[1], totalLines)
		if err != nil {
			return 0, 0, err
		}
		// end需要加1，因为parseLineNumber将其转换为0基，但切片是[ start, end ) 半开区间
		end = end + 1
	}

	// 确保范围有效
	if start < 0 {
		start = 0
	}
	if end > totalLines {
		end = totalLines
	}
	if start > end {
		start, end = end, start
	}

	return start, end, nil
}

// parseLineNumber 解析行号
func parseLineNumber(s string, totalLines int) (int, error) {
	// 支持相对行号（从末尾计算）
	if strings.HasPrefix(s, "-") {
		relative, err := parseLineNumber(strings.TrimPrefix(s, "-"), totalLines)
		if err != nil {
			return 0, err
		}
		return totalLines - relative, nil
	}

	line, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("无效的行号: %s", s)
	}

	// 转换为0基索引，end值需要加1因为切片是[start, end)的半开区间
	if line > 0 {
		return line - 1, nil
	}

	return line, nil
}

// resolvePath 解析文件路径
func resolvePath(filePath, basePath string) (string, error) {
	// 如果是绝对路径，直接返回
	if filepath.IsAbs(filePath) {
		// 清理路径并验证是否存在路径遍历
		cleanPath := filepath.Clean(filePath)
		if strings.Contains(cleanPath, "..") {
			return "", fmt.Errorf("路径包含非法字符 '..'")
		}
		// 执行安全路径检查
		if err := isPathInSafeZone(cleanPath); err != nil {
			return "", fmt.Errorf("安全路径检查失败: %w", err)
		}
		return cleanPath, nil
	}

	// 如果提供了基准路径，使用基准路径
	if basePath != "" {
		// 清理并验证基准路径
		cleanBasePath := filepath.Clean(basePath)
		if strings.Contains(cleanBasePath, "..") {
			return "", fmt.Errorf("基准路径包含非法字符 '..'")
		}
		
		// 确保最终路径在基准路径内部
		joinedPath := filepath.Join(cleanBasePath, filePath)
		cleanPath := filepath.Clean(joinedPath)
		
		// 检查规范化路径是否仍在基准路径内
		relPath, err := filepath.Rel(cleanBasePath, cleanPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return "", fmt.Errorf("路径遍历攻击检测: 尝试访问基准路径外的文件")
		}
		
		return cleanPath, nil
	}

	// 否则使用当前工作目录
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("路径包含非法字符 '..'")
	}
	
	return filepath.Abs(cleanPath)
}

// searchLines 在行中搜索，使用预编译的正则表达式
// 返回匹配的行数
func searchLines(lines []string, pattern string, re *regexp.Regexp, caseSensitive, ignoreCase bool, regex bool, filePath string, outputMode string, showLineNum bool, contextBefore, contextAfter, context int, matches *[]string, matchedFiles *map[string]bool) int {
	// 确定大小写敏感性
	caseInsensitive := ignoreCase || !caseSensitive

	// 计算上下文行数
	before := contextBefore
	after := contextAfter
	if context > 0 {
		before = context
		after = context
	}

	// 预计算大小写转换的模式（仅当非正则表达式时）
	searchPattern := pattern
	if !regex && caseInsensitive {
		searchPattern = strings.ToLower(pattern)
	}

	// 用于跟踪已经添加的行，防止重复添加
	addedLines := make(map[int]bool)
	matchCount := 0

	for i, line := range lines {
		var matched bool

		if regex && re != nil {
			// 使用预编译的正则表达式搜索
			matched = re.MatchString(line)
		} else {
			// 使用字符串匹配
			searchText := line
			if caseInsensitive {
				searchText = strings.ToLower(line)
			}
			matched = strings.Contains(searchText, searchPattern)
		}

		if matched {
			matchCount++
			if outputMode == "files_with_matches" {
				(*matchedFiles)[filePath] = true
			} else {
				// 添加上下文行
				start := i - before
				if start < 0 {
					start = 0
				}
				end := i + after + 1
				if end > len(lines) {
					end = len(lines)
				}

				for j := start; j < end; j++ {
					// 检查是否已经添加过这一行
					if !addedLines[j] {
						contextLine := lines[j]
						if showLineNum {
							if j == i {
								*matches = append(*matches, fmt.Sprintf("%s:%d: %s", filePath, j+1, contextLine))
							} else {
								*matches = append(*matches, fmt.Sprintf("%s:%d- %s", filePath, j+1, contextLine))
							}
						} else {
							*matches = append(*matches, contextLine)
						}
						addedLines[j] = true
					}
				}
			}
		}
	}
	
	return matchCount
}

// isPathInSafeZone 检查路径是否在安全区域内
// 增强安全路径检查，防止路径遍历攻击和访问系统关键文件
func isPathInSafeZone(path string) error {
	// 解析为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("无法解析绝对路径: %w", err)
	}

	// 清理路径
	cleanPath := filepath.Clean(absPath)

	// 检查是否包含路径遍历序列
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("路径包含非法字符 '..'")
	}

	// 检查路径是否为空
	if cleanPath == "" || cleanPath == "/" {
		return fmt.Errorf("路径不能为空或根目录")
	}

	// 检查符号链接
	if isSymlink, err := isSymbolicLink(cleanPath); err == nil && isSymlink {
		return fmt.Errorf("路径包含符号链接，可能存在安全风险: %s", cleanPath)
	}

	// 系统关键目录和文件保护
	unsafePrefixes := []string{
		"/proc",
		"/sys",
		"/dev",
		"/boot",
		"/etc",
		"/var/log",
		"/usr/bin",
		"/usr/sbin",
		"/bin",
		"/sbin",
		"/lib",
		"/lib64",
		"/run",
		"/var/run",
		"/var/tmp",
	}

	// 系统关键文件
	unsafeFiles := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/hosts",
		"/etc/resolv.conf",
		"/root/.ssh",
		"/root/.bashrc",
		"/root/.profile",
		"/root/.bash_history",
		"/etc/ssh/sshd_config",
		"/etc/fstab",
		"/etc/crontab",
	}

	// 检查路径是否以不安全前缀开头
	for _, prefix := range unsafePrefixes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+"/") {
			return fmt.Errorf("尝试访问系统关键目录: %s", cleanPath)
		}
	}

	// 检查是否为系统关键文件
	for _, unsafeFile := range unsafeFiles {
		if cleanPath == unsafeFile {
			return fmt.Errorf("尝试访问系统关键文件: %s", cleanPath)
		}
	}

	// 检查路径深度，防止深层目录遍历
	pathDepth := strings.Count(cleanPath, string(filepath.Separator))
	if pathDepth > 20 {
		return fmt.Errorf("路径深度过大 (%d)，可能存在安全风险", pathDepth)
	}

	// 检查路径是否包含危险字符
	dangerousChars := []string{"~", "$", "|", "&", ";", "`"}
	for _, char := range dangerousChars {
		if strings.Contains(cleanPath, char) {
			return fmt.Errorf("路径包含危险字符 '%s'", char)
		}
	}

	// 检查路径是否在用户安全区域内
	// 这里可以添加更多安全检查，比如限制在特定工作目录内

	return nil
}

// isSymbolicLink 检查路径是否为符号链接
func isSymbolicLink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		// 如果文件不存在，返回false
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

// atomicWriteFile 原子性地写入文件，防止数据损坏
func atomicWriteFile(filePath string, data []byte, perm os.FileMode) error {
	// 生成临时文件路径，使用随机后缀避免冲突
	tempPath := filePath + "." + fmt.Sprintf("%d", time.Now().UnixNano()) + ".tmp"

	// 检查临时文件是否已存在，如果存在则删除
	if _, err := os.Stat(tempPath); err == nil {
		os.Remove(tempPath)
	}

	// 写入临时文件
	if err := os.WriteFile(tempPath, data, perm); err != nil {
		// 失败时清理临时文件
		os.Remove(tempPath)
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// 验证临时文件内容
	tempContent, err := os.ReadFile(tempPath)
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to verify temporary file: %w", err)
	}

	if len(tempContent) != len(data) {
		os.Remove(tempPath)
		return fmt.Errorf("temporary file verification failed: size mismatch")
	}

	// 原子性地重命名临时文件为目标文件
	if err := os.Rename(tempPath, filePath); err != nil {
		// 重命名失败时清理临时文件
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// isRegexComplexitySafe 检查正则表达式复杂度，防止ReDoS攻击
func isRegexComplexitySafe(pattern string) error {
	// 检查模式长度
	if len(pattern) > 1000 {
		return fmt.Errorf("正则表达式过长 (%d 字符)，超过最大限制 1000 字符", len(pattern))
	}
	
	// 检查嵌套量词深度
	nestedQuantifierDepth := 0
	maxNestedDepth := 0
	inGroup := false
	
	for i := 0; i < len(pattern); i++ {
		char := pattern[i]
		
		switch char {
		case '(', '[', '{':
			// 进入组或字符类
			if char == '(' {
				inGroup = true
			}
		case ')', ']', '}':
			// 退出组或字符类
			if char == ')' {
				inGroup = false
				if nestedQuantifierDepth > 0 {
					nestedQuantifierDepth--
				}
			}
		case '*', '+', '?':
			// 量词字符
			if inGroup {
				nestedQuantifierDepth++
				if nestedQuantifierDepth > maxNestedDepth {
					maxNestedDepth = nestedQuantifierDepth
				}
				// 检查嵌套深度
				if maxNestedDepth > 5 {
					return fmt.Errorf("正则表达式嵌套深度过大 (%d)，可能造成ReDoS攻击", maxNestedDepth)
				}
			}
		case '\\':
			// 转义字符，跳过下一个字符
			if i+1 < len(pattern) {
				i++
			}
		}
	}
	
	// 检查回溯爆炸模式
	dangerousPatterns := []string{
		"(a+)+$",
		"(a|a)+$",
		"(a*)*$",
		"(.*)*$",
		"(a+)*$",
		"(.+)*$",
	}
	
	for _, dangerous := range dangerousPatterns {
		if strings.Contains(pattern, dangerous) {
			return fmt.Errorf("检测到潜在ReDoS攻击模式: %s", dangerous)
		}
	}
	
	return nil
}
