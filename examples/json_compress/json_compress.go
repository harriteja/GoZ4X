package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/harriteja/GoZ4X/compress"
)

func main() {
	// Create a simple JSON object
	dataObject := map[string]interface{}{
		"users": []map[string]interface{}{
			{
				"id":     1,
				"name":   "John Doe",
				"email":  "john@example.com",
				"active": true,
			},
			{
				"id":     2,
				"name":   "Jane Smith",
				"email":  "jane@example.com",
				"active": false,
			},
		},
		"settings": map[string]interface{}{
			"notifications": true,
			"theme":         "dark",
			"language":      "en",
		},
		"metadata": map[string]string{
			"version":   "1.0.0",
			"timestamp": "2024-05-10T12:00:00Z",
		},
		// Add some repetitive data for better compression demonstration
		"repetitive_data": []string{
			"This is a repeated string to test compression.",
			"This is a repeated string to test compression.",
			"This is a repeated string to test compression.",
			"This is a repeated string to test compression.",
			"This is a repeated string to test compression.",
		},
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(dataObject, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Original JSON size: %d bytes\n", len(jsonData))

	// v0.1 implementation
	fmt.Println("\n--- v0.1 Implementation ---")
	v1Compressed, err := compress.CompressBlock(jsonData, nil)
	if err != nil {
		fmt.Printf("Error compressing with v0.1: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("v0.1 Compressed size: %d bytes\n", len(v1Compressed))
	fmt.Printf("v0.1 Ratio: %.2f%%\n", float64(len(v1Compressed))/float64(len(jsonData))*100)

	// v0.2 implementation
	fmt.Println("\n--- v0.2 Implementation ---")
	v2Compressed, err := compress.CompressBlockV2(jsonData, nil)
	if err != nil {
		fmt.Printf("Error compressing with v0.2: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("v0.2 Compressed size: %d bytes\n", len(v2Compressed))
	fmt.Printf("v0.2 Ratio: %.2f%%\n", float64(len(v2Compressed))/float64(len(jsonData))*100)

	// Improvement calculation
	improvement := (1 - float64(len(v2Compressed))/float64(len(v1Compressed))) * 100
	fmt.Printf("v0.2 vs v0.1 improvement: %.2f%%\n", improvement)

	// Verify decompression for both versions
	fmt.Println("\n--- Decompression Verification ---")

	// Decompress v0.1
	v1Decompressed, err := compress.DecompressBlock(v1Compressed, nil, len(jsonData))
	if err != nil {
		fmt.Printf("Error decompressing v0.1: %v\n", err)
	} else {
		v1Match := string(v1Decompressed) == string(jsonData)
		fmt.Printf("v0.1 Decompression successful: %v\n", v1Match)
	}

	// Decompress v0.2
	v2Decompressed, err := compress.DecompressBlock(v2Compressed, nil, len(jsonData))
	if err != nil {
		fmt.Printf("Error decompressing v0.2: %v\n", err)
	} else {
		v2Match := string(v2Decompressed) == string(jsonData)
		fmt.Printf("v0.2 Decompression successful: %v\n", v2Match)
	}

	// Test with different compression levels
	fmt.Println("\n--- v0.2 with Different Compression Levels ---")
	levels := []int{1, 6, 12}

	for _, level := range levels {
		compressed, err := compress.CompressBlockV2Level(jsonData, nil, compress.CompressionLevel(level))
		if err != nil {
			fmt.Printf("Error compressing at level %d: %v\n", level, err)
			continue
		}

		ratio := float64(len(compressed)) / float64(len(jsonData)) * 100
		fmt.Printf("Level %d - Compressed size: %d bytes, Ratio: %.2f%%\n",
			level, len(compressed), ratio)
	}
}
