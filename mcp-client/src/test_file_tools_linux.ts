// ============================================================================
// 测试: Linux 版 mcp-file-bash-tools 的 6 个文件/待办工具
//   - read_file  读取文件(支持行范围/行号)
//   - write_file 写入文件(自动建目录/原子写入)
//   - edit_file  编辑文件(单次/全部替换)
//   - glob       模式匹配文件路径
//   - grep       文件/目录文本搜索(内容/文件名/计数模式)
//   - todo_write 待办事项列表管理
// 通过 mcp-client (TypeScript SDK) 连接 dist 二进制进行测试
// ============================================================================
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { SERVER_PATH, parseResult } from './shared.js';

const WORK_DIR = '/tmp/mcp_file_tools_test';
const WRITE_TARGET = `${WORK_DIR}/write_target.txt`;
const NESTED_TARGET = `${WORK_DIR}/nested/deep/target.txt`;

// --------------------------- 结果类型定义 ---------------------------
interface ReadResult {
  content: string;
  total_lines: number;
  lines_returned: number;
}

interface WriteResult {
  message: string;
  bytes_written: number;
  file_path: string;
}

interface EditResult {
  message: string;
  replacements: number;
  file_path: string;
}

interface GlobResult {
  files: string[];
  count: number;
  truncated?: boolean;
}

interface GrepResult {
  matches: string[];
  count: number;
  truncated?: boolean;
  files?: string[];
}

interface TodoWriteResult {
  message: string;
  stats: { total: number; pending: number; in_progress: number; completed: number };
}

// --------------------------- 测试结果收集器 ---------------------------
interface Outcome {
  name: string;
  ok: boolean;
  detail: string;
}

const outcomes: Outcome[] = [];

function record(name: string, ok: boolean, detail = ''): void {
  outcomes.push({ name, ok, detail });
  console.log(`${ok ? '✅' : '❌'} ${name}${detail ? ` — ${detail}` : ''}`);
}

// 调用工具并统一处理错误/工具错误结果
async function callTool(
  client: Client,
  name: string,
  args: Record<string, unknown>,
): Promise<{ data: any; error: string | null }> {
  try {
    const result = await client.callTool({ name, arguments: args });
    if ((result as any).isError) {
      return { data: null, error: (result.content as any)[0]?.text ?? '' };
    }
    return { data: result, error: null };
  } catch (e: any) {
    return { data: null, error: e?.message ?? String(e) };
  }
}

// --------------------------- 测试环境准备 ---------------------------
function setupWorkspace(): void {
  rmSync(WORK_DIR, { recursive: true, force: true });
  mkdirSync(`${WORK_DIR}/sub`, { recursive: true });
  writeFileSync(`${WORK_DIR}/a.txt`, 'alpha\nbeta needle\ngamma\nline four\n');
  writeFileSync(`${WORK_DIR}/b.txt`, 'Needle here\nline two\n');
  writeFileSync(`${WORK_DIR}/sub/c.md`, 'markdown needle content\n');
  writeFileSync(`${WORK_DIR}/data.json`, '{"key":"value"}\n');
}

function cleanupWorkspace(): void {
  rmSync(WORK_DIR, { recursive: true, force: true });
}

