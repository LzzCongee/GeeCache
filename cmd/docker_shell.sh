
# 清理函数，用于脚本退出时清理资源
cleanup() {
    echo "正在清理资源..."
    # 停止并删除etcd容器
    if [ ! -z "$(docker ps -q -f name=geecache-etcd)" ]; then
        docker stop geecache-etcd
        docker rm geecache-etcd
    fi
    # 终止所有后台进程
    if [ ! -z "$SERVER_PIDS" ]; then
        kill $SERVER_PIDS 2>/dev/null
    fi
    # 删除编译生成的可执行文件
    if [ -f "server" ]; then
        rm server
    fi
    echo "清理完成"
}

# 设置退出时的清理
trap cleanup EXIT INT TERM


# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    handle_error "Docker未安装，请先安装Docker"
fi

# 检查是否已有同名容器运行，如有则停止并删除
if [ ! -z "$(docker ps -a -q -f name=geecache-etcd)" ]; then
    echo "发现已有geecache-etcd容器，正在停止并删除..."
    docker stop geecache-etcd
    docker rm geecache-etcd
fi


# 启动etcd容器
echo "正在启动etcd容器..."
docker run -d --name geecache-etcd   -p 2379:2379   -p 2380:2380   --env ALLOW_NONE_AUTHENTICATION=yes   --env ETCD_ADVERTISE_CLIENT_URLS=http://0.0.0.0:2379     --env ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379  bitnami/etcd:latest 

# 等待etcd服务就绪
echo "等待etcd服务就绪..."
sleep 5

# 验证etcd服务是否正常运行
echo "验证etcd服务..."
docker exec geecache-etcd etcdctl put test value || handle_error "etcd服务未就绪"
docker exec geecache-etcd etcdctl get test
echo "etcd服务已就绪"