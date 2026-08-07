// ============================================================================
// kill_shell 终止后台 ping 测试 (Linux)
// 1. 启动后台 ping 127.0.0.1
// 2. 轮询监控输出
// 3. 检查 ping 进程存在
// 4. kill_shell 终止
// 5. 验证 ping 进程已消失
// ============================================================================
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { SERVER_PATH, sleep, parseResult, countPingReplies, findProcess } from './shared.js';

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

async function main(): Promise<void> {
  console.log('=== MCP kill_shell 测试 (Linux, 非 Node 进程) ===\n');
  console.log(`服务器: ${SERVER_PATH}\n`);

  const client = new Client({ name: 'test-ping-linux', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: SERVER_PATH, args: [] });

  await client.connect(transport);
  console.log('✅ 已连接到 MCP 服务器\n');

  // 后台持续 ping 本机
  const testCommand = 'ping 127.0.0.1';
  console.log(`📝 启动后台任务: ${testCommand}`);

  const result = await client.callTool({
    name: 'bash',
    arguments: {
      command: testCommand,
      timeout: 30000,
      run_in_background: true,
      description: '测试 ping 进程终止',
    },
  });

  const data = parseResult<BashResult>(result);
  const shellId = data.shellId;
  if (!shellId) throw new Error('未获取到 shellId');
  console.log(`✅ Shell ID: ${shellId}\n`);

  // 等待并监控输出
  console.log('⏳ 等待并监控输出 (3 秒)...');
  for (let i = 0; i < 3; i++) {
    await sleep(1000);
    console.log(`\n🔍 第 ${i + 1} 次检查输出:`);
    const outputResult = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId },
    });
    const outputData = parseResult<BashOutputResult>(outputResult);
    if (outputData.output) {
      console.log('----------------------------------------');
      process.stdout.write(outputData.output);
      console.log('----------------------------------------');
    } else {
      console.log('   (无新输出)');
    }
  }

  // 检查 ping 进程
  console.log('\n🔍 终止前检查 ping 进程:');
  const beforePing = findProcess('ping');
  if (beforePing.length > 0) {
    console.log(`   ✅ 找到 ${beforePing.length} 个 ping 进程`);
    beforePing.forEach((l) => console.log(`      ${l.trim()}`));
  } else {
    console.log('   ⚠️ 未找到 ping 进程');
  }

  // 终止任务
  console.log(`\n🛑 调用 kill_shell (shell_id: ${shellId})`);
  const killResult = await client.callTool({
    name: 'kill_shell',
    arguments: { shell_id: shellId },
  });
  console.log('   结果:', parseResult<any>(killResult).message);

  await sleep(2000);

  // 再次检查
  console.log('\n🔍 终止后检查 ping 进程:');
  const afterPing = findProcess('ping');
  if (afterPing.length > 0) {
    console.log(`   ❌ 仍有 ${afterPing.length} 个 ping 进程 (失败)`);
    afterPing.forEach((l) => console.log(`      ${l.trim()}`));
  } else {
    console.log('   ✅ 无 ping 进程 (成功终止)');
  }

  await client.close();

  console.log('\n=== 测试完成 ===');
  console.log(afterPing.length === 0 ? '✅ 测试通过!' : '❌ 测试失败!');
}

main().catch(console.error);
