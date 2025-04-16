@echo off
setlocal enabledelayedexpansion

:: GeeCache 分布式缓存系统 Docker 运行脚本 (Windows版)
:: 该脚本用于启动etcd容器和缓存节点进行测试

echo === GeeCache 分布式缓存系统启动脚本 ===

:: 检查Docker是否安装
docker --version > nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo 错误: Docker未安装，请先安装Docker
    exit /b 1
)

:: 清理函数
:cleanup
if "%1"=="cleanup" (
    echo 正在清理资源...
    
    :: 停止并删除etcd容器
    docker ps -q -f name=geecache-etcd > nul 2>&1
    if !ERRORLEVEL! equ 0 (
        docker stop geecache-etcd > nul 2>&1
        docker rm geecache-etcd > nul 2>&1
    )
    
    :: 终止所有后台进程
    taskkill /F /IM server.exe > nul 2>&1
    
    :: 删除编译生成的可执行文件
    if exist server.exe del server.exe
    
    echo 清理完成
    exit /b 0
)

:: 设置退出时的清理
:: 注册Ctrl+C处理
if not "%1"=="cleanup" (
    echo 按Ctrl+C可以退出并清理资源
)

:: 检查是否已有同名容器运行，如有则停止并删除
docker ps -a -q -f name=geecache-etcd > nul 2>&1
if %ERRORLEVEL% equ 0 (
    echo 发现已有geecache-etcd容器，正在停止并删除...
    docker stop geecache-etcd > nul 2>&1
    docker rm geecache-etcd > nul 2>&1
)

:: 启动etcd容器
echo 正在启动etcd容器...
docker run -d --name geecache-etcd ^^
    -p 2379:2379 ^^
    -p 2380:2380 ^^
    --env ALLOW_NONE_AUTHENTICATION=yes ^^
    --env ETCD_ADVERTISE_CLIENT_URLS=http://0.0.0.0:2379 ^^
    --env ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 ^^
    bitnami/etcd:latest

if %ERRORLEVEL% neq 0 (
    echo 错误: 启动etcd容器失败
    call :cleanup cleanup
    exit /b 1
)

:: 等待etcd服务就绪
echo 等待etcd服务就绪...
timeout /t 5 /nobreak > nul

:: 验证etcd服务是否正常运行
echo 验证etcd服务...
docker exec geecache-etcd etcdctl put test value
if %ERRORLEVEL% neq 0 (
    echo 错误: etcd服务未就绪
    call :cleanup cleanup
    exit /b 1
)
docker exec geecache-etcd etcdctl get test
echo etcd服务已就绪

:: 编译服务器程序
echo 正在编译服务器程序...
go build -o server.exe
if %ERRORLEVEL% neq 0 (
    echo 错误: 编译失败
    call :cleanup cleanup
    exit /b 1
)

:: 启动缓存节点
echo 正在启动缓存节点...

:: 启动节点1（端口8001）
start /B cmd /c "server.exe -port=8001 -etcd=true -etcd-endpoints=localhost:2379"
echo 节点1已启动

:: 启动节点2（端口8002）
start /B cmd /c "server.exe -port=8002 -etcd=true -etcd-endpoints=localhost:2379"
echo 节点2已启动

:: 启动API节点（端口8003，API端口9999）
start /B cmd /c "server.exe -port=8003 -api=true -etcd=true -etcd-endpoints=localhost:2379"
echo API节点已启动

:: 等待服务启动
echo 等待服务启动...
timeout /t 3 /nobreak > nul

:: 运行测试请求
echo 正在发送测试请求...
curl "http://localhost:9999/api?key=Tom"
echo.
curl "http://localhost:9999/api?key=Jack"
echo.
curl "http://localhost:9999/api?key=Sam"
echo.

:: 显示服务状态
echo.
echo === 服务状态 ===
echo - etcd 运行在 localhost:2379
echo - 缓存节点1 运行在 localhost:8001
echo - 缓存节点2 运行在 localhost:8002
echo - API节点 运行在 localhost:8003，API端口 9999

echo.
echo 按任意键退出并清理资源
pause > nul

:: 清理资源
call :cleanup cleanup