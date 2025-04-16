# 分布式缓存存储引擎实现文档

## 1. 内存存储引擎实现

### 1.1 基本结构

内存存储引擎（MemoryStorage）是一个基于内存的键值存储实现，它实现了Storage接口定义的所有方法。主要特点包括：

- 基于Go原生的map实现键值存储
- 支持设置过期时间
- 支持最大内存限制
- 线程安全（使用读写锁保护）
- 自动清理过期数据

### 1.2 核心数据结构

```go
type MemoryStorage struct {
	data     map[string][]byte     // 存储键值对
	expiries map[string]time.Time  // 存储过期时间
	maxSize  int64                 // 最大存储大小
	size     int64                 // 当前使用的存储大小
	mu       sync.RWMutex          // 读写锁
	
	// 清理相关
	cleanupInterval time.Duration  // 清理间隔
	stopCleanup     chan struct{}  // 停止清理的信号通道
	cleanupRunning  bool           // 清理协程是否在运行
}
```

### 1.3 主要改进

相比原始实现，我们对内存存储引擎进行了以下改进：

1. **自动清理过期数据**：添加了定时清理机制，定期清理过期的键值对，避免内存泄漏。
2. **更精确的内存管理**：精确计算每个键值对占用的内存大小，确保不超过最大内存限制。
3. **并发安全性增强**：使用读写锁分离读写操作，提高并发性能。
4. **优雅关闭**：添加了停止清理协程的机制，确保资源正确释放。

### 1.4 关键方法实现

#### 创建存储引擎

```go
func NewMemoryStorage(options StorageOptions) (*MemoryStorage, error) {
	storage := &MemoryStorage{
		data:            make(map[string][]byte),
		expiries:        make(map[string]time.Time),
		maxSize:         options.MaxSize,
		cleanupInterval: 5 * time.Minute, // 默认5分钟清理一次过期数据
		stopCleanup:     make(chan struct{}),
	}
	
	// 启动自动清理过期数据的协程
	storage.startCleanupTimer()
	
	return storage, nil
}
```

#### 自动清理过期数据

```go
func (s *MemoryStorage) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	expiredCount := 0
	expiredSize := int64(0)
	
	for key, expiry := range s.expiries {
		if now.After(expiry) {
			if value, ok := s.data[key]; ok {
				expiredSize += int64(len(key) + len(value))
				delete(s.data, key)
				expiredCount++
			}
			delete(s.expiries, key)
		}
	}
	
	// 更新存储大小
	if expiredCount > 0 {
		s.size -= expiredSize
		log.Printf("[MemoryStorage] Cleaned %d expired items, freed %d bytes", expiredCount, expiredSize)
	}
}
```

## 2. 测试程序完善

### 2.1 基础功能测试

基础功能测试覆盖了Storage接口的所有方法，包括：

- Set/Get：设置和获取键值对
- Has：检查键是否存在
- Keys：获取所有键
- Delete：删除指定键
- SetWithExpire：设置带过期时间的键值对
- Clear：清空存储

### 2.2 边界条件测试

边界条件测试主要测试以下场景：

- 存储容量限制：当达到最大容量时，应该自动淘汰旧数据
- 获取不存在的键：应该返回适当的错误
- 删除不存在的键：应该正常处理
- 零长度值：应该能正确处理空值

### 2.3 并发安全测试

并发安全测试通过多个goroutine同时操作存储引擎，验证其在高并发场景下的正确性：

- 并发写入：多个goroutine同时写入不同的键值对
- 并发读取：多个goroutine同时读取刚写入的键值对
- 并发删除：多个goroutine同时删除键值对

### 2.4 性能测试

性能测试使用Go的基准测试功能，测量以下操作的性能：

- Set：设置键值对的性能
- Get：获取键值对的性能
- Delete：删除键值对的性能
- SetWithExpire：设置带过期时间的键值对的性能

## 3. 与分布式缓存系统集成

### 3.1 当前缓存实现分析

目前的分布式缓存系统使用了简单的内存缓存实现（cache.go），主要特点：

- 基于LRU算法实现缓存淘汰
- 使用互斥锁保证并发安全
- 缺乏持久化能力
- 没有过期时间支持

### 3.2 集成存储引擎的优势

将存储引擎与分布式缓存系统集成有以下优势：

1. **多种存储选择**：可以根据需求选择不同的存储引擎（内存、LevelDB、RocksDB等）
2. **过期时间支持**：原生支持设置缓存项的过期时间
3. **更好的内存管理**：精确控制内存使用量
4. **持久化能力**：使用持久化存储引擎可以实现数据持久化
5. **更高的扩展性**：可以轻松添加新的存储引擎实现

### 3.3 集成方案

#### 3.3.1 修改ByteView结构

```go
type ByteView struct {
	b []byte
	// 可以添加过期时间等元数据
	expireAt time.Time
}
```

#### 3.3.2 修改cache结构

```go
type cache struct {
	storage storage.Storage
	// 可以保留一些统计信息
	bytes int64 // 当前缓存使用的字节数
}
```

#### 3.3.3 修改cache方法

```go
func (c *cache) add(key string, value ByteView) {
	// 使用存储引擎存储数据
	if !value.expireAt.IsZero() {
		// 计算过期时间
		expire := time.Until(value.expireAt)
		c.storage.SetWithExpire(key, value.b, expire)
	} else {
		c.storage.Set(key, value.b)
	}
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	// 从存储引擎获取数据
	data, err := c.storage.Get(key)
	if err != nil {
		return ByteView{}, false
	}
	return ByteView{b: cloneBytes(data)}, true
}
```

#### 3.3.4 修改Group初始化

```go
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	
	// 创建存储引擎
	storage, err := storage.NewStorage(storage.StorageTypeMemory, storage.StorageOptions{
		MaxSize: cacheBytes,
	})
	if err != nil {
		panic(fmt.Sprintf("create storage error: %v", err))
	}
	
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			storage: storage,
		},
		loader: &singleflight.Group{},
	}
	
	// 初始化热点数据相关字段
	g.hotSpot.accessCount = make(map[string]int)
	g.hotSpot.hotKeys = make(map[string]bool)
	g.hotSpot.threshold = 100 // 默认阈值
	g.hotSpot.backupCount = 2 // 默认备份节点数
	g.hotSpot.lastCleanTime = time.Now()
	groups[name] = g
	return g
}
```

### 3.4 配置选项

可以添加配置选项，允许用户选择不同的存储引擎：

```go
type GroupOptions struct {
	StorageType storage.StorageType
	StorageOptions storage.StorageOptions
}

func NewGroupWithOptions(name string, options GroupOptions, getter Getter) *Group {
	// 使用指定的存储引擎创建Group
}
```

## 4. 未来扩展

### 4.1 实现更多存储引擎

- **LevelDB存储引擎**：适用于需要持久化但数据量不是特别大的场景
- **RocksDB存储引擎**：适用于大数据量、高性能需求的场景
- **Badger存储引擎**：纯Go实现的高性能键值存储

### 4.2 增强功能

- **数据压缩**：实现数据压缩以节省存储空间
- **批量操作**：支持批量读写操作，提高性能
- **事务支持**：为需要原子操作的场景提供事务支持
- **数据分片**：支持数据分片，提高大数据量下的性能

### 4.3 监控与统计

- **性能指标**：收集各种操作的性能指标
- **容量监控**：监控存储容量使用情况
- **热点数据分析**：分析热点数据访问模式，优化缓存策略