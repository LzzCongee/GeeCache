#ifndef GEECACHE_STORAGE_H
#define GEECACHE_STORAGE_H

#include <string>
#include <vector>
#include <memory>
#include <chrono>

namespace geecache {
namespace storage {

/**
 * @brief 存储引擎接口
 */
class Storage {
public:
    /**
     * @brief 析构函数
     */
    virtual ~Storage() = default;

    /**
     * @brief 获取指定键的值
     * @param key 键
     * @return 值
     */
    virtual std::string Get(const std::string& key) = 0;

    /**
     * @brief 设置指定键的值
     * @param key 键
     * @param value 值
     * @return 是否成功
     */
    virtual bool Set(const std::string& key, const std::string& value) = 0;

    /**
     * @brief 设置指定键的值，并指定过期时间
     * @param key 键
     * @param value 值
     * @param expire 过期时间（毫秒）
     * @return 是否成功
     */
    virtual bool SetWithExpire(const std::string& key, const std::string& value, int64_t expire) = 0;

    /**
     * @brief 删除指定键
     * @param key 键
     * @return 是否成功
     */
    virtual bool Delete(const std::string& key) = 0;

    /**
     * @brief 判断指定键是否存在
     * @param key 键
     * @return 是否存在
     */
    virtual bool Has(const std::string& key) = 0;

    /**
     * @brief 获取所有键
     * @return 键列表
     */
    virtual std::vector<std::string> Keys() = 0;

    /**
     * @brief 清空存储
     * @return 是否成功
     */
    virtual bool Clear() = 0;

    /**
     * @brief 关闭存储
     * @return 是否成功
     */
    virtual bool Close() = 0;
};

/**
 * @brief 存储引擎工厂
 */
class StorageFactory {
public:
    /**
     * @brief 创建内存存储引擎
     * @param max_size 最大存储大小
     * @return 存储引擎
     */
    static std::shared_ptr<Storage> CreateMemoryStorage(int64_t max_size);

    /**
     * @brief 创建LevelDB存储引擎
     * @param path 存储路径
     * @param max_size 最大存储大小
     * @param compression 是否启用压缩
     * @return 存储引擎
     */
    static std::shared_ptr<Storage> CreateLevelDBStorage(const std::string& path, int64_t max_size, bool compression);

    /**
     * @brief 创建RocksDB存储引擎
     * @param path 存储路径
     * @param max_size 最大存储大小
     * @param compression 是否启用压缩
     * @return 存储引擎
     */
    static std::shared_ptr<Storage> CreateRocksDBStorage(const std::string& path, int64_t max_size, bool compression);
};

} // namespace storage
} // namespace geecache

#endif // GEECACHE_STORAGE_H 