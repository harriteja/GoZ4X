package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"runtime"
	"time"

	goz4x "github.com/harriteja/GoZ4X"
)

const (
	// Test data size - 100MB for significant parallel performance
	dataSize = 100 * 1024 * 1024
)

func main() {
	fmt.Printf("GoZ4X v%s - Parallel Compression Example\n", goz4x.Version)
	fmt.Printf("CPU Cores: %d\n\n", runtime.NumCPU())

	// Generate test data with different compression characteristics
	fmt.Println("Generating test data...")
	data := generateCompressibleData(dataSize)
	fmt.Printf("Generated %d bytes of test data\n\n", len(data))

	// Single-threaded compression using v0.2 algorithm
	fmt.Println("Single-threaded compression (v0.2):")
	singleStart := time.Now()
	singleCompressed, err := goz4x.CompressBlockV2(data, nil)
	if err != nil {
		fmt.Printf("Compression error: %v\n", err)
		return
	}
	singleTime := time.Since(singleStart)
	singleRatio := float64(len(data)) / float64(len(singleCompressed))
	fmt.Printf("Compressed to %d bytes (%.2fx ratio) in %v\n",
		len(singleCompressed), singleRatio, singleTime)
	fmt.Printf("Throughput: %.2f MB/s\n\n",
		(float64(len(data))/singleTime.Seconds())/(1024*1024))

	// Parallel compression using v0.3 capabilities
	fmt.Println("Parallel compression (v0.3):")
	parallelStart := time.Now()
	parallelCompressed, err := goz4x.CompressBlockV2Parallel(data, nil)
	if err != nil {
		fmt.Printf("Parallel compression error: %v\n", err)
		return
	}
	parallelTime := time.Since(parallelStart)
	parallelRatio := float64(len(data)) / float64(len(parallelCompressed))
	fmt.Printf("Compressed to %d bytes (%.2fx ratio) in %v\n",
		len(parallelCompressed), parallelRatio, parallelTime)
	fmt.Printf("Throughput: %.2f MB/s\n\n",
		(float64(len(data))/parallelTime.Seconds())/(1024*1024))

	// Calculate speedup
	speedup := singleTime.Seconds() / parallelTime.Seconds()
	fmt.Printf("Parallel speedup: %.2fx\n\n", speedup)

	// Verify both compressed outputs decompress to the original
	fmt.Println("Verifying decompression...")
	singleDecompressed, err := goz4x.DecompressBlock(singleCompressed, nil, len(data))
	if err != nil {
		fmt.Printf("Decompression error: %v\n", err)
		return
	}

	parallelDecompressed, err := goz4x.DecompressBlock(parallelCompressed, nil, len(data))
	if err != nil {
		fmt.Printf("Parallel decompression error: %v\n", err)
		return
	}

	if bytes.Equal(singleDecompressed, data) && bytes.Equal(parallelDecompressed, data) {
		fmt.Println("Verification successful - both methods decompressed correctly")
	} else {
		fmt.Println("Verification failed - decompression mismatch")
	}

	// Test streaming API
	fmt.Println("\nTesting streaming API:")
	testStreamingAPI(data)
}

// generateCompressibleData creates test data with repeated patterns
// that can be effectively compressed
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

// testStreamingAPI compares the performance of regular and parallel streaming API
func testStreamingAPI(data []byte) {
	// Regular streaming API
	var buf1 bytes.Buffer
	regularStart := time.Now()
	w1 := goz4x.NewWriterV2(&buf1)
	_, err := w1.Write(data)
	if err != nil {
		fmt.Printf("Regular stream write error: %v\n", err)
		return
	}
	err = w1.Close()
	if err != nil {
		fmt.Printf("Regular stream close error: %v\n", err)
		return
	}
	regularTime := time.Since(regularStart)
	regularRatio := float64(len(data)) / float64(buf1.Len())
	fmt.Printf("Regular streaming: %d bytes (%.2fx ratio) in %v\n",
		buf1.Len(), regularRatio, regularTime)
	fmt.Printf("Throughput: %.2f MB/s\n\n",
		(float64(len(data))/regularTime.Seconds())/(1024*1024))

	// Parallel streaming API
	var buf2 bytes.Buffer
	parallelStart := time.Now()
	w2 := goz4x.NewParallelWriterV2(&buf2)
	_, err = w2.Write(data)
	if err != nil {
		fmt.Printf("Parallel stream write error: %v\n", err)
		return
	}
	err = w2.Close()
	if err != nil {
		fmt.Printf("Parallel stream close error: %v\n", err)
		return
	}
	parallelTime := time.Since(parallelStart)
	parallelRatio := float64(len(data)) / float64(buf2.Len())
	fmt.Printf("Parallel streaming: %d bytes (%.2fx ratio) in %v\n",
		buf2.Len(), parallelRatio, parallelTime)
	fmt.Printf("Throughput: %.2f MB/s\n\n",
		(float64(len(data))/parallelTime.Seconds())/(1024*1024))

	// Calculate speedup
	streamSpeedup := regularTime.Seconds() / parallelTime.Seconds()
	fmt.Printf("Streaming speedup: %.2fx\n\n", streamSpeedup)

	// Verify decompression of both streams
	fmt.Println("Verifying streaming decompression...")

	// Decompress regular stream
	r1 := goz4x.NewReader(bytes.NewReader(buf1.Bytes()))
	decompressed1 := &bytes.Buffer{}
	_, err = io.Copy(decompressed1, r1)
	if err != nil {
		fmt.Printf("Regular decompression error: %v\n", err)
		return
	}

	// Decompress parallel stream
	r2 := goz4x.NewReader(bytes.NewReader(buf2.Bytes()))
	decompressed2 := &bytes.Buffer{}
	_, err = io.Copy(decompressed2, r2)
	if err != nil {
		fmt.Printf("Parallel decompression error: %v\n", err)
		return
	}

	if bytes.Equal(decompressed1.Bytes(), data) && bytes.Equal(decompressed2.Bytes(), data) {
		fmt.Println("Streaming verification successful")
	} else {
		fmt.Println("Streaming verification failed")
	}
}
