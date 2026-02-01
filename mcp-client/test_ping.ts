import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { spawn, execSync } from 'child_process';

function checkProcess(name: string): string[] {
  try {
    const output = execSync(`tasklist`, { encoding: 'utf8' });
    return output.split('\n').filter(line => line.toLowerCase().includes(name.toLowerCase()));
  } catch {
    return [];
  }
}

async function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function main() {
  console.log('=== MCP Kill Shell 测试 (非 Node 进程) ===\n');
  
  const client = new Client({ name: 'test', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ 
    command: 'H:/mcp/windwos-bash-tools-mcp-go/dist/bash-tools.exe', 
    args: [] 
  });
  
  await client.connect(transport);
  console.log('✅ 已连接到 MCP 服务器\n');
  
  // 测试命令: ping -t (持续 ping)
  const testCommand = 'ping.exe -t 127.0.0.1';
  console.log(`📝 启动后台任务: ${testCommand}`);
  
  const result = await client.callTool({
    name: 'bash',
    arguments: {
      command: testCommand,
      timeout: 30000,
      run_in_background: true,
      description: '测试 ping 进程终止'
    }
  });
  
  const data = JSON.parse((result.content as any)[0].text);
  const shellId = data.shellId;
  console.log(`✅ Shell ID: ${shellId}\n`);
  
  // 等待进程启动并监控输出
  console.log('⏳ 等待并监控输出 (3秒)...');
  for (let i = 0; i < 3; i++) {
    await sleep(1000);
    console.log(`\n🔍 第 ${i + 1} 次检查输出:`);
    const outputResult = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId }
    });
    const outputData = JSON.parse((outputResult.content as any)[0].text);
    if (outputData.output) {
        console.log('----------------------------------------');
        process.stdout.write(outputData.output);
        console.log('----------------------------------------');
    } else {
        console.log('   (无新输出)');
    }
  }
  
  // 检查 PING 进程
  console.log('\n🔍 终止前检查 PING 进程:');
  const beforePing = checkProcess('PING.EXE');
  if (beforePing.length > 0) {
    console.log(`   ✅ 找到 ${beforePing.length} 个 PING 进程`);
    beforePing.forEach(l => console.log(`      ${l.trim()}`));
  } else {
    console.log('   ⚠️ 未找到 PING 进程');
  }
  
  // 终止任务
  console.log(`\n🛑 调用 kill_shell (shell_id: ${shellId})`);
  const killResult = await client.callTool({
    name: 'kill_shell',
    arguments: { shell_id: shellId }
  });
  console.log('   结果:', JSON.parse((killResult.content as any)[0].text).message);
  
  // 等待进程终止
  await sleep(2000);
  
  // 再次检查
  console.log('\n🔍 终止后检查 PING 进程:');
  const afterPing = checkProcess('PING.EXE');
  if (afterPing.length > 0) {
    console.log(`   ❌ 仍有 ${afterPing.length} 个 PING 进程 (失败)`);
    afterPing.forEach(l => console.log(`      ${l.trim()}`));
  } else {
    console.log('   ✅ 无 PING 进程 (成功终止)');
  }
  
  await client.close();
  
  console.log('\n=== 测试完成 ===');
  console.log(afterPing.length === 0 ? '✅ 测试通过!' : '❌ 测试失败!');
}

main().catch(console.error);
