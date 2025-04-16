# GeeCache 分布式缓存系统

GeeCache是一个高性能的分布式缓存系统，支持一致性哈希、服务发现、数据压缩等特性。

## 项目架构

### 核心组件

1. **缓存核心 (geecache)**
   - Group: 缓存命名空间
   - lru: LRU缓存淘汰算法实现
   - cache: 并发安全的缓存
   - singleflight: 防止缓存击穿的并发控制组件
   - consistenthash: 一致性哈希实现，确保分布式环境下的负载均衡

2. **通信和协议 (geecache/geecachepb)**
   - Protocol Buffers定义的缓存数据通信协议
   - HTTP: 基于HTTP的节点间通信

3. **服务发现 (geecache/registry)**
   - 基于etcd的服务注册与发现机制
   - 支持自动注册服务和发现其他节点

4. **压缩模块 (geecache/compression)**
   - 支持多种压缩算法：Gzip、Snappy、LZ4、Zstd
   - 提供统一的压缩接口，可根据性能/压缩率需求选择算法

5. **监控指标 (geecache/metrics)**
   - 支持Prometheus接口
   - 收集请求延迟、缓存命中率等关键指标

6. **存储接口 (geecache/storage)**
   - 抽象存储层，可扩展不同的后端存储

### 架构图

```
                   ┌─────────┐      ┌─────────┐
                   │ Client  │      │ Client  │
                   └─────────┘      └─────────┘
                        │                │
                        ▼                ▼
┌──────────────────────────────────────────────────────┐
│  HTTP                                                │
├──────────────────────────────────────────────────────┤
│                                                      │
│  ┌─────────┐    ┌─────────────┐    ┌──────────────┐  │
│  │  Group  │───▶│SingleFlight │───▶│   Remote     │  │
│  └─────────┘    └─────────────┘    │   Nodes      │  │
│       │                            └──────────────┘  │
│       │                                   ▲          │
│       ▼                                   │          │
│  ┌─────────┐                     ┌──────────────┐    │
│  │  Cache  │                     │ Consistent   │    │
│  └─────────┘                     │     Hash     │    │
│       │                          └──────────────┘    │
│       │                                              │
│       ▼                                              │
│  ┌─────────┐                                         │
│  │   LRU   │                                         │
│  └─────────┘                                         │
│                        GeeCache                      │
└──────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
    ┌─────────────┐                ┌────────────┐
    │    etcd     │                │ Prometheus │
    │ (服务发现)   │                │  (监控)    │
    └─────────────┘                └────────────┘
```

## 项目特点

- **一致性哈希**：使用一致性哈希算法进行缓存数据的分布，确保数据均匀分布并减少节点变化时的缓存迁移
- **防止缓存击穿**：使用singleflight确保对同一个key的多次请求只会触发一次底层查询
- **服务发现**：集成etcd实现服务的自动注册与发现，支持动态伸缩
- **数据压缩**：支持多种压缩算法，优化网络传输和存储效率
- **监控指标**：集成Prometheus监控系统，实时监控缓存性能
- **可扩展性**：良好的接口设计，可轻松扩展不同的后端存储和功能

## 如何运行

### 环境需求
- Go 1.13+
- etcd (服务发现需要)

### 运行服务

1. **启动etcd**
   ```bash
   etcd
   ```

2. **启动缓存节点**
   ```bash
   # 启动节点1（端口8001）
   go run main.go -port=8001 -etcd-endpoints=localhost:2379
   
   # 启动节点2（端口8002）
   go run main.go -port=8002 -etcd-endpoints=localhost:2379
   
   # 启动API节点（端口9999）
   go run main.go -port=8003 -api=true -etcd-endpoints=localhost:2379
   ```

3. **使用自动化脚本启动**
   ```bash
   # Linux/MacOS
   chmod +x tests/start_servers.sh
   ./tests/start_servers.sh
   
   # Windows
   ./tests/start_servers.ps1
   ```

### 运行测试

1. **运行性能测试**
   ```bash
   cd tests/perf
   go run main.go
   ```

2. **测试API**
   ```bash
   # 查询缓存
   curl "http://localhost:9999/api?key=Tom"
   ```

## 性能测试

通过`tests/perf`目录下的测试工具，可以测试缓存系统的性能指标：

1. 压缩算法性能对比
2. 本地缓存性能（吞吐量、响应时间）
3. 分布式缓存性能（吞吐量、成功率）

测试结果会保存在`tests/perf/performance_test.log`文件中。

## 项目结构

```
geecache/
├── byteview.go         # 缓存值的不可变视图
├── cache.go            # 并发安全的缓存
├── compression/        # 压缩模块
│   ├── compression.go  # 压缩接口
│   ├── gzip.go         # Gzip压缩实现
│   ├── lz4.go          # LZ4压缩实现 
│   ├── none.go         # 无压缩实现
│   ├── snappy.go       # Snappy压缩实现
│   └── zstd.go         # Zstd压缩实现
├── consistenthash/     # 一致性哈希
├── geecache.go         # 核心缓存逻辑
├── geecachepb/         # 协议定义
├── go.mod              # 模块定义
├── http.go             # HTTP通信
├── http_pool_discovery.go # 服务发现增强的HTTP节点池
├── lru/                # LRU缓存实现
├── metrics/            # 监控指标
│   └── prometheus.go   # Prometheus客户端
├── peers.go            # 节点接口
├── registry/           # 服务注册与发现
│   ├── discovery.go    # 服务发现
│   ├── etcd.go         # etcd实现
│   └── registry.go     # 服务注册
├── singleflight/       # 防缓存击穿实现
└── storage/            # 存储接口
```

## 开发与调试

### 添加新特性

1. 实现新的压缩算法
   ```go
   // 实现Compressor接口
   type MyCompressor struct {}
   
   func (c *MyCompressor) Compress(data []byte) ([]byte, error) {
       // 实现压缩逻辑
   }
   
   func (c *MyCompressor) Decompress(data []byte) ([]byte, error) {
       // 实现解压逻辑
   }
   ```

2. 添加新存储后端
   ```go
   // 实现Storage接口
   type MyStorage struct {}
   
   func (s *MyStorage) Get(key string) ([]byte, error) {
       // 实现获取逻辑
   }
   
   func (s *MyStorage) Set(key string, value []byte) error {
       // 实现设置逻辑
   }
   ```

### 问题排查

如果遇到服务发现问题，可以通过etcd命令行检查服务注册情况：

```bash
etcdctl get --prefix /geecache/registry/
``` 