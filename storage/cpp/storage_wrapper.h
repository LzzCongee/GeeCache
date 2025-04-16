#ifndef GEECACHE_STORAGE_WRAPPER_H
#define GEECACHE_STORAGE_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 存储引擎句柄
 */
typedef void* storage_t;

/**
 * @brief 创建内存存储引擎
 * @param max_size 最大存储大小
 * @return 存储引擎句柄
 */
storage_t storage_create_memory(long long max_size);

/**
 * @brief 创建LevelDB存储引擎
 * @param path 存储路径
 * @param max_size 最大存储大小
 * @param compression 是否启用压缩
 * @return 存储引擎句柄
 */
storage_t storage_create_leveldb(const char* path, long long max_size, int compression);

/**
 * @brief 创建RocksDB存储引擎
 * @param path 存储路径
 * @param max_size 最大存储大小
 * @param compression 是否启用压缩
 * @return 存储引擎句柄
 */
storage_t storage_create_rocksdb(const char* path, long long max_size, int compression);

/**
 * @brief 获取指定键的值
 * @param storage 存储引擎句柄
 * @param key 键
 * @param key_len 键长度
 * @param value_len 值长度
 * @return 值
 */
const char* storage_get(storage_t storage, const char* key, int key_len, int* value_len);

/**
 * @brief 设置指定键的值
 * @param storage 存储引擎句柄
 * @param key 键
 * @param key_len 键长度
 * @param value 值
 * @param value_len 值长度
 * @return 是否成功
 */
int storage_set(storage_t storage, const char* key, int key_len, const char* value, int value_len);

/**
 * @brief 设置指定键的值，并指定过期时间
 * @param storage 存储引擎句柄
 * @param key 键
 * @param key_len 键长度
 * @param value 值
 * @param value_len 值长度
 * @param expire 过期时间（毫秒）
 * @return 是否成功
 */
int storage_set_with_expire(storage_t storage, const char* key, int key_len, const char* value, int value_len, long long expire);

/**
 * @brief 删除指定键
 * @param storage 存储引擎句柄
 * @param key 键
 * @param key_len 键长度
 * @return 是否成功
 */
int storage_delete(storage_t storage, const char* key, int key_len);

/**
 * @brief 判断指定键是否存在
 * @param storage 存储引擎句柄
 * @param key 键
 * @param key_len 键长度
 * @return 是否存在
 */
int storage_has(storage_t storage, const char* key, int key_len);

/**
 * @brief 获取所有键
 * @param storage 存储引擎句柄
 * @param keys_len 键列表长度
 * @return 键列表
 */
const char** storage_keys(storage_t storage, int* keys_len);

/**
 * @brief 释放键列表
 * @param keys 键列表
 * @param keys_len 键列表长度
 */
void storage_free_keys(const char** keys, int keys_len);

/**
 * @brief 清空存储
 * @param storage 存储引擎句柄
 * @return 是否成功
 */
int storage_clear(storage_t storage);

/**
 * @brief 关闭存储
 * @param storage 存储引擎句柄
 * @return 是否成功
 */
int storage_close(storage_t storage);

/**
 * @brief 释放存储引擎
 * @param storage 存储引擎句柄
 */
void storage_free(storage_t storage);

#ifdef __cplusplus
}
#endif

#endif // GEECACHE_STORAGE_WRAPPER_H 