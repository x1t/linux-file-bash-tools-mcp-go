import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';

async function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function main() {
  console.log('=== MCP Output Test (Simple Loop) ===\n');
  
  const client = new Client({ name: 'test-output', version: '1.0.0' }, { capabilities: {} });
  const transport = new StdioClientTransport({ 
    command: 'H:/mcp/windwos-bash-tools-mcp-go/dist/bash-tools.exe', 
    args: [] 
  });
  
  await client.connect(transport);
  console.log('✅ Connected\n');
  
  // Script that outputs every second
  const testCommand = 'for ($i=0; $i -lt 5; $i++) { Write-Output "Line $i"; Start-Sleep -Seconds 1 }';
  console.log(`📝 Starting background task: ${testCommand}`);
  
  const result = await client.callTool({
    name: 'bash',
    arguments: {
      command: testCommand,
      timeout: 30000,
      run_in_background: true
    }
  });
  
  const shellId = JSON.parse((result.content as any)[0].text).shellId;
  console.log(`✅ Shell ID: ${shellId}\n`);
  
  for (let i = 0; i < 5; i++) {
    await sleep(1100);
    console.log(`\n🔍 Check ${i + 1}:`);
    const outputResult = await client.callTool({
      name: 'bash_output',
      arguments: { bash_id: shellId }
    });
    const data = JSON.parse((outputResult.content as any)[0].text);
    if (data.output) {
        process.stdout.write(data.output);
    } else {
        console.log('   (No new output)');
    }
  }

  await client.callTool({
      name: 'kill_shell',
      arguments: { shell_id: shellId }
  });
  
  await client.close();
}

main().catch(console.error);
