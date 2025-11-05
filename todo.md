// 工具输入类型定义
type ToolInput =
  | AgentInput
  | BashInput
  | BashOutputInput
  | FileEditInput
  | FileReadInput
  | FileWriteInput
  | GlobInput
  | GrepInput
  | KillShellInput
  | WebFetchInput
  | WebSearchInput
  | TodoWriteInput
  | ExitPlanModeInput


// Task工具输入
interface AgentInput {
  /** 任务的简短描述（3-5个词） */
  description: string;
  /** 代理要执行的任务 */
  prompt: string;
  /** 用于此任务的专用代理类型 */
  subagent_type: string;
}

// Bash工具输入
interface BashInput {
  /** 要执行的命令 */
  command: string;
  /** 可选超时时间（毫秒，最大600000） */
  timeout?: number;
  /** 5-10个词简要描述命令功能 */
  description?: string;
  /** 设置为true可在后台运行命令 */
  run_in_background?: boolean;
}

// BashOutput工具输入
interface BashOutputInput {
  /** 要获取输出的后台shell ID */
  bash_id: string;
  /** 用于过滤输出行的可选正则表达式 */
  filter?: string;
}

// Edit工具输入
interface FileEditInput {
  /** 要修改的文件的绝对路径 */
  file_path: string;
  /** 要替换的文本 */
  old_string: string;
  /** 替换后的文本（必须与old_string不同） */
  new_string: string;
  /** 是否替换所有出现的old_string（默认false） */
  replace_all?: boolean;
}

// Read工具输入
interface FileReadInput {
  /** 要读取的文件的绝对路径 */
  file_path: string;
  /** 开始读取的行号 */
  offset?: number;
  /** 要读取的行数 */
  limit?: number;
}

// Write工具输入
interface FileWriteInput {
  /** 要写入的文件的绝对路径 */
  file_path: string;
  /** 要写入文件的内容 */
  content: string;
}

// Glob工具输入
interface GlobInput {
  /** 用于匹配文件的glob模式 */
  pattern: string;
  /** 搜索目录（默认为当前工作目录） */
  path?: string;
}

// Grep工具输入
interface GrepInput {
  /** 要搜索的正则表达式模式 */
  pattern: string;
  /** 要搜索的文件或目录（默认为当前工作目录） */
  path?: string;
  /** 用于过滤文件的glob模式（例如"*.js"） */
  glob?: string;
  /** 要搜索的文件类型（例如"js"、"py"、"rust"） */
  type?: string;
  /** 输出模式："content"、"files_with_matches"或"count" */
  output_mode?: 'content' | 'files_with_matches' | 'count';
  /** 不区分大小写搜索 */
  '-i'?: boolean;
  /** 显示行号（适用于content模式） */
  '-n'?: boolean;
  /** 每个匹配前显示的行数 */
  '-B'?: number;
  /** 每个匹配后显示的行数 */
  '-A'?: number;
  /** 每个匹配前后显示的行数 */
  '-C'?: number;
  /** 限制输出到前N行/条目 */
  head_limit?: number;
  /** 启用多行模式 */
  multiline?: boolean;
}

// KillBash工具输入
interface KillShellInput {
  /** 要终止的后台shell的ID */
  shell_id: string;
}


// WebFetch工具输入
interface WebFetchInput {
  /** 要获取内容的URL */
  url: string;
  /** 用于处理获取内容的提示词 */
  prompt: string;
}

// WebSearch工具输入
interface WebSearchInput {
  /** 搜索查询词 */
  query: string;
  /** 仅包含这些域名的结果 */
  allowed_domains?: string[];
  /** 从不包含这些域名的结果 */
  blocked_domains?: string[];
}

// TodoWrite工具输入
interface TodoWriteInput {
  /** 更新后的待办事项列表 */
  todos: Array<{
    /** 任务描述 */
    content: string;
    /** 任务状态 */
    status: 'pending' | 'in_progress' | 'completed';
    /** 任务描述的主动形式 */
    activeForm: string;
  }>;
}

// ExitPlanMode工具输入
interface ExitPlanModeInput {
  /** 供用户批准的计划 */
  plan: string;
}


// 工具输出类型定义
type ToolOutput =
  | TaskOutput
  | BashOutput
  | BashOutputToolOutput
  | EditOutput
  | ReadOutput
  | WriteOutput
  | GlobOutput
  | GrepOutput
  | KillBashOutput
  | WebFetchOutput
  | WebSearchOutput
  | TodoWriteOutput
  | ExitPlanModeOutput