// --------------------------- 测试: write_file ---------------------------
async function testWriteFile(client: Client): Promise<void> {
  console.log('\n━━━ 测试 write_file ━━━');

  // 1. 正常写入新文件 (5 行 + 4 个换行符 = 34 字节)
  const r1 = await callTool(client, 'write_file', {
    file_path: WRITE_TARGET,
    content: 'line 1\nline 2\nline 3\nline 4\nline 5',
  });
  if (r1.error) {
    record('write_file 正常写入', false, r1.error);
  } else {
    const d = parseResult<WriteResult>(r1.data);
    const onDisk = existsSync(WRITE_TARGET) && readFileSync(WRITE_TARGET, 'utf8').includes('line 3');
    record('write_file 正常写入',
      d.message.includes('Successfully wrote') && d.bytes_written === 34 && onDisk,
      `bytes=${d.bytes_written}, 落盘=${onDisk}`);
  }

  // 2. 自动创建嵌套目录
  const r2 = await callTool(client, 'write_file', {
    file_path: NESTED_TARGET,
    content: 'nested content',
  });
  if (r2.error) {
    record('write_file 自动建目录', false, r2.error);
  } else {
    const d = parseResult<WriteResult>(r2.data);
    record('write_file 自动建目录', existsSync(NESTED_TARGET), `path=${d.file_path}`);
  }

  // 3. 缺少 file_path
  const r3 = await callTool(client, 'write_file', { content: 'x' });
  record('write_file 缺少 file_path', r3.error !== null, r3.error ?? '');

  // 4. 缺少 content
  const r4 = await callTool(client, 'write_file', { file_path: WRITE_TARGET });
  record('write_file 缺少 content', r4.error !== null, r4.error ?? '');

  // 5. 相对路径被拒绝
  const r5 = await callTool(client, 'write_file', { file_path: 'relative/file.txt', content: 'x' });
  record('write_file 相对路径拒绝', r5.error !== null, r5.error ?? '');
}

// --------------------------- 测试: read_file ---------------------------
async function testReadFile(client: Client): Promise<void> {
  console.log('\n━━━ 测试 read_file ━━━');

  // 1. 完整读取
  const r1 = await callTool(client, 'read_file', { file_path: WRITE_TARGET });
  if (r1.error) {
    record('read_file 完整读取', false, r1.error);
  } else {
    const d = parseResult<ReadResult>(r1.data);
    record('read_file 完整读取',
      d.total_lines === 5 && d.lines_returned === 5 && d.content.includes('1: line 1') && d.content.includes('5: line 5'),
      `total=${d.total_lines}, returned=${d.lines_returned}`);
  }

  // 2. offset + limit 范围读取 (第2~4行)
  const r2 = await callTool(client, 'read_file', { file_path: WRITE_TARGET, offset: 2, limit: 3 });
  if (r2.error) {
    record('read_file 范围读取', false, r2.error);
  } else {
    const d = parseResult<ReadResult>(r2.data);
    record('read_file 范围读取',
      d.lines_returned === 3 && d.content.includes('2: line 2') && d.content.includes('4: line 4') && !d.content.includes('1: line 1'),
      `returned=${d.lines_returned}, content="${d.content.replace(/\n/g, ' | ')}"`);
  }

  // 3. 关闭行号
  const r3 = await callTool(client, 'read_file', { file_path: WRITE_TARGET, limit: 1, show_line_numbers: false });
  if (r3.error) {
    record('read_file 关闭行号', false, r3.error);
  } else {
    const d = parseResult<ReadResult>(r3.data);
    record('read_file 关闭行号', d.content.trim() === 'line 1', `content="${d.content.trim()}"`);
  }

  // 4. 文件不存在
  const r4 = await callTool(client, 'read_file', { file_path: '/tmp/mcp_file_tools_test/nope.txt' });
  record('read_file 文件不存在', r4.error !== null, r4.error ?? '');

  // 5. 缺少 file_path / 相对路径
  const r5 = await callTool(client, 'read_file', {});
  record('read_file 缺少 file_path', r5.error !== null, r5.error ?? '');
  const r6 = await callTool(client, 'read_file', { file_path: 'relative.txt' });
  record('read_file 相对路径拒绝', r6.error !== null, r6.error ?? '');
}

