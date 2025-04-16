#ifndef GEECACHE_MEMORY_STORAGE_H
#define GEECACHE_MEMORY_STORAGE_H

#include "storage.h"
#include <unordered_map>
#include <mutex>

namespace geecache {
namespace storage {

/**
 * @brief 内存存储引擎
 */
class MemoryStorage : public Storage {
public:
    /**
     * @brief 构造函数
     * @param max_size 最大存储大小
     */
    explicit MemoryStorage(int64_t max_size);

    /**
     * @brief 析构函数
     */
    ~MemoryStorage() override = default;

    /**
     * @brief 获取指定键的值
     * @param key 键
     * @return 值
     */
    std::string Get(const std::string& key) override;

    /**
     * @brief 设置指定键的值
     * @param key 键
     * @param value 值
     * @return 是否成功
     */
    bool Set(const std::string& key, const std::string& value) override;

    /**
     * @brief 设置指定键的值，并指定过期时间
     * @param key 键
     * @param value 值
     * @param expire 过期时间（毫秒）
     * @return 是否成功
     */
    bool SetWithExpire(const std::string& key, const std::string& value, int64_t expire) override;

    /**
     * @brief 删除指定键
     * @param key 键
     * @return 是否成功
     */
    bool Delete(const std::string& key) override;

    /**
     * @brief 判断指定键是否存在
     * @param key 键
     * @return 是否存在
     */
    bool Has(const std::string& key) override;

    /**
     * @brief 获取所有键
     * @return 键列表
     */
    std::vector<std::string> Keys() override;

    /**
     * @brief 清空存储
     * @return 是否成功
     */
    bool Clear() override;

    /**
     * @brief 关闭存储
     * @return 是否成功
     */
    bool Close() override;

private:
    /**
     * @brief 检查是否过期
     * @param key 键
     * @return 是否过期
     */
    bool IsExpired(const std::string& key);

    /**
     * @brief 清理过期键
     */
    void ClearExpired();

private:
    std::unordered_map<std::string, std::string> data_;
    std::unordered_map<std::string, int64_t> expiries_;
    int64_t max_size_;
    int64_t size_;
    std::mutex mutex_;
};

} // namespace storage
} // namespace geecache

#endif // GEECACHE_MEMORY_STORAGE_H 