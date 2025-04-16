# GeeCache 测试启动脚本 (PowerShell)

Write-Host "启动 GeeCache 分布式缓存测试..." -ForegroundColor Green

# 输出目录信息
Write-Host "当前工作目录: $(Get-Location)" -ForegroundColor Cyan

# 创建日志目录
$logDir = "logs"
if (-not (Test-Path -Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir | Out-Null
    Write-Host "创建日志目录: $logDir" -ForegroundColor Yellow
}

# 设置节点配置
$node1Port = 8001
$node2Port = 8002
$apiPort = 8003

# 确保所有端口未被占用
function Test-PortInUse {
    param($port)
    $connections = Get-NetTCPConnection -ErrorAction SilentlyContinue | Where-Object { $_.LocalPort -eq $port }
    if ($connections) {
        Write-Host "警告: 端口 $port 已被占用，可能会导致服务启动失败" -ForegroundColor Red
        return $true
    }
    return $false
}

Test-PortInUse -port $node1Port
Test-PortInUse -port $node2Port
Test-PortInUse -port $apiPort
Test-PortInUse -port 9999

# 注意：Windows上需要先确保etcd已运行
Write-Host "注意：请确保etcd已经在本地运行，端口为2379" -ForegroundColor Yellow
Write-Host "如果尚未运行etcd，请在另一个命令行窗口中启动它" -ForegroundColor Yellow
Write-Host "按任意键继续..." -ForegroundColor Yellow
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

# 启动缓存节点1
Write-Host "正在启动缓存节点1 (端口 $node1Port)..." -ForegroundColor Cyan
Start-Process -NoNewWindow powershell -ArgumentList "cd $(Get-Location); go run geecache/cmd/main.go -port=$node1Port -etcd=true -etcd-endpoints=localhost:2379 > logs/node1.log 2>&1"

# 启动缓存节点2
Write-Host "正在启动缓存节点2 (端口 $node2Port)..." -ForegroundColor Cyan
Start-Process -NoNewWindow powershell -ArgumentList "cd $(Get-Location); go run geecache/cmd/main.go -port=$node2Port -etcd=true -etcd-endpoints=localhost:2379 > logs/node2.log 2>&1"

# 启动API节点
Write-Host "正在启动API节点 (端口 $apiPort)..." -ForegroundColor Cyan
Start-Process -NoNewWindow powershell -ArgumentList "cd $(Get-Location); go run geecache/cmd/main.go -port=$apiPort -api=true -etcd=true -test=true -etcd-endpoints=localhost:2379 > logs/api.log 2>&1"

# 等待服务启动
Write-Host "等待所有服务启动完成..." -ForegroundColor Cyan
Start-Sleep -Seconds 3

Write-Host "GeeCache 测试环境已启动!" -ForegroundColor Green
Write-Host "- etcd 运行在 localhost:2379" -ForegroundColor Green
Write-Host "- 缓存节点1 运行在 http://localhost:$node1Port" -ForegroundColor Green
Write-Host "- 缓存节点2 运行在 http://localhost:$node2Port" -ForegroundColor Green
Write-Host "- API节点运行在 http://localhost:$apiPort (Web API: http://localhost:9999)" -ForegroundColor Green
Write-Host ""
Write-Host "日志文件保存在 ./logs/ 目录下" -ForegroundColor Yellow
Write-Host ""
Write-Host "你可以使用浏览器或 curl 访问以下接口测试缓存功能:" -ForegroundColor Cyan
Write-Host "- 查询缓存: http://localhost:9999/api?key=Tom" -ForegroundColor Yellow
Write-Host "- 查看统计: http://localhost:9999/stats" -ForegroundColor Yellow
Write-Host ""
Write-Host "测试完成后，请使用任务管理器手动关闭所有 go run 进程" -ForegroundColor Red 