import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';

interface BashResult {
  output: string;
  exitCode: number;
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

async function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// 计算 ping 回复数量
function countPingReplies(output: string): number {
  // 匹配 "来自 xxx 的回复" 或 "Reply from xxx"
  const matches = output.match(/来自|Reply from|bytes from/gi);
  return matches ? matches.length : 0;
}

async function main() {
  console.log('=== MCP bash_output 增量输出测试 ===\n');
  console.log('📋 测试流程:');
  console.log('   1. 启动后台 ping 223.5.5.5');
  console.log('   2. 第一次 bash_output - 查看输出');
  console.log('   3. 等待几秒');
  console.log('   4. 第二次 bash_output - 输出应该增加');
  console.log('   5. kill_shell 终止任务\n');
  
  const client = new Client({ name: 'test', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ 
    command: '/root/mcp/linux-file-bash-tools-mcp-go/dist/mcp-file-bash-tools-linux-arm64', 
    args: [] 
  });
  
  try {
    await client.connect(transport);
    console.log('✅ 已连接到 MCP 服务器\n');
    
    // ========== 步骤 1: 启动后台 ping ==========
    console.log('━'.repeat(50));
    console.log('📝 步骤 1: 启动后台 ping 223.5.5.5');
    
    const bashResult = await client.callTool({
      name: 'bash',
      arguments: {
        command: 'ping 223.5.5.5',
        timeout: 60000,
        run_in_background: true,
        description: '持续 ping 阿里 DNS'
      }
    });
    
    const bashData = JSON.parse((bashResult.content as any)[0].text) as BashResult;
    const shellId = bashData.shellId!;
    console.log(`✅ 后台任务已启动`);
    console.log(`   Shell ID: ${shellId}`);
    
    // 等待 ping 开始产生输出
    console.log('   ⏳ 等待 4 秒让 ping 产生输出...\n');
    await sleep(4000);
    
    // ========== 步骤 2: 第一次 bash_output ==========
    console.log('━'.repeat(50));
    console.log('📝 步骤 2: 第一次 bash_output');
    
    const output1Result = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId }
    });
    
    const output1Data = JSON.parse((output1Result.content as any)[0].text) as BashOutputResult;
    const pingCount1 = countPingReplies(output1Data.output);
    
    console.log(`   状态: ${output1Data.status}`);
    console.log(`   📊 Ping 回复数: ${pingCount1}`);
    console.log('   输出预览:');
    const preview1 = output1Data.output.split('\n').slice(0, 6).join('\n');
    console.log(preview1.split('\n').map(l => '      ' + l).join('\n'));
    
    // 等待更多 ping
    console.log('\n   ⏳ 等待 3 秒让 ping 继续产生输出...\n');
    await sleep(3000);
    
    // ========== 步骤 3: 第二次 bash_output ==========
    console.log('━'.repeat(50));
    console.log('📝 步骤 3: 第二次 bash_output');
    
    const output2Result = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId }
    });
    
    const output2Data = JSON.parse((output2Result.content as any)[0].text) as BashOutputResult;
    const pingCount2 = countPingReplies(output2Data.output);
    
    console.log(`   状态: ${output2Data.status}`);
    console.log(`   📊 Ping 回复数: ${pingCount2}`);
    console.log('   输出预览 (最后几行):');
    const lines = output2Data.output.trim().split('\n');
    const preview2 = lines.slice(-6).join('\n');
    console.log(preview2.split('\n').map(l => '      ' + l).join('\n'));
    
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
      arguments: { shell_id: shellId }
    });
    
    const killData = JSON.parse((killResult.content as any)[0].text) as KillShellResult;
    console.log(`   ✅ ${killData.message}`);
    
    // 等待进程终止
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
    
  } catch (error) {
    console.error('❌ 测试过程中发生错误:', error);
  } finally {
    await client.close();
    console.log('\n🧹 MCP 客户端已关闭');
  }
}

main().catch(console.error);
