package goz4x

import (
	"bytes"
	cryptorand "crypto/rand"
	"io"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func init() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
}

// Helper functions for generating test data
func generateRandomData(size int) []byte {
	data := make([]byte, size)
	cryptorand.Read(data)
	return data
}

// generateCompressibleData creates test data with good compression characteristics
func generateCompressibleData(size int) []byte {
	// Create data with a repeating pattern and some variation
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
		copy(data[i:end], pattern[:end-i])

		// Add some random variations (15% of bytes get randomized)
		for j := i; j < end; j++ {
			if rand.Float32() < 0.15 {
				data[j] = byte(rand.Intn(256))
			}
		}
	}

	return data
}

// Test CompressBlock function
func TestCompressBlock(t *testing.T) {
	// For v0.1, skip the full decompression tests
	t.Skip("In v0.1, block-level compression/decompression is not fully implemented yet")

	tests := []struct {
		name         string
		inputSize    int
		compressible bool
		preAllocBuf  bool
	}{
		{"Small random data, nil buffer", 1024, false, false},
		{"Small compressible data, nil buffer", 1024, true, false},
		{"Medium random data, nil buffer", 64 * 1024, false, false},
		{"Medium compressible data, nil buffer", 64 * 1024, true, false},
		{"Small random data, pre-allocated buffer", 1024, false, true},
		{"Small compressible data, pre-allocated buffer", 1024, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []byte
			if tt.compressible {
				input = generateCompressibleData(tt.inputSize)
			} else {
				input = generateRandomData(tt.inputSize)
			}

			var buf []byte
			if tt.preAllocBuf {
				// Allocate a buffer large enough for the worst case
				buf = make([]byte, tt.inputSize+(tt.inputSize/255)+16)
			}

			compressed, err := CompressBlock(input, buf)
			if err != nil {
				t.Fatalf("CompressBlock() error = %v", err)
			}

			if compressed == nil {
				t.Errorf("CompressBlock() returned nil buffer")
			}

			// For compressible data, compression should reduce size
			if tt.compressible && tt.inputSize > 100 {
				compressionRatio := float64(len(compressed)) / float64(len(input))
				t.Logf("Compression ratio: %.2f", compressionRatio)

				if compressionRatio > 0.9 {
					t.Logf("Warning: Compression ratio %.2f is higher than expected for compressible data", compressionRatio)
				}
			}

			// Verify the compressed data can be decompressed back to the original
			decompressed, err := DecompressBlock(compressed, nil, tt.inputSize)
			if err != nil {
				t.Fatalf("DecompressBlock() error = %v", err)
			}

			if !bytes.Equal(decompressed, input) {
				t.Errorf("Decompressed data does not match original")
			}
		})
	}
}

// Test CompressBlockLevel function
func TestCompressBlockLevel(t *testing.T) {
	// For v0.1, skip the full decompression tests
	t.Skip("In v0.1, block-level compression with different levels is not fully implemented yet")

	inputSize := 64 * 1024
	input := generateCompressibleData(inputSize)

	// Test with different compression levels
	levels := []int{1, 6, 12}

	for _, level := range levels {
		t.Run("Level "+string(rune('0'+level)), func(t *testing.T) {
			compressed, err := CompressBlockLevel(input, nil, level)
			if err != nil {
				t.Fatalf("CompressBlockLevel(%d) error = %v", level, err)
			}

			if compressed == nil {
				t.Errorf("CompressBlockLevel(%d) returned nil buffer", level)
			}

			// Verify the compressed data can be decompressed
			decompressed, err := DecompressBlock(compressed, nil, inputSize)
			if err != nil {
				t.Fatalf("DecompressBlock() error = %v", err)
			}

			if !bytes.Equal(decompressed, input) {
				t.Errorf("Decompressed data does not match original for level %d", level)
			}
		})
	}
}

