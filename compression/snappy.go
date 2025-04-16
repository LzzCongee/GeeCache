package compression

import (
	"github.com/golang/snappy"
)

// SnappyCompressor Snappy压缩器
type SnappyCompressor struct{}

// NewSnappyCompressor 创建一个新的Snappy压缩器
func NewSnappyCompressor(_ CompressionLevel) (*SnappyCompressor, error) {
	return &SnappyCompressor{}, nil
}

// Compress 压缩数据
func (c *SnappyCompressor) Compress(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

// Decompress 解压数据
func (c *SnappyCompressor) Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}
