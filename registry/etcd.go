package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"maps"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceInfo 表示一个服务节点的信息
type ServiceInfo struct {
	Addr     string            // 服务地址
	Metadata map[string]string // 服务元数据
}

// EtcdRegistry 实现基于ETCD的服务注册与发现
type EtcdRegistry struct {
	client      *clientv3.Client // ETCD 客户端
	leaseID     clientv3.LeaseID // 租约ID
	serviceTTL  int64            // 服务存活时间
	serviceKey  string           // 服务键名
	serviceInfo ServiceInfo      // 服务信息
	stopSignal  chan struct{}    // 停止信号
	registered  bool             // 是否已注册
	mu          sync.Mutex       // 互斥锁
}

// NewEtcdRegistry 创建一个新的ETCD注册中心客户端
func NewEtcdRegistry(endpoints []string, serviceTTL int64) (*EtcdRegistry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &EtcdRegistry{
		client:     client,
		serviceTTL: serviceTTL,
		stopSignal: make(chan struct{}),
	}, nil
}

// Register 注册服务到ETCD
func (e *EtcdRegistry) Register(serviceKey string, info ServiceInfo) error {
	e.mu.Lock() // 使用互斥锁保证并发安全。
	defer e.mu.Unlock()

	if e.registered {
		// 检查服务是否已注册，避免重复注册。
		return fmt.Errorf("service already registered")
	}

	// context.Background()创建了一个空的上下文对象，作为父上下文。父上下文通常用于传递请求的初始参数和取消信号。
	// WithTimeout函数创建了一个新的子上下文ctx，并设置了一个5秒的超时时间。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 手动取消上下文

	// 创建租约
	resp, err := e.client.Grant(ctx, e.serviceTTL)
	if err != nil {
		return fmt.Errorf("create lease error: %v", err)
	}
	e.leaseID = resp.ID

	// 序列化服务信息
	value, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal service info error: %v", err)
	}

	// 注册服务, Put 方法将服务信息存储到ETCD，并关联租约。
	_, err = e.client.Put(ctx, serviceKey, string(value), clientv3.WithLease(e.leaseID))
	if err != nil {
		return fmt.Errorf("put service info error: %v", err)
	}

	// 保持租约
	// e.client.KeepAlive(context.Background(), e.leaseID)方法为租约启动一个续约机制，确保租约在TTL到期前不会被删除。
	// keepAliveCh是一个通道，用于接收租约续约的响应。如果启动续约机制失败，返回一个包含错误信息的错误对象。
	keepAliveCh, err := e.client.KeepAlive(context.Background(), e.leaseID)
	if err != nil {
		return fmt.Errorf("keep alive error: %v", err)
	}

	e.serviceKey = serviceKey
	e.serviceInfo = info
	e.registered = true

	// 启动保持租约的goroutine
	go e.keepAlive(keepAliveCh)

	log.Printf("Service registered: %s -> %s\n", serviceKey, info.Addr)
	return nil
}

// 保持租约活跃
func (e *EtcdRegistry) keepAlive(keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		// 使用 for 循环持续监听 keepAliveCh 通道。
		select {
		case <-e.stopSignal:
			// 监听 e.stopSignal 通道，如果收到停止信号，则退出循环。
			return
		case resp, ok := <-keepAliveCh:
			if !ok {
				log.Println("Keep alive channel closed, trying to re-register...")
				// 尝试重新注册
				if err := e.reRegister(); err != nil {
					log.Printf("Re-register failed: %v\n", err)
				}
				return
			}
			log.Printf("Lease renewed: %d\n", resp.ID)
		}
	}
}

// 重新注册服务
func (e *EtcdRegistry) reRegister() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.registered = false
	return e.Register(e.serviceKey, e.serviceInfo)
}

// Deregister 注销服务
func (e *EtcdRegistry) Deregister() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.registered {
		return nil
	}

	close(e.stopSignal) // 关闭停止信号，停止租约续约的goroutine。

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 撤销租约，自动删除关联的key
	_, err := e.client.Revoke(ctx, e.leaseID)
	if err != nil {
		return fmt.Errorf("revoke lease error: %v", err)
	}

	e.registered = false
	log.Printf("Service deregistered: %s\n", e.serviceKey)
	return nil
}

