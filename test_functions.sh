#!/bin/bash

echo "========== 测试1: ANSI转义序列清理 =========="
echo "测试命令: Get-ChildItem -Path ."
echo ""

# 模拟通过MCP调用bash工具
# 这里我们直接测试PowerShell命令的输出
pwsh.exe -NoProfile -Command "Get-ChildItem -Path ." 2>&1 | head -10

echo ""
echo "========== 测试2: EstimatedTime功能 =========="
echo "测试一个长时间运行的命令（预计时间5秒，实际执行10秒）"
echo ""

# 测试长时间运行命令
timeout 15s pwsh.exe -NoProfile -Command "Start-Sleep 10; Write-Output '完成'" 2>&1

echo ""
echo "========== 测试完成 =========="
