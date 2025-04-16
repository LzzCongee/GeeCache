package main

import (
	"flag"
	"fmt"
	"geecache"
	"geecache/registry"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建缓存组
func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动简单模式的缓存服务器
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动支持服务发现的缓存服务器
func startCacheServerWithDiscovery(addr string, etcdEndpoints []string, gee *geecache.Group) (*registry.EtcdRegistry, error) {
	// 创建服务发现客户端
	discovery, err := registry.NewDiscovery(registry.RegistryTypeEtcd, etcdEndpoints, registry.DefaultServicePrefix)
	if err != nil {
		return nil, fmt.Errorf("create discovery error: %v", err)
	}

	// 创建支持服务发现的HTTP节点池
	peers := geecache.NewHTTPPoolWithDiscovery(addr, discovery, registry.DefaultServicePrefix)
	gee.RegisterPeers(peers)

	// 创建服务注册客户端
	r, err := registry.NewRegistry(registry.RegistryTypeEtcd, etcdEndpoints, 10)
	if err != nil {
		return nil, fmt.Errorf("create registry error: %v", err)
	}

	// 注册服务
	err = geecache.RegisterService(r, addr, registry.DefaultServicePrefix, map[string]string{
		"group": gee.Name(),
	})
	if err != nil {
		return nil, fmt.Errorf("register service error: %v", err)
	}

	// 启动HTTP服务
	go func() {
		log.Println("geecache is running at", addr)
		log.Fatal(http.ListenAndServe(addr[7:], peers))
	}()

	return r.(*registry.EtcdRegistry), nil
}

// 启动API服务器
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))

	// 添加一个简单的指标接口
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := gee.GetStats()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"cache_size": %d, "hits": %d, "misses": %d}`,
			stats.Size, stats.Hits, stats.Misses)
	})

	log.Println("API server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

// 测试分布式功能
func testDistributed(apiAddr string) {
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	keys := []string{"Tom", "Jack", "Sam", "kkk"}

	for _, key := range keys {
		url := fmt.Sprintf("%s/api?key=%s", apiAddr, key)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Failed to get %s: %v\n", key, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("Successfully retrieved key %s\n", key)
		} else {
			log.Printf("Failed to get key %s, status: %d\n", key, resp.StatusCode)
		}
	}
}

func main() {
	var port int
	var api bool
	var useEtcd bool
	var runTests bool
	var etcdEndpoints string

	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start an API server?")
	flag.BoolVar(&useEtcd, "etcd", false, "Use etcd for service discovery?")
	flag.BoolVar(&runTests, "test", false, "Run distributed tests after startup")
	flag.StringVar(&etcdEndpoints, "etcd-endpoints", "localhost:2379", "Etcd endpoints, separated by comma")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建缓存组
	gee := createGroup()

	// 启动API服务器
	if api {
		go startAPIServer(apiAddr, gee)
		log.Printf("API server started at %s\n", apiAddr)

		// 如果指定了运行测试
		if runTests {
			time.Sleep(1 * time.Second) // 等待API服务器启动
			log.Println("Running distributed tests...")
			testDistributed(apiAddr)
		}
	}

	// 使用ETCD进行服务注册与发现
	if useEtcd {
		endpoints := strings.Split(etcdEndpoints, ",")
		log.Printf("Using etcd service discovery with endpoints: %v\n", endpoints)

		// 直接基于port构建地址，不依赖于硬编码的addrMap
		addr := fmt.Sprintf("http://localhost:%d", port)
		log.Printf("Starting cache server at %s with etcd service discovery\n", addr)

		reg, err := startCacheServerWithDiscovery(addr, endpoints, gee)
		if err != nil {
			log.Fatalf("Failed to start cache server with discovery: %v", err)
		}

		// 优雅退出 设置信号监听，等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 注册信号监听
		log.Println("Press Ctrl+C to shut down server...")
		<-quit // 阻塞在这里，直到收到中断信号
		log.Println("Shutting down server...")

		// 注销服务
		err = reg.Deregister()
		if err != nil {
			log.Printf("Failed to deregister service: %v", err)
		}
		log.Println("Server exited")
		return
	} else {
		// 使用硬编码方式指定节点地址
		log.Printf("Starting cache server at %s with peers %v\n", addrMap[port], addrs)
		startCacheServer(addrMap[port], addrs, gee)
	}
}
