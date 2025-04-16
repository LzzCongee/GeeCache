package perf

import (
	"fmt"
	"geecache"
	"geecache/compression"
	"geecache/registry"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// 测试数据生成
// 生成随机长度的值，模拟不同大小的缓存项
// 每个值的大小在 100B 到 10KB 之间，模拟不同大小的缓存项
func generateTestData(size int) map[string]string {
	data := make(map[string]string)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key-%d", i)
		// 生成随机长度的值，模拟不同大小的缓存项
		valueSize := rand.Intn(10000) + 100 // 100B 到 10KB
		value := make([]byte, valueSize)
		rand.Read(value)
		data[key] = string(value)
	}
	return data
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

func createNewNode(testData map[string]string, port int) {
	useEtcd := true
	etcdEndpoints := "localhost:2379"
	gee := geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := testData[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
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
	}
}

// 初始化日志文件
func setupLogFile() *os.File {
	// 创建日志文件
	logFile, err := os.Create("performance_test.log")
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	// 使用MultiWriter将日志同时输出到文件和标准输出
	// mw := io.MultiWriter(os.Stdout, logFile)
	// log.SetOutput(mw)
	log.SetOutput(logFile)
	log.Printf("=== 性能测试开始于 %s ===\n", time.Now().Format(time.RFC3339))
	return logFile
}

// 基准测试：本地缓存性能
// 初始化测试数据并创建缓存组。
// 使用 b.RunParallel 并发执行基准测试，模拟多个 goroutine 同时访问缓存。
// 记录操作数、耗时和吞吐量，并计算缓存命中率。
func BenchmarkLocalCache(b *testing.B) {
	logFile := setupLogFile()
	defer logFile.Close()

	// 初始化测试数据
	testData := generateTestData(1000)

	// 创建缓存组
	group := geecache.NewGroup("test-local", 10<<20, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			if v, ok := testData[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("key not found")
		}))

	// 记录开始时间
	startTime := time.Now()
	b.ResetTimer()

	// 执行基准测试
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			view, err := group.Get(key)
			if err != nil {
				b.Fatalf("获取缓存数据失败: %v", err)
			}
			_ = view.String() // 使用结果防止编译器优化
			i++
		}
	})

	duration := time.Since(startTime)
	ops := float64(b.N) / duration.Seconds()

	// 记录测试结果
	log.Printf("本地缓存测试完成，操作数: %d, 耗时: %v, 吞吐量: %.2f ops/s\n",
		b.N, duration, ops)

	// 记录缓存命中率等指标
	stats := group.GetStats()
	hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
	log.Printf("缓存命中率: %.2f%%\n", hitRate)
	log.Printf("缓存大小: %d bytes\n", stats.Size)
}

