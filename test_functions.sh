#!/bin/bash

echo "========== 测试1: ANSI转义序列清理 =========="
echo "测试命令: ls -la"
echo ""

# 模拟通过MCP调用bash工具
# 这里我们直接测试Linux命令的输出
ls -la 2>&1 | head -10

echo ""
echo "========== 测试2: 长时间运行命令功能 =========="
echo "测试一个长时间运行的命令（预计时间5秒，实际执行10秒）"
echo ""

# 测试长时间运行命令
timeout 15s bash -c "sleep 10; echo '完成'" 2>&1

echo ""
echo "========== 测试完成 =========="
