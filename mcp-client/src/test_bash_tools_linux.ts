// ============================================================================
// 测试: Linux 版 mcp-file-bash-tools 的 3 个 bash 工具
//   - bash        前台/后台/超时转后台/参数校验
//   - bash_output 增量输出/正则过滤/参数校验
//   - kill_shell  终止后台进程/进程组清理/参数校验
// 通过 mcp-client (TypeScript SDK) 连接 dist 二进制进行测试
// ============================================================================
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { spawnSync } from 'child_process';
import { readFileSync, unlinkSync, existsSync } from 'fs';

const serverPath = '/root/mcp/linux-file-bash-tools-mcp-go/dist/mcp-file-bash-tools-linux-amd64';
const pidFile = '/tmp/mcp_bash_kill_test.pid';

// --------------------------- 结果类型定义 ---------------------------
interface BashResult {
  output: string;
  exitCode: number;
  killed: boolean;
  shellId?: string;
}

interface BashOutputResult {
  output: string;
  status: string;
  exitCode?: number;
}

interface KillShellResult {
  message: string;
  shell_id: string;
}

// 统一解析返回: 优先 structuredContent, 回退到 content[0].text 的 JSON
function parseResult<T>(result: any): T {
  if (result.structuredContent) return result.structuredContent as T;
  return JSON.parse((result.content as any)[0].text) as T;
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
  const mark = ok ? '✅' : '❌';
  console.log(`${mark} ${name}${detail ? ` — ${detail}` : ''}`);
}

// --------------------------- 工具函数 ---------------------------
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// 检查 pid 进程是否存活 (kill -0 成功=存活)
function isProcessAlive(pid: number): boolean {
  const res = spawnSync('bash', ['-c', `kill -0 ${pid} 2>/dev/null`]);
  return res.status === 0;
}

function cleanupPidFile(): void {
  if (existsSync(pidFile)) unlinkSync(pidFile);
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
      const text = (result.content as any)[0]?.text ?? '';
      return { data: null, error: text };
    }
    return { data: result, error: null };
  } catch (e: any) {
    return { data: null, error: e?.message ?? String(e) };
  }
}

// --------------------------- 测试: bash 前台 ---------------------------
async function testBashForeground(client: Client): Promise<void> {
  console.log('\n━━━ 测试 bash 前台执行 ━━━');

  // 1. 简单命令
  const r1 = await callTool(client, 'bash', { command: 'echo "hello from MCP bash"', timeout: 10000 });
  if (r1.error) {
    record('bash 前台简单命令', false, r1.error);
  } else {
    const d = parseResult<BashResult>(r1.data);
    record('bash 前台简单命令', d.exitCode === 0 && d.output.includes('hello from MCP bash'),
      `exitCode=${d.exitCode}, output="${d.output.trim()}"`);
  }

  // 2. 合并 stdout + stderr
  const r2 = await callTool(client, 'bash', {
    command: 'echo "to-stdout"; echo "to-stderr" >&2',
    timeout: 10000,
  });
  if (r2.error) {
    record('bash 合并 stdout/stderr', false, r2.error);
  } else {
    const d = parseResult<BashResult>(r2.data);
    record('bash 合并 stdout/stderr',
      d.exitCode === 0 && d.output.includes('to-stdout') && d.output.includes('to-stderr'),
      `exitCode=${d.exitCode}, output="${d.output.replace(/\n/g, ' | ')}"`);
  }

  // 3. 非零退出码
  const r3 = await callTool(client, 'bash', { command: 'exit 3', timeout: 10000 });
  if (r3.error) {
    record('bash 非零退出码', false, r3.error);
  } else {
    const d = parseResult<BashResult>(r3.data);
    record('bash 非零退出码', d.exitCode === 3, `exitCode=${d.exitCode}`);
  }

  // 4. 缺少 command 参数
  const r4 = await callTool(client, 'bash', { timeout: 10000 });
  record('bash 缺少 command 参数', r4.error !== null, r4.error ?? '');

  // 5. 非法 timeout (0)
  const r5 = await callTool(client, 'bash', { command: 'echo hi', timeout: 0 });
  record('bash 非法 timeout=0', r5.error !== null, r5.error ?? '');

  // 6. 非法 timeout (超上限 600000)
  const r6 = await callTool(client, 'bash', { command: 'echo hi', timeout: 600001 });
  record('bash 非法 timeout>600000', r6.error !== null, r6.error ?? '');
}

