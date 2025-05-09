package main

import (
	"bytes"
	"fmt"
	"runtime"
	"time"

	goz4x "github.com/harriteja/GoZ4X"
	"github.com/harriteja/GoZ4X/v04/simd"
)

func main() {
	// Print CPU information
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())

	// Detect CPU features
	features := simd.DetectFeatures()
	fmt.Printf("CPU Features: SSE2=%v, SSE4.1=%v, AVX2=%v, AVX512=%v, NEON=%v\n",
		features.HasSSE2, features.HasSSE41, features.HasAVX2, features.HasAVX512, features.HasNEON)

	// Create test data (10MB of somewhat compressible data)
	const dataSize = 10 * 1024 * 1024
	data := make([]byte, dataSize)

	// Fill with some compressible pattern
	for i := 0; i < dataSize; i++ {
		data[i] = byte((i * 7) % 251) // Create a repeating but not too obvious pattern
	}

	// Benchmark compression with different methods
	fmt.Println("\nCompression benchmarks:")
	fmt.Println("------------------------")

	// v0.1 basic compression
	v1Time, v1Size := benchmarkCompression("v0.1 Basic", data, func(src, dst []byte) ([]byte, error) {
		return goz4x.CompressBlock(src, dst)
	})

	// v0.2 improved compression
	v2Time, v2Size := benchmarkCompression("v0.2 Improved", data, func(src, dst []byte) ([]byte, error) {
		return goz4x.CompressBlockV2(src, dst)
	})

	// v0.3 parallel compression
	v3Time, v3Size := benchmarkCompression("v0.3 Parallel", data, func(src, dst []byte) ([]byte, error) {
		return goz4x.CompressBlockV2Parallel(src, dst)
	})

	// v0.4 SIMD-optimized compression
	v4Time, v4Size := benchmarkCompression("v0.4 SIMD", data, func(src, dst []byte) ([]byte, error) {
		return goz4x.CompressBlockV4(src, dst)
	})

	// v0.4 SIMD + parallel
	v4pTime, v4pSize := benchmarkCompression("v0.4 SIMD+Parallel", data, func(src, dst []byte) ([]byte, error) {
		return goz4x.CompressBlockV4Parallel(src, dst)
	})

	// Print results
	fmt.Println("\nResults summary:")
	fmt.Println("----------------")
	fmt.Printf("Original size: %d bytes\n", dataSize)

	printResult("v0.1 Basic", v1Time, v1Size, dataSize)
	printResult("v0.2 Improved", v2Time, v2Size, dataSize)
	printResult("v0.3 Parallel", v3Time, v3Size, dataSize)
	printResult("v0.4 SIMD", v4Time, v4Size, dataSize)
	printResult("v0.4 SIMD+Parallel", v4pTime, v4pSize, dataSize)

	// Show speedups
	fmt.Println("\nSpeedups:")
	fmt.Println("---------")
	fmt.Printf("v0.2 vs v0.1: %.2fx\n", float64(v1Time)/float64(v2Time))
	fmt.Printf("v0.3 vs v0.2: %.2fx\n", float64(v2Time)/float64(v3Time))
	fmt.Printf("v0.4 vs v0.3: %.2fx\n", float64(v3Time)/float64(v4Time))
	fmt.Printf("v0.4+P vs v0.3: %.2fx\n", float64(v3Time)/float64(v4pTime))

	// Show space savings improvements
	fmt.Println("\nCompression ratio improvements:")
	fmt.Println("------------------------------")
	v1Ratio := float64(dataSize) / float64(v1Size)
	fmt.Printf("v0.1 ratio: %.2f:1\n", v1Ratio)
	fmt.Printf("v0.2 ratio: %.2f:1 (%.2f%% better)\n",
		float64(dataSize)/float64(v2Size),
		(float64(dataSize)/float64(v2Size)-v1Ratio)/v1Ratio*100)
	fmt.Printf("v0.4 ratio: %.2f:1 (%.2f%% better)\n",
		float64(dataSize)/float64(v4Size),
		(float64(dataSize)/float64(v4Size)-v1Ratio)/v1Ratio*100)
}

// Helper function to benchmark a compression function
func benchmarkCompression(name string, data []byte, compressFunc func([]byte, []byte) ([]byte, error)) (time.Duration, int) {
	fmt.Printf("Testing %s compression...\n", name)

	// Allocate destination buffer
	dst := make([]byte, len(data)*2) // Allocate plenty of space

	// Time the compression
	start := time.Now()
	compressed, err := compressFunc(data, dst)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return elapsed, 0
	}

	// Verify the compressed data by decompressing
	decompressed, err := goz4x.DecompressBlock(compressed, nil, len(data))
	if err != nil {
		fmt.Printf("  Decompression error: %v\n", err)
		return elapsed, len(compressed)
	}

	if !bytes.Equal(data, decompressed) {
		fmt.Printf("  Data verification failed! Decompressed data doesn't match original.\n")
	} else {
		fmt.Printf("  Verified: Data decompressed correctly.\n")
	}

	return elapsed, len(compressed)
}

// Helper function to print benchmark results
func printResult(name string, elapsed time.Duration, compressedSize, originalSize int) {
	ratio := float64(originalSize) / float64(compressedSize)
	speedMBps := float64(originalSize) / elapsed.Seconds() / 1024 / 1024

	fmt.Printf("%s:\n", name)
	fmt.Printf("  Compressed size: %d bytes (%.2f%% of original)\n",
		compressedSize, float64(compressedSize)*100/float64(originalSize))
	fmt.Printf("  Compression ratio: %.2f:1\n", ratio)
	fmt.Printf("  Compression time: %v\n", elapsed)
	fmt.Printf("  Throughput: %.2f MB/s\n", speedMBps)
}