// Task工具输出
interface TaskOutput {
  /** 子代理的最终结果消息 */
  result: string;
  /** 令牌使用统计 */
  usage?: {
    input_tokens: number;
    output_tokens: number;
    cache_creation_input_tokens?: number;
    cache_read_input_tokens?: number;
  };
  /** 总费用（USD） */
  total_cost_usd?: number;
  /** 执行持续时间（毫秒） */
  duration_ms?: number;
}

// Bash工具输出
interface BashOutput {
  /** 合并的stdout和stderr输出 */
  output: string;
  /** 命令的退出代码 */
  exitCode: number;
  /** 命令是否因超时而被终止 */
  killed?: boolean;
  /** 后台进程的shell ID */
  shellId?: string;
}

// BashOutput工具输出
interface BashOutputToolOutput {
  /** 自上次检查以来的新输出 */
  output: string;
  /** 当前shell状态 */
  status: 'running' | 'completed' | 'failed';
  /** 退出代码（完成时） */
  exitCode?: number;
}

// Edit工具输出
interface EditOutput {
  /** 确认消息 */
  message: string;
  /** 替换次数 */
  replacements: number;
  /** 被编辑的文件路径 */
  file_path: string;
}

// Read工具输出
type ReadOutput =
  | TextFileOutput
  | ImageFileOutput
  | PDFFileOutput


interface TextFileOutput {
  /** 带行号的文件内容 */
  content: string;
  /** 文件总行数 */
  total_lines: number;
  /** 实际返回的行数 */
  lines_returned: number;
}

interface ImageFileOutput {
  /** Base64编码的图像数据 */
  image: string;
  /** 图像MIME类型 */
  mime_type: string;
  /** 文件大小（字节） */
  file_size: number;
}

interface PDFFileOutput {
  /** 页面内容数组 */
  pages: Array<{
    page_number: number;
    text?: string;
    images?: Array<{
      image: string;
      mime_type: string;
    }>;
  }>;
  /** 总页数 */
  total_pages: number;
}


// Write工具输出
interface WriteOutput {
  /** 成功消息 */
  message: string;
  /** 写入的字节数 */
  bytes_written: number;
  /** 被写入的文件路径 */
  file_path: string;
}

// Glob工具输出
interface GlobOutput {
  /** 匹配的文件路径数组 */
  matches: string[];
  /** 找到的匹配数 */
  count: number;
  /** 使用的搜索目录 */
  search_path: string;
}

// Grep工具输出
type GrepOutput =
  | GrepContentOutput
  | GrepFilesOutput
  | GrepCountOutput;

interface GrepContentOutput {
  /** 带上下文的匹配行 */
  matches: Array<{
    file: string;
    line_number?: number;
    line: string;
    before_context?: string[];
    after_context?: string[];
  }>;
  /** 总匹配数 */
  total_matches: number;
}

interface GrepFilesOutput {
  /** 包含匹配项的文件 */
  files: string[];
  /** 有匹配项的文件数 */
  count: number;
}

interface GrepCountOutput {
  /** 每个文件的匹配计数 */
  counts: Array<{
    file: string;
    count: number;
  }>;
  /** 所有文件的总匹配数 */
  total: number;
}

// KillBash工具输出
interface KillBashOutput {
  /** 成功消息 */
  message: string;
  /** 被终止的shell ID */
  shell_id: string;
}


// WebFetch工具输出
interface WebFetchOutput {
  /** AI模型对提示词的响应 */
  response: string;
  /** 被获取的URL */
  url: string;
  /** 重定向后的最终URL */
  final_url?: string;
  /** HTTP状态码 */
  status_code?: number;
}

// WebSearch工具输出
interface WebSearchOutput {
  /** 搜索结果 */
  results: Array<{
    title: string;
    url: string;
    snippet: string;
    /** 可用的附加元数据 */
    metadata?: Record<string, any>;
  }>;
  /** 结果总数 */
  total_results: number;
  /** 搜索的查询词 */
  query: string;
}

// TodoWrite工具输出
interface TodoWriteOutput {
  /** 成功消息 */
  message: string;
  /** 当前待办事项统计 */
  stats: {
    total: number;
    pending: number;
    in_progress: number;
    completed: number;
  };
}

// ExitPlanMode工具输出
interface ExitPlanModeOutput {
  /** 确认消息 */
  message: string;
  /** 用户是否批准了计划 */
  approved?: boolean;
}
