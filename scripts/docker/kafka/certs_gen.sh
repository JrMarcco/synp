#!/bin/bash
set -e

echo "=== 生成 Kafka 3 节点集群 RSA TLS 证书 ( 仅 DNS 配置 )==="

VALIDITY_DAYS=365

# 创建证书目录
CERT_DIR="./certs"
mkdir -p $CERT_DIR
cd $CERT_DIR

# 清理旧证书
rm -f *.key *.pem *.csr *.srl *.conf *.crt 2>/dev/null

echo "1. 生成 CA 私钥 (RSA 4096)"
openssl genrsa -out ca.key 4096

echo "2. 生成 CA 证书"
openssl req -new -x509 -key ca.key -days "$VALIDITY_DAYS" -out ca.crt \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=Synp/OU=CA/CN=kafka-ca"

echo "3. 生成 kafka broker 私钥 (RSA 4096)"
openssl genrsa -out kafka.key 4096

echo "4. 创建服务器证书配置文件（仅 DNS 名称）"
cat > server.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = CN
ST = Beijing
L = Beijing
O = Synp
OU = synp-kafka-server
CN = synp-kafka-server

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
# ===== 本地回环（保留 localhost 用于测试）=====
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1

# ===== Docker 容器名称，集群情况下配置多个 =====
DNS.2 = kafka-1
DNS.3 = kafka-2
DNS.4 = kafka-3

# ===== 如果需要外部访问，添加主机名 或 IP =====
# DNS.5 = kafka.yourdomain.com
IP.3 = 192.168.3.3
EOF

echo "5. 生成 kafka 服务器证书请求"
openssl req -new -key kafka.key -out kafka.csr -config server.conf

echo "6. 签发 kafka 服务器证书"
openssl x509 -req -in kafka.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out kafka.crt -days "$VALIDITY_DAYS" \
    -extensions v3_req -extfile server.conf


echo "7. 创建 PEM 格式文件 (Bitnami Kafka 要求)"

# kafka.keystore.pem (私钥 + 证书 + CA证书，完整证书链)
cat kafka.key kafka.crt ca.crt > kafka.keystore.pem

# kafka.keystore.key (私钥单独文件)
cp kafka.key kafka.keystore.key

# kafka.truststore.pem (CA 证书)
cp ca.crt kafka.truststore.pem

echo "✅ PEM 文件生成完成 (Bitnami Kafka 格式)"
echo "   - kafka.keystore.pem (私钥 + 证书)"
echo "   - kafka.keystore.key (私钥)"
echo "   - kafka.truststore.pem (CA 证书)"
echo ""


echo "8. 验证证书"
openssl verify -CAfile ca.crt kafka.crt

if [ $? -eq 0 ]; then
    echo "✅ 证书验证成功！"
else
    echo "❌ 证书验证失败！"
    exit 1
fi
echo ""
echo "📋 证书信息:"
echo ""
echo "--- CA 证书 ---"
openssl x509 -in ca.crt -noout -subject -issuer -dates
echo ""
echo "--- Kafka 证书 ---"
openssl x509 -in kafka.crt -noout -subject -issuer -dates
echo ""
echo "--- SAN (Subject Alternative Names) ---"
openssl x509 -in kafka.crt -noout -text | grep -A1 "Subject Alternative Name"
echo ""

echo "9. 设置证书文件权限"
chmod 600 *.key
chmod 644 *.crt
chmod 644 *.pem

echo "🔒 文件权限已设置"
echo ""

echo "10. 清理临时文件"
rm -f kafka.csr server.conf ca.srl

echo "📦 生成的文件列表:"
ls -lh *.crt *.key *.pem 2>/dev/null || true
echo ""
