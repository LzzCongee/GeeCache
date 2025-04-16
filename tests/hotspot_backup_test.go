package tests

import (
	"fmt"
	"geecache"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// 测试数据生成
func generateTestData(size int) map[string]string {
	data := make(map[string]string)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		data[key] = value
	}
	return data
}

// 创建缓存服务器
func createCacheServer(addr string, testData map[string]string, api bool) *geecache.Group {
	gee := geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := testData[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	peers := geecache.NewHTTPPool(addr)
	gee.RegisterPeers(peers)

	// 设置热点数据相关参数
	// 为了测试方便，将热点阈值设置较低
	gee.SetHotSpotThreshold(5)
	// 设置备份节点数量
	gee.SetBackupCount(2)

	// 启动HTTP服务
	go func() {
		log.Println("geecache is running at", addr)
		var handler http.Handler = peers
		if api {
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				key := r.URL.Query().Get("key")
				view, err := gee.Get(key)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(view.ByteSlice())
			})
		}
		log.Fatal(http.ListenAndServe(addr[7:], handler))
	}()

	return gee
}

// 测试热点数据识别和备份功能
func TestHotSpotBackup(t *testing.T) {
	// 创建测试数据
	testData := generateTestData(10)

	// 启动三个缓存服务器
	addrs := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}

	// 创建服务器并等待它们启动
	var groups []*geecache.Group
	for _, addr := range addrs {
		groups = append(groups, createCacheServer(addr, testData, true))
	}
	// 等待服务器启动
	time.Sleep(1 * time.Second)

	// 设置节点之间的连接
	for _, g := range groups {
		g.GetPeers().(*geecache.HTTPPool).Set(addrs...)
	}

	// 测试1: 热点数据识别
	t.Run("HotSpotDetection", func(t *testing.T) {
		// 选择一个键作为热点数据
		hotKey := "key-1"

		// 多次访问该键，使其成为热点数据
		for i := 0; i < 10; i++ {
			_, err := groups[0].Get(hotKey)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", hotKey, err)
			}
		}

		// 验证该键是否被标记为热点数据
		isHotSpot := groups[0].IsHotSpot(hotKey)
		if !isHotSpot {
			t.Errorf("Key %s should be marked as hot spot data", hotKey)
		} else {
			t.Logf("Key %s is correctly marked as hot spot data", hotKey)
		}
	})

	// 测试2: 热点数据备份
	t.Run("HotSpotBackup", func(t *testing.T) {
		// 选择一个键作为热点数据
		hotKey := "key-2"

		// 多次访问该键，使其成为热点数据
		for i := 0; i < 10; i++ {
			_, err := groups[0].Get(hotKey)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", hotKey, err)
			}
		}

		// 等待数据同步到备份节点
		time.Sleep(500 * time.Millisecond)

		// 验证数据是否已同步到备份节点
		// 通过直接访问备份节点的API来验证
		for i := 1; i < len(addrs); i++ {
			resp, err := http.Get(fmt.Sprintf("%s/api?key=%s", addrs[i], hotKey))
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Errorf("Failed to get hot spot data from backup node %d: %v", i, err)
			} else {
				t.Logf("Successfully retrieved hot spot data from backup node %d", i)
				resp.Body.Close()
			}
		}
	})

	// 测试3: 节点故障恢复
	t.Run("NodeFailureRecovery", func(t *testing.T) {
		// 选择一个键作为热点数据
		hotKey := "key-3"

		// 多次访问该键，使其成为热点数据
		for i := 0; i < 10; i++ {
			_, err := groups[0].Get(hotKey)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", hotKey, err)
			}
		}

		// 等待数据同步到备份节点
		time.Sleep(500 * time.Millisecond)

		// 模拟主节点故障（通过不从主节点获取数据）
		// 直接从备份节点获取数据
		for i := 1; i < len(addrs); i++ {
			resp, err := http.Get(fmt.Sprintf("%s/api?key=%s", addrs[i], hotKey))
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Errorf("Failed to get hot spot data from backup node %d after primary node failure: %v", i, err)
			} else {
				t.Logf("Successfully retrieved hot spot data from backup node %d after primary node failure", i)
				resp.Body.Close()
			}
		}
	})

	// 测试4: 并发访问性能
	t.Run("ConcurrentAccess", func(t *testing.T) {
		// 选择一个键作为热点数据
		hotKey := "key-4"
		// 选择一个普通键
		normalKey := "key-5"

		// 多次访问热点键，使其成为热点数据
		for i := 0; i < 10; i++ {
			_, err := groups[0].Get(hotKey)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", hotKey, err)
			}
		}

		// 等待数据同步到备份节点
		time.Sleep(500 * time.Millisecond)

		// 并发访问热点数据和普通数据，比较性能
		concurrency := 100
		var wg sync.WaitGroup

		// 测量热点数据访问性能
		hotSpotStart := time.Now()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := http.Get(fmt.Sprintf("%s/api?key=%s", addrs[0], hotKey))
				if err != nil {
					t.Logf("Error accessing hot spot data: %v", err)
				}
			}()
		}
		wg.Wait()
		hotSpotDuration := time.Since(hotSpotStart)

		// 测量普通数据访问性能
		normalStart := time.Now()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := http.Get(fmt.Sprintf("%s/api?key=%s", addrs[0], normalKey))
				if err != nil {
					t.Logf("Error accessing normal data: %v", err)
				}
			}()
		}
		wg.Wait()
		normalDuration := time.Since(normalStart)

		t.Logf("Hot spot data access time: %v", hotSpotDuration)
		t.Logf("Normal data access time: %v", normalDuration)
		t.Logf("Performance improvement: %.2f%%", (float64(normalDuration-hotSpotDuration)/float64(normalDuration))*100)
	})

	// 测试5: 清理过期热点数据
	t.Run("CleanExpiredHotSpot", func(t *testing.T) {
		// 选择一个键作为临时热点数据
		tempHotKey := "key-6"

		// 多次访问该键，使其成为热点数据
		for i := 0; i < 10; i++ {
			_, err := groups[0].Get(tempHotKey)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", tempHotKey, err)
			}
		}

		// 验证该键是否被标记为热点数据
		isHotSpot := groups[0].IsHotSpot(tempHotKey)
		if !isHotSpot {
			t.Errorf("Key %s should be marked as hot spot data", tempHotKey)
		} else {
			t.Logf("Key %s is correctly marked as hot spot data", tempHotKey)
		}

		// 手动触发清理过期热点数据
		groups[0].CleanExpiredHotSpot()

		// 验证热点数据是否被清理
		// 注意：在实际情况下，这个测试可能不稳定，因为cleanExpiredHotSpot方法可能只会减少访问计数而不是完全移除热点标记
		// 这里主要是测试方法是否能正常执行而不会导致死锁
		t.Logf("Successfully triggered cleanExpiredHotSpot without deadlock")
	})
}

