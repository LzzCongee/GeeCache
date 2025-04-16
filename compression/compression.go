package compression

import (
	"fmt"
)

// CompressionType 压缩类型
type CompressionType string

const (
	// CompressionTypeNone 不压缩
	CompressionTypeNone CompressionType = "none"
	// CompressionTypeGzip Gzip压缩
	CompressionTypeGzip CompressionType = "gzip"
	// CompressionTypeSnappy Snappy压缩
	CompressionTypeSnappy CompressionType = "snappy"
	// CompressionTypeLZ4 LZ4压缩
	CompressionTypeLZ4 CompressionType = "lz4"
	// CompressionTypeZstd Zstandard压缩
	CompressionTypeZstd CompressionType = "zstd"
)

// CompressionLevel 压缩级别
type CompressionLevel int

const (
	// CompressionLevelDefault 默认压缩级别
	CompressionLevelDefault CompressionLevel = 0
	// CompressionLevelBestSpeed 最快压缩级别
	CompressionLevelBestSpeed CompressionLevel = 1
	// CompressionLevelBestCompression 最佳压缩级别
	CompressionLevelBestCompression CompressionLevel = 9
)

// CompressionOptions 压缩选项
type CompressionOptions struct {
	// Type 压缩类型
	Type CompressionType
	// Level 压缩级别
	Level CompressionLevel
}

// Compressor 压缩器接口
type Compressor interface {
	// Compress 压缩数据
	Compress(data []byte) ([]byte, error)
	// Decompress 解压数据
	Decompress(data []byte) ([]byte, error)
}

// NewCompressor 创建一个新的压缩器
func NewCompressor(options CompressionOptions) (Compressor, error) {
	switch options.Type {
	case CompressionTypeNone:
		return NewNoneCompressor(), nil
	case CompressionTypeGzip:
		return NewGzipCompressor(options.Level)
	case CompressionTypeSnappy:
		return NewSnappyCompressor(options.Level)
	case CompressionTypeLZ4:
		return NewLZ4Compressor(options.Level)
	case CompressionTypeZstd:
		return NewZstdCompressor(options.Level)
	default:
		return nil, fmt.Errorf("unknown compression type: %s", options.Type)
	}
}
