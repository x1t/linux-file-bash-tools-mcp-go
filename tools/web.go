package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// API配置
const (
	API_BASE_URL = "https://s.101818.xyz"
	BEARER_TOKEN = "960423wc"
)

// WebSearchParams 定义WebSearch工具参数
type WebSearchParams struct {
	Query          string   `json:"query" jsonschema:"搜索查询词" jsonschema:"required"`
	AllowedDomains []string `json:"allowed_domains,omitempty" jsonschema:"仅包含这些域名的结果"`
	BlockedDomains []string `json:"blocked_domains,omitempty" jsonschema:"从不包含这些域名的结果"`
}

// WebFetchParams 定义WebFetch工具参数
type WebFetchParams struct {
	URL    string `json:"url" jsonschema:"要获取内容的URL" jsonschema:"required"`
	Prompt string `json:"prompt" jsonschema:"用于处理获取内容的提示词"`
}

// WebSearchResultItem 定义搜索结果项
type WebSearchResultItem struct {
	Title   string                 `json:"title"`
	URL     string                 `json:"url"`
	Snippet string                 `json:"snippet"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WebSearchResult 定义WebSearch工具结果
type WebSearchResult struct {
	Query        string                `json:"query"`
	Results      []WebSearchResultItem `json:"results"`
	TotalResults int                  `json:"total_results"`
}

// WebFetchResult 定义WebFetch工具结果
type WebFetchResult struct {
	Response    string `json:"response"`
	URL         string `json:"url"`
	FinalURL    string `json:"final_url,omitempty"`
	StatusCode  int    `json:"status_code,omitempty"`
}

// API响应结构体
type SearchResponse struct {
	Results []SearchResultItem `json:"results"`
}

type SearchResultItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type CrawlRequest struct {
	URLs           []string          `json:"urls"`
	BrowserConfig  BrowserConfig     `json:"browser_config"`
	CrawlerConfig  map[string]interface{} `json:"crawler_config"`
}

type BrowserConfig struct {
	Headless bool `json:"headless"`
}

type CrawlResponse struct {
	Success bool           `json:"success"`
	Results []CrawlResult  `json:"results"`
	Error   string         `json:"error,omitempty"`
}

type CrawlResult struct {
	HTML     string                 `json:"html"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AddWebTools 注册Web工具
func AddWebTools(server *mcp.Server) {
	// WebSearch工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "web_search",
		Description: "搜索网络 - file-bash-tools.web_search (MCP)(query: \"搜索关键词\", allowed_domains: [\"example.com\"]) - 获取搜索结果",
	}, webSearchHandler)

	// WebFetch工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "web_fetch",
		Description: "获取网页内容并用AI处理 - file-bash-tools.web_fetch (MCP)(url: \"https://example.com\", prompt: \"提取主要信息\") - AI处理内容",
	}, webFetchHandler)
}

// webSearchHandler 处理WebSearch命令
func webSearchHandler(ctx context.Context, req *mcp.CallToolRequest, params WebSearchParams) (*mcp.CallToolResult, WebSearchResult, error) {
	// 验证输入参数
	if params.Query == "" {
		return nil, WebSearchResult{}, fmt.Errorf("query parameter cannot be empty")
	}

	// 构建搜索URL
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json", API_BASE_URL, url.QueryEscape(params.Query))

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, WebSearchResult{}, fmt.Errorf("failed to create search request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", "Bearer "+BEARER_TOKEN)
	httpReq.Header.Set("Accept", "application/json")

	// 设置HTTP客户端超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, WebSearchResult{}, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, WebSearchResult{}, fmt.Errorf("search API error: %d %s", resp.StatusCode, resp.Status)
	}

	// 解析响应
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, WebSearchResult{}, fmt.Errorf("failed to parse search response: %w", err)
	}

	// 转换结果格式并应用域名过滤
	results := make([]WebSearchResultItem, 0)
	if searchResp.Results != nil {
		for _, item := range searchResp.Results {
			// 域名过滤检查
			if !shouldIncludeDomain(item.URL, params.AllowedDomains, params.BlockedDomains) {
				continue
			}

			result := WebSearchResultItem{
				Title:   item.Title,
				URL:     item.URL,
				Snippet: item.Content,
				Metadata: make(map[string]interface{}),
			}
			results = append(results, result)
		}
	}

	// 限制结果数量为10个
	if len(results) > 10 {
		results = results[:10]
	}

	return nil, WebSearchResult{
		Query:        params.Query,
		Results:      results,
		TotalResults: len(results),
	}, nil
}

