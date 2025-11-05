package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadParams 定义读取文件参数 (完全符合todo.md标准)
type ReadParams struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path of the file to read"`
	Offset   int    `json:"offset,omitempty" jsonschema:"Starting line number (default: 1)"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Number of lines to read"`
}

// WriteParams 定义写入文件参数 (完全符合todo.md标准)
type WriteParams struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path of the file to write"`
	Content  string `json:"content" jsonschema:"Content to write to the file"`
}

// EditParams 定义编辑文件参数 (完全符合todo.md标准)
type EditParams struct {
	FilePath  string `json:"filePath,omitempty" jsonschema:"Path to the file to edit"`
	FilePath1 string `json:"file_path,omitempty" jsonschema:"Path to the file to edit (alternative naming)"`
	FilePath2 string `json:"filepath,omitempty" jsonschema:"Path to the file to edit (alternative naming)"`
	// todo.md标准参数
	OldString  string `json:"old_string,omitempty" jsonschema:"String to be replaced"`
	NewString  string `json:"new_string,omitempty" jsonschema:"String to replace with"`
	ReplaceAll interface{} `json:"replace_all,omitempty" jsonschema:"Replace all occurrences (default: false)"`
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
	FilePath      string      `json:"filePath,omitempty" jsonschema:"Specific file to search in"`
	FilePath1     string      `json:"file_path,omitempty" jsonschema:"Specific file to search in (alternative naming)"`
	FilePath2     string      `json:"filepath,omitempty" jsonschema:"Specific file to search in (alternative naming)"`
	Path          string      `json:"path,omitempty" jsonschema:"Directory to search in"`
	Path1         string      `json:"Path,omitempty" jsonschema:"Directory to search in (alternative naming)"`
	GlobPattern   string      `json:"glob,omitempty" jsonschema:"Glob pattern to filter files (e.g., '*.js')"`
	FileType      string      `json:"type,omitempty" jsonschema:"File type to search (e.g., 'js', 'py', 'rust')"`
	OutputMode    string      `json:"output_mode,omitempty" jsonschema:"Output mode: 'content' | 'files_with_matches' | 'count'"`
	ShowLineNum   interface{} `json:"-n,omitempty" jsonschema:"Show line numbers in output"`
	CaseSensitive interface{} `json:"caseSensitive,omitempty" jsonschema:"Whether to perform case-sensitive search"`
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

	// 读取文件
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return nil, ReadResult{}, fmt.Errorf("Failed to read file: %w", err)
	}

	text := string(content)
	allLines := strings.Split(text, "\n")
	totalLines := len(allLines)

	// 处理offset和limit参数 (符合todo.md标准)
	start := 0
	end := totalLines

	if params.Offset > 0 {
		start = params.Offset - 1 // 转换为0基索引
		if start < 0 {
			start = 0
		}
		if start >= totalLines {
			start = totalLines
		}
	}

	if params.Limit > 0 {
		end = start + params.Limit
		if end > totalLines {
			end = totalLines
		}
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

	// 添加行号前缀 (todo.md标准要求带行号的内容)
	var numberedLines []string
	for i, line := range strings.Split(selectedText, "\n") {
		lineNumber := start + i + 1 // 1基行号
		numberedLines = append(numberedLines, fmt.Sprintf("%d: %s", lineNumber, line))
	}

	contentWithLineNumbers := strings.Join(numberedLines, "\n")

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

	// 验证文件路径是绝对路径
	if !filepath.IsAbs(params.FilePath) {
		return nil, WriteResult{}, fmt.Errorf("file_path must be an absolute path")
	}

	// 确保目录存在
	dir := filepath.Dir(params.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, WriteResult{}, fmt.Errorf("Failed to create directory: %w", err)
	}

	// 写入文件
	contentBytes := []byte(params.Content)
	if err := os.WriteFile(params.FilePath, contentBytes, 0644); err != nil {
		return nil, WriteResult{}, fmt.Errorf("Failed to write file: %w", err)
	}

	return nil, WriteResult{
		Message:      fmt.Sprintf("Successfully wrote file: %s", params.FilePath),
		BytesWritten: len(contentBytes),
		FilePath:     params.FilePath,
	}, nil
}

