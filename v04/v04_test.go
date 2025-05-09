package v04

import (
	"bytes"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
	v03 "github.com/harriteja/GoZ4X/v03"
	"github.com/harriteja/GoZ4X/v04/simd"
)

var (
	// Test data sizes
	smallSize  = 1 * 1024           // 1KB
	mediumSize = 512 * 1024         // 512KB
	largeSize  = 4*1024*1024 - 1024 // ~3.999MB (just under 4MB limit)
)

// generateCompressibleData creates compressible data of the specified size
func generateCompressibleData(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		// Create a compressible pattern with some repetition
		data[i] = byte((i * 7) % 251)
	}
	return data
}

// generateIncompressibleData creates random-looking data that's hard to compress
func generateIncompressibleData(size int) []byte {
	data := make([]byte, size)
	// Use a simple PRNG to generate "random" data
	x := uint32(0xDE4D10CC) // Seed
	for i := 0; i < size; i++ {
		x = x ^ (x << 13)
		x = x ^ (x >> 17)
		x = x ^ (x << 5)
		data[i] = byte(x & 0xFF)
	}
	return data
}

func TestCompressDecompress(t *testing.T) {
	testCases := []struct {
		name string
		size int
	}{
		{"Small", smallSize},
		{"Medium", mediumSize},
		{"Large", largeSize},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := generateCompressibleData(tc.size)

			// Test v0.4 compression and decompression
			compressed, err := CompressBlock(input, nil)
			if err != nil {
				t.Fatalf("v0.4 compression failed: %v", err)
			}

			decompressed, err := compress.DecompressBlock(compressed, nil, len(input))
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			if !bytes.Equal(input, decompressed) {
				t.Fatalf("Data mismatch after v0.4 compress/decompress")
			}

			// Test v0.4 parallel compression and decompression
			compressedParallel, err := CompressBlockParallel(input, nil)
			if err != nil {
				t.Fatalf("v0.4 parallel compression failed: %v", err)
			}

			decompressedParallel, err := compress.DecompressBlock(compressedParallel, nil, len(input))
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			if !bytes.Equal(input, decompressedParallel) {
				t.Fatalf("Data mismatch after v0.4 parallel compress/decompress")
			}
		})
	}
}

