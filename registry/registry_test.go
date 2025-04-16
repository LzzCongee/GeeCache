package registry

import (
	"fmt"
	"os/exec"
	"testing"
	"time"
)

// 测试配置
var (
	testEndpoints = []string{"127.0.0.1:2379"} // 使用 IPv4 地址
	testTTL       = int64(10)
	testPrefix    = DefaultServicePrefix
	testKey       = testPrefix + "test-service"
	testAddr      = "http://localhost:8001"
)

// 启动 ETCD 服务器
func startEtcdServer() error {
	// 检查是否已经存在 ETCD 容器
	checkCmd := exec.Command("docker", "ps", "-q", "-f", "name=etcd-test")
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("check existing container error: %v", err)
	}

	// 如果容器已存在，先停止并删除
	if len(output) > 0 {
		stopCmd := exec.Command("docker", "stop", "etcd-test")
		if err := stopCmd.Run(); err != nil {
			return fmt.Errorf("stop existing container error: %v", err)
		}
		rmCmd := exec.Command("docker", "rm", "etcd-test")
		if err := rmCmd.Run(); err != nil {
			return fmt.Errorf("remove existing container error: %v", err)
		}
	}

	// 启动新的 ETCD 容器
	startCmd := exec.Command("docker", "run", "-d",
		"--name", "etcd-test",
		"-p", "2379:2379",
		"-p", "2380:2380",
		"quay.io/coreos/etcd:v3.5.0",
		"etcd",
		"--advertise-client-urls", "http://0.0.0.0:2379",
		"--listen-client-urls", "http://0.0.0.0:2379",
		"--data-dir", "/etcd-data")

	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("start etcd container error: %v", err)
	}

	// 等待 ETCD 服务器启动
	time.Sleep(2 * time.Second)
	return nil
}

// 停止 ETCD 服务器
func stopEtcdServer() error {
	stopCmd := exec.Command("docker", "stop", "etcd-test")
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("stop etcd container error: %v", err)
	}

	rmCmd := exec.Command("docker", "rm", "etcd-test")
	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("remove etcd container error: %v", err)
	}

	return nil
}

// 测试 ETCD 服务注册与发现
func TestEtcdRegistryAndDiscovery(t *testing.T) {
	// 启动 ETCD 服务器
	if err := startEtcdServer(); err != nil {
		t.Fatalf("Failed to start ETCD server: %v", err)
	}
	defer stopEtcdServer()

	// 创建注册中心客户端
	registry, err := NewRegistry(RegistryTypeEtcd, testEndpoints, testTTL)
	if err != nil {
		t.Fatalf("Create registry error: %v", err)
	}
	defer registry.Close()

	// 注册服务
	err = registry.Register(testKey, ServiceInfo{
		Addr: testAddr,
		Metadata: map[string]string{
			"group": "test",
		},
	})
	if err != nil {
		t.Fatalf("Register service error: %v", err)
	}

	// 创建服务发现客户端
	discovery, err := NewDiscovery(RegistryTypeEtcd, testEndpoints, testPrefix)
	if err != nil {
		t.Fatalf("Create discovery error: %v", err)
	}
	defer discovery.Close()

	// 等待服务注册完成
	time.Sleep(1 * time.Second)

	// 获取服务
	service, ok := discovery.GetService(testKey)
	if !ok {
		t.Fatalf("Service not found: %s", testKey)
	}

	// 验证服务信息
	if service.Addr != testAddr {
		t.Errorf("Service address mismatch: got %s, want %s", service.Addr, testAddr)
	}
	if service.Metadata["group"] != "test" {
		t.Errorf("Service metadata mismatch: got %s, want %s", service.Metadata["group"], "test")
	}

	// 测试 GetServicesByPrefix
	services := discovery.GetServicesByPrefix(testPrefix)
	if len(services) == 0 {
		t.Error("No services found with prefix")
	}

	// 注销服务
	err = registry.Deregister()
	if err != nil {
		t.Fatalf("Deregister service error: %v", err)
	}

	// 等待服务注销完成
	time.Sleep(1 * time.Second)

	// 验证服务已注销
	_, ok = discovery.GetService(testKey)
	if ok {
		t.Errorf("Service still exists after deregistration: %s", testKey)
	}
}

// 测试服务发现的基本功能
func TestServiceDiscovery(t *testing.T) {
	// 启动 ETCD 服务器
	if err := startEtcdServer(); err != nil {
		t.Fatalf("Failed to start ETCD server: %v", err)
	}
	defer stopEtcdServer()

	// 创建服务发现客户端
	discovery, err := NewDiscovery(RegistryTypeEtcd, testEndpoints, testPrefix)
	if err != nil {
		t.Fatalf("Create discovery error: %v", err)
	}
	defer discovery.Close()

	// 测试获取所有服务
	services := discovery.GetServices()
	if services == nil {
		t.Error("GetServices returned nil map")
	}

	// 测试获取不存在的服务
	_, ok := discovery.GetService("non-existent-service")
	if ok {
		t.Error("GetService returned true for non-existent service")
	}
}