// Close 关闭ETCD客户端
func (e *EtcdRegistry) Close() error {
	if e.registered {
		// 如果服务已注册，先注销服务
		if err := e.Deregister(); err != nil {
			return err
		}
	}
	return e.client.Close()
}

// ServiceDiscovery 服务发现接口
type ServiceDiscovery struct {
	client    *clientv3.Client       // ETCD 客户端
	services  map[string]ServiceInfo // 服务缓存,服务列表，存储已发现的服务信息
	watchChan clientv3.WatchChan     // 监听通道
	mu        sync.RWMutex           // 读写锁
	prefix    string                 // 服务前缀，用于过滤服务
}

// NewServiceDiscovery 创建一个新的服务发现客户端
func NewServiceDiscovery(endpoints []string, prefix string) (*ServiceDiscovery, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	sd := &ServiceDiscovery{
		client:   client,
		services: make(map[string]ServiceInfo),
		prefix:   prefix,
	}

	// 初始化服务列表
	// 调用 fetchServices 方法从 etcd 中获取当前注册的服务列表，并将其存储在 services 映射中
	if err := sd.fetchServices(); err != nil {
		// 如果获取服务列表失败，关闭 etcd 客户端并返回错误
		client.Close()
		return nil, err
	}

	// 监听服务变化
	// watchServices 方法监听 etcd 中服务的变化。当服务列表发生变化时，自动更新本地的服务列表。
	go sd.watchServices()

	return sd, nil
}

// 获取所有服务
// 使用 Get 方法获取符合前缀的服务信息
// 解析服务信息并更新服务列表
func (sd *ServiceDiscovery) fetchServices() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := sd.client.Get(ctx, sd.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		var service ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			log.Printf("Unmarshal service error: %v\n", err)
			continue
		}
		sd.mu.Lock()
		sd.services[string(kv.Key)] = service
		sd.mu.Unlock()
	}
	return nil
}

// 监听服务变化
func (sd *ServiceDiscovery) watchServices() {
	// sd.prefix：存储服务信息的路径前缀（例如 "/services/cache/"）
	// WithPrefix()：监听 所有以 sd.prefix 开头的键，确保监控整个服务目录的变化
	sd.watchChan = sd.client.Watch(context.Background(), sd.prefix, clientv3.WithPrefix())
	for wresp := range sd.watchChan {
		// 处理服务变化事件
		// wresp 代表一次 服务变更事件的集合
		for _, ev := range wresp.Events {
			// wresp.Events：ETCD 可能在一次通知里包含多个变更事件（多个服务同时发生变更）
			// 遍历 Events，依次处理每个 新增/更新/删除 事件
			switch ev.Type {
			case clientv3.EventTypePut:
				// 表示 ETCD 新增或更新了某个服务节点
				var service ServiceInfo
				// ev.Kv.Value：ETCD 里存储的服务信息（JSON格式）
				if err := json.Unmarshal(ev.Kv.Value, &service); err != nil {
					log.Printf("Unmarshal service error: %v\n", err)
					continue
				}
				sd.mu.Lock()
				sd.services[string(ev.Kv.Key)] = service
				sd.mu.Unlock()
				log.Printf("Service added/updated: %s -> %s\n", ev.Kv.Key, service.Addr)
			case clientv3.EventTypeDelete:
				// 当某个服务节点 从 ETCD 里删除 时，表示该服务下线
				sd.mu.Lock()
				delete(sd.services, string(ev.Kv.Key))
				sd.mu.Unlock()
				log.Printf("Service removed: %s\n", ev.Kv.Key)
			}
		}
	}
}

// GetServices 获取所有服务
func (sd *ServiceDiscovery) GetServices() map[string]ServiceInfo {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	services := make(map[string]ServiceInfo, len(sd.services))
	maps.Copy(services, sd.services)
	return services
}

// GetService 获取指定服务
func (sd *ServiceDiscovery) GetService(serviceKey string) (ServiceInfo, bool) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	service, ok := sd.services[serviceKey]
	return service, ok
}

// GetServicesByPrefix 获取指定前缀的服务
func (sd *ServiceDiscovery) GetServicesByPrefix(prefix string) map[string]ServiceInfo {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	services := make(map[string]ServiceInfo)
	for k, v := range sd.services {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			services[k] = v
		}
	}
	return services
}

// Close 关闭服务发现客户端
func (sd *ServiceDiscovery) Close() error {
	return sd.client.Close()
}
