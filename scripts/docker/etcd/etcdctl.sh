#!/bin/bash
# scripts/etcdctl.sh

# ================ ↓ 连接参数 ↓ ================ #
ETCD_ENDPOINTS="192.168.3.3:42379,192.168.3.3:42381,192.168.3.3:42383"
ETCD_USER="root"
ETCD_PASSWORD="<root_passwd>"

# TLS 证书路径
CERT_DIR="$HOME/Projects/synp/etc/certs/etcd"
CLIENT_CERT="$CERT_DIR/client.pem"
CLIENT_KEY="$CERT_DIR/client-key.pem"
CA_CERT="$CERT_DIR/ca.pem"
# ================ ↑ 连接参数 ↑ ================ #

# 自动检测是否启用 TLS
USE_TLS=false
if [ -f "$CLIENT_CERT" ] && [ -f "$CLIENT_KEY" ] && [ -f "$CA_CERT" ]; then
    USE_TLS=true
    echo "🔒 检测到 TLS 证书，自动启用 TLS 连接"
    ETCD_ENDPOINTS=$(echo "$ETCD_ENDPOINTS" | sed 's/\([^,]*\)/https:\/\/\1/g')
else
    echo "🔓 未检测到 TLS 证书，使用普通连接"
fi


# 构建 etcdctl 命令
if [ "$USE_TLS" = true ]; then
    ETCDCTL_CMD="etcdctl --endpoints=$ETCD_ENDPOINTS \
                        --cert=$CLIENT_CERT \
                        --key=$CLIENT_KEY \
                        --cacert=$CA_CERT \
                        --user=$ETCD_USER:$ETCD_PASSWORD"
else
    ETCDCTL_CMD="etcdctl --endpoints=$ETCD_ENDPOINTS \
                        --user=$ETCD_USER:$ETCD_PASSWORD"
fi

# 显示帮助信息
show_help() {
    echo "📚 etcd 客户端便捷脚本"
    echo ""
    echo "使用方法: $0 [命令]"
    echo ""
    echo "🔍 集群管理命令："
    echo "  health                    检查集群健康状态"
    echo "  status                    显示集群状态"
    echo "  members                   显示集群成员"
    echo "  version                   显示 etcd 版本信息"
    echo "  metrics                   显示集群指标"
    echo "  alarms                    显示告警信息"
    echo ""
    echo "📋 键值操作命令："
    echo "  list                      列出所有键"
    echo "  list-values               列出所有键值对"
    echo "  put <key> <value>         设置键值"
    echo "  get <key>                 获取键值"
    echo "  get-prefix <prefix>       获取指定前缀的所有键值"
    echo "  get-range <key1> <key2>   获取范围内的键值"
    echo "  del <key>                 删除键值"
    echo "  del-prefix <prefix>       删除指定前缀的所有键值"
    echo "  count                     统计所有键的数量"
    echo "  count-prefix <prefix>     统计指定前缀键的数量"
    echo ""
    echo "👀 监听和事务命令："
    echo "  watch <key>               监听键的变化"
    echo "  watch-prefix <prefix>     监听指定前缀键的变化"
    echo "  txn                       执行事务操作"
    echo ""
    echo "🔧 维护和管理命令："
    echo "  compact <revision>        压缩历史版本"
    echo "  defrag                    碎片整理"
    echo "  snapshot <file>           创建快照"
    echo "  restore <file>            恢复快照"
    echo "  check                     检查数据一致性"
    echo ""
    echo "📊 租约管理命令："
    echo "  lease-grant <ttl>         创建租约"
    echo "  lease-revoke <id>         撤销租约"
    echo "  lease-list                列出所有租约"
    echo "  lease-keepalive <id>      保持租约活跃"
    echo "  lease-timetolive <id>     查看租约剩余时间"
    echo ""
    echo "👥 用户和权限管理："
    echo "  user-list                 列出所有用户"
    echo "  user-add <name>           添加用户"
    echo "  user-delete <name>        删除用户"
    echo "  user-get <name>           获取用户信息"
    echo "  role-list                 列出所有角色"
    echo "  role-add <name>           添加角色"
    echo "  role-delete <name>        删除角色"
    echo "  auth-enable               启用认证"
    echo "  auth-disable              禁用认证"
    echo ""
    echo "📈 性能测试命令："
    echo "  benchmark-put             PUT 操作性能测试"
    echo "  benchmark-get             GET 操作性能测试"
    echo "  benchmark-mixed           混合操作性能测试"
    echo ""
    echo "💡 示例："
    echo "  $0 health"
    echo "  $0 put /app/config '{\"key\": \"value\"}'"
    echo "  $0 get-prefix /app/"
    echo "  $0 watch-prefix /app/"
    echo "  $0 lease-grant 60"
    echo "  $0 benchmark-put"
}

