package storage

/*
#cgo CXXFLAGS: -std=c++11
#cgo LDFLAGS: -lstdc++
#include <stdlib.h>
#include "cpp/storage_wrapper.h"
*/
import "C"
import (
	"errors"
	"runtime"
	"unsafe"
)

// CppStorage C++存储引擎
type CppStorage struct {
	handle C.storage_t
}

// NewCppMemoryStorage 创建一个新的C++内存存储引擎
func NewCppMemoryStorage(options StorageOptions) (*CppStorage, error) {
	handle := C.storage_create_memory(C.longlong(options.MaxSize))
	if handle == nil {
		return nil, errors.New("failed to create memory storage")
	}

	storage := &CppStorage{handle: handle}
	runtime.SetFinalizer(storage, (*CppStorage).Close)
	return storage, nil
}

// NewCppLevelDBStorage 创建一个新的C++ LevelDB存储引擎
func NewCppLevelDBStorage(options StorageOptions) (*CppStorage, error) {
	cPath := C.CString(options.Path)
	defer C.free(unsafe.Pointer(cPath))

	compression := 0
	if options.Compression {
		compression = 1
	}

	handle := C.storage_create_leveldb(cPath, C.longlong(options.MaxSize), C.int(compression))
	if handle == nil {
		return nil, errors.New("failed to create leveldb storage")
	}

	storage := &CppStorage{handle: handle}
	runtime.SetFinalizer(storage, (*CppStorage).Close)
	return storage, nil
}

// NewCppRocksDBStorage 创建一个新的C++ RocksDB存储引擎
func NewCppRocksDBStorage(options StorageOptions) (*CppStorage, error) {
	cPath := C.CString(options.Path)
	defer C.free(unsafe.Pointer(cPath))

	compression := 0
	if options.Compression {
		compression = 1
	}

	handle := C.storage_create_rocksdb(cPath, C.longlong(options.MaxSize), C.int(compression))
	if handle == nil {
		return nil, errors.New("failed to create rocksdb storage")
	}

	storage := &CppStorage{handle: handle}
	runtime.SetFinalizer(storage, (*CppStorage).Close)
	return storage, nil
}

// Get 获取指定键的值
func (s *CppStorage) Get(key string) ([]byte, error) {
	if s.handle == nil {
		return nil, errors.New("storage is closed")
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var valueLen C.int
	cValue := C.storage_get(s.handle, cKey, C.int(len(key)), &valueLen)
	if cValue == nil {
		return nil, errors.New("key not found")
	}
	defer C.free(unsafe.Pointer(cValue))

	value := C.GoBytes(unsafe.Pointer(cValue), valueLen)
	return value, nil
}

// Set 设置指定键的值
func (s *CppStorage) Set(key string, value []byte) error {
	if s.handle == nil {
		return errors.New("storage is closed")
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	cValue := C.CBytes(value)
	defer C.free(unsafe.Pointer(cValue))

	result := C.storage_set(s.handle, cKey, C.int(len(key)), (*C.char)(cValue), C.int(len(value)))
	if result == 0 {
		return errors.New("failed to set value")
	}

	return nil
}

// SetWithExpire 设置指定键的值，并指定过期时间
func (s *CppStorage) SetWithExpire(key string, value []byte, expire int64) error {
	if s.handle == nil {
		return errors.New("storage is closed")
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	cValue := C.CBytes(value)
	defer C.free(unsafe.Pointer(cValue))

	result := C.storage_set_with_expire(s.handle, cKey, C.int(len(key)), (*C.char)(cValue), C.int(len(value)), C.longlong(expire))
	if result == 0 {
		return errors.New("failed to set value with expire")
	}

	return nil
}

// Delete 删除指定键
func (s *CppStorage) Delete(key string) error {
	if s.handle == nil {
		return errors.New("storage is closed")
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	result := C.storage_delete(s.handle, cKey, C.int(len(key)))
	if result == 0 {
		return errors.New("failed to delete key")
	}

	return nil
}

// Has 判断指定键是否存在
func (s *CppStorage) Has(key string) (bool, error) {
	if s.handle == nil {
		return false, errors.New("storage is closed")
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	result := C.storage_has(s.handle, cKey, C.int(len(key)))
	return result != 0, nil
}

// Keys 获取所有键
func (s *CppStorage) Keys() ([]string, error) {
	if s.handle == nil {
		return nil, errors.New("storage is closed")
	}

	var keysLen C.int
	cKeys := C.storage_keys(s.handle, &keysLen)
	if cKeys == nil {
		return []string{}, nil
	}
	defer C.storage_free_keys(cKeys, keysLen)

	keys := make([]string, int(keysLen))
	for i := 0; i < int(keysLen); i++ {
		cKey := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cKeys)) + uintptr(i)*unsafe.Sizeof(cKeys)))
		keys[i] = C.GoString(cKey)
	}

	return keys, nil
}

// Clear 清空存储
func (s *CppStorage) Clear() error {
	if s.handle == nil {
		return errors.New("storage is closed")
	}

	result := C.storage_clear(s.handle)
	if result == 0 {
		return errors.New("failed to clear storage")
	}

	return nil
}

// Close 关闭存储
func (s *CppStorage) Close() error {
	if s.handle == nil {
		return nil
	}

	result := C.storage_close(s.handle)
	C.storage_free(s.handle)
	s.handle = nil

	if result == 0 {
		return errors.New("failed to close storage")
	}

	return nil
}
