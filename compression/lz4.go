package compression

import (
	"github.com/pierrec/lz4/v4"
)

// LZ4Compressor LZ4压缩器
type LZ4Compressor struct {
	level int
}

// NewLZ4Compressor 创建一个新的LZ4压缩器
func NewLZ4Compressor(level CompressionLevel) (*LZ4Compressor, error) {
	return &LZ4Compressor{
		level: int(level),
	}, nil
}

// Compress 压缩数据
func (c *LZ4Compressor) Compress(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, buf, nil)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Decompress 解压数据
func (c *LZ4Compressor) Decompress(data []byte) ([]byte, error) {
	// 由于LZ4压缩不保存原始大小，我们需要估计解压后的大小
	// 这里使用一个保守的估计，实际应用中可能需要更精确的方法
	decompressedSize := len(data) * 4
	dst := make([]byte, decompressedSize)

	n, err := lz4.UncompressBlock(data, dst)
	if err != nil {
		return nil, err
	}

	return dst[:n], nil
}
