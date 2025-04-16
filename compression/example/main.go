package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"geecache/compression"
)

func main() {
	// 准备测试数据
	data, err := ioutil.ReadFile("example.go")
	if err != nil {
		// 如果文件不存在，使用当前文件作为测试数据
		data, err = ioutil.ReadFile("main.go")
		if err != nil {
			// 如果还是不存在，使用一个字符串作为测试数据
			data = []byte(`
			package main

			import (
				"fmt"
				"github.com/geecache/pkg/compression"
				"io/ioutil"
				"os"
				"time"
			)

			func main() {
				// 这是一个示例程序，用于测试各种压缩算法的性能和压缩率
				// 这里有一些重复的文本，以便更好地测试压缩效果
				// 压缩算法通常对重复内容有更好的压缩效果
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
				// 这是一些重复的文本
			}
			`)
		}
	}

	fmt.Printf("原始数据大小: %d 字节\n", len(data))
	fmt.Println("测试各种压缩算法...\n")

	// 定义要测试的压缩类型和级别
	compressionTypes := []compression.CompressionType{
		compression.CompressionTypeNone,
		compression.CompressionTypeGzip,
		compression.CompressionTypeSnappy,
		compression.CompressionTypeLZ4,
		compression.CompressionTypeZstd,
	}

	compressionLevels := []compression.CompressionLevel{
		compression.CompressionLevelDefault,
		compression.CompressionLevelBestSpeed,
		compression.CompressionLevelBestCompression,
	}

	// 打印表头
	fmt.Printf("%-10s %-15s %-15s %-15s %-15s\n", "算法", "压缩级别", "压缩率", "压缩时间", "解压时间")
	fmt.Println(strings.Repeat("-", 70))

	// 测试每种压缩算法和级别
	for _, cType := range compressionTypes {
		for _, cLevel := range compressionLevels {
			// 对于不支持压缩级别的算法，只测试默认级别
			if (cType == compression.CompressionTypeSnappy || cType == compression.CompressionTypeNone) &&
				cLevel != compression.CompressionLevelDefault {
				continue
			}

			options := compression.CompressionOptions{
				Type:  cType,
				Level: cLevel,
			}

			compressor, err := compression.NewCompressor(options)
			if err != nil {
				fmt.Printf("创建压缩器失败: %v\n", err)
				continue
			}

			// 测量压缩时间
			startCompress := time.Now()
			compressed, err := compressor.Compress(data)
			compressTime := time.Since(startCompress)
			if err != nil {
				fmt.Printf("压缩失败: %v\n", err)
				continue
			}

			// 测量解压时间
			startDecompress := time.Now()
			decompressed, err := compressor.Decompress(compressed)
			decompressTime := time.Since(startDecompress)
			if err != nil {
				fmt.Printf("解压失败: %v\n", err)
				continue
			}

			// 验证解压后的数据是否与原始数据相同
			if string(decompressed) != string(data) {
				fmt.Printf("警告: %s 解压后的数据与原始数据不匹配!\n", cType)
			}

			// 计算压缩率
			ratio := float64(len(compressed)) / float64(len(data)) * 100

			// 获取压缩级别的字符串表示
			levelStr := "Default"
			if cLevel == compression.CompressionLevelBestSpeed {
				levelStr = "BestSpeed"
			} else if cLevel == compression.CompressionLevelBestCompression {
				levelStr = "BestCompression"
			}

			// 打印结果
			fmt.Printf("%-10s %-15s %-15.2f%% %-15s %-15s\n",
				cType, levelStr, ratio,
				compressTime.String(), decompressTime.String())
		}
	}
}
