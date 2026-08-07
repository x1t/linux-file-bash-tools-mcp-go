// ============================================================================
// bash_output 持续输出展示测试 (Linux)
// 启动一个每秒输出一行的后台任务, 轮询 5 次查看增量输出
// ============================================================================
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { SERVER_PATH, sleep, parseResult } from './shared.js';

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
  console.log('=== MCP bash_output 持续输出测试 (Linux) ===\n');
  console.log(`服务器: ${SERVER_PATH}\n`);

  const client = new Client({ name: 'test-output-linux', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: SERVER_PATH, args: [] });

  try {
    await client.connect(transport);
    console.log('✅ 已连接\n');

    // 每秒输出一行的后台任务
    const testCommand = 'for i in $(seq 1 5); do echo "Line $i"; sleep 1; done';
    console.log(`📝 启动后台任务: ${testCommand}\n`);

    const result = await client.callTool({
      name: 'bash',
      arguments: {
        command: testCommand,
        timeout: 30000,
        run_in_background: true,
      },
    });
    const data = parseResult<BashResult>(result);
    const shellId = data.shellId;
    if (!shellId) throw new Error('未获取到 shellId');
    console.log(`✅ Shell ID: ${shellId}\n`);

    for (let i = 0; i < 5; i++) {
      await sleep(1100);
      console.log(`🔍 检查 ${i + 1}:`);
      const outputResult = await client.callTool({
        name: 'bash_output',
        arguments: { bash_id: shellId },
      });
      const outData = parseResult<BashOutputResult>(outputResult);
      console.log(`   状态: ${outData.status}`);
      if (outData.output) {
        process.stdout.write(outData.output);
      } else {
        console.log('   (无新输出)');
      }
      console.log('');
    }

    const killResult = await client.callTool({
      name: 'kill_shell',
      arguments: { shell_id: shellId },
    });
    console.log(`🛑 已终止: ${parseResult<any>(killResult).message}`);
    await client.close();
  } catch (error: any) {
    console.error('❌ 测试过程中发生错误:', error?.message ?? error);
    await client.close();
  }
}

main().catch(console.error);
