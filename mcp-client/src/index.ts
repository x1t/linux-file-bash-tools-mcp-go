// ============================================================================
// kill_shell 测试 (Linux): 验证终止后台任务时整个进程组(含子进程)都被清理,
// 防止孤儿进程泄漏
// ============================================================================
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { existsSync, readFileSync, unlinkSync } from 'fs';
import { SERVER_PATH, sleep, parseResult, isProcessAlive } from './shared.js';

interface BashResult {
  output: string;
  exitCode: number;
  killed: boolean;
  shellId?: string;
}

interface KillShellResult {
  message: string;
  shell_id: string;
}

const parentPidFile = '/tmp/mcp_index_parent.pid';
const childPidFile = '/tmp/mcp_index_child.pid';

function cleanupPidFiles(): void {
  for (const f of [parentPidFile, childPidFile]) {
    if (existsSync(f)) unlinkSync(f);
  }
}

function readPid(file: string): number {
  if (!existsSync(file)) return NaN;
  return Number(readFileSync(file, 'utf8').trim());
}

async function main(): Promise<void> {
  console.log('=== MCP kill_shell 进程组清理测试 (Linux) ===\n');
  console.log(`服务器: ${SERVER_PATH}\n`);

  cleanupPidFiles();

  const client = new Client({ name: 'test-kill-shell-linux', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: SERVER_PATH, args: [] });

  try {
    await client.connect(transport);
    console.log('✅ 已连接 MCP 服务器\n');

    // 步骤 1: 启动后台任务, 记录父进程与子进程 PID
    console.log('📝 步骤 1: 启动后台任务 (含子进程)');
    const bgCommand =
      `echo $$ > ${parentPidFile}; ` +
      `sleep 300 & echo $! > ${childPidFile}; ` +
      `for i in $(seq 1 1000); do echo "tick $i"; sleep 1; done`;
    const bg = await client.callTool({
      name: 'bash',
      arguments: {
        command: bgCommand,
        timeout: 60000,
        run_in_background: true,
        description: 'kill_shell 进程组清理测试',
      },
    });
    const bgData = parseResult<BashResult>(bg);
    const shellId = bgData.shellId;
    if (!shellId) throw new Error('未获取到 shellId');
    console.log(`   Shell ID: ${shellId}\n`);

    // 读取 PID 并确认存活
    await sleep(800);
    const parentPid = readPid(parentPidFile);
    const childPid = readPid(childPidFile);
    const startedOk =
      !Number.isNaN(parentPid) && !Number.isNaN(childPid) &&
      isProcessAlive(parentPid) && isProcessAlive(childPid);
    console.log(`   父进程 PID=${parentPid}, 子进程 PID=${childPid}`);
    console.log(`   父进程存活=${isProcessAlive(parentPid)}, 子进程存活=${isProcessAlive(childPid)}`);
    console.log(`${startedOk ? '✅' : '❌'} 后台任务与子进程均已启动\n`);

    // 步骤 2: kill_shell 终止
    console.log('🛑 步骤 2: kill_shell 终止后台任务');
    const kill = await client.callTool({ name: 'kill_shell', arguments: { shell_id: shellId } });
    const killData = parseResult<KillShellResult>(kill);
    console.log(`   ${killData.message}\n`);

    // 步骤 3: 验证进程组全部终止
    await sleep(1000);
    const parentAlive = isProcessAlive(parentPid);
    const childAlive = isProcessAlive(childPid);
    console.log(`   父进程存活=${parentAlive}, 子进程存活=${childAlive}`);
    const killedOk = !parentAlive && !childAlive;
    console.log(`${killedOk ? '✅' : '❌'} 进程组(含子进程)已被全部终止\n`);

    console.log('=== 测试总结 ===');
    console.log(
      startedOk && killedOk
        ? '   🎉 全部通过! kill_shell 成功清理了整个进程组'
        : '   ❌ 测试失败!',
    );
  } catch (e: any) {
    console.error('❌ 测试过程中发生错误:', e?.message ?? e);
  } finally {
    cleanupPidFiles();
    await client.close();
    console.log('\n🧹 客户端已关闭');
  }
}

main().catch(console.error);