// --------------------------- 测试: edit_file ---------------------------
async function testEditFile(client: Client): Promise<void> {
  console.log('\n━━━ 测试 edit_file ━━━');

  // 准备: 写入含重复文本的文件
  await callTool(client, 'write_file', {
    file_path: `${WORK_DIR}/edit_target.txt`,
    content: 'foo bar baz\nfoo again\nend foo\n',
  });

  // 1. 替换第一个出现
  const r1 = await callTool(client, 'edit_file', {
    file_path: `${WORK_DIR}/edit_target.txt`,
    old_string: 'foo',
    new_string: 'XXX',
  });
  if (r1.error) {
    record('edit_file 替换首个', false, r1.error);
  } else {
    const d = parseResult<EditResult>(r1.data);
    const content = readFileSync(`${WORK_DIR}/edit_target.txt`, 'utf8');
    const onlyFirst = content.startsWith('XXX bar baz') && content.includes('foo again') && content.includes('end foo');
    record('edit_file 替换首个', d.replacements === 1 && onlyFirst, `replacements=${d.replacements}`);
  }

  // 2. replace_all 全部替换
  const r2 = await callTool(client, 'edit_file', {
    file_path: `${WORK_DIR}/edit_target.txt`,
    old_string: 'foo',
    new_string: 'YYY',
    replace_all: true,
  });
  if (r2.error) {
    record('edit_file 全部替换', false, r2.error);
  } else {
    const d = parseResult<EditResult>(r2.data);
    const content = readFileSync(`${WORK_DIR}/edit_target.txt`, 'utf8');
    record('edit_file 全部替换', d.replacements === 2 && !content.includes('foo'), `replacements=${d.replacements}`);
  }

  // 3. old_string 不存在
  const r3 = await callTool(client, 'edit_file', {
    file_path: `${WORK_DIR}/edit_target.txt`,
    old_string: 'not-present',
    new_string: 'x',
  });
  record('edit_file 未找到 old_string', r3.error !== null, r3.error ?? '');

  // 4. old_string == new_string
  const r4 = await callTool(client, 'edit_file', {
    file_path: `${WORK_DIR}/edit_target.txt`,
    old_string: 'YYY',
    new_string: 'YYY',
  });
  record('edit_file 新旧相同', r4.error !== null, r4.error ?? '');

  // 5. 缺少必需参数
  const r5 = await callTool(client, 'edit_file', { file_path: `${WORK_DIR}/edit_target.txt` });
  record('edit_file 缺少 old_string', r5.error !== null, r5.error ?? '');
}

// --------------------------- 测试: glob ---------------------------
async function testGlob(client: Client): Promise<void> {
  console.log('\n━━━ 测试 glob ━━━');

  // 1. 单层模式 *.txt
  const r1 = await callTool(client, 'glob', { pattern: '*.txt', path: WORK_DIR });
  if (r1.error) {
    record('glob *.txt', false, r1.error);
  } else {
    const d = parseResult<GlobResult>(r1.data);
    const names = d.files.map((f) => f.replace(WORK_DIR + '/', ''));
    record('glob *.txt', d.count >= 2 && names.includes('a.txt') && names.includes('b.txt'),
      `count=${d.count}, files=${names.join(',')}`);
  }

  // 2. 递归模式 **/*.md
  const r2 = await callTool(client, 'glob', { pattern: '**/*.md', path: WORK_DIR });
  if (r2.error) {
    record('glob **/*.md', false, r2.error);
  } else {
    const d = parseResult<GlobResult>(r2.data);
    const hasSub = d.files.some((f) => f.includes('sub') && f.endsWith('c.md'));
    record('glob **/*.md', d.count >= 1 && hasSub, `count=${d.count}, 含sub/c.md=${hasSub}`);
  }

  // 3. head_limit 分页
  const r3 = await callTool(client, 'glob', { pattern: '**/*', path: WORK_DIR, head_limit: 2 });
  if (r3.error) {
    record('glob head_limit 分页', false, r3.error);
  } else {
    const d = parseResult<GlobResult>(r3.data);
    record('glob head_limit 分页', d.count === 2, `count=${d.count}`);
  }

  // 4. 缺少 pattern
  const r4 = await callTool(client, 'glob', { path: WORK_DIR });
  record('glob 缺少 pattern', r4.error !== null, r4.error ?? '');

  // 5. 路径不存在
  const r5 = await callTool(client, 'glob', { pattern: '*.txt', path: '/tmp/mcp_file_tools_test/not_exist' });
  record('glob 路径不存在', r5.error !== null, r5.error ?? '');
}

