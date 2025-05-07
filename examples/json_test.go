package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
)

func TestJSONCompression(t *testing.T) {
	// Create a sample JSON structure
	testData := map[string]interface{}{
		"name":    "GoZ4X Test",
		"version": 1.0,
		"features": []string{
			"generics",
			"SIMD",
			"parallelism",
			"high compression",
		},
		"metadata": map[string]interface{}{
			"author":  "Test User",
			"created": "2024-05-01",
			"settings": map[string]interface{}{
				"compressionLevel": 6,
				"windowSize":       65535,
				"blockSize":        262144,
			},
		},
		"nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": "deeply nested value",
				},
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Printf("Original JSON size: %d bytes\n", len(jsonData))

	// Compress the JSON data with default level
	compressed, err := compress.CompressBlock(jsonData, nil)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	fmt.Printf("Compressed size: %d bytes\n", len(compressed))
	fmt.Printf("Compression ratio: %.2f%%\n", float64(len(compressed))/float64(len(jsonData))*100)

	// Verify the issue: The Block object is created but not used
	// The current implementation in CompressBlockLevel only creates a Block but doesn't use it
	// Instead it just copies the data without actual compression

	// Verify by decompressing
	decompressed, err := compress.DecompressBlock(compressed, nil, len(jsonData))
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	// Check if decompressed data matches original
	if string(decompressed) != string(jsonData) {
		t.Fatalf("Decompressed data doesn't match original")
	}

	fmt.Println("Decompression successful, data integrity verified")

	// Inspect what's actually happening in the CompressBlockLevel function
	// The issue: block object created but not used for compression
	var source, destination []byte
	source = jsonData

	// This recreates what happens inside CompressBlockLevel
	block, err := compress.NewBlock(source, compress.DefaultLevel)
	if err != nil {
		t.Fatalf("Block creation failed: %v", err)
	}

	// In the actual implementation, this block object is created but never used
	fmt.Printf("Block object created with compression level: %v\n", block.GetLevel())

	// Current implementation just copies data without compression
	// This simulates what the current placeholder does
	worstCaseSize := len(source) + (len(source) / 255) + 16
	destination = make([]byte, worstCaseSize)
	copied := copy(destination, source)
	result := destination[:copied]

	fmt.Printf("Current 'compression' just copies %d bytes\n", copied)

	// Verify the placeholder implementation matches the public function
	if string(result) != string(compressed) {
		t.Fatalf("Our simulation doesn't match actual implementation")
	} else {
		fmt.Println("Verified: current implementation just copies data")
	}
}

// Create a temporary accessor to verify the block's level
// We need to add this method to the compress.Block type
func init() {
	// Register a method to access the internal Block level for testing
	compress.RegisterTestHelpers()
}
