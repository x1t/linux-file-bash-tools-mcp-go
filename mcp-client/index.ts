import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { spawn } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

interface BashResult {
  output: string;
  exitCode: number;
  killed?: boolean;
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

async function checkProcesses(stage: string): Promise<void> {
  console.log(`\n🔍 [${stage}] 检查进程状态:`);
  
  // 检查 node 进程
  const nodeCheck = spawn('ps', ['-ef']);
  let nodeOutput = '';
  
  nodeCheck.stdout.on('data', (data) => {
    nodeOutput += data.toString();
  });
  
  await new Promise<void>((resolve) => {
    nodeCheck.on('close', () => {
      const nodeLines = nodeOutput.split('\n').filter(line => 
        line.toLowerCase().includes('node ') && !line.includes('grep')
      );
      
      if (nodeLines.length > 0) {
        console.log(`   ❌ 发现 ${nodeLines.length} 个 node 进程:`);
        nodeLines.forEach(line => {
          console.log(`      ${line.trim()}`);
        });
      } else {
        console.log('   ✅ 未发现 node 进程');
      }
      resolve();
    });
  });
  
  // 检查端口部分移除，因为 netstat可能不存在
  console.log('   ℹ️ 跳过端口检查 (netstat未必可用)');
}

async function main() {
  console.log('=== MCP Bash Tools Kill Shell 测试 ===\n');
  
  // 服务器路径
  const serverPath = '/root/mcp/linux-file-bash-tools-mcp-go/dist/mcp-file-bash-tools-linux-arm64';
  console.log(`✅ 服务器路径: ${serverPath}\n`);
  
  // 创建客户端
  const client = new Client({
    name: 'test-client',
    version: '1.0.0'
  }, {
    capabilities: {}
  });
  
  // 创建 stdio 传输
  const transport = new StdioClientTransport({
    command: serverPath,
    args: []
  });
  
  try {
    // 连接到服务器
    console.log('🚀 连接到 MCP 服务器...');
    await client.connect(transport);
    console.log('✅ 连接成功\n');
    
    // 步骤 0: 初始进程检查
    await checkProcesses('初始');

    // 步骤 1: 启动后台任务
    console.log('📝 步骤 1: 启动后台任务 (ping)');
    console.log('   命令: ping 223.5.5.5');
    
    const bashResult = await client.callTool({
      name: 'bash',
      arguments: {
        command: 'ping 223.5.5.5',
        timeout: 30000,
        run_in_background: true,
        description: '启动 Ping'
      }
    });
    
    const bashData = JSON.parse((bashResult.content as any)[0].text) as BashResult;
    const shellId = bashData.shellId;
    
    if (!shellId) {
      throw new Error('未获取到 shellId');
    }
    
    console.log('✅ 后台任务已启动');
    console.log(`   Shell ID: ${shellId}`);
    console.log(`   输出: ${bashData.output}`);
    
    // 步骤 2: 等待服务器启动
    console.log('\n⏳ 步骤 2: 等待服务器启动 (10秒)');
    for (let i = 1; i <= 10; i++) {
      await sleep(1000);
      console.log(`   等待中... ${i}/10`);
      
      // 每隔 2 秒检查一次输出
      if (i % 2 === 0) {
        try {
          const outputResult = await client.callTool({
            name: 'bash_output',
            arguments: {
              bash_id: shellId
            }
          });
          
          const outputData = JSON.parse((outputResult.content as any)[0].text) as BashOutputResult;
          console.log(`   状态: ${outputData.status}, ExitCode: ${outputData.exitCode}`);
          
          if (outputData.output) {
            console.log(`   FULL OUTPUT:\n${outputData.output}`);
          }
        } catch (err) {
          console.log(`   ⚠️ 获取输出失败: ${err}`);
        }
      }
    }
    
    // 步骤 3: 检查进程状态（终止前）
    await checkProcesses('终止前');
    
    // 步骤 4: 终止任务
    console.log('\n🛑 步骤 3: 终止后台任务');
    console.log(`   Shell ID: ${shellId}`);
    
    const killResult = await client.callTool({
      name: 'kill_shell',
      arguments: {
        shell_id: shellId
      }
    });
    
    const killData = JSON.parse((killResult.content as any)[0].text) as KillShellResult;
    console.log(`✅ ${killData.message}`);
    
    // 步骤 5: 等待进程终止
    console.log('\n⏳ 步骤 4: 等待进程终止 (5秒)');
    for (let i = 1; i <= 5; i++) {
      await sleep(1000);
      console.log(`   等待中... ${i}/5`);
    }
    
    // 步骤 6: 验证进程是否被终止
    await checkProcesses('终止后');
    
    // 总结
    console.log('\n=== 测试总结 ===');
    
    // 再次检查进程
    const finalNodeCheck = spawn('ps', ['-ef']);
    let finalOutput = '';
    
    finalNodeCheck.stdout.on('data', (data) => {
      finalOutput += data.toString();
    });
    
    await new Promise<void>((resolve) => {
      finalNodeCheck.on('close', () => {
        const hasNode = finalOutput.toLowerCase().includes('node ') && !finalOutput.includes('grep');
        
        if (!hasNode) {
          console.log('✅ 测试通过: kill_shell 成功终止了所有进程');
        } else {
          console.log('❌ 测试失败: kill_shell 未能完全终止进程');
          console.log('   - 仍有 node 进程在运行');
        }
        resolve();
      });
    });
    
  } catch (error) {
    console.error('\n❌ 测试过程中发生错误:', error);
    if (error instanceof Error) {
      console.error('   错误详情:', error.message);
      console.error('   堆栈:', error.stack);
    }
  } finally {
    // 清理：关闭客户端
    console.log('\n🧹 清理: 关闭 MCP 客户端...');
    await client.close();
    console.log('✅ 客户端已关闭');
  }
  
  console.log('\n=== 测试完成 ===');
}

main().catch(console.error);