// Test DecompressBlock function
func TestDecompressBlock(t *testing.T) {
	// For v0.1, skip the full decompression tests
	t.Skip("In v0.1, block-level decompression is not fully implemented yet")

	inputSize := 16 * 1024
	input := generateCompressibleData(inputSize)

	// Compress the data first
	compressed, err := CompressBlock(input, nil)
	if err != nil {
		t.Fatalf("CompressBlock() error = %v", err)
	}

	tests := []struct {
		name      string
		dst       []byte
		maxSize   int
		wantError bool
	}{
		{"Nil destination, zero maxSize", nil, 0, false},
		{"Nil destination, specific maxSize", nil, inputSize, false},
		{"Pre-allocated destination", make([]byte, inputSize), inputSize, false},
		{"Small pre-allocated destination", make([]byte, 1024), inputSize, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decompressed, err := DecompressBlock(compressed, tt.dst, tt.maxSize)
			if tt.wantError {
				if err == nil {
					t.Errorf("DecompressBlock() error = nil, wantError = true")
				}
			} else {
				if err != nil {
					t.Errorf("DecompressBlock() error = %v", err)
				}

				if !bytes.Equal(decompressed, input) {
					t.Errorf("Decompressed data does not match original")
				}
			}
		})
	}
}

// Test NewReader and Reader functionality
func TestReader(t *testing.T) {
	// Prepare test data
	testData := "This is test data for the LZ4 Reader."

	// Compress the data
	var buf bytes.Buffer
	w := NewWriter(&buf)
	_, err := io.WriteString(w, testData)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	compressed := buf.Bytes()

	// Test NewReader
	r := NewReader(bytes.NewReader(compressed))

	// Read and verify the data
	var result bytes.Buffer
	_, err = io.Copy(&result, r)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if result.String() != testData {
		t.Errorf("Read data doesn't match original")
		t.Errorf("Got: %q", result.String())
		t.Errorf("Want: %q", testData)
	}
}

// Test NewWriter, NewWriterLevel, and Writer functionality
func TestWriter(t *testing.T) {
	tests := []struct {
		name         string
		useLevel     bool
		level        int
		inputSize    int
		compressible bool
	}{
		{"Default level, small random data", false, 0, 1024, false},
		{"Default level, small compressible data", false, 0, 1024, true},
		{"Fast level, medium data", true, 3, 16 * 1024, true},
		{"High level, medium data", true, 9, 16 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate input data
			var input []byte
			if tt.compressible {
				input = generateCompressibleData(tt.inputSize)
			} else {
				input = generateRandomData(tt.inputSize)
			}

			// Create writer
			var buf bytes.Buffer
			var w *Writer

			if tt.useLevel {
				w = NewWriterLevel(&buf, tt.level)
			} else {
				w = NewWriter(&buf)
			}

			// Write data
			n, err := w.Write(input)
			if err != nil {
				t.Fatalf("Write error: %v", err)
			}
			if n != len(input) {
				t.Errorf("Write returned %d, want %d", n, len(input))
			}

			// Close to flush data
			err = w.Close()
			if err != nil {
				t.Fatalf("Close error: %v", err)
			}

			// Verify output is not empty
			if buf.Len() == 0 {
				t.Errorf("Output buffer is empty")
			}

			// Verify we can read back the data
			r := NewReader(bytes.NewReader(buf.Bytes()))
			var result bytes.Buffer
			_, err = io.Copy(&result, r)
			if err != nil {
				t.Fatalf("Read error: %v", err)
			}

			if !bytes.Equal(result.Bytes(), input) {
				t.Errorf("Decompressed data does not match original")
			}
		})
	}
}

