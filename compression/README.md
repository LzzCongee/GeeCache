# 数据压缩模块

这个模块提供了统一的数据压缩和解压缩接口，支持多种压缩算法和压缩级别。

## 支持的压缩算法

- **None**: 不进行压缩，直接返回原始数据
- **Gzip**: 使用标准库的 `compress/gzip` 实现
- **Snappy**: 使用 `github.com/golang/snappy` 实现
- **LZ4**: 使用 `github.com/pierrec/lz4/v4` 实现
- **Zstd**: 使用 `github.com/klauspost/compress/zstd` 实现

## 压缩级别

- **Default**: 默认压缩级别，平衡压缩率和速度
- **BestSpeed**: 最快压缩速度，但压缩率较低
- **BestCompression**: 最高压缩率，但速度较慢

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/geecache/pkg/compression"
)

func main() {
    // 创建压缩选项
    options := compression.CompressionOptions{
        Type:  compression.CompressionTypeGzip,
        Level: compression.CompressionLevelDefault,
    }
    
    // 创建压缩器
    compressor, err := compression.NewCompressor(options)
    if err != nil {
        panic(err)
    }
    
    // 压缩数据
    data := []byte("Hello, World!")
    compressed, err := compressor.Compress(data)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("原始大小: %d 字节\n", len(data))
    fmt.Printf("压缩后大小: %d 字节\n", len(compressed))
    
    // 解压数据
    decompressed, err := compressor.Decompress(compressed)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("解压后数据: %s\n", string(decompressed))
}
```

## 性能比较

以下是各种压缩算法在测试数据上的压缩率比较：

| 算法 | 压缩级别 | 压缩率 |
|------|----------|--------|
| None | Default | 100.00% |
| Gzip | Default | 144.44% |
| Gzip | BestSpeed | 133.33% |
| Gzip | BestCompression | 133.33% |
| Snappy | Default | 104.76% |
| LZ4 | Default | 103.17% |
| Zstd | Default | 120.63% |
| Zstd | BestSpeed | 120.63% |
| Zstd | BestCompression | 111.11% |

注意：压缩率是指压缩后大小与原始大小的比率，小于100%表示压缩有效，大于100%表示压缩后反而变大。对于小数据量，压缩后可能会变大，这是因为压缩算法需要存储额外的元数据。 