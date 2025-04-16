#!/bin/bash
# GeeCache 测试启动脚本

echo "启动 GeeCache 测试环境..."

# 确保在根目录执行
cd $(dirname $0)/..

# 启动 etcd (假设已安装)
echo "正在启动 etcd 服务..."
etcd &
ETCD_PID=$!
sleep 2

# 启动第一个缓存节点 (端口 8001)
echo "正在启动缓存节点 1 (端口 8001)..."
go run main.go -port=8001 -etcd-endpoints=localhost:2379 &
NODE1_PID=$!

# 启动第二个缓存节点 (端口 8002)
echo "正在启动缓存节点 2 (端口 8002)..."
go run main.go -port=8002 -etcd-endpoints=localhost:2379 &
NODE2_PID=$!

# 启动 API 节点 (端口 9999)
echo "正在启动 API 节点 (端口 9999)..."
go run main.go -port=8003 -api=true -etcd-endpoints=localhost:2379 &
API_PID=$!

# 等待所有服务启动
echo "等待所有服务启动完成..."
sleep 3

echo "GeeCache 测试环境已启动!"
echo "- etcd 运行在 localhost:2379"
echo "- 缓存节点 1 运行在 http://localhost:8001"
echo "- 缓存节点 2 运行在 http://localhost:8002"
echo "- API 节点运行在 http://localhost:9999"
echo
echo "按 Ctrl+C 停止所有服务"

# 捕获终止信号
function cleanup {
    echo "正在停止所有服务..."
    kill $NODE1_PID $NODE2_PID $API_PID $ETCD_PID
    echo "已停止所有服务"
    exit 0
}

trap cleanup INT TERM

# 保持脚本运行
wait 