// --------------------------- 测试: bash 超时自动转后台 ---------------------------
async function testBashTimeoutAutoBackground(client: Client): Promise<void> {
  console.log('\n━━━ 测试 bash 超时自动转后台 ━━━');

  // sleep 3 秒, 超时 500ms -> 应自动转后台并返回 shellId
  const r = await callTool(client, 'bash', {
    command: 'sleep 3; echo "finished after timeout"',
    timeout: 500,
  });
  if (r.error) {
    record('bash 超时自动转后台', false, r.error);
    return;
  }
  const d = parseResult<BashResult>(r.data);
  const hasShellId = !!d.shellId;
  const isAutoBg = d.output.includes('automatically converted to background task');
  record('bash 超时自动转后台', hasShellId && isAutoBg,
    `shellId=${d.shellId ?? '无'}, output="${d.output.replace(/\n/g, ' | ').slice(0, 80)}"`);

  if (hasShellId) {
    // 稍等进程自然完成, 再确认 bash_output 能读到最终输出
    await sleep(4000);
    const poll = await callTool(client, 'bash_output', { bash_id: d.shellId! });
    if (poll.error) {
      record('超时后台任务 bash_output', false, poll.error);
    } else {
      const od = parseResult<BashOutputResult>(poll.data);
      record('超时后台任务 bash_output',
        od.output.includes('finished after timeout') && od.status !== 'running',
        `status=${od.status}, exitCode=${od.exitCode}, output="${od.output.replace(/\n/g, ' | ')}"`);
    }
  }
}

// --------------------------- 测试: bash 后台 + bash_output ---------------------------
async function testBashBackgroundAndOutput(client: Client): Promise<void> {
  console.log('\n━━━ 测试 bash 后台 + bash_output 增量输出 ━━━');

  // 启动后台任务: 每秒输出一行, 共 30 行
  const r = await callTool(client, 'bash', {
    command: 'for i in $(seq 1 30); do echo "progress line $i"; sleep 0.5; done',
    timeout: 60000,
    run_in_background: true,
    description: '后台增量输出测试',
  });
  if (r.error) {
    record('bash 启动后台任务', false, r.error);
    return;
  }
  const d = parseResult<BashResult>(r.data);
  const shellId = d.shellId;
  record('bash 启动后台任务返回 shellId', !!shellId, `shellId=${shellId ?? '无'}`);

  if (!shellId) return;

  // 第一次 bash_output
  await sleep(1600);
  const p1 = await callTool(client, 'bash_output', { bash_id: shellId });
  if (p1.error) {
    record('bash_output 第一次读取', false, p1.error);
    return;
  }
  const o1 = parseResult<BashOutputResult>(p1.data);
  const count1 = (o1.output.match(/progress line/g) ?? []).length;
  record('bash_output 第一次读取(running)',
    o1.status === 'running' && count1 > 0,
    `status=${o1.status}, 行数=${count1}`);

  // 第二次 bash_output (输出应增长)
  await sleep(1600);
  const p2 = await callTool(client, 'bash_output', { bash_id: shellId });
  if (p2.error) {
    record('bash_output 第二次读取', false, p2.error);
    return;
  }
  const o2 = parseResult<BashOutputResult>(p2.data);
  const count2 = (o2.output.match(/progress line/g) ?? []).length;
  record('bash_output 第二次读取(输出增长)', count2 > count1,
    `行数 ${count1} → ${count2}`);

  // 正则过滤: 只保留偶数行
  const p3 = await callTool(client, 'bash_output', { bash_id: shellId, filter: 'progress line [0-9]*[02468]' });
  if (p3.error) {
    record('bash_output 正则过滤', false, p3.error);
  } else {
    const o3 = parseResult<BashOutputResult>(p3.data);
    const hasOdd = /progress line [0-9]*[13579]/.test(o3.output);
    const hasEven = /progress line [0-9]*[02468]/.test(o3.output);
    record('bash_output 正则过滤(仅偶数行)', hasEven && !hasOdd,
      `含奇数行=${hasOdd}, 含偶数行=${hasEven}`);
  }

  // 等任务完成, 验证 status=completed
  await sleep(14000);
  const p4 = await callTool(client, 'bash_output', { bash_id: shellId });
  if (p4.error) {
    record('bash_output 任务完成状态', false, p4.error);
  } else {
    const o4 = parseResult<BashOutputResult>(p4.data);
    record('bash_output 任务完成状态',
      o4.status === 'completed' && (o4.exitCode === 0 || o4.exitCode === undefined),
      `status=${o4.status}, exitCode=${o4.exitCode}`);
  }

  // 参数校验
  const p5 = await callTool(client, 'bash_output', {});
  record('bash_output 缺少 bash_id', p5.error !== null, p5.error ?? '');

  const p6 = await callTool(client, 'bash_output', { bash_id: 'nonexistent-id-123' });
  record('bash_output 不存在的 bash_id', p6.error !== null, p6.error ?? '');
}

