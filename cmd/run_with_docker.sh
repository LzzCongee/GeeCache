#!/bin/bash
# GeeCache 分布式缓存系统 Docker 运行脚本
# 该脚本用于启动etcd容器和缓存节点进行测试

# 错误处理函数
handle_error() {
    echo "错误: $1"
    exit 1
}


echo "=== GeeCache 分布式缓存系统启动脚本 ==="

# go run main.go -etcd=true -etcd-endpoints=localhost:2379  -port=8001
# go run main.go -etcd=true -etcd-endpoints=localhost:2379  -port=8002
# go run main.go -etcd=true -etcd-endpoints=localhost:2379  -port=8003 -api=true


# 启动缓存节点
echo "正在启动缓存节点..."
SERVER_PIDS=""

# 启动节点1（端口8001）
./server -port=8001 -etcd=true -etcd-endpoints=localhost:2379 &
SERVER_PIDS="$SERVER_PIDS $!"
echo "节点1已启动，PID: $!"

# 启动节点2（端口8002）
./server -port=8002 -etcd=true -etcd-endpoints=localhost:2379 &
SERVER_PIDS="$SERVER_PIDS $!"
echo "节点2已启动，PID: $!"

# 启动API节点（端口8003，API端口9999）
./server -port=8003 -api=true -etcd=true -etcd-endpoints=localhost:2379 &
SERVER_PIDS="$SERVER_PIDS $!"
echo "API节点已启动，PID: $!"

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 运行测试请求
echo "正在发送测试请求..."
curl "http://localhost:9999/api?key=Tom"
echo ""
curl "http://localhost:9999/api?key=Jack"
echo ""
curl "http://localhost:9999/api?key=Sam"
echo ""

# 显示服务状态
echo "\n=== 服务状态 ==="
echo "- etcd 运行在 localhost:2379"
echo "- 缓存节点1 运行在 localhost:8001"
echo "- 缓存节点2 运行在 localhost:8002"
echo "- API节点 运行在 localhost:8003，API端口 9999"

echo "\n按Ctrl+C退出并清理资源"

# 保持脚本运行，直到用户按Ctrl+C
wait