# 处理快捷命令
case "$1" in
    "help" | "-h" | "--help" | "")
        show_help
        exit 0
        ;;

    # 集群管理命令
    "health")
        echo "🔍 检查 etcd 集群健康状态..."
        $ETCDCTL_CMD endpoint health
        ;;
    "status")
        echo "📊 显示 etcd 集群状态..."
        $ETCDCTL_CMD endpoint status --write-out=table
        ;;
    "members")
        echo "👥 显示 etcd 集群成员..."
        $ETCDCTL_CMD member list --write-out=table
        ;;
    "version")
        echo "🏷️ 显示 etcd 版本信息..."
        $ETCDCTL_CMD version
        ;;
    "metrics")
        echo "📈 显示集群指标..."
        $ETCDCTL_CMD endpoint metrics
        ;;
    "alarms")
        echo "🚨 显示告警信息..."
        $ETCDCTL_CMD alarm list
        ;;

    # 键值操作命令
    "list")
        echo "📋 列出所有键..."
        $ETCDCTL_CMD get --prefix "" --keys-only
        ;;
    "list-values")
        echo "📋 列出所有键值对..."
        $ETCDCTL_CMD get --prefix ""
        ;;
    "get-prefix")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供前缀参数"
            echo "用法: $0 get-prefix <prefix>"
            exit 1
        fi
        echo "🔍 获取前缀为 '$2' 的所有键值..."
        $ETCDCTL_CMD get --prefix "$2"
        ;;
    "get-range")
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "❌ 错误: 请提供起始键和结束键"
            echo "用法: $0 get-range <start_key> <end_key>"
            exit 1
        fi
        echo "🔍 获取范围 '$2' 到 '$3' 的键值..."
        $ETCDCTL_CMD get "$2" "$3"
        ;;
    "put")
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "❌ 错误: 请提供键和值参数"
            echo "用法: $0 put <key> <value>"
            exit 1
        fi
        echo "📝 设置键值: $2 = $3"
        $ETCDCTL_CMD put "$2" "$3"
        echo "✅ 键值设置成功"
        ;;
    "get")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供键参数"
            echo "用法: $0 get <key>"
            exit 1
        fi
        echo "🔍 获取键值: $2"
        $ETCDCTL_CMD get "$2"
        ;;
    "del")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供键参数"
            echo "用法: $0 del <key>"
            exit 1
        fi
        echo "🗑️ 删除键值: $2"
        $ETCDCTL_CMD del "$2"
        echo "✅ 键值删除成功"
        ;;
    "del-prefix")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供前缀参数"
            echo "用法: $0 del-prefix <prefix>"
            exit 1
        fi
        echo "🗑️ 删除前缀为 '$2' 的所有键值..."
        echo "⚠️  警告: 这将删除所有匹配的键值，请确认 (y/N): "
        read -r confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            $ETCDCTL_CMD del --prefix "$2"
            echo "✅ 前缀键值删除成功"
        else
            echo "❌ 操作已取消"
        fi
        ;;
    "count")
        echo "🔢 统计所有键的数量..."
        count=$($ETCDCTL_CMD get --prefix "" --keys-only | wc -l)
        echo "📊 总计: $count 个键"
        ;;
    "count-prefix")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供前缀参数"
            echo "用法: $0 count-prefix <prefix>"
            exit 1
        fi
        echo "🔢 统计前缀为 '$2' 的键数量..."
        count=$($ETCDCTL_CMD get --prefix "$2" --keys-only | wc -l)
        echo "📊 前缀 '$2': $count 个键"
        ;;

    # 监听和事务命令
    "watch")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供键参数"
            echo "用法: $0 watch <key>"
            exit 1
        fi
        echo "👀 监听键值变化: $2 (按 Ctrl+C 退出)"
        $ETCDCTL_CMD watch "$2"
        ;;
    "watch-prefix")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供前缀参数"
            echo "用法: $0 watch-prefix <prefix>"
            exit 1
        fi
        echo "👀 监听前缀键值变化: $2 (按 Ctrl+C 退出)"
        $ETCDCTL_CMD watch --prefix "$2"
        ;;
    "txn")
        echo "🔄 执行事务操作..."
        echo "请输入事务内容 (输入 'help' 查看事务语法):"
        $ETCDCTL_CMD txn -i
        ;;

    # 维护和管理命令
    "compact")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供修订版本号"
            echo "用法: $0 compact <revision>"
            exit 1
        fi
        echo "🗜️ 压缩历史版本到: $2"
        $ETCDCTL_CMD compact "$2"
        echo "✅ 压缩完成"
        ;;
    "defrag")
        echo "🔧 执行碎片整理..."
        $ETCDCTL_CMD defrag
        echo "✅ 碎片整理完成"
        ;;
    "snapshot")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供快照文件名"
            echo "用法: $0 snapshot <filename>"
            exit 1
        fi
        echo "📷 创建快照: $2"
        $ETCDCTL_CMD snapshot save "$2"
        echo "✅ 快照创建成功: $2"
        echo "📋 快照信息:"
        $ETCDCTL_CMD snapshot status "$2" --write-out=table
        ;;
    "restore")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供快照文件名"
            echo "用法: $0 restore <filename>"
            exit 1
        fi
        echo "🔄 恢复快照: $2"
        $ETCDCTL_CMD snapshot restore "$2"
        echo "✅ 快照恢复成功"
        ;;
    "check")
        echo "🔍 检查数据一致性..."
        $ETCDCTL_CMD check perf
        ;;

    # 租约管理命令
    "lease-grant")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供 TTL 参数"
            echo "用法: $0 lease-grant <ttl_seconds>"
            exit 1
        fi
        echo "⏰ 创建租约，TTL: $2 秒"
        $ETCDCTL_CMD lease grant "$2"
        ;;
    "lease-revoke")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供租约 ID"
            echo "用法: $0 lease-revoke <lease_id>"
            exit 1
        fi
        echo "❌ 撤销租约: $2"
        $ETCDCTL_CMD lease revoke "$2"
        echo "✅ 租约撤销成功"
        ;;
    "lease-list")
        echo "📋 列出所有租约..."
        $ETCDCTL_CMD lease list
        ;;
    "lease-keepalive")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供租约 ID"
            echo "用法: $0 lease-keepalive <lease_id>"
            exit 1
        fi
        echo "💓 保持租约活跃: $2 (按 Ctrl+C 退出)"
        $ETCDCTL_CMD lease keep-alive "$2"
        ;;
    "lease-timetolive")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供租约 ID"
            echo "用法: $0 lease-timetolive <lease_id>"
            exit 1
        fi
        echo "⏱️ 查看租约剩余时间: $2"
        $ETCDCTL_CMD lease timetolive "$2"
        ;;

    # 用户和权限管理
    "user-list")
        echo "👤 列出所有用户..."
        $ETCDCTL_CMD user list
        ;;
    "user-add")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供用户名"
            echo "用法: $0 user-add <username>"
            exit 1
        fi
        echo "👤 添加用户: $2"
        $ETCDCTL_CMD user add "$2"
        echo "✅ 用户添加成功"
        ;;
    "user-delete")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供用户名"
            echo "用法: $0 user-delete <username>"
            exit 1
        fi
        echo "🗑️ 删除用户: $2"
        $ETCDCTL_CMD user delete "$2"
        echo "✅ 用户删除成功"
        ;;
    "user-get")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供用户名"
            echo "用法: $0 user-get <username>"
            exit 1
        fi
        echo "👤 获取用户信息: $2"
        $ETCDCTL_CMD user get "$2"
        ;;
    "role-list")
        echo "🎭 列出所有角色..."
        $ETCDCTL_CMD role list
        ;;
    "role-add")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供角色名"
            echo "用法: $0 role-add <rolename>"
            exit 1
        fi
        echo "🎭 添加角色: $2"
        $ETCDCTL_CMD role add "$2"
        echo "✅ 角色添加成功"
        ;;
    "role-delete")
        if [ -z "$2" ]; then
            echo "❌ 错误: 请提供角色名"
            echo "用法: $0 role-delete <rolename>"
            exit 1
        fi
        echo "🗑️ 删除角色: $2"
        $ETCDCTL_CMD role delete "$2"
        echo "✅ 角色删除成功"
        ;;
    "auth-enable")
        echo "🔐 启用认证..."
        $ETCDCTL_CMD auth enable
        echo "✅ 认证已启用"
        ;;
    "auth-disable")
        echo "🔓 禁用认证..."
        $ETCDCTL_CMD auth disable
        echo "✅ 认证已禁用"
        ;;

    # 性能测试命令
    "benchmark-put")
        echo "🚀 执行 PUT 操作性能测试..."
        $ETCDCTL_CMD benchmark put --total=10000 --val-size=256
        ;;
    "benchmark-get")
        echo "🚀 执行 GET 操作性能测试..."
        $ETCDCTL_CMD benchmark get --total=10000
        ;;
    "benchmark-mixed")
        echo "🚀 执行混合操作性能测试..."
        $ETCDCTL_CMD benchmark put --total=10000 --val-size=256 &
        $ETCDCTL_CMD benchmark get --total=10000 &
        wait
        echo "✅ 混合性能测试完成"
        ;;

    *)
        # 直接执行 etcdctl 命令
        echo "🔧 执行 etcdctl 命令: $*"
        $ETCDCTL_CMD "$@"
        ;;
esac