// --------------------------- 测试: kill_shell ---------------------------
async function testKillShell(client: Client): Promise<void> {
  console.log('\n━━━ 测试 kill_shell 终止后台进程 ━━━');

  cleanupPidFile();

  // 启动后台长任务, 记录 PID
  const r = await callTool(client, 'bash', {
    command: `echo $$ > ${pidFile}; for i in $(seq 1 300); do echo "tick $i"; sleep 1; done`,
    timeout: 60000,
    run_in_background: true,
    description: 'kill_shell 测试用长任务',
  });
  if (r.error) {
    record('kill_shell 前置: 启动后台任务', false, r.error);
    return;
  }
  const d = parseResult<BashResult>(r.data);
  const shellId = d.shellId;
  if (!shellId) {
    record('kill_shell 前置: 未获取 shellId', false, '');
    return;
  }

  await sleep(800);
  const pid = existsSync(pidFile) ? Number(readFileSync(pidFile, 'utf8').trim()) : NaN;
  record('kill_shell 前置: 获取后台进程 PID', !Number.isNaN(pid) && isProcessAlive(pid),
    Number.isNaN(pid) ? '无法读取 PID 文件' : `pid=${pid}, 存活=${isProcessAlive(pid)}`);

  // 执行 kill_shell
  const k = await callTool(client, 'kill_shell', { shell_id: shellId });
  if (k.error) {
    record('kill_shell 终止后台进程', false, k.error);
    return;
  }
  const kd = parseResult<KillShellResult>(k.data);
  record('kill_shell 终止后台进程', kd.message.includes('Successfully killed'),
    kd.message);

  // 等待进程组退出
  await sleep(1000);

  // 验证进程已消失
  const aliveAfter = !Number.isNaN(pid) && isProcessAlive(pid);
  record('kill_shell 后进程已终止', !aliveAfter,
    `pid=${pid}, 存活=${aliveAfter}`);

  // 再次 kill 同一个 shell_id (已被移除) -> 应报错
  const k2 = await callTool(client, 'kill_shell', { shell_id: shellId });
  record('kill_shell 重复终止(已移除)', k2.error !== null, k2.error ?? '');

  // 参数校验
  const k3 = await callTool(client, 'kill_shell', {});
  record('kill_shell 缺少 shell_id', k3.error !== null, k3.error ?? '');

  const k4 = await callTool(client, 'kill_shell', { shell_id: 'nonexistent-id-456' });
  record('kill_shell 不存在的 shell_id', k4.error !== null, k4.error ?? '');

  cleanupPidFile();
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
  console.log('=== MCP File-Bash-Tools (Linux) 3 个 bash 工具测试 ===');
  console.log(`服务器: ${serverPath}\n`);

  const client = new Client({ name: 'linux-test-client', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: serverPath, args: [] });

  try {
    console.log('🚀 连接 MCP 服务器...');
    await client.connect(transport);
    console.log('✅ 连接成功\n');

    await testBashForeground(client);
    await testBashTimeoutAutoBackground(client);
    await testBashBackgroundAndOutput(client);
    await testKillShell(client);

    printSummary();
  } catch (e: any) {
    console.error('❌ 测试过程异常:', e?.message ?? e);
    if (e?.stack) console.error(e.stack);
  } finally {
    cleanupPidFile();
    await client.close();
    console.log('\n🧹 客户端已关闭');
  }
}

main().then(() => {
  const failed = outcomes.filter((o) => !o.ok).length;
  process.exit(failed === 0 ? 0 : 1);
});
