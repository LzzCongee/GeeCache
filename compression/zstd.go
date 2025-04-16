package compression

import (
	"github.com/klauspost/compress/zstd"
)

// ZstdCompressor Zstd压缩器
type ZstdCompressor struct {
	level zstd.EncoderLevel
}

// NewZstdCompressor 创建一个新的Zstd压缩器
func NewZstdCompressor(level CompressionLevel) (*ZstdCompressor, error) {
	var zLevel zstd.EncoderLevel
	switch level {
	case CompressionLevelBestSpeed:
		zLevel = zstd.SpeedFastest
	case CompressionLevelBestCompression:
		zLevel = zstd.SpeedBestCompression
	default:
		zLevel = zstd.SpeedDefault
	}

	return &ZstdCompressor{
		level: zLevel,
	}, nil
}

// Compress 压缩数据
func (c *ZstdCompressor) Compress(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(c.level))
	if err != nil {
		return nil, err
	}
	return encoder.EncodeAll(data, nil), nil
}

// Decompress 解压数据
func (c *ZstdCompressor) Decompress(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	return decoder.DecodeAll(data, nil)
}