// BenchmarkCompress benchmarks compression performance for different versions
func BenchmarkCompress(b *testing.B) {
	testCases := []struct {
		name string
		size int
		data []byte
	}{
		{"SmallCompressible", smallSize, generateCompressibleData(smallSize)},
		{"MediumCompressible", mediumSize, generateCompressibleData(mediumSize)},
		{"LargeCompressible", largeSize, generateCompressibleData(largeSize)},
		{"SmallIncompressible", smallSize, generateIncompressibleData(smallSize)},
		{"MediumIncompressible", mediumSize, generateIncompressibleData(mediumSize)},
		{"LargeIncompressible", largeSize, generateIncompressibleData(largeSize)},
	}

	// Get SIMD features for reporting
	features := simd.DetectFeatures()
	b.Logf("CPU Features: SSE2=%v, SSE4.1=%v, AVX2=%v, AVX512=%v, NEON=%v",
		features.HasSSE2, features.HasSSE41, features.HasAVX2, features.HasAVX512, features.HasNEON)

	for _, tc := range testCases {
		// v0.1 (baseline)
		b.Run("v0.1/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data)*2) // Pre-allocate to avoid allocation overhead
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compress.CompressBlock(tc.data, dst)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// v0.1 with level 6 (comparable to v0.2/v0.3/v0.4 default)
		b.Run("v0.1-L6/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data)*2)
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compress.CompressBlockLevel(tc.data, dst, compress.CompressionLevel(6))
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// v0.2 (improved match finding)
		b.Run("v0.2/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data)*2)
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compress.CompressBlockV2(tc.data, dst)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// v0.3 (parallel)
		b.Run("v0.3/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data)*2)
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := v03.CompressBlockV2Parallel(tc.data, dst)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// v0.4 (SIMD)
		b.Run("v0.4/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data)*2)
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := CompressBlock(tc.data, dst)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// v0.4 parallel (SIMD + parallel)
		b.Run("v0.4-Parallel/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data)*2)
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := CompressBlockParallel(tc.data, dst)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDecompress benchmarks decompression performance
func BenchmarkDecompress(b *testing.B) {
	testCases := []struct {
		name string
		size int
		data []byte
	}{
		{"SmallCompressible", smallSize, generateCompressibleData(smallSize)},
		{"MediumCompressible", mediumSize, generateCompressibleData(mediumSize)},
		{"LargeCompressible", largeSize, generateCompressibleData(largeSize)},
	}

	for _, tc := range testCases {
		// Pre-compress data with different versions
		v1Compressed, _ := compress.CompressBlock(tc.data, nil)
		v2Compressed, _ := compress.CompressBlockV2(tc.data, nil)
		v4Compressed, _ := CompressBlock(tc.data, nil)

		// Benchmark decompression
		b.Run("FromV1/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data))
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compress.DecompressBlock(v1Compressed, dst, len(tc.data))
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("FromV2/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data))
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compress.DecompressBlock(v2Compressed, dst, len(tc.data))
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("FromV4/"+tc.name, func(b *testing.B) {
			dst := make([]byte, len(tc.data))
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compress.DecompressBlock(v4Compressed, dst, len(tc.data))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCompressionRatio measures compression ratios of different versions
func BenchmarkCompressionRatio(b *testing.B) {
	testCases := []struct {
		name string
		size int
		data []byte
	}{
		{"SmallCompressible", smallSize, generateCompressibleData(smallSize)},
		{"MediumCompressible", mediumSize, generateCompressibleData(mediumSize)},
		{"LargeCompressible", largeSize, generateCompressibleData(largeSize)},
		{"SmallIncompressible", smallSize, generateIncompressibleData(smallSize)},
		{"MediumIncompressible", mediumSize, generateIncompressibleData(mediumSize)},
		{"LargeIncompressible", largeSize, generateIncompressibleData(largeSize)},
	}

	for _, tc := range testCases {
		// Test each version once to measure compression ratio
		b.Run(tc.name, func(b *testing.B) {
			// Only run once - we're just measuring size
			b.N = 1

			// v0.1
			v1Compressed, _ := compress.CompressBlock(tc.data, nil)
			v1Ratio := float64(len(tc.data)) / float64(len(v1Compressed))

			// v0.2
			v2Compressed, _ := compress.CompressBlockV2(tc.data, nil)
			v2Ratio := float64(len(tc.data)) / float64(len(v2Compressed))

			// v0.3 (v0.2 with parallel)
			v3Compressed, _ := v03.CompressBlockV2Parallel(tc.data, nil)
			v3Ratio := float64(len(tc.data)) / float64(len(v3Compressed))

			// v0.4
			v4Compressed, _ := CompressBlock(tc.data, nil)
			v4Ratio := float64(len(tc.data)) / float64(len(v4Compressed))

			// Log results
			b.Logf("Compression ratio - v0.1: %.2f:1, v0.2: %.2f:1, v0.3: %.2f:1, v0.4: %.2f:1",
				v1Ratio, v2Ratio, v3Ratio, v4Ratio)

			b.Logf("Size reduction - v0.1: %.2f%%, v0.2: %.2f%%, v0.3: %.2f%%, v0.4: %.2f%%",
				100-float64(len(v1Compressed))*100/float64(len(tc.data)),
				100-float64(len(v2Compressed))*100/float64(len(tc.data)),
				100-float64(len(v3Compressed))*100/float64(len(tc.data)),
				100-float64(len(v4Compressed))*100/float64(len(tc.data)))

			// Improvements over v0.1
			b.Logf("Improvements over v0.1 - v0.2: %.2f%%, v0.3: %.2f%%, v0.4: %.2f%%",
				(v2Ratio-v1Ratio)/v1Ratio*100,
				(v3Ratio-v1Ratio)/v1Ratio*100,
				(v4Ratio-v1Ratio)/v1Ratio*100)
		})
	}
}
