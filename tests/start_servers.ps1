# GeeCache 测试启动脚本 (PowerShell版本)

Write-Host "启动 GeeCache 测试环境..." -ForegroundColor Green

# 确保在根目录执行
Set-Location $PSScriptRoot/..

# 注意：Windows上需要先启动etcd
Write-Host "注意：请确保etcd已经在本地运行，端口为2379" -ForegroundColor Yellow
Write-Host "如果尚未运行etcd，请在另一个命令行窗口中启动它" -ForegroundColor Yellow
Write-Host "按任意键继续..." -ForegroundColor Yellow
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

# 启动第一个缓存节点 (端口 8001)
Write-Host "正在启动缓存节点 1 (端口 8001)..." -ForegroundColor Cyan
Start-Process -NoNewWindow powershell -ArgumentList "go run main.go -port=8001 -etcd-endpoints=localhost:2379"

# 启动第二个缓存节点 (端口 8002)
Write-Host "正在启动缓存节点 2 (端口 8002)..." -ForegroundColor Cyan
Start-Process -NoNewWindow powershell -ArgumentList "go run main.go -port=8002 -etcd-endpoints=localhost:2379"

# 启动 API 节点 (端口 9999)
Write-Host "正在启动 API 节点 (端口 9999)..." -ForegroundColor Cyan
Start-Process -NoNewWindow powershell -ArgumentList "go run main.go -port=8003 -api=true -etcd-endpoints=localhost:2379"

# 等待所有服务启动
Write-Host "等待所有服务启动完成..." -ForegroundColor Cyan
Start-Sleep -Seconds 3

Write-Host "GeeCache 测试环境已启动!" -ForegroundColor Green
Write-Host "- etcd 运行在 localhost:2379" -ForegroundColor Green
Write-Host "- 缓存节点 1 运行在 http://localhost:8001" -ForegroundColor Green
Write-Host "- 缓存节点 2 运行在 http://localhost:8002" -ForegroundColor Green
Write-Host "- API 节点运行在 http://localhost:9999" -ForegroundColor Green
Write-Host ""
Write-Host "现在可以运行性能测试了：" -ForegroundColor Green
Write-Host "   cd tests/perf" -ForegroundColor Yellow
Write-Host "   go run main.go" -ForegroundColor Yellow
Write-Host ""
Write-Host "测试完成后，你需要手动关闭所有启动的进程" -ForegroundColor Red 