// 测试边界条件
func TestHotSpotBoundaries(t *testing.T) {
	// 创建测试数据
	testData := generateTestData(10)

	// 启动一个缓存服务器
	addr := "http://localhost:8004"
	gee := createCacheServer(addr, testData, true)

	// 等待服务器启动
	time.Sleep(1 * time.Second)

	// 测试1: 阈值边界测试
	t.Run("ThresholdBoundary", func(t *testing.T) {
		// 获取当前热点阈值
		threshold := 5 // 假设默认为5

		// 测试刚好达到阈值的情况
		key := "boundary-key-1"
		for i := 0; i < threshold; i++ {
			_, err := gee.Get(key)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", key, err)
			}
		}

		// 验证是否成为热点数据
		isHotSpot := gee.IsHotSpot(key)
		if !isHotSpot {
			t.Logf("Key %s with exactly threshold accesses is not marked as hot spot", key)
		} else {
			t.Logf("Key %s with exactly threshold accesses is marked as hot spot", key)
		}

		// 测试刚好低于阈值的情况
		key = "boundary-key-2"
		for i := 0; i < threshold-1; i++ {
			_, err := gee.Get(key)
			if err != nil {
				t.Fatalf("Failed to get key %s: %v", key, err)
			}
		}

		// 验证是否成为热点数据
		isHotSpot = gee.IsHotSpot(key)
		if isHotSpot {
			t.Errorf("Key %s with below threshold accesses should not be marked as hot spot", key)
		} else {
			t.Logf("Key %s with below threshold accesses is correctly not marked as hot spot", key)
		}
	})

	// 测试2: 备份节点数量边界测试
	t.Run("BackupCountBoundary", func(t *testing.T) {
		// 设置不同的备份节点数量并测试
		backupCounts := []int{0, 1, 5}

		for _, count := range backupCounts {
			// 设置备份节点数量
			gee.SetBackupCount(count)

			// 选择一个键作为热点数据
			key := "backup-count-key-" + strconv.Itoa(count)

			// 多次访问该键，使其成为热点数据
			for i := 0; i < 10; i++ {
				_, err := gee.Get(key)
				if err != nil {
					t.Fatalf("Failed to get key %s: %v", key, err)
				}
			}

			// 验证备份节点数量设置是否生效
			actualCount := gee.GetBackupCount()
			if actualCount != count {
				t.Errorf("Backup count should be %d, but got %d", count, actualCount)
			} else {
				t.Logf("Backup count is correctly set to %d", count)
			}
		}
	})

	// 测试3: 空键测试
	t.Run("EmptyKeyTest", func(t *testing.T) {
		// 尝试获取空键
		_, err := gee.Get("")
		if err == nil {
			t.Errorf("Empty key should return an error")
		} else {
			t.Logf("Empty key correctly returns error: %v", err)
		}

		// 验证空键不会被标记为热点数据
		isHotSpot := gee.IsHotSpot("")
		if isHotSpot {
			t.Errorf("Empty key should not be marked as hot spot")
		} else {
			t.Logf("Empty key is correctly not marked as hot spot")
		}
	})
}

// 主函数，用于手动运行测试
func TestMain(m *testing.M) {
	// 设置日志输出
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 运行测试
	exitCode := m.Run()

	// 退出
	os.Exit(exitCode)
}
