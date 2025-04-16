package registry

import (
	"fmt"
)

// Registry 服务注册接口
type Registry interface {
	// Register 注册服务
	Register(serviceKey string, info ServiceInfo) error
	// Deregister 注销服务
	Deregister() error
	// Close 关闭注册中心客户端
	Close() error
}

// Discovery 服务发现接口
type Discovery interface {
	// GetServices 获取所有服务
	GetServices() map[string]ServiceInfo
	// GetService 获取指定服务
	GetService(serviceKey string) (ServiceInfo, bool)
	// GetServicesByPrefix 获取指定前缀的服务
	GetServicesByPrefix(prefix string) map[string]ServiceInfo
	// Close 关闭服务发现客户端
	Close() error
}

// RegistryType 注册中心类型
type RegistryType string

const (
	// RegistryTypeEtcd ETCD注册中心
	RegistryTypeEtcd RegistryType = "etcd"
	// RegistryTypeConsul Consul注册中心
	RegistryTypeConsul RegistryType = "consul"
	// RegistryTypeZookeeper Zookeeper注册中心
	RegistryTypeZookeeper RegistryType = "zookeeper"
)

// ServicePrefix 服务前缀
const (
	DefaultServicePrefix = "/services/geecache/"
)

// NewRegistry 创建一个新的注册中心客户端
func NewRegistry(registryType RegistryType, endpoints []string, serviceTTL int64) (Registry, error) {
	switch registryType {
	case RegistryTypeEtcd:
		return NewEtcdRegistry(endpoints, serviceTTL)
	default:
		return NewEtcdRegistry(endpoints, serviceTTL)
	}
}

// NewDiscovery 创建一个新的服务发现客户端
func NewDiscovery(registryType RegistryType, endpoints []string, prefix string) (Discovery, error) {
	switch registryType {
	case RegistryTypeEtcd:
		return NewServiceDiscovery(endpoints, prefix)
	case RegistryTypeConsul:
		return nil, fmt.Errorf("consul discovery not implemented yet")
	case RegistryTypeZookeeper:
		return nil, fmt.Errorf("zookeeper discovery not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", registryType)
	}
}
