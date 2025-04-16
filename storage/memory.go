package storage

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MemoryStorage 内存存储引擎
type MemoryStorage struct {
	data     map[string][]byte
	expiries map[string]time.Time
	maxSize  int64
	size     int64
	mu       sync.RWMutex

	// 清理相关
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupRunning  bool
}

// NewMemoryStorage 创建一个新的内存存储引擎
func NewMemoryStorage(options StorageOptions) (*MemoryStorage, error) {
	storage := &MemoryStorage{
		data:            make(map[string][]byte),
		expiries:        make(map[string]time.Time),
		maxSize:         options.MaxSize,
		cleanupInterval: 5 * time.Minute, // 默认5分钟清理一次过期数据
		stopCleanup:     make(chan struct{}),
	}

	// 启动自动清理过期数据的协程
	storage.startCleanupTimer()

	return storage, nil
}

// Get 获取指定键的值
func (s *MemoryStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查是否过期
	if expiry, ok := s.expiries[key]; ok && time.Now().After(expiry) {
		delete(s.data, key)
		delete(s.expiries, key)
		return nil, fmt.Errorf("key %s not found", key)
	}

	value, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found", key)
	}

	return value, nil
}

// Set 设置指定键的值
func (s *MemoryStorage) Set(key string, value []byte) error {
	return s.SetWithExpire(key, value, 0)
}

// SetWithExpire 设置指定键的值，并指定过期时间
func (s *MemoryStorage) SetWithExpire(key string, value []byte, expire time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查存储大小
	if s.maxSize > 0 {
		// 如果已存在，先减去旧值的大小
		if oldValue, ok := s.data[key]; ok {
			s.size -= int64(len(key) + len(oldValue))
		}

		// 计算新值的大小
		newSize := s.size + int64(len(key)+len(value))
		if newSize > s.maxSize {
			return fmt.Errorf("storage size limit exceeded")
		}
		s.size = newSize
	}

	// 设置值
	s.data[key] = value

	// 设置过期时间
	if expire > 0 {
		s.expiries[key] = time.Now().Add(expire)
	} else {
		delete(s.expiries, key)
	}

	return nil
}

// Delete 删除指定键
func (s *MemoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if oldValue, ok := s.data[key]; ok {
		s.size -= int64(len(key) + len(oldValue))
		delete(s.data, key)
		delete(s.expiries, key)
	}

	return nil
}

// Has 判断指定键是否存在
func (s *MemoryStorage) Has(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查是否过期
	if expiry, ok := s.expiries[key]; ok && time.Now().After(expiry) {
		delete(s.data, key)
		delete(s.expiries, key)
		return false, nil
	}

	_, ok := s.data[key]
	return ok, nil
}

// Keys 获取所有键
func (s *MemoryStorage) Keys() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		// 检查是否过期
		if expiry, ok := s.expiries[key]; ok && now.After(expiry) {
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// Clear 清空存储
func (s *MemoryStorage) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string][]byte)
	s.expiries = make(map[string]time.Time)
	s.size = 0

	return nil
}

// Close 关闭存储
func (s *MemoryStorage) Close() error {
	// 停止清理协程
	s.stopCleanupTimer()
	return s.Clear()
}

// startCleanupTimer 启动定时清理过期数据的定时器
func (s *MemoryStorage) startCleanupTimer() {
	if s.cleanupInterval <= 0 {
		return
	}

	s.mu.Lock()
	if s.cleanupRunning {
		s.mu.Unlock()
		return
	}
	s.cleanupRunning = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanExpired()
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// stopCleanupTimer 停止清理定时器
func (s *MemoryStorage) stopCleanupTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cleanupRunning {
		s.stopCleanup <- struct{}{}
		s.cleanupRunning = false
	}
}

// cleanExpired 清理过期数据
func (s *MemoryStorage) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expiredCount := 0
	expiredSize := int64(0)

	for key, expiry := range s.expiries {
		if now.After(expiry) {
			if value, ok := s.data[key]; ok {
				expiredSize += int64(len(key) + len(value))
				delete(s.data, key)
				expiredCount++
			}
			delete(s.expiries, key)
		}
	}

	// 更新存储大小
	if expiredCount > 0 {
		s.size -= expiredSize
		log.Printf("[MemoryStorage] Cleaned %d expired items, freed %d bytes", expiredCount, expiredSize)
	}
}
