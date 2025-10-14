#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Etcd 集群初始化脚本"
echo ""

# 1. 生成 .env 文件
echo "📝 生成 .env 配置文件..."
cat > .env << 'EOF'
ETCD_NAME=synp
ETCD_ROOT_PASSWORD=<root_passwd>
EOF
echo "✅ .env 文件已生成"
echo ""

# 2. 创建数据目录
NUM_DIRS=${1:-3}
echo "📁 创建 $NUM_DIRS 个数据目录..."
for i in $(seq 1 $NUM_DIRS); do
    DATA_DIR="./data-$i"
    if [ -d "$DATA_DIR" ]; then
        echo "  📁 数据目录 $DATA_DIR 已存在"
    else
        echo "  📁 创建数据目录 $DATA_DIR"
        mkdir -p "$DATA_DIR"
    fi
done
echo "✅ 数据目录创建完成"
echo ""

# 3. 生成证书
if [ ! -f "./certs/server.pem" ] || [ ! -f "./certs/ca.pem" ]; then
    echo "🔐 生成 TLS 证书..."
    chmod +x ./certs_gen.sh
    sudo ./certs_gen.sh
    echo "✅ 证书生成完成"
    echo ""
else
    echo "ℹ️  证书已存在，跳过生成"
    echo ""
fi

# 4. 设置文件权限（Bitnami Etcd 需要 UID 1001）
echo "🔒 设置文件权限..."
sudo chown 1001:1001 -R ./data-* 2>/dev/null || true
sudo chown 1001:1001 -R ./certs 2>/dev/null || true
echo "✅ 权限设置完成"
echo ""

echo "🎉 初始化完成！"