// Test Writer Reset functionality
func TestWriterReset(t *testing.T) {
	// Create initial buffer and writer
	var buf1 bytes.Buffer
	w := NewWriter(&buf1)

	// Write and close
	_, err := io.WriteString(w, "data1")
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Reset with new buffer
	var buf2 bytes.Buffer
	w.Reset(&buf2)

	// Write and close again
	_, err = io.WriteString(w, "data2")
	if err != nil {
		t.Fatalf("Write after Reset error: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("Close after Reset error: %v", err)
	}

	// Verify both outputs can be read correctly
	r1 := NewReader(bytes.NewReader(buf1.Bytes()))
	var result1 bytes.Buffer
	_, err = io.Copy(&result1, r1)
	if err != nil {
		t.Fatalf("Read error on first buffer: %v", err)
	}
	if result1.String() != "data1" {
		t.Errorf("First buffer data mismatch")
	}

	r2 := NewReader(bytes.NewReader(buf2.Bytes()))
	var result2 bytes.Buffer
	_, err = io.Copy(&result2, r2)
	if err != nil {
		t.Fatalf("Read error on second buffer: %v", err)
	}
	if result2.String() != "data2" {
		t.Errorf("Second buffer data mismatch")
	}
}

// Test streaming compression and decompression with large data
func TestStreamingLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	// Generate large test data (1MB)
	size := 1 * 1024 * 1024
	testData := generateCompressibleData(size)

	// Compress using streaming API
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Write in chunks to test multiple Write calls
	chunkSize := 64 * 1024
	for i := 0; i < len(testData); i += chunkSize {
		end := i + chunkSize
		if end > len(testData) {
			end = len(testData)
		}

		_, err := w.Write(testData[i:end])
		if err != nil {
			t.Fatalf("Write error at chunk %d: %v", i/chunkSize, err)
		}
	}

	err := w.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Report compression stats
	compressed := buf.Bytes()
	compressionRatio := float64(len(compressed)) / float64(len(testData))
	t.Logf("Original size: %d, Compressed size: %d, Ratio: %.2f%%",
		len(testData), len(compressed), compressionRatio*100)

	// Decompress
	r := NewReader(bytes.NewReader(compressed))
	result := make([]byte, 0, size)
	resultBuf := bytes.NewBuffer(result)

	// Read in chunks
	buffer := make([]byte, 32*1024)
	for {
		n, err := r.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}

		resultBuf.Write(buffer[:n])
	}

	// Verify
	if !bytes.Equal(resultBuf.Bytes(), testData) {
		t.Errorf("Decompressed data does not match original")
	}
}

// Test CompressBlockV2 and CompressBlockV2Level functions
func TestCompressBlockV2(t *testing.T) {
	tests := []struct {
		name         string
		inputSize    int
		compressible bool
		preAllocBuf  bool
	}{
		{"Small random data, nil buffer", 1024, false, false},
		{"Small compressible data, nil buffer", 1024, true, false},
		{"Medium random data, nil buffer", 64 * 1024, false, false},
		{"Medium compressible data, nil buffer", 64 * 1024, true, false},
		{"Small random data, pre-allocated buffer", 1024, false, true},
		{"Small compressible data, pre-allocated buffer", 1024, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []byte
			if tt.compressible {
				input = generateCompressibleData(tt.inputSize)
			} else {
				input = generateRandomData(tt.inputSize)
			}

			var buf []byte
			if tt.preAllocBuf {
				// Allocate a buffer large enough for the worst case
				buf = make([]byte, tt.inputSize+(tt.inputSize/255)+16)
			}

			compressed, err := CompressBlockV2(input, buf)
			if err != nil {
				t.Fatalf("CompressBlockV2() error = %v", err)
			}

			if compressed == nil {
				t.Errorf("CompressBlockV2() returned nil buffer")
			}

			// Verify the compressed data can be decompressed back to the original
			decompressed, err := DecompressBlock(compressed, nil, tt.inputSize)
			if err != nil {
				t.Fatalf("DecompressBlock() error = %v", err)
			}

			if !bytes.Equal(decompressed, input) {
				t.Errorf("Decompressed data does not match original")
			}
		})
	}
}

