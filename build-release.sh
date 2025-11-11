#!/bin/bash

# 生产版本构建脚本（仅当前架构，但包含完整的发布流程）
set -euo pipefail

PROJECT_NAME="mcp-file-bash-tools"
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
DIST_DIR="dist"

echo "🚀 构建生产版本 ${PROJECT_NAME} v${VERSION}"
echo "📅 构建时间: ${BUILD_TIME}"
echo "🔗 Git 提交: ${GIT_COMMIT}"
echo ""

# 清理构建目录
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

# 设置环境变量
export CGO_ENABLED=1
export CC=musl-gcc

# 构建参数
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"
BUILD_FLAGS="-s -w"

# 获取当前架构
ARCH=$(go env GOOS)-$(go env GOARCH)
BINARY_NAME="${PROJECT_NAME}-${ARCH}"
BINARY_PATH="${DIST_DIR}/${BINARY_NAME}"

echo "📦 构建目标: ${ARCH}"
echo "🔧 开始编译..."

go build \
    -tags "netgo osusergo static_build" \
    -ldflags "${LDFLAGS} ${BUILD_FLAGS} -extldflags '-static'" \
    -o "${BINARY_PATH}" \
    .

echo "✅ 构建完成!"
echo ""

# 验证二进制文件
echo "🧪 验证二进制文件..."

# 检查静态链接 (跨平台兼容)
OS_NAME=$(uname -s)
if [[ "$OS_NAME" == "Linux" ]]; then
    # Linux 使用 ldd 检查
    if ldd "${BINARY_PATH}" 2>&1 | grep -q "not a dynamic executable"; then
        echo "🔒 静态链接: ✅"
    else
        echo "⚠️ 静态链接: 可能不完全"
    fi
elif [[ "$OS_NAME" == "Darwin" ]]; then
    # macOS 使用 otool 检查
    if otool -L "${BINARY_PATH}" 2>&1 | grep -q "no dynamic libraries"; then
        echo "🔒 静态链接: ✅"
    else
        echo "⚠️ 静态链接: 可能不完全"
    fi
else
    echo "⚠️ 静态链接: 未知平台，无法验证"
fi

# 文件信息
if command -v file >/dev/null 2>&1; then
    echo "📊 文件信息: $(file ${BINARY_PATH})"
else
    echo "📊 文件信息: file command not available"
fi
echo "📏 文件大小: $(ls -lh ${BINARY_PATH} | cut -d' ' -f5)"

# 功能测试
echo "🔧 功能测试..."
if "${BINARY_PATH}" --version > /dev/null 2>&1; then
    echo "✅ 版本信息正常"
else
    echo "❌ 版本信息异常"
fi

if "${BINARY_PATH}" --help > /dev/null 2>&1; then
    echo "✅ 帮助信息正常"
else
    echo "❌ 帮助信息异常"
fi

echo ""

# 创建发布包
echo "📦 创建发布包..."

cd "${DIST_DIR}"

# 1. 创建压缩包
tar -czf "${BINARY_NAME}.tar.gz" "${BINARY_NAME}"
echo "✅ 压缩包: ${BINARY_NAME}.tar.gz"

# 2. 生成校验和
sha256sum "${BINARY_NAME}" > "${BINARY_NAME}.sha256"
sha256sum "${BINARY_NAME}.tar.gz" > "${BINARY_NAME}.tar.gz.sha256"
echo "✅ 校验和: ${BINARY_NAME}.sha256"

# 3. 生成构建信息
STATIC_LINKED="否"
if [[ "$OS_NAME" == "Linux" ]]; then
    if ldd "${BINARY_PATH}" 2>&1 | grep -q "not a dynamic executable"; then
        STATIC_LINKED="是"
    fi
elif [[ "$OS_NAME" == "Darwin" ]]; then
    if otool -L "${BINARY_PATH}" 2>&1 | grep -q "no dynamic libraries"; then
        STATIC_LINKED="是"
    fi
fi

cat > "build-info.txt" << EOF
项目名称: ${PROJECT_NAME}
版本: ${VERSION}
构建时间: ${BUILD_TIME}
Git 提交: ${GIT_COMMIT}
目标架构: ${ARCH}
Go 版本: $(go version)
构建主机: $(hostname)
操作系统: $(uname -a)
二进制大小: $(ls -lh "${BINARY_NAME}" | cut -d' ' -f5)
静态链接: ${STATIC_LINKED}
EOF
echo "✅ 构建信息: build-info.txt"

cd ..

# 显示结果
echo ""
echo "🎉 构建完成！"
echo ""
echo "📁 构建产物:"
ls -la "${DIST_DIR}/"
echo ""
echo "📋 文件清单:"
cd "${DIST_DIR}"
for file in *; do
    if [[ -f "$file" ]]; then
        size=$(ls -lh "$file" | cut -d' ' -f5)
        echo "  📄 $file ($size)"
    fi
done
cd ..

echo ""
echo "✅ 生产版本构建完成，文件已保存到 ${DIST_DIR}/ 目录"