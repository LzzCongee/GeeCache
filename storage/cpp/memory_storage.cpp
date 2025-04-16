#include "memory_storage.h"
#include <chrono>
#include <stdexcept>

namespace geecache {
namespace storage {

MemoryStorage::MemoryStorage(int64_t max_size)
    : max_size_(max_size), size_(0) {
}

std::string MemoryStorage::Get(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);

    // 检查是否过期
    if (IsExpired(key)) {
        Delete(key);
        throw std::runtime_error("key not found");
    }

    auto it = data_.find(key);
    if (it == data_.end()) {
        throw std::runtime_error("key not found");
    }

    return it->second;
}

bool MemoryStorage::Set(const std::string& key, const std::string& value) {
    return SetWithExpire(key, value, 0);
}

bool MemoryStorage::SetWithExpire(const std::string& key, const std::string& value, int64_t expire) {
    std::lock_guard<std::mutex> lock(mutex_);

    // 检查存储大小
    if (max_size_ > 0) {
        // 如果已存在，先减去旧值的大小
        auto it = data_.find(key);
        if (it != data_.end()) {
            size_ -= key.size() + it->second.size();
        }

        // 计算新值的大小
        int64_t new_size = size_ + key.size() + value.size();
        if (new_size > max_size_) {
            return false;
        }
        size_ = new_size;
    }

    // 设置值
    data_[key] = value;

    // 设置过期时间
    if (expire > 0) {
        int64_t expire_time = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()).count() + expire;
        expiries_[key] = expire_time;
    } else {
        expiries_.erase(key);
    }

    return true;
}

bool MemoryStorage::Delete(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);

    auto it = data_.find(key);
    if (it != data_.end()) {
        size_ -= key.size() + it->second.size();
        data_.erase(it);
        expiries_.erase(key);
        return true;
    }

    return false;
}

bool MemoryStorage::Has(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);

    // 检查是否过期
    if (IsExpired(key)) {
        Delete(key);
        return false;
    }

    return data_.find(key) != data_.end();
}

std::vector<std::string> MemoryStorage::Keys() {
    std::lock_guard<std::mutex> lock(mutex_);

    // 清理过期键
    ClearExpired();

    std::vector<std::string> keys;
    keys.reserve(data_.size());
    for (const auto& kv : data_) {
        keys.push_back(kv.first);
    }

    return keys;
}

bool MemoryStorage::Clear() {
    std::lock_guard<std::mutex> lock(mutex_);

    data_.clear();
    expiries_.clear();
    size_ = 0;

    return true;
}

bool MemoryStorage::Close() {
    return Clear();
}

bool MemoryStorage::IsExpired(const std::string& key) {
    auto it = expiries_.find(key);
    if (it == expiries_.end()) {
        return false;
    }

    int64_t now = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    return it->second <= now;
}

void MemoryStorage::ClearExpired() {
    int64_t now = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    for (auto it = expiries_.begin(); it != expiries_.end();) {
        if (it->second <= now) {
            Delete(it->first);
            it = expiries_.begin();
        } else {
            ++it;
        }
    }
}

} // namespace storage
} // namespace geecache 