// --------------------------- 测试: grep ---------------------------
async function testGrep(client: Client): Promise<void> {
  console.log('\n━━━ 测试 grep ━━━');

  // 1. 单文件内容模式 (默认忽略大小写)
  const r1 = await callTool(client, 'grep', { pattern: 'needle', file_path: `${WORK_DIR}/b.txt` });
  if (r1.error) {
    record('grep 单文件内容', false, r1.error);
  } else {
    const d = parseResult<GrepResult>(r1.data);
    record('grep 单文件内容(默认忽略大小写)',
      d.count === 1 && d.matches.length === 1 && d.matches[0].includes('Needle here'),
      `count=${d.count}, matches=${JSON.stringify(d.matches)}`);
  }

  // 2. 目录搜索
  const r2 = await callTool(client, 'grep', { pattern: 'needle', path: WORK_DIR });
  if (r2.error) {
    record('grep 目录搜索', false, r2.error);
  } else {
    const d = parseResult<GrepResult>(r2.data);
    record('grep 目录搜索', d.count >= 3, `count=${d.count}`);
  }

  // 3. files_with_matches 模式
  const r3 = await callTool(client, 'grep', { pattern: 'needle', path: WORK_DIR, output_mode: 'files_with_matches' });
  if (r3.error) {
    record('grep files_with_matches', false, r3.error);
  } else {
    const d = parseResult<GrepResult>(r3.data);
    record('grep files_with_matches',
      d.matches.length >= 3 && d.matches.some((f) => f.endsWith('a.txt')) && d.matches.some((f) => f.endsWith('c.md')),
      `files=${d.matches.join(',')}`);
  }

  // 4. count 模式
  const r4 = await callTool(client, 'grep', { pattern: 'line', file_path: `${WORK_DIR}/a.txt`, output_mode: 'count' });
  if (r4.error) {
    record('grep count 模式', false, r4.error);
  } else {
    const d = parseResult<GrepResult>(r4.data);
    record('grep count 模式', d.count === 1, `count=${d.count}`);
  }

  // 5. regex 模式
  const r5 = await callTool(client, 'grep', { pattern: 'N.+.le', file_path: `${WORK_DIR}/b.txt`, regex: true });
  if (r5.error) {
    record('grep regex 模式', false, r5.error);
  } else {
    const d = parseResult<GrepResult>(r5.data);
    record('grep regex 模式', d.count === 1 && d.matches.some((m) => m.includes('Needle')), `count=${d.count}`);
  }

  // 6. case_sensitive: true 时大小写不匹配
  const r6 = await callTool(client, 'grep', {
    pattern: 'needle',
    file_path: `${WORK_DIR}/b.txt`,
    case_sensitive: true,
  });
  if (r6.error) {
    record('grep case_sensitive 区分大小写', false, r6.error);
  } else {
    const d = parseResult<GrepResult>(r6.data);
    record('grep case_sensitive 区分大小写', d.count === 0, `count=${d.count} (期望 0)`);
  }

  // 7. 缺少 pattern
  const r7 = await callTool(client, 'grep', { file_path: `${WORK_DIR}/a.txt` });
  record('grep 缺少 pattern', r7.error !== null, r7.error ?? '');
}

