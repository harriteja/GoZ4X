package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	goz4x "github.com/harriteja/GoZ4X"
)

func main() {
	fmt.Println("GoZ4X v0.2 vs v0.1 Compression Test")
	fmt.Println("===================================")

	// Test with different data types
	testCompression("Text data", generateTextData(), true)
	testCompression("Repetitive data", generateRepetitiveData(), true)
	testCompression("Mixed data", generateMixedData(), true)
}

// testCompression compares v0.1 and v0.2 compression
func testCompression(dataType string, data []byte, showDecompressTime bool) {
	fmt.Printf("\n%s (%d bytes):\n", dataType, len(data))
	fmt.Println(strings.Repeat("-", len(dataType)+15))

	// v0.1 Block Compression
	v1Start := time.Now()
	v1Compressed, _ := goz4x.CompressBlock(data, nil)
	v1Time := time.Since(v1Start)
	v1Ratio := float64(len(data)) / float64(len(v1Compressed))

	// v0.2 Block Compression
	v2Start := time.Now()
	v2Compressed, _ := goz4x.CompressBlockV2(data, nil)
	v2Time := time.Since(v2Start)
	v2Ratio := float64(len(data)) / float64(len(v2Compressed))

	// Results
	fmt.Printf("v0.1 Compression: %d bytes (%.2fx ratio) in %v\n",
		len(v1Compressed), v1Ratio, v1Time)
	fmt.Printf("v0.2 Compression: %d bytes (%.2fx ratio) in %v\n",
		len(v2Compressed), v2Ratio, v2Time)

	// Calculate improvement
	compressionImprovement := (v2Ratio - v1Ratio) / v1Ratio * 100
	speedDiff := (v1Time.Seconds() - v2Time.Seconds()) / v1Time.Seconds() * 100

	fmt.Printf("v0.2 Improvement: %.2f%% better compression ", compressionImprovement)
	if speedDiff > 0 {
		fmt.Printf("and %.2f%% faster\n", speedDiff)
	} else {
		fmt.Printf("but %.2f%% slower\n", -speedDiff)
	}

	// Decompression time (optional)
	if showDecompressTime {
		// Decompress v0.1
		decompStart := time.Now()
		v1Decompressed, _ := goz4x.DecompressBlock(v1Compressed, nil, len(data))
		v1DecompTime := time.Since(decompStart)

		// Decompress v0.2
		decompStart = time.Now()
		v2Decompressed, _ := goz4x.DecompressBlock(v2Compressed, nil, len(data))
		v2DecompTime := time.Since(decompStart)

		// Verify decompression worked
		if !bytes.Equal(v1Decompressed, data) || !bytes.Equal(v2Decompressed, data) {
			fmt.Println("ERROR: Decompression verification failed!")
		}

		fmt.Printf("v0.1 Decompression: %v\n", v1DecompTime)
		fmt.Printf("v0.2 Decompression: %v\n", v2DecompTime)
	}

	// Streaming API comparison
	fmt.Println("\nStreaming API comparison:")

	// v0.1 Streaming
	var buf1 bytes.Buffer
	streamStart := time.Now()
	w1 := goz4x.NewWriter(&buf1)
	w1.Write(data)
	w1.Close()
	v1StreamTime := time.Since(streamStart)
	v1StreamRatio := float64(len(data)) / float64(buf1.Len())

	// v0.2 Streaming
	var buf2 bytes.Buffer
	streamStart = time.Now()
	w2 := goz4x.NewWriterV2(&buf2)
	w2.Write(data)
	w2.Close()
	v2StreamTime := time.Since(streamStart)
	v2StreamRatio := float64(len(data)) / float64(buf2.Len())

	// Results
	fmt.Printf("v0.1 Stream: %d bytes (%.2fx ratio) in %v\n",
		buf1.Len(), v1StreamRatio, v1StreamTime)
	fmt.Printf("v0.2 Stream: %d bytes (%.2fx ratio) in %v\n",
		buf2.Len(), v2StreamRatio, v2StreamTime)

	// Calculate improvement
	streamImprovement := (v2StreamRatio - v1StreamRatio) / v1StreamRatio * 100
	streamSpeedDiff := (v1StreamTime.Seconds() - v2StreamTime.Seconds()) / v1StreamTime.Seconds() * 100

	fmt.Printf("v0.2 Stream Improvement: %.2f%% better compression ", streamImprovement)
	if streamSpeedDiff > 0 {
		fmt.Printf("and %.2f%% faster\n", streamSpeedDiff)
	} else {
		fmt.Printf("but %.2f%% slower\n", -streamSpeedDiff)
	}
}

// Data generation functions
func generateTextData() []byte {
	return bytes.Repeat([]byte(
		"GoZ4X is a pure-Go implementation of the LZ4 compression algorithm. "+
			"It's designed for modern workloads and offers high performance with "+
			"easy-to-use APIs for both block and streaming compression. "+
			"The v0.2 version includes better match finding algorithms for improved compression ratio. "+
			"This example demonstrates the improvements of v0.2 over v0.1 in terms of "+
			"compression ratio and performance.\n"), 20)
}

func generateRepetitiveData() []byte {
	// Create a pattern that should compress well
	pattern := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	return bytes.Repeat(pattern, 1000)
}

func generateMixedData() []byte {
	// Create a buffer with mixed content
	var buf bytes.Buffer

	// Add some repetitive patterns
	for i := 0; i < 10; i++ {
		buf.WriteString(strings.Repeat("pattern", 20))
		buf.WriteByte('\n')
	}

	// Add some text
	buf.WriteString(string(generateTextData()[:1000]))

	// Add binary-like data with some repetition
	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			buf.WriteByte(byte(i % 256))
		} else {
			buf.WriteByte(byte(42))
		}
	}

	return buf.Bytes()
}
