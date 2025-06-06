# GeeCache 项目更新说明

## 目录结构调整

1. 将 `tests` 目录移到 `geecache` 目录下，便于统一管理测试用例
2. 在 `geecache/cmd` 目录下创建专用的测试主程序
3. 更新 `go.mod` 以解决导入路径问题

## 功能增强

1. 添加指标统计功能：
   - 在 `Group` 中增加统计计数器
   - 添加 `GetStats()` 方法获取统计信息
   - 为 `lru.Cache` 添加 `Size()` 方法返回缓存大小

2. 增强测试工具：
   - 创建 `run_test.ps1` PowerShell 脚本，便于启动分布式测试环境
   - 支持日志文件记录，便于观察测试结果

## 使用指南

1. 启动测试环境：
   ```powershell
   # 在项目根目录下执行
   .\geecache\tests\run_test.ps1
   ```

2. 测试接口：
   - 查询缓存: http://localhost:9999/api?key=Tom
   - 查看统计: http://localhost:9999/stats

3. 查看日志：
   - 所有日志文件保存在 `logs` 目录下

## 架构说明

GeeCache 是一个分布式缓存系统，主要特点：

1. **分布式节点**：使用一致性哈希算法分布数据
2. **服务发现**：支持通过 etcd 实现自动服务发现
3. **多种压缩**：支持多种压缩算法优化传输
4. **统计指标**：记录缓存命中率、大小等关键指标

此更新对项目进行了整体优化，使之更易于测试和维护，并添加了性能监控能力。 