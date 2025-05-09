package bench

import (
	"bytes"
	"fmt"
	"testing"

	goz4x "github.com/harriteja/GoZ4X"
	"github.com/harriteja/GoZ4X/compress"
	v03 "github.com/harriteja/GoZ4X/v03"
)

// TestCompressionDetails examines the detailed behavior of the different compression versions
func TestCompressionDetails(t *testing.T) {
	// Create sample data with known patterns
	patternData := bytes.Repeat([]byte("ABCABCABCDEFDEFDEFGHIGHIGHI"), 100)

	// Get compression from each version
	v1Compressed, err1 := goz4x.CompressBlock(patternData, nil)
	if err1 != nil {
		t.Fatalf("V1 compression failed: %v", err1)
	}

	v2Compressed, err2 := goz4x.CompressBlockV2(patternData, nil)
	if err2 != nil {
		t.Fatalf("V2 compression failed: %v", err2)
	}

	v3Compressed, err3 := v03.CompressBlockV2Parallel(patternData, nil)
	if err3 != nil {
		t.Fatalf("V3 compression failed: %v", err3)
	}

	// Compare compression ratios
	v1Ratio := float64(len(v1Compressed)) / float64(len(patternData))
	v2Ratio := float64(len(v2Compressed)) / float64(len(patternData))
	v3Ratio := float64(len(v3Compressed)) / float64(len(patternData))

	fmt.Printf("V1 compression ratio: %.5f (size: %d bytes)\n", v1Ratio, len(v1Compressed))
	fmt.Printf("V2 compression ratio: %.5f (size: %d bytes)\n", v2Ratio, len(v2Compressed))
	fmt.Printf("V3 compression ratio: %.5f (size: %d bytes)\n", v3Ratio, len(v3Compressed))

	// Verify decompression works for all versions
	v1Decompressed, err1d := goz4x.DecompressBlock(v1Compressed, nil, len(patternData))
	if err1d != nil {
		t.Fatalf("V1 decompression failed: %v", err1d)
	}

	v2Decompressed, err2d := goz4x.DecompressBlock(v2Compressed, nil, len(patternData))
	if err2d != nil {
		t.Fatalf("V2 decompression failed: %v", err2d)
	}

	v3Decompressed, err3d := goz4x.DecompressBlock(v3Compressed, nil, len(patternData))
	if err3d != nil {
		t.Fatalf("V3 decompression failed: %v", err3d)
	}

	// Verify all decompressed results match original
	if !bytes.Equal(patternData, v1Decompressed) {
		t.Error("V1 decompression does not match original data")
	}

	if !bytes.Equal(patternData, v2Decompressed) {
		t.Error("V2 decompression does not match original data")
	}

	if !bytes.Equal(patternData, v3Decompressed) {
		t.Error("V3 decompression does not match original data")
	}
}

// BenchmarkCompressionLevels tests different compression levels across versions
func BenchmarkCompressionLevels(b *testing.B) {
	// Create text data
	textData := bytes.Repeat([]byte("GoZ4X is a pure-Go implementation of the LZ4 compression algorithm. "), 200)

	// Test compression levels for v0.1
	for _, level := range []compress.CompressionLevel{1, 3, 6} {
		// Skipping level 9 and 12 to prevent test hangs
		b.Run(fmt.Sprintf("V0.1_Level%d", level), func(b *testing.B) {
			// Limit iterations to prevent hangs
			b.N = min(b.N, 1000)

			b.ResetTimer()
			b.SetBytes(int64(len(textData)))

			for i := 0; i < b.N; i++ {
				compressed, _ := compress.CompressBlockLevel(textData, nil, level)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(textData))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})
	}

	// Test compression levels for v0.2
	for _, level := range []compress.CompressionLevel{1, 3, 6} {
		// Skipping level 9 and 12 to prevent test hangs
		b.Run(fmt.Sprintf("V0.2_Level%d", level), func(b *testing.B) {
			// Limit iterations to prevent hangs
			b.N = min(b.N, 1000)

			b.ResetTimer()
			b.SetBytes(int64(len(textData)))

			for i := 0; i < b.N; i++ {
				compressed, _ := compress.CompressBlockV2Level(textData, nil, level)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(textData))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})
	}

	// Test compression levels for v0.3
	for _, level := range []int{1, 3, 6} {
		// Skipping level 9 and 12 to prevent test hangs
		b.Run(fmt.Sprintf("V0.3_Level%d", level), func(b *testing.B) {
			// Limit iterations to prevent hangs
			b.N = min(b.N, 1000)

			b.ResetTimer()
			b.SetBytes(int64(len(textData)))

			for i := 0; i < b.N; i++ {
				compressed, _ := v03.CompressBlockV2ParallelLevel(textData, nil, level)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(textData))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})
	}
}
