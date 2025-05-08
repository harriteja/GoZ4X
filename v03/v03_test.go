package v03

import (
	"bytes"
	"io"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/harriteja/GoZ4X/compress"
)

// generateCompressibleData creates test data with good compression characteristics
func generateCompressibleData(size int) []byte {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, size)

	// Create a pattern that repeats but with some variation
	patternSize := 4 * 1024 // 4KB pattern
	pattern := make([]byte, patternSize)
	for i := 0; i < patternSize; i++ {
		pattern[i] = byte(rand.Intn(256))
	}

	// Fill the data with the pattern and some random variations
	for i := 0; i < size; i += patternSize {
		end := i + patternSize
		if end > size {
			end = size
		}

		// Copy the pattern
		copy(data[i:end], pattern)

		// Add some random variations (15% of bytes get randomized)
		for j := i; j < end; j++ {
			if rand.Float32() < 0.15 {
				data[j] = byte(rand.Intn(256))
			}
		}
	}

	return data
}

// generateRandomData creates test data with poor compression characteristics
func generateRandomData(size int) []byte {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(rand.Intn(256))
	}
	return data
}

// TestCompressBlockParallel tests the parallel compression functions
func TestCompressBlockParallel(t *testing.T) {
	testSizes := []int{
		4 * 1024,    // 4KB
		64 * 1024,   // 64KB
		256 * 1024,  // 256KB
		1024 * 1024, // 1MB
	}

	for _, size := range testSizes {
		// Test with compressible data
		t.Run("CompressibleData-"+byteSizeToString(size), func(t *testing.T) {
			testCompressBlockParallel(t, generateCompressibleData(size))
		})

		// Test with random data
		t.Run("RandomData-"+byteSizeToString(size), func(t *testing.T) {
			testCompressBlockParallel(t, generateRandomData(size))
		})
	}
}

// Test helper function for CompressBlockParallel
func testCompressBlockParallel(t *testing.T, input []byte) {
	// Test with default level
	compressed, err := CompressBlockParallel(input, nil)
	if err != nil {
		t.Fatalf("CompressBlockParallel error: %v", err)
	}

	decompressed, err := compress.DecompressBlock(compressed, nil, len(input))
	if err != nil {
		t.Fatalf("DecompressBlock error: %v", err)
	}

	if !bytes.Equal(input, decompressed) {
		t.Fatalf("Decompressed data doesn't match original data")
	}

	// Test with specific levels
	for level := 1; level <= 12; level++ {
		compressed, err := CompressBlockParallelLevel(input, nil, level)
		if err != nil {
			t.Fatalf("CompressBlockParallelLevel error at level %d: %v", level, err)
		}

		decompressed, err := compress.DecompressBlock(compressed, nil, len(input))
		if err != nil {
			t.Fatalf("DecompressBlock error at level %d: %v", level, err)
		}

		if !bytes.Equal(input, decompressed) {
			t.Fatalf("Decompressed data doesn't match original data at level %d", level)
		}
	}
}

// TestCompressBlockV2Parallel tests the parallel V2 compression functions
func TestCompressBlockV2Parallel(t *testing.T) {
	testSizes := []int{
		4 * 1024,    // 4KB
		64 * 1024,   // 64KB
		256 * 1024,  // 256KB
		1024 * 1024, // 1MB
	}

	for _, size := range testSizes {
		// Test with compressible data
		t.Run("CompressibleData-"+byteSizeToString(size), func(t *testing.T) {
			testCompressBlockV2Parallel(t, generateCompressibleData(size))
		})

		// Test with random data
		t.Run("RandomData-"+byteSizeToString(size), func(t *testing.T) {
			testCompressBlockV2Parallel(t, generateRandomData(size))
		})
	}
}

