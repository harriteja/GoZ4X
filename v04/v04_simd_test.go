package v04

import (
	"bytes"
	"runtime"
	"testing"
	"time"

	"github.com/harriteja/GoZ4X/compress"
	"github.com/harriteja/GoZ4X/v04/simd"
)

// TestSIMDImplementationSelection tests that v0.4 correctly selects
// the appropriate SIMD implementation based on CPU features
func TestSIMDImplementationSelection(t *testing.T) {
	// Get the detected CPU features
	features := simd.DetectFeatures()
	t.Logf("CPU Features: SSE2=%v, SSE4.1=%v, AVX2=%v, AVX512=%v, NEON=%v",
		features.HasSSE2, features.HasSSE41, features.HasAVX2, features.HasAVX512, features.HasNEON)

	// Best SIMD implementation available
	bestImpl := simd.BestImplementation()
	t.Logf("Best SIMD implementation: %s (%d)", simd.ImplementationName(bestImpl), bestImpl)

	// Create sample data
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 64) // Simple repeating pattern
	}

	// Create different compression options with specific SIMD implementations
	opts := []struct {
		name    string
		options Options
	}{
		{"Default", DefaultOptions()},
		{"Generic", Options{Level: DefaultLevel, UseV2: true, SIMDImpl: simd.ImplGeneric}},
	}

	// Add architecture-specific implementations
	if runtime.GOARCH == "amd64" && features.HasSSE41 {
		opts = append(opts, struct {
			name    string
			options Options
		}{"SSE4.1", Options{Level: DefaultLevel, UseV2: true, SIMDImpl: simd.ImplSSE41}})
	}

	if runtime.GOARCH == "arm64" && features.HasNEON {
		opts = append(opts, struct {
			name    string
			options Options
		}{"NEON", Options{Level: DefaultLevel, UseV2: true, SIMDImpl: simd.ImplNEON}})
	}

	// Test compression with each implementation
	var results [][]byte

	for _, o := range opts {
		t.Run(o.name, func(t *testing.T) {
			compressed, err := CompressBlockWithOptions(data, nil, o.options)
			if err != nil {
				t.Fatalf("Compression with %s failed: %v", o.name, err)
			}

			// Store result for comparison
			results = append(results, compressed)

			// Verify decompression works
			decompressed, err := compress.DecompressBlock(compressed, nil, len(data))
			if err != nil {
				t.Fatalf("Decompression with %s failed: %v", o.name, err)
			}

			// Ensure compression is lossless
			if !bytes.Equal(decompressed, data) {
				t.Errorf("Data mismatch with %s implementation", o.name)
			}

			t.Logf("%s compressed size: %d bytes (%.2f%%)",
				o.name, len(compressed), float64(len(compressed))*100/float64(len(data)))
		})
	}

	// Compare results from different implementations
	// They should be identical or very similar in size
	if len(results) > 1 {
		baseSize := len(results[0])
		for i := 1; i < len(results); i++ {
			sizeRatio := float64(len(results[i])) / float64(baseSize)
			t.Logf("Size ratio %s vs %s: %.2f", opts[0].name, opts[i].name, sizeRatio)

			// Results should be reasonably close in size
			if sizeRatio < 0.8 || sizeRatio > 1.2 {
				t.Logf("Warning: Significant size difference between implementations")
			}
		}
	}
}

// TestParallelSIMDImplementation tests the parallel SIMD implementation
func TestParallelSIMDImplementation(t *testing.T) {
	// Create large sample data for parallel processing
	const dataSize = 4 * 1024 * 1024 // 4MB
	data := make([]byte, dataSize)

	// Fill with compressible data
	for i := range data {
		data[i] = byte((i * 7) % 251)
	}

	// Create options
	defaultOpts := DefaultOptions()

	// Test parallel compression with default options
	compressed, err := CompressBlockParallel(data, nil)
	if err != nil {
		t.Fatalf("Parallel compression failed: %v", err)
	}

	// Log compression stats
	t.Logf("Original size: %d bytes", len(data))
	t.Logf("Compressed size: %d bytes (%.2f%%)",
		len(compressed), float64(len(compressed))*100/float64(len(data)))

	// Verify we can decompress the data correctly
	decompressed, err := compress.DecompressBlock(compressed, nil, len(data))
	if err != nil {
		t.Fatalf("Failed to decompress parallel compressed data: %v", err)
	}

	// Verify the decompressed data matches the original
	if !bytes.Equal(data, decompressed) {
		t.Fatal("Data mismatch after parallel compression/decompression")
	}

	// Test with different worker counts
	workerCounts := []int{1, 2, 4, runtime.NumCPU()}
	for _, count := range workerCounts {
		t.Run("Workers-"+string(rune('0'+count)), func(t *testing.T) {
			opts := defaultOpts
			opts.NumWorkers = count

			start := time.Now()
			workerCompressed, err := CompressBlockParallelWithOptions(data, nil, opts)
			if err != nil {
				t.Fatalf("Compression with %d workers failed: %v", count, err)
			}
			elapsed := time.Since(start)

			t.Logf("Workers=%d: compressed %d bytes in %v (%.2f MB/s)",
				count, len(data), elapsed,
				float64(len(data))/elapsed.Seconds()/(1024*1024))

			// Verify the data can be decompressed
			_, err = compress.DecompressBlock(workerCompressed, nil, len(data))
			if err != nil {
				t.Fatalf("Failed to decompress data compressed with %d workers: %v", count, err)
			}
		})
	}
}