// webFetchHandler 处理WebFetch命令
func webFetchHandler(ctx context.Context, req *mcp.CallToolRequest, params WebFetchParams) (*mcp.CallToolResult, WebFetchResult, error) {
	// 验证URL参数
	if params.URL == "" {
		return nil, WebFetchResult{}, fmt.Errorf("url parameter cannot be empty")
	}

	// 验证URL格式
	if _, err := url.Parse(params.URL); err != nil {
		return nil, WebFetchResult{}, fmt.Errorf("invalid URL format: %w", err)
	}

	// 构建爬取请求
	crawlReq := CrawlRequest{
		URLs: []string{params.URL},
		BrowserConfig: BrowserConfig{
			Headless: true,
		},
		CrawlerConfig: make(map[string]interface{}),
	}

	// 序列化请求体
	reqBody, err := json.Marshal(crawlReq)
	if err != nil {
		return nil, WebFetchResult{}, fmt.Errorf("failed to marshal crawl request: %w", err)
	}

	// 创建HTTP请求
	crawlURL := API_BASE_URL + "/crawl"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", crawlURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, WebFetchResult{}, fmt.Errorf("failed to create crawl request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", "Bearer "+BEARER_TOKEN)
	httpReq.Header.Set("Content-Type", "application/json")

	// 设置HTTP客户端超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, WebFetchResult{}, fmt.Errorf("crawl request failed: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	// 读取原始响应用于调试
	var rawResponse bytes.Buffer
	tee := io.TeeReader(resp.Body, &rawResponse)

	// 解析响应
	var crawlResp CrawlResponse
	if err := json.NewDecoder(tee).Decode(&crawlResp); err != nil {
		return nil, WebFetchResult{}, fmt.Errorf("failed to parse crawl response: %w", err)
	}

	// 检查爬取是否成功
	if !crawlResp.Success {
		return nil, WebFetchResult{}, fmt.Errorf("crawl request failed: %s", crawlResp.Error)
	}

	// 检查结果
	if len(crawlResp.Results) == 0 {
		return nil, WebFetchResult{}, fmt.Errorf("no results returned from crawl API")
	}

	crawlResult := crawlResp.Results[0]
	if crawlResult.HTML == "" {
		return nil, WebFetchResult{}, fmt.Errorf("no HTML content returned from crawl API")
	}

	// 提取网页内容
	title := extractTitle(crawlResult.HTML)
	textContent := extractTextContent(crawlResult.HTML)
	description := extractDescription(crawlResult)

	// 根据prompt处理内容
	response := processContent(params.Prompt, title, description, params.URL, textContent)

	return nil, WebFetchResult{
		Response:   response,
		URL:        params.URL,
		FinalURL:   params.URL, // API不返回重定向后的URL，使用原始URL
		StatusCode: statusCode,
	}, nil
}

// shouldIncludeDomain 检查域名是否应该包含在结果中
func shouldIncludeDomain(itemURL string, allowedDomains, blockedDomains []string) bool {
	if len(allowedDomains) == 0 && len(blockedDomains) == 0 {
		return true
	}

	parsedURL, err := url.Parse(itemURL)
	if err != nil {
		return false
	}

	domain := parsedURL.Hostname()

	// 检查阻止域名
	for _, blocked := range blockedDomains {
		if strings.HasSuffix(domain, blocked) {
			return false
		}
	}

	// 检查允许域名
	if len(allowedDomains) > 0 {
		for _, allowed := range allowedDomains {
			if strings.HasSuffix(domain, allowed) {
				return true
			}
		}
		return false
	}

	return true
}

// extractTitle 从HTML中提取标题
func extractTitle(html string) string {
	re := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return "无标题"
}

// extractTextContent 从HTML中提取纯文本内容
func extractTextContent(html string) string {
	// 移除script标签
	html = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`).ReplaceAllString(html, "")
	// 移除style标签
	html = regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`).ReplaceAllString(html, "")
	// 移除HTML标签
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, " ")
	// 规范化空白字符
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
	return strings.TrimSpace(html)
}

// extractDescription 从元数据中提取描述
func extractDescription(crawlResult CrawlResult) string {
	if crawlResult.Metadata != nil {
		if desc, ok := crawlResult.Metadata["description"].(string); ok && desc != "" {
			return desc
		}
	}
	return "无描述"
}

// processContent 根据prompt处理网页内容
func processContent(prompt, title, description, url, content string) string {
	if prompt == "" {
		// 默认格式化
		return fmt.Sprintf("网页标题: %s\n网页描述: %s\n网页URL: %s\n\n%s", title, description, url, content)
	}

	// 简单的内容处理逻辑
	content = fmt.Sprintf("标题: %s\n描述: %s\nURL: %s\n\n内容:\n%s", title, description, url, content)

	// 根据prompt提供不同的处理方式
	if strings.Contains(strings.ToLower(prompt), "summary") || strings.Contains(strings.ToLower(prompt), "摘要") {
		// 生成简单摘要（截取前500字符）
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		return fmt.Sprintf("网页摘要:\n%s", content)
	}

	if strings.Contains(strings.ToLower(prompt), "title") || strings.Contains(strings.ToLower(prompt), "标题") {
		return fmt.Sprintf("网页标题: %s\nURL: %s", title, url)
	}

	// 默认返回完整内容
	return content
}