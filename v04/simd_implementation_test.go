package v04

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
	"github.com/harriteja/GoZ4X/v04/simd"
)

// TestSIMDImplementationChoice tests that the correct SIMD implementation is chosen
// based on the options and available CPU features
func TestSIMDImplementationChoice(t *testing.T) {
	// Get available features
	features := simd.DetectFeatures()
	t.Logf("CPU Features: SSE2=%v, SSE4.1=%v, AVX2=%v, AVX512=%v, NEON=%v",
		features.HasSSE2, features.HasSSE41, features.HasAVX2, features.HasAVX512, features.HasNEON)

	// Create test data
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 251)
	}

	// Test with explicitly specified implementations
	testCases := []struct {
		name             string
		simdImpl         int
		shouldWork       bool
		compressionLevel int
	}{
		{"Generic", simd.ImplGeneric, true, 6},
		{"SSE4.1", simd.ImplSSE41, runtime.GOARCH == "amd64" && features.HasSSE41, 6},
		{"AVX2", simd.ImplAVX2, runtime.GOARCH == "amd64" && features.HasAVX2, 6},
		{"NEON", simd.ImplNEON, runtime.GOARCH == "arm64" && features.HasNEON, 6},
		// Test different compression levels
		{"Level1", simd.BestImplementation(), true, 1},
		{"Level9", simd.BestImplementation(), true, 9},
		{"Level12", simd.BestImplementation(), true, 12},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := DefaultOptions()
			opts.SIMDImpl = tc.simdImpl
			opts.Level = CompressionLevel(tc.compressionLevel)

			// Try compressing with the specified implementation
			compressed, err := CompressBlockWithOptions(data, nil, opts)

			if !tc.shouldWork {
				if err == nil {
					t.Logf("Warning: Expected %s implementation to fail, but it worked", tc.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("Compression with %s implementation failed: %v", tc.name, err)
			}

			// Try decompressing
			decompressed, err := compress.DecompressBlock(compressed, nil, len(data))
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			// Verify data integrity
			if !bytes.Equal(data, decompressed) {
				t.Fatalf("Data mismatch with %s implementation", tc.name)
			}

			t.Logf("%s implementation with level %d: %d -> %d bytes (%.2f%%)",
				tc.name, tc.compressionLevel, len(data), len(compressed),
				float64(len(compressed))*100/float64(len(data)))
		})
	}
}

// TestCompressionWithSIMDOptions tests compression with various SIMD-specific options
func TestCompressionWithSIMDOptions(t *testing.T) {
	// Create test data with different patterns
	dataSize := 1 * 1024 * 1024 // 1MB

	// Create compressible data (repeating patterns)
	compressibleData := make([]byte, dataSize)
	for i := range compressibleData {
		compressibleData[i] = byte((i / 100) % 255) // Repeats every 25500 bytes
	}

	// Create less compressible data (quasi-random)
	lessCompressibleData := make([]byte, dataSize)
	x := uint32(12345)
	for i := range lessCompressibleData {
		// Simple PRNG
		x = x ^ (x << 13)
		x = x ^ (x >> 17)
		x = x ^ (x << 5)
		lessCompressibleData[i] = byte(x & 0xFF)
	}

	// Test different combinations of options
	testCases := []struct {
		name    string
		data    []byte
		options Options
	}{
		{"Default-Compressible", compressibleData, DefaultOptions()},
		{"Default-LessCompressible", lessCompressibleData, DefaultOptions()},
		{"Level1-Compressible", compressibleData, Options{Level: 1, UseV2: true, SIMDImpl: simd.BestImplementation()}},
		{"Level9-Compressible", compressibleData, Options{Level: 9, UseV2: true, SIMDImpl: simd.BestImplementation()}},
		{"NoV2-Compressible", compressibleData, Options{Level: 6, UseV2: false, SIMDImpl: simd.BestImplementation()}},
		{"SmallWindow-Compressible", compressibleData, Options{Level: 6, UseV2: true, SIMDImpl: simd.BestImplementation(), WindowSize: 16 * 1024}},
		{"LargeWindow-Compressible", compressibleData, Options{Level: 6, UseV2: true, SIMDImpl: simd.BestImplementation(), WindowSize: 256 * 1024}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Compress with specific options
			compressed, err := CompressBlockWithOptions(tc.data, nil, tc.options)
			if err != nil {
				t.Fatalf("Compression failed: %v", err)
			}

			// Verify decompression works
			decompressed, err := compress.DecompressBlock(compressed, nil, len(tc.data))
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			// Verify data integrity
			if !bytes.Equal(tc.data, decompressed) {
				t.Fatalf("Data integrity check failed")
			}

			// Log compression metrics
			t.Logf("%s: %d -> %d bytes (%.2f%%), level=%d, useV2=%v, window=%d",
				tc.name, len(tc.data), len(compressed),
				float64(len(compressed))*100/float64(len(tc.data)),
				tc.options.Level, tc.options.UseV2, tc.options.WindowSize)
		})
	}
}

// TestParallelCompressionWithSIMD tests that parallel compression works with SIMD optimizations
func TestParallelCompressionWithSIMD(t *testing.T) {
	// Create a large test data set that benefits from parallel compression
	dataSize := 8 * 1024 * 1024 // 8MB
	data := make([]byte, dataSize)

	// Fill with compressible pattern
	for i := range data {
		data[i] = byte((i / 1024) % 251) // Repeats every ~261K bytes
	}

	// Test with different worker counts
	workerCounts := []int{1, 2, 4, runtime.NumCPU()}

	for _, workers := range workerCounts {
		t.Run("Workers-"+string(rune('0'+workers)), func(t *testing.T) {
			opts := DefaultOptions()
			opts.NumWorkers = workers

			// Use parallel compression
			compressed, err := CompressBlockParallelWithOptions(data, nil, opts)
			if err != nil {
				t.Fatalf("Parallel compression with %d workers failed: %v", workers, err)
			}

			// Verify decompression works
			decompressed, err := compress.DecompressBlock(compressed, nil, len(data))
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			// Verify data integrity
			if !bytes.Equal(data, decompressed) {
				t.Fatalf("Data integrity check failed with %d workers", workers)
			}

			t.Logf("Workers=%d: %d -> %d bytes (%.2f%%)",
				workers, len(data), len(compressed),
				float64(len(compressed))*100/float64(len(data)))
		})
	}
}
