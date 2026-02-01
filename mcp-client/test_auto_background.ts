import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';

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

async function main() {
  console.log('=== MCP Auto-Background on Timeout Test ===\n');
  
  const client = new Client({ name: 'test-auto-bg', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ 
    command: '/root/mcp/linux-file-bash-tools-mcp-go/dist/mcp-file-bash-tools-linux-arm64', 
    args: [] 
  });
  
  try {
    await client.connect(transport);
    console.log('✅ Connected to MCP server\n');
    
    // Step 1: Execute foreground command with short timeout
    const timeoutMs = 3000;
    console.log(`📝 Step 1: Running 'ping' in foreground with ${timeoutMs}ms timeout`);
    console.log('   Expected behavior: Wait 3s, then receive auto-background conversion message.');
    
    const startTime = Date.now();
    
    const bashResult = await client.callTool({
      name: 'bash',
      arguments: {
        command: 'ping 223.5.5.5',
        timeout: timeoutMs,
        run_in_background: false,
        description: 'Foreground Ping Test'
      }
    });
    
    const duration = Date.now() - startTime;
    console.log(`   ⏱️ Execution took: ${duration}ms`);
    
    const bashData = JSON.parse((bashResult.content as any)[0].text) as BashResult;
    console.log('   📩 Response:', bashData);
    
    // Verify response format
    const expectedMsgPart = 'automatically converted to background task';
    if (bashData.output.includes(expectedMsgPart)) {
        console.log('   ✅ Success: Received auto-background conversion message.');
    } else {
        console.error('   ❌ Failure: Did not receive conversion message.');
        throw new Error('Test Failed');
    }

    if (!bashData.shellId) {
        console.error('   ❌ Failure: No ShellID returned.');
        throw new Error('Test Failed');
    }
    
    const shellId = bashData.shellId;
    console.log(`   ✅ Shell ID: ${shellId}`);
    
    // Step 2: Verify process is running in background
    console.log('\n📝 Step 2: Verifying process status via bash_output');
    
    const outputResult = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId }
    });
    
    const outputData = JSON.parse((outputResult.content as any)[0].text) as BashOutputResult;
    console.log('   📩 Output status:', outputData.status);
    
    if (outputData.status === 'running') {
        console.log('   ✅ Success: Process is still running in background.');
    } else {
        console.error(`   ❌ Failure: Process status is ${outputData.status}`);
    }

    // Step 3: Terminate process
    console.log('\n📝 Step 3: Terminating process');
    
    const killResult = await client.callTool({
        name: 'kill_shell',
        arguments: { shell_id: shellId }
    });
    
    const killData = JSON.parse((killResult.content as any)[0].text) as KillShellResult;
    console.log(`   ✅ ${killData.message}`);
    
    // Final check
    await sleep(1000);
    console.log('\n🎉 Test Passed: Auto-background conversion works!');

  } catch (error) {
    console.error('\n❌ Test execution failed:', error);
  } finally {
    await client.close();
    console.log('\n🧹 Client closed');
  }
}

main().catch(console.error);
