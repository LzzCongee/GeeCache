package compression

// NoneCompressor 不压缩压缩器
type NoneCompressor struct {
}

// NewNoneCompressor 创建一个新的不压缩压缩器
func NewNoneCompressor() *NoneCompressor {
	return &NoneCompressor{}
}

// Compress 压缩数据
func (c *NoneCompressor) Compress(data []byte) ([]byte, error) {
	return data, nil
}

// Decompress 解压数据
func (c *NoneCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}