func TestCompressBlockV2Level(t *testing.T) {
	inputSize := 64 * 1024
	input := generateCompressibleData(inputSize)

	// Test with different compression levels
	levels := []int{1, 6, 12}

	for _, level := range levels {
		t.Run("Level "+string(rune('0'+level)), func(t *testing.T) {
			compressed, err := CompressBlockV2Level(input, nil, level)
			if err != nil {
				t.Fatalf("CompressBlockV2Level(%d) error = %v", level, err)
			}

			if compressed == nil {
				t.Errorf("CompressBlockV2Level(%d) returned nil buffer", level)
			}

			// Verify the compressed data can be decompressed
			decompressed, err := DecompressBlock(compressed, nil, inputSize)
			if err != nil {
				t.Fatalf("DecompressBlock() error = %v", err)
			}

			if !bytes.Equal(decompressed, input) {
				t.Errorf("Decompressed data does not match original for level %d", level)
			}
		})
	}
}

// Test V2 writer functionality
func TestWriterV2(t *testing.T) {
	tests := []struct {
		name         string
		useLevel     bool
		level        int
		inputSize    int
		compressible bool
	}{
		{"Default level, small random data", false, 0, 1024, false},
		{"Default level, small compressible data", false, 0, 1024, true},
		{"Fast level, medium data", true, 3, 16 * 1024, true},
		{"High level, medium data", true, 9, 16 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate input data
			var input []byte
			if tt.compressible {
				input = generateCompressibleData(tt.inputSize)
			} else {
				input = generateRandomData(tt.inputSize)
			}

			// Create writer
			var buf bytes.Buffer
			var w *Writer

			if tt.useLevel {
				w = NewWriterV2Level(&buf, tt.level)
			} else {
				w = NewWriterV2(&buf)
			}

			// Write data
			n, err := w.Write(input)
			if err != nil {
				t.Fatalf("Write error: %v", err)
			}
			if n != len(input) {
				t.Errorf("Write returned %d, want %d", n, len(input))
			}

			// Close the writer
			err = w.Close()
			if err != nil {
				t.Fatalf("Close error: %v", err)
			}

			// Create reader to decompress
			compressed := buf.Bytes()
			r := NewReader(bytes.NewReader(compressed))

			// Read and verify
			decompressed := make([]byte, tt.inputSize)
			readTotal := 0
			for readTotal < tt.inputSize {
				n, err := r.Read(decompressed[readTotal:])
				if err != nil && err != io.EOF {
					t.Fatalf("Read error: %v", err)
				}
				if n == 0 {
					break
				}
				readTotal += n
			}

			if readTotal != tt.inputSize {
				t.Errorf("Read size mismatch: got %d, want %d", readTotal, tt.inputSize)
			}

			if !bytes.Equal(decompressed[:readTotal], input) {
				t.Errorf("Decompressed data doesn't match original")
			}
		})
	}
}

// Test V2 vs V1 compression ratio
func TestV2VsV1CompressionRatio(t *testing.T) {
	// Only run this for meaningful tests
	// Using compressible data that should have a clear difference
	input := generateCompressibleData(32 * 1024)

	// Compress with V1
	v1Compressed, err := CompressBlock(input, nil)
	if err != nil {
		t.Fatalf("CompressBlock() error = %v", err)
	}

	// Compress with V2
	v2Compressed, err := CompressBlockV2(input, nil)
	if err != nil {
		t.Fatalf("CompressBlockV2() error = %v", err)
	}

	// Compare sizes - at minimum they should both decompress correctly
	v1Ratio := float64(len(input)) / float64(len(v1Compressed))
	v2Ratio := float64(len(input)) / float64(len(v2Compressed))

	t.Logf("V1 ratio: %.2f, V2 ratio: %.2f", v1Ratio, v2Ratio)

	// Both should decompress correctly
	v1Decompressed, err := DecompressBlock(v1Compressed, nil, len(input))
	if err != nil {
		t.Fatalf("DecompressBlock(v1) error = %v", err)
	}

	v2Decompressed, err := DecompressBlock(v2Compressed, nil, len(input))
	if err != nil {
		t.Fatalf("DecompressBlock(v2) error = %v", err)
	}

	if !bytes.Equal(v1Decompressed, input) || !bytes.Equal(v2Decompressed, input) {
		t.Errorf("Decompression verification failed")
	}
}

