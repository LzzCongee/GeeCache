package geecache

import (
	"fmt"
	"geecache/registry"
	"log"
	"sync"
	"time"
)

// HTTPPoolWithDiscovery 实现基于服务发现的HTTP节点池
type HTTPPoolWithDiscovery struct {
	*HTTPPool
	discovery       registry.Discovery // 服务发现客户端
	servicePrefix   string             // 服务前缀
	refreshInterval time.Duration      // 刷新间隔
	stopSignal      chan struct{}      // 停止信号
	mu              sync.Mutex         // 互斥锁
}

// NewHTTPPoolWithDiscovery 创建一个支持服务发现的HTTP节点池
func NewHTTPPoolWithDiscovery(self string, discovery registry.Discovery, servicePrefix string) *HTTPPoolWithDiscovery {
	if servicePrefix == "" {
		servicePrefix = registry.DefaultServicePrefix
	}

	pool := &HTTPPoolWithDiscovery{
		HTTPPool:        NewHTTPPool(self),
		discovery:       discovery,
		servicePrefix:   servicePrefix,
		refreshInterval: 10 * time.Second,
		stopSignal:      make(chan struct{}),
	}

	// 初始化节点列表
	pool.refreshPeers()

	// 启动定期刷新节点的goroutine
	go pool.refreshPeersLoop()

	return pool
}

// 刷新节点列表
func (p *HTTPPoolWithDiscovery) refreshPeers() {
	p.mu.Lock()
	defer p.mu.Unlock()

	services := p.discovery.GetServicesByPrefix(p.servicePrefix)
	if len(services) == 0 {
		log.Println("No services found with prefix:", p.servicePrefix)
		return
	}

	var peers []string
	for _, service := range services {
		peers = append(peers, service.Addr)
	}

	// 更新节点列表
	p.HTTPPool.Set(peers...)
	log.Printf("Refreshed peers: %v\n", peers)
}

// 定期刷新节点列表
func (p *HTTPPoolWithDiscovery) refreshPeersLoop() {
	// 创建一个定时器，固定时间间隔内周期性触发，并向 ticker.C 这个通道发送信号
	ticker := time.NewTicker(p.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopSignal:
			// 监听 stopSignal，收到信号则退出循环，终止刷新任务
			return
		case <-ticker.C:
			// 每隔一段时间（refreshInterval），刷新节点列表
			p.refreshPeers()
		}
	}
}

// SetRefreshInterval 设置刷新间隔
func (p *HTTPPoolWithDiscovery) SetRefreshInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refreshInterval = interval
}

// Close 关闭服务发现
func (p *HTTPPoolWithDiscovery) Close() error {
	close(p.stopSignal)
	return nil
}

// RegisterService 注册服务到注册中心
func RegisterService(r registry.Registry, addr string, servicePrefix string, metadata map[string]string) error {
	if servicePrefix == "" {
		servicePrefix = registry.DefaultServicePrefix
	}

	// 生成服务键
	serviceKey := fmt.Sprintf("%s%s", servicePrefix, addr)

	// 注册服务
	return r.Register(serviceKey, registry.ServiceInfo{
		Addr:     addr,
		Metadata: metadata,
	})
}
