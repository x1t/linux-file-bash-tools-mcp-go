// ============================================================================
// bash_output 增量输出测试 (Linux)
// 1. 启动后台 ping 127.0.0.1
// 2. 第一次 bash_output 查看输出
// 3. 等待几秒后第二次 bash_output, 输出应增加
// 4. kill_shell 终止任务
// ============================================================================
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { SERVER_PATH, sleep, parseResult, countPingReplies } from './shared.js';

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

async function main(): Promise<void> {
  console.log('=== MCP bash_output 增量输出测试 (Linux) ===\n');
  console.log('📋 测试流程:');
  console.log('   1. 启动后台 ping 127.0.0.1');
  console.log('   2. 第一次 bash_output - 查看输出');
  console.log('   3. 等待几秒');
  console.log('   4. 第二次 bash_output - 输出应该增加');
  console.log('   5. kill_shell 终止任务\n');
  console.log(`服务器: ${SERVER_PATH}\n`);

  const client = new Client({ name: 'test-bash-output-linux', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: SERVER_PATH, args: [] });

  try {
    await client.connect(transport);
    console.log('✅ 已连接到 MCP 服务器\n');

    // ========== 步骤 1: 启动后台 ping ==========
    console.log('━'.repeat(50));
    console.log('📝 步骤 1: 启动后台 ping 127.0.0.1');
    const bashResult = await client.callTool({
      name: 'bash',
      arguments: {
        command: 'ping 127.0.0.1',
        timeout: 60000,
        run_in_background: true,
        description: '持续 ping 本机',
      },
    });
    const bashData = parseResult<BashResult>(bashResult);
    const shellId = bashData.shellId;
    if (!shellId) throw new Error('未获取到 shellId');
    console.log('✅ 后台任务已启动');
    console.log(`   Shell ID: ${shellId}`);

    // 等待 ping 开始产生输出
    console.log('   ⏳ 等待 3 秒让 ping 产生输出...\n');
    await sleep(3000);

    // ========== 步骤 2: 第一次 bash_output ==========
    console.log('━'.repeat(50));
    console.log('📝 步骤 2: 第一次 bash_output');
    const output1Result = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId },
    });
    const output1Data = parseResult<BashOutputResult>(output1Result);
    const pingCount1 = countPingReplies(output1Data.output);
    console.log(`   状态: ${output1Data.status}`);
    console.log(`   📊 Ping 回复数: ${pingCount1}`);
    console.log('   输出预览:');
    const preview1 = output1Data.output.split('\n').slice(0, 6).join('\n');
    console.log(preview1.split('\n').map((l) => `      ${l}`).join('\n'));

    // 等待更多 ping
    console.log('\n   ⏳ 等待 2 秒让 ping 继续产生输出...\n');
    await sleep(2000);

    // ========== 步骤 3: 第二次 bash_output ==========
    console.log('━'.repeat(50));
    console.log('📝 步骤 3: 第二次 bash_output');
    const output2Result = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId },
    });
    const output2Data = parseResult<BashOutputResult>(output2Result);
    const pingCount2 = countPingReplies(output2Data.output);
    console.log(`   状态: ${output2Data.status}`);
    console.log(`   📊 Ping 回复数: ${pingCount2}`);
    console.log('   输出预览 (最后几行):');
    const lines = output2Data.output.trim().split('\n');
    const preview2 = lines.slice(-6).join('\n');
    console.log(preview2.split('\n').map((l) => `      ${l}`).join('\n'));

    // ========== 步骤 4: 验证输出增长 ==========
    console.log('\n' + '━'.repeat(50));
    console.log('📝 步骤 4: 验证输出增长');
    if (pingCount2 > pingCount1) {
      console.log(`   ✅ 输出正常增长: ${pingCount1} → ${pingCount2} (增加了 ${pingCount2 - pingCount1} 条)`);
    } else {
      console.log(`   ⚠️ 输出未增长: 第一次=${pingCount1}, 第二次=${pingCount2}`);
    }

    // ========== 步骤 5: kill_shell ==========
    console.log('\n' + '━'.repeat(50));
    console.log('📝 步骤 5: kill_shell 终止任务');
    const killResult = await client.callTool({
      name: 'kill_shell',
      arguments: { shell_id: shellId },
    });
    const killData = parseResult<KillShellResult>(killResult);
    console.log(`   ✅ ${killData.message}`);

    await sleep(1000);

    // ========== 总结 ==========
    console.log('\n' + '═'.repeat(50));
    console.log('📋 测试总结:');
    console.log(`   第一次 bash_output: ${pingCount1} 条 ping 回复`);
    console.log(`   第二次 bash_output: ${pingCount2} 条 ping 回复`);
    const success = pingCount1 > 0 && pingCount2 > pingCount1;
    if (success) {
      console.log('\n   🎉 测试通过! bash_output 正确获取了增量输出!');
    } else {
      console.log('\n   ❌ 测试失败!');
      if (pingCount1 === 0) console.log('      原因: 第一次获取没有输出');
      if (pingCount2 <= pingCount1) console.log('      原因: 第二次输出没有增长');
    }
    console.log('═'.repeat(50));
  } catch (error: any) {
    console.error('❌ 测试过程中发生错误:', error?.message ?? error);
  } finally {
    await client.close();
    console.log('\n🧹 MCP 客户端已关闭');
  }
}

main().catch(console.error);
