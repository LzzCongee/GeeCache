package storage

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// 检查是否启用了C++测试
func isCppTestEnabled() bool {
	return os.Getenv("ENABLE_CPP_TEST") == "1"
}

// 如果C++测试未启用，则跳过测试
func skipIfCppTestDisabled(t *testing.T) {
	if !isCppTestEnabled() {
		t.Skip("Skipping C++ test. Set ENABLE_CPP_TEST=1 to run this test.")
	}
}

func TestMemoryStorage(t *testing.T) {
	// 这是Go内存存储引擎的测试，不需要跳过
	// 创建内存存储引擎
	storage, err := NewStorage(StorageTypeMemory, StorageOptions{
		MaxSize: 1024 * 1024, // 1MB
	})
	if err != nil {
		t.Fatalf("Create memory storage error: %v", err)
	}
	defer storage.Close()

	// 测试Set和Get
	key := "test_key"
	value := []byte("test_value")
	err = storage.Set(key, value)
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}

	getValue, err := storage.Get(key)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if string(getValue) != string(value) {
		t.Errorf("Get value mismatch: got %s, want %s", string(getValue), string(value))
	}

	// 测试Has
	has, err := storage.Has(key)
	if err != nil {
		t.Fatalf("Has error: %v", err)
	}
	if !has {
		t.Errorf("Has should return true")
	}

	// 测试Keys
	keys, err := storage.Keys()
	if err != nil {
		t.Fatalf("Keys error: %v", err)
	}
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("Keys mismatch: got %v, want [%s]", keys, key)
	}

	// 测试Delete
	err = storage.Delete(key)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	has, err = storage.Has(key)
	if err != nil {
		t.Fatalf("Has error after delete: %v", err)
	}
	if has {
		t.Errorf("Has should return false after delete")
	}

	// 测试SetWithExpire
	err = storage.SetWithExpire(key, value, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SetWithExpire error: %v", err)
	}

	has, err = storage.Has(key)
	if err != nil {
		t.Fatalf("Has error after SetWithExpire: %v", err)
	}
	if !has {
		t.Errorf("Has should return true after SetWithExpire")
	}

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	has, err = storage.Has(key)
	if err != nil {
		t.Fatalf("Has error after expire: %v", err)
	}
	if has {
		t.Errorf("Has should return false after expire")
	}

	// 测试Clear
	err = storage.Set(key, value)
	if err != nil {
		t.Fatalf("Set error before clear: %v", err)
	}

	err = storage.Clear()
	if err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	has, err = storage.Has(key)
	if err != nil {
		t.Fatalf("Has error after clear: %v", err)
	}
	if has {
		t.Errorf("Has should return false after clear")
	}
}

// 测试内存存储引擎的边界条件
func TestMemoryStorageEdgeCases(t *testing.T) {
	// 这是Go内存存储引擎的测试，不需要跳过
	// 创建一个小容量的内存存储引擎
	storage, err := NewStorage(StorageTypeMemory, StorageOptions{
		MaxSize: 100, // 只允许存储100字节
	})
	if err != nil {
		t.Fatalf("Create memory storage error: %v", err)
	}
	defer storage.Close()

	// 测试存储容量限制
	key1 := "key1"
	value1 := []byte("value1") // 6字节
	err = storage.Set(key1, value1)
	if err != nil {
		t.Fatalf("Set key1 error: %v", err)
	}

	// 添加一个大值，应该成功但会导致之前的键被删除
	key2 := "key2"
	value2 := make([]byte, 90) // 90字节的值
	for i := range value2 {
		value2[i] = byte(i % 256)
	}

	err = storage.Set(key2, value2)
	if err != nil {
		t.Fatalf("Set key2 error: %v", err)
	}

	// 检查key1是否还存在（应该不存在，因为容量限制）
	_, err = storage.Get(key1)
	if err == nil {
		t.Errorf("key1 should be evicted due to size limit")
	}

	// 测试获取不存在的键
	_, err = storage.Get("non_existent_key")
	if err == nil {
		t.Errorf("Get non-existent key should return error")
	}

	// 测试删除不存在的键
	err = storage.Delete("non_existent_key")
	if err != nil {
		t.Errorf("Delete non-existent key should not return error: %v", err)
	}

	// 测试零长度值
	key3 := "key3"
	value3 := []byte{}
	err = storage.Set(key3, value3)
	if err != nil {
		t.Fatalf("Set empty value error: %v", err)
	}

	getValue3, err := storage.Get(key3)
	if err != nil {
		t.Fatalf("Get empty value error: %v", err)
	}
	if len(getValue3) != 0 {
		t.Errorf("Empty value should have zero length")
	}
}

