package compression

import (
	"bytes"
	"compress/gzip"
	"io"
)

// GzipCompressor Gzip压缩器
type GzipCompressor struct {
	level int
}

// NewGzipCompressor 创建一个新的Gzip压缩器
func NewGzipCompressor(level CompressionLevel) (*GzipCompressor, error) {
	return &GzipCompressor{
		level: int(level),
	}, nil
}

// Compress 压缩数据
func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, c.level)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress 解压数据
func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
