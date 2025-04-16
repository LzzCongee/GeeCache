#include "storage.h"
#include "memory_storage.h"
#include <stdexcept>

namespace geecache {
namespace storage {

std::shared_ptr<Storage> StorageFactory::CreateMemoryStorage(int64_t max_size) {
    return std::make_shared<MemoryStorage>(max_size);
}

std::shared_ptr<Storage> StorageFactory::CreateLevelDBStorage(const std::string& path, int64_t max_size, bool compression) {
    throw std::runtime_error("LevelDB storage not implemented yet");
}

std::shared_ptr<Storage> StorageFactory::CreateRocksDBStorage(const std::string& path, int64_t max_size, bool compression) {
    throw std::runtime_error("RocksDB storage not implemented yet");
}

} // namespace storage
} // namespace geecache 