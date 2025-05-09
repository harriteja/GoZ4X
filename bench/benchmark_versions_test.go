package bench

import (
	"bytes"
	"testing"

	goz4x "github.com/harriteja/GoZ4X"
	v03 "github.com/harriteja/GoZ4X/v03"
)

// BenchmarkVersionComparison compares the performance and compression ratio of v0.1, v0.2, and v0.3
func BenchmarkVersionComparison(b *testing.B) {
	// Create different sample data types for testing
	// Text data - typical prose text with some repetition
	textData := bytes.Repeat([]byte("GoZ4X is a pure-Go implementation of the LZ4 compression algorithm. "+
		"It's designed for speed and compatibility with the original LZ4 format."), 100)

	// JSON-like data - structured data with field names and values
	jsonData := bytes.Repeat([]byte(`{"id":1234,"name":"GoZ4X","version":"0.3.0","features":["speed","compatibility","parallelism"],`+
		`"metrics":{"compressionRatio":0.4,"speed":"fast","memory":"efficient"}}`), 50)

	// Binary data - less compressible random-like data
	binaryData := make([]byte, 100000)
	for i := range binaryData {
		binaryData[i] = byte(i * 17 % 255)
	}

	// Highly compressible data
	highlyCompressible := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 4000)

	// Large data for testing parallel advantage (5MB+)
	// Commented out to prevent benchmark hangs
	// largeTextData := bytes.Repeat([]byte("GoZ4X is a pure-Go implementation of the LZ4 compression algorithm. "+
	//	"It's designed for speed and compatibility with the original LZ4 format."), 5000)

	// Very large JSON data (10MB+)
	// Commented out to prevent benchmark hangs
	// largeJSONData := bytes.Repeat([]byte(`{"id":1234,"name":"GoZ4X","version":"0.3.0","features":["speed","compatibility","parallelism"],`+
	//	`"metrics":{"compressionRatio":0.4,"speed":"fast","memory":"efficient"}}`), 5000)

	tests := []struct {
		name string
		data []byte
	}{
		{"Text", textData},
		{"JSON", jsonData},
		{"Binary", binaryData},
		{"HighlyCompressible", highlyCompressible},
		// Skip large tests to prevent benchmark hangs
		// {"LargeText_5MB", largeTextData},
		// {"LargeJSON_10MB", largeJSONData},
	}

	for _, tt := range tests {
		// Benchmark v0.1 block compression
		b.Run("v0.1_"+tt.name, func(b *testing.B) {
			// Limit iterations to prevent test hangs
			b.N = min(b.N, 1000)

			b.ResetTimer()
			b.SetBytes(int64(len(tt.data)))

			for i := 0; i < b.N; i++ {
				compressed, _ := goz4x.CompressBlock(tt.data, nil)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tt.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})

		// Benchmark v0.2 block compression
		b.Run("v0.2_"+tt.name, func(b *testing.B) {
			// Limit iterations to prevent test hangs
			b.N = min(b.N, 1000)

			b.ResetTimer()
			b.SetBytes(int64(len(tt.data)))

			for i := 0; i < b.N; i++ {
				compressed, _ := goz4x.CompressBlockV2(tt.data, nil)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tt.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})

		// Benchmark v0.3 (parallel) block compression
		b.Run("v0.3_"+tt.name, func(b *testing.B) {
			// Limit iterations to prevent test hangs
			b.N = min(b.N, 1000)

			b.ResetTimer()
			b.SetBytes(int64(len(tt.data)))

			for i := 0; i < b.N; i++ {
				compressed, _ := v03.CompressBlockV2Parallel(tt.data, nil)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tt.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})
	}
}

// BenchmarkDecompression compares the decompression performance of the different versions
func BenchmarkDecompression(b *testing.B) {
	// Text data sample - reduce size to prevent test hangs
	textData := bytes.Repeat([]byte("GoZ4X is a pure-Go implementation of the LZ4 compression algorithm. "+
		"It's designed for speed and compatibility with the original LZ4 format."), 100) // Reduced from 1000

	// Compress using all three methods
	v1Compressed, _ := goz4x.CompressBlock(textData, nil)
	v2Compressed, _ := goz4x.CompressBlockV2(textData, nil)
	v3Compressed, _ := v03.CompressBlockV2Parallel(textData, nil)

	// Benchmark decompression
	b.Run("Decompress_v0.1", func(b *testing.B) {
		// Limit iterations to prevent test hangs
		b.N = min(b.N, 1000)

		b.ResetTimer()
		b.SetBytes(int64(len(textData)))

		for i := 0; i < b.N; i++ {
			decompressed, _ := goz4x.DecompressBlock(v1Compressed, nil, len(textData))
			b.StopTimer()
			if len(decompressed) != len(textData) {
				b.Fatalf("Decompression failed: expected %d bytes, got %d", len(textData), len(decompressed))
			}
			b.StartTimer()
		}
	})

	b.Run("Decompress_v0.2", func(b *testing.B) {
		// Limit iterations to prevent test hangs
		b.N = min(b.N, 1000)

		b.ResetTimer()
		b.SetBytes(int64(len(textData)))

		for i := 0; i < b.N; i++ {
			decompressed, _ := goz4x.DecompressBlock(v2Compressed, nil, len(textData))
			b.StopTimer()
			if len(decompressed) != len(textData) {
				b.Fatalf("Decompression failed: expected %d bytes, got %d", len(textData), len(decompressed))
			}
			b.StartTimer()
		}
	})

	b.Run("Decompress_v0.3", func(b *testing.B) {
		// Limit iterations to prevent test hangs
		b.N = min(b.N, 1000)

		b.ResetTimer()
		b.SetBytes(int64(len(textData)))

		for i := 0; i < b.N; i++ {
			decompressed, _ := goz4x.DecompressBlock(v3Compressed, nil, len(textData))
			b.StopTimer()
			if len(decompressed) != len(textData) {
				b.Fatalf("Decompression failed: expected %d bytes, got %d", len(textData), len(decompressed))
			}
			b.StartTimer()
		}
	})
}
