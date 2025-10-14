#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Kafka 集群初始化脚本"
echo ""

# 1. 生成 .env 文件
echo "📝 生成 .env 配置文件..."
cat > .env << 'EOF'
KAFKA_CLUSTER_ID=synp-kafka-cluster

KAFKA_ADMIN_USER=admin
KAFKA_ADMIN_PASSWORD=admin-secret

KAFKA_CLIENT_USER=synp-kafka-user
KAFKA_CLIENT_PASSWORD=synp-kafka-secret

KAFKA_SUPER_USERS=User:admin;User:synp-kafka-user
KAFKA_CLIENT_USERS=admin,synp-kafka-user
KAFKA_CLIENT_PASSWORDS=admin-secret,synp-kafka-secret
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
if [ ! -f "./certs/kafka.keystore.pem" ] || [ ! -f "./certs/ca.crt" ]; then
    echo "🔐 生成 TLS 证书..."
    chmod +x ./certs_gen.sh
    sudo ./certs_gen.sh
    echo "✅ 证书生成完成"
    echo ""
else
    echo "ℹ️  证书已存在，跳过生成"
    echo ""
fi

# 4. 设置文件权限（Bitnami Kafka 需要 UID 1001）
echo "🔒 设置文件权限..."
sudo chown 1001:1001 -R ./data-* 2>/dev/null || true
sudo chown 1001:1001 -R ./certs 2>/dev/null || true
echo "✅ 权限设置完成"
echo ""

echo "🎉 初始化完成！"
