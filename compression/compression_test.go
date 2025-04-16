package compression

import (
	"bytes"
	"testing"
)

func TestCompressors(t *testing.T) {
	testData := []byte("Hello, World! This is a test string for compression algorithms.")

	tests := []struct {
		name            string
		compressionType CompressionType
		level           CompressionLevel
	}{
		{"None", CompressionTypeNone, CompressionLevelDefault},
		{"Gzip-Default", CompressionTypeGzip, CompressionLevelDefault},
		{"Gzip-Speed", CompressionTypeGzip, CompressionLevelBestSpeed},
		{"Gzip-Compression", CompressionTypeGzip, CompressionLevelBestCompression},
		{"Snappy", CompressionTypeSnappy, CompressionLevelDefault},
		{"LZ4", CompressionTypeLZ4, CompressionLevelDefault},
		{"Zstd-Default", CompressionTypeZstd, CompressionLevelDefault},
		{"Zstd-Speed", CompressionTypeZstd, CompressionLevelBestSpeed},
		{"Zstd-Compression", CompressionTypeZstd, CompressionLevelBestCompression},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := CompressionOptions{
				Type:  tt.compressionType,
				Level: tt.level,
			}

			compressor, err := NewCompressor(options)
			if err != nil {
				t.Fatalf("Failed to create compressor: %v", err)
			}

			compressed, err := compressor.Compress(testData)
			if err != nil {
				t.Fatalf("Compression failed: %v", err)
			}

			// 对于None压缩器，压缩后的数据应该与原始数据相同
			if tt.compressionType == CompressionTypeNone {
				if !bytes.Equal(compressed, testData) {
					t.Errorf("None compressor should not change data")
				}
			}

			// 对于其他压缩器，压缩后的数据应该与原始数据不同（除非数据太小或太简单）
			if tt.compressionType != CompressionTypeNone && len(testData) > 20 {
				if bytes.Equal(compressed, testData) {
					t.Logf("Warning: Compressed data is identical to original data for %s", tt.name)
				}
			}

			// 测试解压
			decompressed, err := compressor.Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			// 解压后的数据应该与原始数据相同
			if !bytes.Equal(decompressed, testData) {
				t.Errorf("Decompressed data does not match original data for %s", tt.name)
			}

			// 打印压缩率
			compressionRatio := float64(len(compressed)) / float64(len(testData)) * 100
			t.Logf("%s compression ratio: %.2f%% (%d -> %d bytes)",
				tt.name, compressionRatio, len(testData), len(compressed))
		})
	}
}
