// ============================================================================
// 共享配置与工具函数 (Linux 版 mcp-client)
// 统一管理 MCP 服务器二进制路径，避免各测试脚本重复写死路径导致平台分叉
// ============================================================================
import path from 'path';
import { spawnSync } from 'child_process';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// 编译后位于 mcp-client/dist，向上两级为项目根目录 dist/ 下的 Linux 二进制
export const SERVER_PATH = path.resolve(
  __dirname,
  '../../dist/mcp-file-bash-tools-linux-amd64',
);

export async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// 统一解析工具返回: 优先 structuredContent, 回退到 content[0].text 的 JSON
export function parseResult<T = any>(result: any): T {
  if (result.structuredContent) return result.structuredContent as T;
  return JSON.parse((result.content as any)[0].text) as T;
}

// 检查 pid 进程是否存活 (kill -0 成功 = 存活)
export function isProcessAlive(pid: number): boolean {
  const res = spawnSync('bash', ['-c', `kill -0 ${pid} 2>/dev/null`]);
  return res.status === 0;
}

// 统计 Linux ping 输出中的回复数 (如 "64 bytes from 127.0.0.1: icmp_seq=1 ...")
export function countPingReplies(output: string): number {
  const matches = output.match(/bytes from/gi);
  return matches ? matches.length : 0;
}

// 检查进程是否存在于进程表中 (ps aux + 单词边界匹配)
// 使用 \b 防止误匹配如 "smokeping" (含 ping 子串但与 ping 无关)
export function findProcess(name: string): string[] {
  const res = spawnSync('ps', ['aux'], { encoding: 'utf8' });
  const re = new RegExp(`\\b${name}\\b`, 'i');
  return res.stdout.split('\n').filter((line) => re.test(line));
}