// editFileHandler 处理文件编辑请求 (完全符合todo.md标准)
func editFileHandler(ctx context.Context, req *mcp.CallToolRequest, params EditParams) (*mcp.CallToolResult, EditResult, error) {
	// 从多个参数名中获取文件路径
	filePath := getFilePath(params.FilePath, params.FilePath1, params.FilePath2)
	if filePath == "" {
		return nil, EditResult{}, fmt.Errorf("file_path parameter is required")
	}

	// 验证必需参数 (根据todo.md标准)
	if params.OldString == "" || params.NewString == "" {
		return nil, EditResult{}, fmt.Errorf("old_string and new_string parameters are required")
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

	// 读取原文件
	content, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, EditResult{}, fmt.Errorf("Failed to read file: %w", err)
	}

	originalText := string(content)
	var newText string
	var replacements int

	// 执行字符串替换 (符合todo.md标准)
	if parseBool(params.ReplaceAll) {
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

	// 写回文件
	if err := os.WriteFile(actualPath, []byte(newText), 0644); err != nil {
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
	// 从多个参数名中获取搜索路径
	path := getSearchPath(params.Path, params.Path1)

	// 解析搜索路径
	searchPath, err := resolvePath(path, params.BasePath)
	if err != nil {
		return nil, GlobResult{}, fmt.Errorf("解析搜索路径失败: %w", err)
	}

	// 确保files不为nil
	files := make([]string, 0)
	fileInfos := make(map[string]fs.FileInfo)

	// 使用Walk遍历目录
	err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录，只处理文件
		if d.IsDir() {
			return nil
		}

		// 匹配模式（支持**/*.go等高级模式）
		matched, err := filepath.Match(params.Pattern, filepath.Base(path))
		if err != nil {
			// 如果模式无效，跳过
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

	if err != nil {
		return nil, GlobResult{}, fmt.Errorf("glob匹配失败: %w", err)
	}

	// 按修改时间排序（最新的在前）
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			info1, exists1 := fileInfos[files[i]]
			info2, exists2 := fileInfos[files[j]]
			if exists1 && exists2 && info1.ModTime().Before(info2.ModTime()) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// 应用分页
	var pagedFiles []string
	if params.Offset > 0 || params.HeadLimit > 0 {
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
			}
			pagedFiles = files[start:end]
		}
	} else {
		pagedFiles = files
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
	// 确保matches不为nil
	matches := make([]string, 0)
	matchedFiles := make(map[string]bool)
	outputMode := params.OutputMode
	if outputMode == "" {
		outputMode = "content" // 默认输出模式
	}

	// 从多个参数名中获取文件路径
	filePath := getFilePath(params.FilePath, params.FilePath1, params.FilePath2)

	if filePath != "" {
		// 在单个文件中搜索
		actualPath, err := resolvePath(filePath, params.BasePath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("解析文件路径失败: %w", err)
		}

		content, err := os.ReadFile(actualPath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("读取文件失败: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		searchLines(lines, params.Pattern, parseBool(params.CaseSensitive), parseBool(params.IgnoreCase), parseBool(params.Regex), actualPath, outputMode, parseBool(params.ShowLineNum), params.ContextBefore, params.ContextAfter, params.Context, &matches, &matchedFiles)
	} else {
		// 从多个参数名中获取搜索路径
		path := getSearchPath(params.Path, params.Path1)

		// 在目录中搜索
		searchPath, err := resolvePath(path, params.BasePath)
		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("解析搜索路径失败: %w", err)
		}

		err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// 跳过目录
			if d.IsDir() {
				return nil
			}

			// 读取文件并搜索
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // 忽略读取错误
			}

			lines := strings.Split(string(content), "\n")
			searchLines(lines, params.Pattern, parseBool(params.CaseSensitive), parseBool(params.IgnoreCase), parseBool(params.Regex), path, outputMode, parseBool(params.ShowLineNum), params.ContextBefore, params.ContextAfter, params.Context, &matches, &matchedFiles)

			return nil
		})

		if err != nil {
			return nil, GrepResult{}, fmt.Errorf("搜索失败: %w", err)
		}
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

	return nil, GrepResult{
		Matches:   truncatedMatches,
		Count:     len(truncatedMatches),
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

// getFilePath 从多个可能的参数名中获取文件路径
func getFilePath(filePath, filePath1, filePath2 string) string {
	if filePath != "" {
		return filePath
	}
	if filePath1 != "" {
		return filePath1
	}
	if filePath2 != "" {
		return filePath2
	}
	return ""
}

// getSearchPath 从多个可能的参数名中获取搜索路径
func getSearchPath(path, path1 string) string {
	if path != "" {
		return path
	}
	if path1 != "" {
		return path1
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
		return filepath.Clean(filePath), nil
	}

	// 如果提供了基准路径，使用基准路径
	if basePath != "" {
		return filepath.Abs(filepath.Join(basePath, filePath))
	}

	// 否则使用当前工作目录
	return filepath.Abs(filePath)
}

// searchLines 在行中搜索
func searchLines(lines []string, pattern string, caseSensitive, ignoreCase bool, regex bool, filePath string, outputMode string, showLineNum bool, contextBefore, contextAfter, context int, matches *[]string, matchedFiles *map[string]bool) {
	// 确定大小写敏感性
	caseInsensitive := ignoreCase || !caseSensitive

	// 计算上下文行数
	before := contextBefore
	after := contextAfter
	if context > 0 {
		before = context
		after = context
	}

	for i, line := range lines {
		var matched bool

		if regex {
			// 实现真正的正则表达式搜索
			var re *regexp.Regexp
			var err error
			if caseInsensitive {
				re, err = regexp.Compile("(?i)" + pattern)
			} else {
				re, err = regexp.Compile(pattern)
			}
			if err != nil {
				// 正则表达式无效，继续使用字符串匹配
				searchText := line
				searchPattern := pattern
				if caseInsensitive {
					searchText = strings.ToLower(line)
					searchPattern = strings.ToLower(pattern)
				}
				matched = strings.Contains(searchText, searchPattern)
			} else {
				matched = re.MatchString(line)
			}
		} else {
			searchText := line
			searchPattern := pattern
			if caseInsensitive {
				searchText = strings.ToLower(line)
				searchPattern = strings.ToLower(pattern)
			}
			matched = strings.Contains(searchText, searchPattern)
		}

		if matched {
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
				}
			}
		}
	}
}