// --------------------------- 测试: todo_write ---------------------------
async function testTodoWrite(client: Client): Promise<void> {
  console.log('\n━━━ 测试 todo_write ━━━');

  // 1. 数组格式, 统计计算
  const r1 = await callTool(client, 'todo_write', {
    todos: [
      { content: '任务A', status: 'pending', activeForm: '处理任务A' },
      { content: '任务B', status: 'in_progress', activeForm: '处理任务B' },
      { content: '任务C', status: 'completed', activeForm: '处理任务C' },
    ],
  });
  if (r1.error) {
    record('todo_write 数组格式', false, r1.error);
  } else {
    const d = parseResult<TodoWriteResult>(r1.data);
    record('todo_write 数组格式统计',
      d.message.includes('3') && d.stats.total === 3 && d.stats.pending === 1 && d.stats.in_progress === 1 && d.stats.completed === 1,
      `total=${d.stats.total}, pending=${d.stats.pending}, in_progress=${d.stats.in_progress}, completed=${d.stats.completed}`);
  }

  // 2. JSON 字符串格式
  const r2 = await callTool(client, 'todo_write', {
    todos: JSON.stringify([
      { content: '任务X', status: 'pending', activeForm: '处理任务X' },
      { content: '任务Y', status: 'completed', activeForm: '处理任务Y' },
    ]),
  });
  if (r2.error) {
    record('todo_write JSON字符串格式', false, r2.error);
  } else {
    const d = parseResult<TodoWriteResult>(r2.data);
    record('todo_write JSON字符串格式', d.stats.total === 2 && d.stats.completed === 1, `total=${d.stats.total}`);
  }

  // 3. 非法 status
  const r3 = await callTool(client, 'todo_write', {
    todos: [{ content: '任务', status: 'unknown', activeForm: '处理任务' }],
  });
  record('todo_write 非法 status', r3.error !== null, r3.error ?? '');

  // 4. 缺少 content
  const r4 = await callTool(client, 'todo_write', {
    todos: [{ status: 'pending', activeForm: '处理任务' }],
  });
  record('todo_write 缺少 content', r4.error !== null, r4.error ?? '');

  // 5. 缺少 todos
  const r5 = await callTool(client, 'todo_write', {});
  record('todo_write 缺少 todos', r5.error !== null, r5.error ?? '');
}

// --------------------------- 汇总输出 ---------------------------
function printSummary(): void {
  const passed = outcomes.filter((o) => o.ok).length;
  const failed = outcomes.filter((o) => !o.ok).length;

  console.log('\n' + '═'.repeat(60));
  console.log('📋 测试总结');
  console.log('═'.repeat(60));
  console.log(`   总用例: ${outcomes.length}`);
  console.log(`   ✅ 通过: ${passed}`);
  console.log(`   ❌ 失败: ${failed}`);

  if (failed > 0) {
    console.log('\n  失败用例:');
    for (const o of outcomes.filter((o) => !o.ok)) {
      console.log(`     - ${o.name}: ${o.detail}`);
    }
  }
  console.log('═'.repeat(60));
  if (failed === 0) {
    console.log('🎉 全部通过! 无警告、无错误!');
  }
}

// --------------------------- 主流程 ---------------------------
async function main(): Promise<void> {
  console.log('=== MCP File-Bash-Tools (Linux) 6 个文件/待办工具测试 ===');
  console.log(`服务器: ${SERVER_PATH}\n`);

  setupWorkspace();
  console.log(`📁 测试工作区: ${WORK_DIR}\n`);

  const client = new Client({ name: 'linux-file-tools-test', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: SERVER_PATH, args: [] });

  try {
    console.log('🚀 连接 MCP 服务器...');
    await client.connect(transport);
    console.log('✅ 连接成功\n');

    await testWriteFile(client);
    await testReadFile(client);
    await testEditFile(client);
    await testGlob(client);
    await testGrep(client);
    await testTodoWrite(client);

    printSummary();
  } catch (e: any) {
    console.error('❌ 测试过程异常:', e?.message ?? e);
    if (e?.stack) console.error(e.stack);
  } finally {
    cleanupWorkspace();
    await client.close();
    console.log('\n🧹 测试工作区已清理, 客户端已关闭');
  }
}

main().then(() => {
  const failed = outcomes.filter((o) => !o.ok).length;
  process.exit(failed === 0 ? 0 : 1);
});
