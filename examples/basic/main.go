// Example of using GoZ4X for streaming compression and decompression
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/harriteja/GoZ4X/compress"
)

func main() {
	// Create sample data
	testData := strings.Repeat("GoZ4X is a pure Go implementation of LZ4 compression algorithm. ", 20)
	fmt.Printf("Original size: %d bytes\n", len(testData))

	// Compress using streaming API
	var compressedBuf bytes.Buffer
	writer := compress.NewWriter(&compressedBuf)

	// Write data
	_, err := io.Copy(writer, strings.NewReader(testData))
	if err != nil {
		log.Fatalf("Compression failed: %v", err)
	}

	// Close to ensure all data is flushed
	err = writer.Close()
	if err != nil {
		log.Fatalf("Failed to close writer: %v", err)
	}

	compressedData := compressedBuf.Bytes()
	fmt.Printf("Compressed size: %d bytes\n", len(compressedData))
	fmt.Printf("Compression ratio: %.2f%%\n", float64(len(compressedData))*100/float64(len(testData)))

	// Decompress
	reader := compress.NewReader(bytes.NewReader(compressedData))
	var decompressedBuf bytes.Buffer
	_, err = io.Copy(&decompressedBuf, reader)
	if err != nil {
		log.Fatalf("Decompression failed: %v", err)
	}

	decompressed := decompressedBuf.String()

	// Verify the data
	if decompressed == testData {
		fmt.Println("Decompression successful, data integrity verified!")
		fmt.Println("GoZ4X v0.1 streaming compression/decompression is working correctly.")
	} else {
		log.Fatalf("Decompressed data doesn't match original!")
	}
}
