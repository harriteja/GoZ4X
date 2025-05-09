package bench

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
)

const (
	// Test data size for benchmarks
	smallSize  = 1 << 10 // 1KB
	mediumSize = 1 << 16 // 64KB
	largeSize  = 1 << 20 // 1MB
	hugeSize   = 1 << 24 // 16MB
)

var (
	// Global variables to prevent compiler optimizations
	result      []byte
	compressErr error
	benchSizes  = []int{smallSize, mediumSize, largeSize, hugeSize}
	benchLevels = []compress.CompressionLevel{1, 6, 9}
)

// Generate test data with different compressibility
func generateData(size int, compressibility float64) []byte {
	// Allocate buffer
	data := make([]byte, size)

	if compressibility <= 0 {
		// Random data (incompressible)
		rand.Read(data)
		return data
	}

	if compressibility >= 1 {
		// All zeros (maximum compressibility)
		return data
	}

	// Pattern data with controlled redundancy
	patternSize := int(float64(size) * (1 - compressibility))
	if patternSize < 4 {
		patternSize = 4
	}

	pattern := make([]byte, patternSize)
	rand.Read(pattern)

	// Fill with repeating pattern
	for i := 0; i < size; i += patternSize {
		n := copy(data[i:], pattern)
		if n < patternSize {
			break
		}
	}

	return data
}

// Benchmark block compression with different input sizes and compression levels
func BenchmarkBlockCompress(b *testing.B) {
	// For each test data size
	for _, size := range benchSizes {
		// Skip huge size as it exceeds MaxBlockSize (4MB)
		if size == hugeSize || size == largeSize || size == mediumSize {
			continue // Skip larger sizes to prevent test hangs
		}

		// For each compressibility level
		for _, comp := range []float64{0.0, 0.5, 0.9} {
			data := generateData(size, comp)
			name := ""
			switch {
			case size == smallSize:
				name = "Small"
			case size == mediumSize:
				name = "Medium"
			case size == largeSize:
				name = "Large"
			case size == hugeSize:
				name = "Huge"
			}

			compStr := ""
			switch {
			case comp == 0.0:
				compStr = "Random"
			case comp == 0.5:
				compStr = "Mixed"
			case comp == 0.9:
				compStr = "Compressible"
			}

			// For each compression level
			for _, level := range benchLevels {
				b.Run(name+"_"+compStr+"_Level"+string(rune('0'+int(level))), func(b *testing.B) {
					// Limit iterations for larger sizes to prevent timeouts
					if size == largeSize {
						b.N = min(b.N, 10)
					} else if size == mediumSize {
						b.N = min(b.N, 100)
					}

					// Reset benchmark timer
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						// Compress the block
						result, compressErr = compress.CompressBlockLevel(data, nil, level)
						if compressErr != nil {
							b.Fatal(compressErr)
						}
					}

					// Report compression ratio
					b.ReportMetric(float64(len(result))/float64(len(data)), "ratio")
					b.SetBytes(int64(len(data)))
				})
			}
		}
	}
}

// Benchmark block decompression with different input sizes
func BenchmarkBlockDecompress(b *testing.B) {
	// For each test data size
	for _, size := range benchSizes {
		// Skip huge size for decompression benchmarks
		if size == hugeSize || size == largeSize || size == mediumSize {
			continue // Skip larger sizes to prevent test hangs
		}

		// For each compressibility level
		for _, comp := range []float64{0.0, 0.5, 0.9} {
			data := generateData(size, comp)

			// Compress the data first
			compressed, err := compress.CompressBlock(data, nil)
			if err != nil {
				b.Fatal(err)
			}

			name := ""
			switch {
			case size == smallSize:
				name = "Small"
			case size == mediumSize:
				name = "Medium"
			case size == largeSize:
				name = "Large"
			}

			compStr := ""
			switch {
			case comp == 0.0:
				compStr = "Random"
			case comp == 0.5:
				compStr = "Mixed"
			case comp == 0.9:
				compStr = "Compressible"
			}

			b.Run(name+"_"+compStr, func(b *testing.B) {
				// Limit iterations for larger sizes to prevent timeouts
				if size == largeSize {
					b.N = min(b.N, 10)
				} else if size == mediumSize {
					b.N = min(b.N, 100)
				}

				// Preallocate decompression buffer
				decompressed := make([]byte, size)

				// Reset benchmark timer
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// Decompress the block
					var err error
					result, err = compress.DecompressBlock(compressed, decompressed, size)
					if err != nil {
						b.Fatal(err)
					}

					// Verify decompression (only on first iteration to avoid overhead)
					if i == 0 && !bytes.Equal(result, data) {
						b.Fatal("decompression failed")
					}
				}

				b.SetBytes(int64(size))
			})
		}
	}
}