// 基准测试：压缩性能
// 生成 1MB 的测试数据。
// 定义多个压缩算法及其级别。
// 使用 b.Run 进行基准测试，记录压缩和解压的时间，并计算压缩率。
func BenchmarkCompression(b *testing.B) {
	logFile := setupLogFile()
	defer logFile.Close()

	// 生成测试数据
	data := make([]byte, 1<<20) // 1MB 数据
	rand.Read(data)

	// 测试不同压缩算法
	compressors := []struct {
		name       string
		level      compression.CompressionLevel
		compressor compression.Compressor
	}{
		{"gzip-default", compression.CompressionLevelDefault, mustNewCompressor(compression.CompressionTypeGzip, compression.CompressionLevelDefault)},
		{"gzip-speed", compression.CompressionLevelBestSpeed, mustNewCompressor(compression.CompressionTypeGzip, compression.CompressionLevelBestSpeed)},
		{"gzip-compression", compression.CompressionLevelBestCompression, mustNewCompressor(compression.CompressionTypeGzip, compression.CompressionLevelBestCompression)},
		{"snappy", compression.CompressionLevelDefault, mustNewCompressor(compression.CompressionTypeSnappy, compression.CompressionLevelDefault)},
		{"lz4", compression.CompressionLevelDefault, mustNewCompressor(compression.CompressionTypeLZ4, compression.CompressionLevelDefault)},
		{"zstd-default", compression.CompressionLevelDefault, mustNewCompressor(compression.CompressionTypeZstd, compression.CompressionLevelDefault)},
		{"zstd-speed", compression.CompressionLevelBestSpeed, mustNewCompressor(compression.CompressionTypeZstd, compression.CompressionLevelBestSpeed)},
		{"zstd-compression", compression.CompressionLevelBestCompression, mustNewCompressor(compression.CompressionTypeZstd, compression.CompressionLevelBestCompression)},
	}

	log.Printf("压缩基准测试 - 数据大小: %d 字节\n", len(data))
	log.Println("算法\t压缩级别\t压缩率\t压缩时间\t解压时间")
	log.Println("------------------------------------------------------------")

	for _, c := range compressors {
		b.Run(fmt.Sprintf("%s-%d", c.name, c.level), func(b *testing.B) {
			var compressedSize int
			var compressTime, decompressTime time.Duration

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// 测试压缩性能
				start := time.Now()
				compressed, err := c.compressor.Compress(data)
				if err != nil {
					b.Fatalf("压缩失败: %v", err)
				}
				compressTime += time.Since(start)
				compressedSize = len(compressed)

				// 测试解压性能
				start = time.Now()
				_, err = c.compressor.Decompress(compressed)
				if err != nil {
					b.Fatalf("解压失败: %v", err)
				}
				decompressTime += time.Since(start)
			}

			// 计算平均值
			avgCompressTime := compressTime / time.Duration(b.N)
			avgDecompressTime := decompressTime / time.Duration(b.N)
			compressionRatio := float64(compressedSize) / float64(len(data)) * 100

			log.Printf("%s\t%d\t%.2f%%\t%v\t%v\n",
				c.name, c.level, compressionRatio, avgCompressTime, avgDecompressTime)
		})
	}
}

// mustNewCompressor 创建一个新的压缩器，如果失败则panic
func mustNewCompressor(typ compression.CompressionType, level compression.CompressionLevel) compression.Compressor {
	options := compression.CompressionOptions{
		Type:  typ,
		Level: level,
	}
	comp, err := compression.NewCompressor(options)
	if err != nil {
		panic(err)
	}
	return comp
}

// 基准测试：分布式缓存性能
func BenchmarkDistributedCache(b *testing.B) {
	// 注意：这个测试需要先启动 etcd 服务和多个缓存节点
	// 此处仅提供框架，实际运行需要在环境中执行

	const (
		cacheAddr1 = "http://localhost:8001"
		cacheAddr2 = "http://localhost:8002"
		apiAddr    = "http://localhost:9999"
	)

	logFile := setupLogFile()
	defer logFile.Close()

	// 等待缓存服务启动
	log.Println("等待分布式缓存服务启动...")
	time.Sleep(2 * time.Second)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 准备数据并注册一个新节点
	testData := generateTestData(500)
	go createNewNode(testData, 8009)
	// 等待节点注册完成
	time.Sleep(5 * time.Second)
	// 准备测试
	keys := make([]string, 0, len(testData))
	for key := range testData {
		keys = append(keys, key)
	}

	// 并发测试分布式缓存
	var wg sync.WaitGroup
	startTime := time.Now()
	requestCount := 1000
	successCount := 0
	hitCount := 0
	var mutex sync.Mutex

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := keys[idx%len(keys)]
			url := fmt.Sprintf("%s/api?key=%s", apiAddr, key)

			resp, err := client.Get(url)
			if err != nil {
				log.Printf("请求失败: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err == nil && string(body) == testData[key] {
					mutex.Lock()
					hitCount++
					mutex.Unlock()
				}
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 计算统计信息
	successRate := float64(successCount) / float64(requestCount) * 100
	hitRate := float64(hitCount) / float64(requestCount) * 100
	throughput := float64(requestCount) / duration.Seconds()

	log.Printf("分布式缓存测试完成:\n")
	log.Printf("总请求数: %d\n", requestCount)
	log.Printf("成功请求数: %d\n", successCount)
	log.Printf("成功率: %.2f%%\n", successRate)
	log.Printf("缓存命中数: %d\n", hitCount)
	log.Printf("缓存命中率: %.2f%%\n", hitRate)
	log.Printf("总耗时: %v\n", duration)
	log.Printf("吞吐量: %.2f 请求/秒\n", throughput)
	time.Sleep(5 * time.Second)
}