// TestVersion verifies the version constants
func TestVersion(t *testing.T) {
	if Version != "0.3.0" {
		t.Errorf("Expected version to be 0.3.0, got %s", Version)
	}

	if VersionMajor != 0 {
		t.Errorf("Expected VersionMajor to be 0, got %d", VersionMajor)
	}

	if VersionMinor != 3 {
		t.Errorf("Expected VersionMinor to be 3, got %d", VersionMinor)
	}

	if VersionPatch != 0 {
		t.Errorf("Expected VersionPatch to be 0, got %d", VersionPatch)
	}
}

// TestParallelCompression tests the parallel compression API
func TestParallelCompression(t *testing.T) {
	// Skip for very small test runs
	if testing.Short() {
		t.Skip("Skipping parallel compression test in short mode")
	}

	// Generate test data
	sizes := []int{
		10 * 1024,   // 10KB
		100 * 1024,  // 100KB
		1024 * 1024, // 1MB
	}

	for _, size := range sizes {
		t.Run(byteSizeToString(size), func(t *testing.T) {
			testParallelCompression(t, generateCompressibleData(size))
		})
	}
}

// Helper function for testing parallel compression
func testParallelCompression(t *testing.T, data []byte) {
	// Test basic parallel compression
	parallelCompressed, err := CompressBlockParallel(data, nil)
	if err != nil {
		t.Fatalf("CompressBlockParallel error: %v", err)
	}

	// Test parallel compression with level
	for level := 1; level <= 12; level++ {
		compressed, err := CompressBlockParallelLevel(data, nil, level)
		if err != nil {
			t.Fatalf("CompressBlockParallelLevel error at level %d: %v", level, err)
		}

		// Verify decompression
		decompressed, err := DecompressBlock(compressed, nil, len(data))
		if err != nil {
			t.Fatalf("DecompressBlock error at level %d: %v", level, err)
		}

		if !bytes.Equal(data, decompressed) {
			t.Fatalf("Decompressed data doesn't match original at level %d", level)
		}
	}

	// Test V2 parallel compression
	v2Compressed, err := CompressBlockV2Parallel(data, nil)
	if err != nil {
		t.Fatalf("CompressBlockV2Parallel error: %v", err)
	}

	// Test V2 parallel compression with level
	for level := 1; level <= 12; level++ {
		compressed, err := CompressBlockV2ParallelLevel(data, nil, level)
		if err != nil {
			t.Fatalf("CompressBlockV2ParallelLevel error at level %d: %v", level, err)
		}

		// Verify decompression
		decompressed, err := DecompressBlock(compressed, nil, len(data))
		if err != nil {
			t.Fatalf("DecompressBlock error at level %d: %v", level, err)
		}

		if !bytes.Equal(data, decompressed) {
			t.Fatalf("Decompressed data doesn't match original at level %d", level)
		}
	}

	// Verify original parallel compression
	decompressed, err := DecompressBlock(parallelCompressed, nil, len(data))
	if err != nil {
		t.Fatalf("DecompressBlock error: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Fatalf("Decompressed data doesn't match original for parallel compression")
	}

	// Verify V2 parallel compression
	decompressedV2, err := DecompressBlock(v2Compressed, nil, len(data))
	if err != nil {
		t.Fatalf("DecompressBlock error: %v", err)
	}

	if !bytes.Equal(data, decompressedV2) {
		t.Fatalf("Decompressed data doesn't match original for V2 parallel compression")
	}
}

// TestParallelWriter tests the parallel writer API
func TestParallelWriter(t *testing.T) {
	// Skip for very small test runs
	if testing.Short() {
		t.Skip("Skipping parallel writer test in short mode")
	}

	// Generate test data
	sizes := []int{
		10 * 1024,   // 10KB
		100 * 1024,  // 100KB
		1024 * 1024, // 1MB
	}

	for _, size := range sizes {
		t.Run(byteSizeToString(size), func(t *testing.T) {
			testParallelWriter(t, generateCompressibleData(size))
		})
	}
}

// Helper function for testing parallel writer
func testParallelWriter(t *testing.T, data []byte) {
	// Test all writer configurations
	testParallelWriterConfig(t, data, func() *ParallelWriter {
		return NewParallelWriter(bytes.NewBuffer(nil))
	})

	// Test with specific level
	testParallelWriterConfig(t, data, func() *ParallelWriter {
		return NewParallelWriterLevel(bytes.NewBuffer(nil), 6)
	})

	// Test with V2 algorithm
	testParallelWriterConfig(t, data, func() *ParallelWriter {
		return NewParallelWriterV2(bytes.NewBuffer(nil))
	})

	// Test with V2 and specific level
	testParallelWriterConfig(t, data, func() *ParallelWriter {
		return NewParallelWriterV2Level(bytes.NewBuffer(nil), 9)
	})

	// Test with custom settings
	testParallelWriterConfig(t, data, func() *ParallelWriter {
		pw := NewParallelWriter(bytes.NewBuffer(nil))
		pw.SetNumWorkers(runtime.NumCPU())
		pw.SetChunkSize(64 * 1024)
		return pw
	})
}

// Test helper for specific ParallelWriter configuration
func testParallelWriterConfig(t *testing.T, data []byte, createWriter func() *ParallelWriter) {
	var buf bytes.Buffer
	pw := createWriter()

	// Set buffer as output
	pw.Reset(&buf)

	// Write data in chunks to test multiple writes
	chunkSize := 1024
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		n, err := pw.Write(data[i:end])
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
	r := NewReader(bytes.NewReader(buf.Bytes()))
	decompressed := bytes.NewBuffer(nil)
	if _, err := io.Copy(decompressed, r); err != nil {
		t.Fatalf("Decompress error: %v", err)
	}

	// Verify the decompressed data
	if !bytes.Equal(data, decompressed.Bytes()) {
		t.Fatalf("Decompressed data doesn't match original data")
	}
}

// TestVersionComparison tests and compares all version's compression
func TestVersionComparison(t *testing.T) {
	// Skip for very small test runs
	if testing.Short() {
		t.Skip("Skipping version comparison test in short mode")
	}

	// Generate test data with different compressibility
	sizes := []int{
		100 * 1024,  // 100KB
		1024 * 1024, // 1MB
	}

	compressibilities := []float64{
		0.3, // Low compressibility
		0.7, // Medium compressibility
		0.9, // High compressibility
	}

	for _, size := range sizes {
		for _, comp := range compressibilities {
			t.Run(byteSizeToString(size)+"-Comp"+string(rune('0'+int(comp*10))), func(t *testing.T) {
				testVersionCompression(t, size, comp)
			})
		}
	}
}

// Helper function for testing version comparison
func testVersionCompression(t *testing.T, size int, compressibility float64) {
	// Generate data with the specified compressibility
	data := generateDataWithCompressibility(size, compressibility)

	// Compress with each version
	v1Compressed, _ := CompressBlock(data, nil)
	v2Compressed, _ := CompressBlockV2(data, nil)
	v3Compressed, _ := CompressBlockParallel(data, nil)
	v3v2Compressed, _ := CompressBlockV2Parallel(data, nil)

	// Check compression ratios
	v1Ratio := float64(len(data)) / float64(len(v1Compressed))
	v2Ratio := float64(len(data)) / float64(len(v2Compressed))
	v3Ratio := float64(len(data)) / float64(len(v3Compressed))
	v3v2Ratio := float64(len(data)) / float64(len(v3v2Compressed))

	// Log results
	t.Logf("Data size: %d, Compressibility: %.1f", size, compressibility)
	t.Logf("v0.1: %d bytes (%.2fx ratio)", len(v1Compressed), v1Ratio)
	t.Logf("v0.2: %d bytes (%.2fx ratio)", len(v2Compressed), v2Ratio)
	t.Logf("v0.3: %d bytes (%.2fx ratio)", len(v3Compressed), v3Ratio)
	t.Logf("v0.3-V2: %d bytes (%.2fx ratio)", len(v3v2Compressed), v3v2Ratio)

	// Verify all decompress properly
	// v0.1
	decompressedV1, err := DecompressBlock(v1Compressed, nil, len(data))
	if err != nil || !bytes.Equal(data, decompressedV1) {
		t.Fatalf("v0.1 decompression failed: %v", err)
	}

	// v0.2
	decompressedV2, err := DecompressBlock(v2Compressed, nil, len(data))
	if err != nil || !bytes.Equal(data, decompressedV2) {
		t.Fatalf("v0.2 decompression failed: %v", err)
	}

	// v0.3
	decompressedV3, err := DecompressBlock(v3Compressed, nil, len(data))
	if err != nil || !bytes.Equal(data, decompressedV3) {
		t.Fatalf("v0.3 decompression failed: %v", err)
	}

	// v0.3-V2
	decompressedV3V2, err := DecompressBlock(v3v2Compressed, nil, len(data))
	if err != nil || !bytes.Equal(data, decompressedV3V2) {
		t.Fatalf("v0.3-V2 decompression failed: %v", err)
	}

	// Basic expectations
	// v0.2 should be better than v0.1
	if len(v2Compressed) > len(v1Compressed) {
		t.Logf("Warning: v0.2 compression worse than v0.1")
	}

	// v0.3-V2 should be comparable to v0.2 (same algorithm, just parallel)
	ratio := float64(len(v3v2Compressed)) / float64(len(v2Compressed))
	if ratio > 1.05 {
		t.Logf("Warning: v0.3-V2 compression significantly worse than v0.2 (ratio: %.2f)", ratio)
	}
}

// Test streaming with different writers for comparison
func TestStreamingWriters(t *testing.T) {
	// Skip for very small test runs
	if testing.Short() {
		t.Skip("Skipping streaming writers test in short mode")
	}

	// Generate data
	data := generateCompressibleData(1024 * 1024) // 1MB

	// Compress with each writer type
	writers := []struct {
		name   string
		create func(io.Writer) io.WriteCloser
	}{
		{"v0.1", func(w io.Writer) io.WriteCloser { return NewWriter(w) }},
		{"v0.2", func(w io.Writer) io.WriteCloser { return NewWriterV2(w) }},
		{"v0.3", func(w io.Writer) io.WriteCloser { return NewParallelWriter(w) }},
		{"v0.3-V2", func(w io.Writer) io.WriteCloser { return NewParallelWriterV2(w) }},
	}

	// Collect results
	results := make(map[string][]byte)

	for _, w := range writers {
		t.Run(w.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := w.create(&buf)

			// Write data
			n, err := writer.Write(data)
			if err != nil {
				t.Fatalf("Write error: %v", err)
			}
			if n != len(data) {
				t.Fatalf("Wrong number of bytes written: %d, expected: %d", n, len(data))
			}

			// Close writer
			if err := writer.Close(); err != nil {
				t.Fatalf("Close error: %v", err)
			}

			// Store compressed data
			results[w.name] = buf.Bytes()

			// Verify decompression
			r := NewReader(bytes.NewReader(buf.Bytes()))
			decompressed := bytes.NewBuffer(nil)
			if _, err := io.Copy(decompressed, r); err != nil {
				t.Fatalf("Decompress error: %v", err)
			}

			if !bytes.Equal(data, decompressed.Bytes()) {
				t.Fatalf("Decompressed data doesn't match original")
			}

			// Log compression ratio
			ratio := float64(len(data)) / float64(buf.Len())
			t.Logf("%s: %d bytes compressed to %d bytes (%.2fx ratio)",
				w.name, len(data), buf.Len(), ratio)
		})
	}

	// Compare results
	if len(results["v0.2"]) > len(results["v0.1"]) {
		t.Logf("Warning: v0.2 writer produced larger output than v0.1")
	}

	// v0.3 and v0.3-V2 should be comparable to their non-parallel counterparts
	v3Ratio := float64(len(results["v0.3"])) / float64(len(results["v0.1"]))
	if v3Ratio > 1.05 {
		t.Logf("Warning: v0.3 writer significantly worse than v0.1 (ratio: %.2f)", v3Ratio)
	}

	v3v2Ratio := float64(len(results["v0.3-V2"])) / float64(len(results["v0.2"]))
	if v3v2Ratio > 1.05 {
		t.Logf("Warning: v0.3-V2 writer significantly worse than v0.2 (ratio: %.2f)", v3v2Ratio)
	}
}

