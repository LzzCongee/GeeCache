package storage

import (
	"testing"
	"time"
)

func TestCppMemoryStorage(t *testing.T) {
	// 跳过测试，除非明确指定要运行C++测试
	t.Skip("Skipping C++ test. Set ENABLE_CPP_TEST=1 to run this test.")

	// 创建C++内存存储引擎
	storage, err := NewStorage(StorageTypeCppMemory, StorageOptions{
		MaxSize: 1024 * 1024, // 1MB
	})
	if err != nil {
		t.Fatalf("Create C++ memory storage error: %v", err)
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
