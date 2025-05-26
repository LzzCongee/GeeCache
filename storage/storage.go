package storage

import (
	"fmt"
	"time"
)

// Storage 存储引擎接口
type Storage interface {
	// Get 获取指定键的值
	Get(key string) ([]byte, error)
	// Set 设置指定键的值
	Set(key string, value []byte) error
	// SetWithExpire 设置指定键的值，并指定过期时间
	SetWithExpire(key string, value []byte, expire time.Duration) error
	// Delete 删除指定键
	Delete(key string) error
	// Has 判断指定键是否存在
	Has(key string) (bool, error)
	// Keys 获取所有键
	Keys() ([]string, error)
	// Clear 清空存储
	Clear() error
	// Close 关闭存储
	Close() error
}

// StorageType 存储引擎类型
type StorageType string

const (
	// StorageTypeMemory 内存存储
	StorageTypeMemory StorageType = "memory"
	// StorageTypeLevelDB LevelDB存储
	StorageTypeLevelDB StorageType = "leveldb"
	// StorageTypeRocksDB RocksDB存储
	StorageTypeRocksDB StorageType = "rocksdb"
	// StorageTypeBadger Badger存储
	StorageTypeBadger StorageType = "badger"
	// StorageTypeMemory C++内存存储 - 跳表结构
	StorageTypeCppSkipList StorageType = "skiplist"
	// StorageTypeCppMemory C++内存存储
	StorageTypeCppMemory StorageType = "cpp_memory"
	// StorageTypeCppLevelDB C++ LevelDB存储
	StorageTypeCppLevelDB StorageType = "cpp_leveldb"
	// StorageTypeCppRocksDB C++ RocksDB存储
	StorageTypeCppRocksDB StorageType = "cpp_rocksdb"
)

// StorageOptions 存储引擎选项
type StorageOptions struct {
	// Path 存储路径
	Path string
	// MaxSize 最大存储大小
	MaxSize int64
	// Compression 是否启用压缩
	Compression bool
}

// NewStorage 创建一个新的存储引擎
func NewStorage(storageType StorageType, options StorageOptions) (Storage, error) {
	switch storageType {
	case StorageTypeMemory:
		return NewMemoryStorage(options)
	case StorageTypeCppSkipList:
		return NewCppSkipListStorage(options)
	case StorageTypeCppMemory:
		// return NewCppMemoryStorage(options)
		return nil, fmt.Errorf("storage type %s not implemented yet", storageType)
	case StorageTypeCppLevelDB:
		// return NewCppLevelDBStorage(options)
		return nil, fmt.Errorf("storage type %s not implemented yet", storageType)
	case StorageTypeCppRocksDB:
		// return NewCppRocksDBStorage(options)
		return nil, fmt.Errorf("storage type %s not implemented yet", storageType)
	case StorageTypeLevelDB, StorageTypeRocksDB, StorageTypeBadger:
		return nil, fmt.Errorf("storage type %s not implemented yet", storageType)
	default:
		return NewMemoryStorage(options)
	}
}