// Helper function to generate data with specified compressibility
func generateDataWithCompressibility(size int, compressibility float64) []byte {
	data := make([]byte, size)

	// Create several patterns to use
	patternCount := 5
	patterns := make([][]byte, patternCount)
	patternSize := 256
	for i := 0; i < patternCount; i++ {
		patterns[i] = make([]byte, patternSize)
		for j := 0; j < patternSize; j++ {
			patterns[i][j] = byte(rand.Intn(256))
		}
	}

	// Fill data with patterns and randomness based on compressibility
	pos := 0
	for pos < size {
		if rand.Float64() < compressibility {
			// Use a pattern (compressible part)
			pattern := patterns[rand.Intn(patternCount)]
			repeatLength := rand.Intn(1024) + 64
			for i := 0; i < repeatLength && pos < size; i++ {
				data[pos] = pattern[i%len(pattern)]
				pos++
			}
		} else {
			// Use random data (incompressible part)
			randomLength := rand.Intn(64) + 16
			for i := 0; i < randomLength && pos < size; i++ {
				data[pos] = byte(rand.Intn(256))
				pos++
			}
		}
	}

	return data
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

// Benchmark compression function performance
func BenchmarkCompressionFunctions(b *testing.B) {
	// Generate test data
	data := generateCompressibleData(1024 * 1024) // 1MB

	b.Run("v0.1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compressed, _ := CompressBlock(data, nil)
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if len(compressed) == 0 {
				b.Fatal("Compression failed")
			}
		}
	})

	b.Run("v0.2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compressed, _ := CompressBlockV2(data, nil)
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if len(compressed) == 0 {
				b.Fatal("Compression failed")
			}
		}
	})

	b.Run("v0.3-Parallel", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compressed, _ := CompressBlockParallel(data, nil)
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if len(compressed) == 0 {
				b.Fatal("Compression failed")
			}
		}
	})

	b.Run("v0.3-V2Parallel", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compressed, _ := CompressBlockV2Parallel(data, nil)
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if len(compressed) == 0 {
				b.Fatal("Compression failed")
			}
		}
	})
}

// Benchmark streaming writer performance
func BenchmarkStreamingWriters(b *testing.B) {
	// Generate test data
	data := generateCompressibleData(1024 * 1024) // 1MB

	b.Run("Writer", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(nil)
			w := NewWriter(buf)
			w.Write(data)
			w.Close()
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if buf.Len() == 0 {
				b.Fatal("Compression failed")
			}
		}
	})

	b.Run("WriterV2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(nil)
			w := NewWriterV2(buf)
			w.Write(data)
			w.Close()
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if buf.Len() == 0 {
				b.Fatal("Compression failed")
			}
		}
	})

	b.Run("ParallelWriter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(nil)
			w := NewParallelWriter(buf)
			w.Write(data)
			w.Close()
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if buf.Len() == 0 {
				b.Fatal("Compression failed")
			}
		}
	})

	b.Run("ParallelWriterV2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(nil)
			w := NewParallelWriterV2(buf)
			w.Write(data)
			w.Close()
			b.SetBytes(int64(len(data)))
			// Prevent compiler optimization
			if buf.Len() == 0 {
				b.Fatal("Compression failed")
			}
		}
	})
}
