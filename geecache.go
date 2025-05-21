package geecache

import (
	"context"
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
	"log"
	"sync"
	"time"
)

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group

	// 统计信息
	stats struct {
		hits   int64        // 缓存命中次数
		misses int64        // 缓存未命中次数
		mu     sync.RWMutex // 用于保护统计字段
	}

	// 热点数据相关
	hotSpot struct {
		mu            sync.RWMutex    // 用于保护热点数据字段
		accessCount   map[string]int  // 记录每个key的访问次数
		hotKeys       map[string]bool // 热点key集合
		threshold     int             // 热点判定阈值
		backupCount   int             // 热点数据备份节点数量
		lastCleanTime time.Time       // 上次清理时间
	}
}

func (g *Group) IsHotSpot(key string) bool {
	g.hotSpot.mu.RLock()
	defer g.hotSpot.mu.RUnlock()
	_, exists := g.hotSpot.hotKeys[key]
	return exists
}

// A Getter loads data for a key.
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group) // 一个缓存节点可以有多个命名组
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
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

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Name returns the name of the group
func (g *Group) Name() string {
	return g.name
}

// Stats represents cache statistics
type Stats struct {
	Hits   int64 // number of cache hits
	Misses int64 // number of cache misses
	Size   int64 // current size of cache
}

// GetStats returns a copy of current statistics
func (g *Group) GetStats() Stats {
	g.stats.mu.RLock()
	defer g.stats.mu.RUnlock()
	return Stats{
		Hits:   g.stats.hits,
		Misses: g.stats.misses,
		Size:   g.mainCache.bytes,
	}
}

// 记录key的访问次数并检查是否为热点数据
func (g *Group) recordAccess(key string) bool {
	g.hotSpot.mu.Lock()
	defer g.hotSpot.mu.Unlock()

	// 如果已经是热点数据，直接返回
	if g.hotSpot.hotKeys[key] {
		return true
	}

	// 增加访问计数
	g.hotSpot.accessCount[key]++

	// 检查是否达到热点阈值
	if g.hotSpot.accessCount[key] >= g.hotSpot.threshold {
		g.hotSpot.hotKeys[key] = true
		log.Printf("[GeeCache] Key %s becomes hot spot data", key)
		return true
	}

	// 每隔一段时间清理过期的访问计数，使用goroutine异步执行，避免长时间持有锁
	if time.Since(g.hotSpot.lastCleanTime) > 10*time.Minute {
		// 更新清理时间，避免短时间内多次触发清理
		g.hotSpot.lastCleanTime = time.Now()
		go func() {
			// 使用defer捕获可能的panic，确保异步操作的可靠性
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[GeeCache] Panic in CleanExpiredHotSpot: %v", r)
				}
			}()
			g.CleanExpiredHotSpot()
		}()
	}

	return false
}

// 清理过期的热点数据和访问计数
func (g *Group) CleanExpiredHotSpot() {
	// 独立获取锁，避免在 recordAccess 中长时间持有锁
	g.hotSpot.mu.Lock()
	defer g.hotSpot.mu.Unlock()

	// 更新清理时间
	g.hotSpot.lastCleanTime = time.Now()

	// 清理访问计数
	newAccessCount := make(map[string]int)
	for k, v := range g.hotSpot.accessCount {
		// 保留热点数据和访问次数较高的数据
		if g.hotSpot.hotKeys[k] || v > g.hotSpot.threshold/2 {
			newAccessCount[k] = v / 2 // 衰减访问次数
		}
	}
	g.hotSpot.accessCount = newAccessCount

	// 重新评估热点数据
	newHotKeys := make(map[string]bool)
	for k, v := range g.hotSpot.accessCount {
		if v >= g.hotSpot.threshold {
			newHotKeys[k] = true
		}
	}
	g.hotSpot.hotKeys = newHotKeys

	log.Printf("[GeeCache] Cleaned expired hot spot data, remaining %d hot keys", len(g.hotSpot.hotKeys))
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		// 记录缓存命中
		g.stats.mu.Lock()
		g.stats.hits++
		g.stats.mu.Unlock()

		// 记录访问并检查是否为热点数据
		g.recordAccess(key)

		log.Println("[GeeCache] hit")
		return v, nil
	}

	// 记录缓存未命中
	g.stats.mu.Lock()
	g.stats.misses++
	g.stats.mu.Unlock()

	return g.load(key)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	// 即确保一定时间范围内对同一key的请求只执行一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 检查是否为热点数据
		isHotSpot := g.recordAccess(key)
		// TODO：添加超時上下文
		//     ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // 示例：设置3秒超时
		//     defer cancel()
		if g.peers != nil {
			// 如果是热点数据且有多个节点可用，使用并行查询(即同时向主数据源和备份源发起请求)
			if isHotSpot {
				peers, ok := g.peers.PickPeers(key, g.hotSpot.backupCount)
				if ok && len(peers) > 0 {
					log.Printf("[GeeCache] Fetching hot spot data %s from %d peers", key, len(peers))
					if value, err = g.getFromPeers(peers, key); err == nil {
						return value, nil
					}
					log.Println("[GeeCache] Failed to get hot spot data from peers", err)
				}
			} else {
				// 非热点数据，使用单节点查询
				if peer, ok := g.peers.PickPeer(key); ok {
					if value, err = g.getFromPeer(peer, key); err == nil {
						return value, nil
					}
					log.Println("[GeeCache] Failed to get from peer", err)
				}
			}
		}

		// 从本地获取数据
		value, err := g.getLocally(key)
		if err == nil && isHotSpot && g.peers != nil {
			// 如果是热点数据，异步将数据同步到备份节点
			go g.syncToBackupPeers(key, value)
		}
		return value, err
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)

	// 检查是否为热点数据，如果是则同步到备份节点
	g.hotSpot.mu.RLock()
	isHotSpot := g.hotSpot.hotKeys[key]
	g.hotSpot.mu.RUnlock()

	if isHotSpot && g.peers != nil {
		go g.syncToBackupPeers(key, value)
	}
}