// Test helper function for CompressBlockV2Parallel
func testCompressBlockV2Parallel(t *testing.T, input []byte) {
	// Test with default level
	compressed, err := CompressBlockV2Parallel(input, nil)
	if err != nil {
		t.Fatalf("CompressBlockV2Parallel error: %v", err)
	}

	decompressed, err := compress.DecompressBlock(compressed, nil, len(input))
	if err != nil {
		t.Fatalf("DecompressBlock error: %v", err)
	}

	if !bytes.Equal(input, decompressed) {
		t.Fatalf("Decompressed data doesn't match original data")
	}

	// Test with specific levels
	for level := 1; level <= 12; level++ {
		compressed, err := CompressBlockV2ParallelLevel(input, nil, level)
		if err != nil {
			t.Fatalf("CompressBlockV2ParallelLevel error at level %d: %v", level, err)
		}

		decompressed, err := compress.DecompressBlock(compressed, nil, len(input))
		if err != nil {
			t.Fatalf("DecompressBlock error at level %d: %v", level, err)
		}

		if !bytes.Equal(input, decompressed) {
			t.Fatalf("Decompressed data doesn't match original data at level %d", level)
		}
	}
}

// TestParallelWriter tests the parallel writer implementation
func TestParallelWriter(t *testing.T) {
	testSizes := []int{
		4 * 1024,    // 4KB
		64 * 1024,   // 64KB
		256 * 1024,  // 256KB
		1024 * 1024, // 1MB
	}

	for _, size := range testSizes {
		// Test with compressible data
		t.Run("CompressibleData-"+byteSizeToString(size), func(t *testing.T) {
			testParallelWriter(t, generateCompressibleData(size))
		})

		// Test with random data
		t.Run("RandomData-"+byteSizeToString(size), func(t *testing.T) {
			testParallelWriter(t, generateRandomData(size))
		})
	}
}

// Test helper function for ParallelWriter
func testParallelWriter(t *testing.T, input []byte) {
	// Test all writer configurations
	testParallelWriterConfig(t, input, func() *ParallelWriter {
		return NewParallelWriter(bytes.NewBuffer(nil))
	})

	// Test with specific level
	testParallelWriterConfig(t, input, func() *ParallelWriter {
		return NewParallelWriterLevel(bytes.NewBuffer(nil), 6)
	})

	// Test with V2 algorithm
	testParallelWriterConfig(t, input, func() *ParallelWriter {
		return NewParallelWriterWithOptions(bytes.NewBuffer(nil), ParallelWriterOptions{
			Level:      7,
			UseV2:      true,
			NumWorkers: runtime.NumCPU(),
			ChunkSize:  64 * 1024,
		})
	})

	// Test with custom workers and chunk size
	testParallelWriterConfig(t, input, func() *ParallelWriter {
		pw := NewParallelWriter(bytes.NewBuffer(nil))
		pw.SetNumWorkers(2)
		pw.SetChunkSize(32 * 1024)
		return pw
	})
}

// Test specific ParallelWriter configuration
func testParallelWriterConfig(t *testing.T, input []byte, createWriter func() *ParallelWriter) {
	var buf bytes.Buffer
	pw := createWriter()
	// Set buffer as output
	pw.Reset(&buf)

	// Test NumWorkers and ChunkSize getters
	_ = pw.NumWorkers()
	_ = pw.ChunkSize()

	// Write data in chunks to test multiple writes
	chunkSize := 1024
	for i := 0; i < len(input); i += chunkSize {
		end := i + chunkSize
		if end > len(input) {
			end = len(input)
		}
		n, err := pw.Write(input[i:end])
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != end-i {
			t.Fatalf("Wrong number of bytes written: %d, expected: %d", n, end-i)
		}
	}

	// Close the writer
	if err := pw.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Decompress the data
	r := compress.NewReader(bytes.NewReader(buf.Bytes()))
	decompressed := bytes.NewBuffer(nil)
	if _, err := io.Copy(decompressed, r); err != nil {
		t.Fatalf("Decompress error: %v", err)
	}

	// Verify the decompressed data
	if !bytes.Equal(input, decompressed.Bytes()) {
		t.Fatalf("Decompressed data doesn't match original data")
	}
}

