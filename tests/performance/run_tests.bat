@echo off

:: 创建日志目录
mkdir logs 2>nul

:: 运行所有测试
echo Running cache performance tests...
go test -v -run TestCachePerformance > logs\cache_test.log 2>&1

echo Running compression performance tests...
go test -v -run TestCompressionPerformance > logs\compression_test.log 2>&1

echo Running service discovery performance tests...
go test -v -run TestServiceDiscoveryPerformance > logs\discovery_test.log 2>&1

echo Running concurrent performance tests...
go test -v -run TestConcurrentPerformance > logs\concurrent_test.log 2>&1

:: 生成测试报告
echo Generating test report...
go test -v -json > logs\test_report.json

echo All tests completed. Check logs directory for results.
pause 