// 相当于从数据库中获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res) // 从远程节点获取指定值
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

// 从多个节点并行获取数据
func (g *Group) getFromPeers(peers []PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan ByteView, 1)
	errChan := make(chan error, len(peers))
	var wg sync.WaitGroup

	// 并行从多个节点获取数据
	for _, peer := range peers {
		wg.Add(1)
		go func(p PeerGetter) {
			defer wg.Done()
			res := &pb.Response{}
			err := p.Get(req, res)
			if err != nil {
				errChan <- err
				return
			}
			// 一旦有一个节点返回结果，就取消其他请求
			select {
			case resultChan <- ByteView{b: res.Value}:
				cancel()
			case <-ctx.Done():
				// 其他goroutine已经获取到结果
			}
		}(peer)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(resultChan)
		close(errChan)
	}()

	// 等待结果或超时
	select {
	case result := <-resultChan:
		return result, nil
	case <-time.After(500 * time.Millisecond):
		// 所有节点都超时或失败
		return ByteView{}, fmt.Errorf("timeout waiting for peers")
	}
}

// 将热点数据同步到备份节点
// TODO: pb文件新增Set方法
func (g *Group) syncToBackupPeers(key string, value ByteView) {
	if g.peers == nil {
		return
	}

	// 获取备份节点
	peers, ok := g.peers.PickPeers(key, g.hotSpot.backupCount)
	if !ok || len(peers) == 0 {
		return
	}

	log.Printf("[GeeCache] Syncing hot spot data %s to %d backup peers", key, len(peers))

	// 构建请求
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{
		Value: value.ByteSlice(),
	}

	// 异步将数据同步到每个备份节点
	for _, peer := range peers {
		go func(p PeerGetter) {
			// 这里假设PeerGetter接口有一个Set方法用于设置数据
			// 实际上可能需要扩展PeerGetter接口或使用HTTP PUT请求
			// 这里简化处理，仅演示概念
			// 检查 peer 是否实现了 Set 方法 (可能需要扩展 PeerGetter 或在 httpGetter 中实现)
			if setter, ok := p.(interface {
				Set(*pb.Request, *pb.Response) error
			}); ok {
				err := setter.Set(req, res)
				if err != nil {
					// 如果 PeerGetter 没有 Set 方法，可能需要记录日志或采取其他措施
					log.Printf("[GeeCache] Failed to sync hot spot data to peer: %v", err)
				}
			}
		}(peer)
	}
}

// SetHotSpotThreshold 设置热点数据判定阈值
func (g *Group) SetHotSpotThreshold(threshold int) {
	g.hotSpot.mu.Lock()
	defer g.hotSpot.mu.Unlock()
	g.hotSpot.threshold = threshold
}

// SetBackupCount 设置热点数据备份节点数量
func (g *Group) SetBackupCount(count int) {
	g.hotSpot.mu.Lock()
	defer g.hotSpot.mu.Unlock()
	g.hotSpot.backupCount = count
}

// GetBackupCount 获取当前的备份节点数量
func (g *Group) GetBackupCount() int {
	g.hotSpot.mu.RLock()
	defer g.hotSpot.mu.RUnlock()
	return g.hotSpot.backupCount
}

func (g *Group) GetPeers() PeerPicker {
	return g.peers
}