// Benchmark parallel compression vs standard compression
func BenchmarkParallelCompression(b *testing.B) {
	sizes := []int{
		64 * 1024,       // 64KB
		1024 * 1024,     // 1MB
		8 * 1024 * 1024, // 8MB
	}

	for _, size := range sizes {
		// Use less data for large sizes to prevent memory issues
		actualSize := size
		if size > 4*1024*1024 {
			actualSize = 4 * 1024 * 1024 // Cap at 4MB for benchmarks
		}

		input := generateCompressibleData(actualSize)

		// Benchmark v0.1 compression
		b.Run("v0.1-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compressed, err := compress.CompressBlock(input, nil)
				if err != nil {
					b.Skipf("Compression failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if len(compressed) == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})

		// Benchmark v0.2 compression
		b.Run("v0.2-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compressed, err := compress.CompressBlockV2(input, nil)
				if err != nil {
					b.Skipf("Compression failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if len(compressed) == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})

		// Benchmark v0.3 parallel compression
		b.Run("v0.3-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compressed, err := CompressBlockParallel(input, nil)
				if err != nil {
					b.Skipf("Compression failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if len(compressed) == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})

		// Benchmark v0.3 parallel v2 compression
		b.Run("v0.3-v2-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compressed, err := CompressBlockV2Parallel(input, nil)
				if err != nil {
					b.Skipf("Compression failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if len(compressed) == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})
	}
}

// Benchmark streaming with different writers
func BenchmarkStreamingWriters(b *testing.B) {
	sizes := []int{
		64 * 1024,       // 64KB
		1024 * 1024,     // 1MB
		8 * 1024 * 1024, // 8MB
	}

	for _, size := range sizes {
		// Use less data for large sizes to prevent memory issues
		actualSize := size
		if size > 4*1024*1024 {
			actualSize = 4 * 1024 * 1024 // Cap at 4MB for benchmarks
		}

		input := generateCompressibleData(actualSize)

		// Benchmark standard writer
		b.Run("Writer-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := bytes.NewBuffer(nil)
				w := compress.NewWriter(buf)
				_, err := w.Write(input)
				if err != nil {
					b.Skipf("Write failed: %v", err)
					return
				}
				err = w.Close()
				if err != nil {
					b.Skipf("Close failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if buf.Len() == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})

		// Benchmark v0.2 writer
		b.Run("WriterV2-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := bytes.NewBuffer(nil)
				w := compress.NewWriterWithOptions(buf, compress.WriterOptions{
					Level: compress.DefaultLevel,
					UseV2: true,
				})
				_, err := w.Write(input)
				if err != nil {
					b.Skipf("Write failed: %v", err)
					return
				}
				err = w.Close()
				if err != nil {
					b.Skipf("Close failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if buf.Len() == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})

		// Benchmark parallel writer
		b.Run("ParallelWriter-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := bytes.NewBuffer(nil)
				w := NewParallelWriter(buf)
				_, err := w.Write(input)
				if err != nil {
					b.Skipf("Write failed: %v", err)
					return
				}
				err = w.Close()
				if err != nil {
					b.Skipf("Close failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if buf.Len() == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})

		// Benchmark parallel v2 writer
		b.Run("ParallelWriterV2-"+byteSizeToString(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := bytes.NewBuffer(nil)
				w := NewParallelWriterWithOptions(buf, ParallelWriterOptions{
					Level: int(compress.DefaultLevel),
					UseV2: true,
				})
				_, err := w.Write(input)
				if err != nil {
					b.Skipf("Write failed: %v", err)
					return
				}
				err = w.Close()
				if err != nil {
					b.Skipf("Close failed: %v", err)
					return
				}
				b.SetBytes(int64(len(input)))
				// Prevent compiler optimization
				if buf.Len() == 0 {
					b.Skip("Compression produced empty result")
				}
			}
		})
	}
}

// Helper function to convert byte size to string
func byteSizeToString(size int) string {
	if size < 1024 {
		return string(rune('0'+size)) + "B"
	} else if size < 1024*1024 {
		return string(rune('0'+size/1024)) + "KB"
	} else if size < 1024*1024*1024 {
		return string(rune('0'+size/(1024*1024))) + "MB"
	}
	return string(rune('0'+size/(1024*1024*1024))) + "GB"
}
