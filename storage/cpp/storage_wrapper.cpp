#include "storage_wrapper.h"
#include "storage.h"
#include <string>
#include <vector>
#include <memory>
#include <cstring>

using namespace geecache::storage;

// 存储引擎句柄
struct storage_handle {
    std::shared_ptr<Storage> storage;
    std::string last_error;
};

// 创建内存存储引擎
storage_t storage_create_memory(long long max_size) {
    try {
        auto storage = StorageFactory::CreateMemoryStorage(max_size);
        auto handle = new storage_handle{storage, ""};
        return handle;
    } catch (const std::exception& e) {
        auto handle = new storage_handle{nullptr, e.what()};
        return handle;
    }
}

// 创建LevelDB存储引擎
storage_t storage_create_leveldb(const char* path, long long max_size, int compression) {
    try {
        auto storage = StorageFactory::CreateLevelDBStorage(path, max_size, compression != 0);
        auto handle = new storage_handle{storage, ""};
        return handle;
    } catch (const std::exception& e) {
        auto handle = new storage_handle{nullptr, e.what()};
        return handle;
    }
}

// 创建RocksDB存储引擎
storage_t storage_create_rocksdb(const char* path, long long max_size, int compression) {
    try {
        auto storage = StorageFactory::CreateRocksDBStorage(path, max_size, compression != 0);
        auto handle = new storage_handle{storage, ""};
        return handle;
    } catch (const std::exception& e) {
        auto handle = new storage_handle{nullptr, e.what()};
        return handle;
    }
}

// 获取指定键的值
const char* storage_get(storage_t storage, const char* key, int key_len, int* value_len) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return nullptr;
    }

    try {
        std::string k(key, key_len);
        std::string v = handle->storage->Get(k);
        *value_len = static_cast<int>(v.size());
        char* result = new char[v.size()];
        std::memcpy(result, v.data(), v.size());
        return result;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return nullptr;
    }
}

// 设置指定键的值
int storage_set(storage_t storage, const char* key, int key_len, const char* value, int value_len) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return 0;
    }

    try {
        std::string k(key, key_len);
        std::string v(value, value_len);
        return handle->storage->Set(k, v) ? 1 : 0;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return 0;
    }
}

// 设置指定键的值，并指定过期时间
int storage_set_with_expire(storage_t storage, const char* key, int key_len, const char* value, int value_len, long long expire) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return 0;
    }

    try {
        std::string k(key, key_len);
        std::string v(value, value_len);
        return handle->storage->SetWithExpire(k, v, expire) ? 1 : 0;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return 0;
    }
}

// 删除指定键
int storage_delete(storage_t storage, const char* key, int key_len) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return 0;
    }

    try {
        std::string k(key, key_len);
        return handle->storage->Delete(k) ? 1 : 0;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return 0;
    }
}

// 判断指定键是否存在
int storage_has(storage_t storage, const char* key, int key_len) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return 0;
    }

    try {
        std::string k(key, key_len);
        return handle->storage->Has(k) ? 1 : 0;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return 0;
    }
}

// 获取所有键
const char** storage_keys(storage_t storage, int* keys_len) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        *keys_len = 0;
        return nullptr;
    }

    try {
        std::vector<std::string> keys = handle->storage->Keys();
        *keys_len = static_cast<int>(keys.size());
        if (keys.empty()) {
            return nullptr;
        }

        const char** result = new const char*[keys.size()];
        for (size_t i = 0; i < keys.size(); ++i) {
            char* key = new char[keys[i].size()];
            std::memcpy(key, keys[i].data(), keys[i].size());
            result[i] = key;
        }
        return result;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        *keys_len = 0;
        return nullptr;
    }
}

// 释放键列表
void storage_free_keys(const char** keys, int keys_len) {
    if (!keys) {
        return;
    }

    for (int i = 0; i < keys_len; ++i) {
        delete[] keys[i];
    }
    delete[] keys;
}

// 清空存储
int storage_clear(storage_t storage) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return 0;
    }

    try {
        return handle->storage->Clear() ? 1 : 0;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return 0;
    }
}

// 关闭存储
int storage_close(storage_t storage) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle || !handle->storage) {
        return 0;
    }

    try {
        return handle->storage->Close() ? 1 : 0;
    } catch (const std::exception& e) {
        handle->last_error = e.what();
        return 0;
    }
}

// 释放存储引擎
void storage_free(storage_t storage) {
    auto handle = static_cast<storage_handle*>(storage);
    if (!handle) {
        return;
    }

    delete handle;
} 