// 测试内存存储引擎的并发安全性
func TestMemoryStorageConcurrency(t *testing.T) {
	// 这是Go内存存储引擎的测试，不需要跳过
	storage, err := NewStorage(StorageTypeMemory, StorageOptions{
		MaxSize: 10 * 1024 * 1024, // 10MB
	})
	if err != nil {
		t.Fatalf("Create memory storage error: %v", err)
	}
	defer storage.Close()

	// 并发写入
	const goroutines = 10
	const keysPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine; i++ {
				key := fmt.Sprintf("key_%d_%d", gid, i)
				value := []byte(fmt.Sprintf("value_%d_%d", gid, i))

				// 设置值
				err := storage.Set(key, value)
				if err != nil {
					t.Errorf("Concurrent Set error: %v", err)
					return
				}

				// 立即读取
				getValue, err := storage.Get(key)
				if err != nil {
					t.Errorf("Concurrent Get error: %v", err)
					return
				}
				if string(getValue) != string(value) {
					t.Errorf("Concurrent Get value mismatch: got %s, want %s", string(getValue), string(value))
					return
				}
			}
		}(g)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 验证所有键都存在
	for g := 0; g < goroutines; g++ {
		for i := 0; i < keysPerGoroutine; i++ {
			key := fmt.Sprintf("key_%d_%d", g, i)
			expectedValue := fmt.Sprintf("value_%d_%d", g, i)

			getValue, err := storage.Get(key)
			if err != nil {
				t.Errorf("Get after concurrency test error: %v", err)
				continue
			}
			if string(getValue) != expectedValue {
				t.Errorf("Get after concurrency test value mismatch: got %s, want %s", string(getValue), expectedValue)
			}
		}
	}

	// 测试并发删除
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine; i++ {
				key := fmt.Sprintf("key_%d_%d", gid, i)
				err := storage.Delete(key)
				if err != nil {
					t.Errorf("Concurrent Delete error: %v", err)
				}
			}
		}(g)
	}

	wg.Wait()

	// 验证所有键都已删除
	keys, err := storage.Keys()
	if err != nil {
		t.Fatalf("Keys error after concurrency test: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after deletion, got %d", len(keys))
	}
}

// 测试内存存储引擎的性能
func BenchmarkMemoryStorage(b *testing.B) {
	// 这是Go内存存储引擎的基准测试，不需要跳过
	storage, err := NewStorage(StorageTypeMemory, StorageOptions{
		MaxSize: 100 * 1024 * 1024, // 100MB
	})
	if err != nil {
		b.Fatalf("Create memory storage error: %v", err)
	}
	defer storage.Close()

	// 准备测试数据
	keys := make([]string, 1000)
	values := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("bench_key_%d", i)
		values[i] = []byte(fmt.Sprintf("bench_value_%d", i))
	}

	// 基准测试：Set操作
	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			index := i % 1000
			err := storage.Set(keys[index], values[index])
			if err != nil {
				b.Fatalf("Set error: %v", err)
			}
		}
	})

	// 预先填充数据
	for i := 0; i < 1000; i++ {
		storage.Set(keys[i], values[i])
	}

	// 基准测试：Get操作
	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			index := i % 1000
			_, err := storage.Get(keys[index])
			if err != nil {
				b.Fatalf("Get error: %v", err)
			}
		}
	})

	// 基准测试：Delete操作
	b.Run("Delete", func(b *testing.B) {
		// 重新填充数据
		for i := 0; i < 1000; i++ {
			storage.Set(keys[i], values[i])
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			index := i % 1000
			storage.Delete(keys[index])
		}
	})

	// 基准测试：SetWithExpire操作
	b.Run("SetWithExpire", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			index := i % 1000
			err := storage.SetWithExpire(keys[index], values[index], time.Hour)
			if err != nil {
				b.Fatalf("SetWithExpire error: %v", err)
			}
		}
	})
}
