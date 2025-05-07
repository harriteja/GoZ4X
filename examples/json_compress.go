package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/harriteja/GoZ4X/compress"
)

func main2() {
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
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(dataObject, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Original JSON size: %d bytes\n", len(jsonData))

	// Current implementation - creates but doesn't use the Block object
	fmt.Println("\n--- Current implementation (placeholder) ---")
	compressed, err := compress.CompressBlock(jsonData, nil)
	if err != nil {
		fmt.Printf("Error compressing: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("'Compressed' size: %d bytes\n", len(compressed))
	fmt.Printf("Ratio: %.2f%%\n", float64(len(compressed))/float64(len(jsonData))*100)

	// Verify that it's just copying the data
	fmt.Printf("Is just a copy? %v\n", string(compressed) == string(jsonData))

	// How it SHOULD be implemented
	fmt.Println("\n--- Proper implementation (example) ---")

	// Create a Block with the input
	block, err := compress.NewBlock(jsonData, compress.DefaultLevel)
	if err != nil {
		fmt.Printf("Error creating block: %v\n", err)
		os.Exit(1)
	}

	// In a proper implementation, we would call a method on the block to compress
	// For example (this doesn't exist yet):
	// properlyCompressed, err := block.Compress()

	// For now, simulate a basic LZ4-style compression
	// This is just to demonstrate that the block should be used, not actual LZ4 compression
	fmt.Printf("Block compression level: %d\n", block.GetLevel())
	fmt.Println("Note: The CompressBlockLevel function creates this Block but doesn't use it")
	fmt.Println("In a proper implementation, the Block object would be used for compression")

	// In the actual implementation, proper LZ4 compression would use:
	// 1. The HCMatcher from compress/hc.go
	// 2. Match finding logic from the block
	// 3. Literal/match encoding according to LZ4 format

	fmt.Println("\nRecommendation: Update CompressBlockLevel to actually use the Block object")
	fmt.Println("Example fix:")
	fmt.Println("```go")
	fmt.Println("func CompressBlockLevel(src []byte, dst []byte, level CompressionLevel) ([]byte, error) {")
	fmt.Println("    block, err := NewBlock(src, level)")
	fmt.Println("    if err != nil {")
	fmt.Println("        return nil, err")
	fmt.Println("    }")
	fmt.Println("    ")
	fmt.Println("    // Prepare destination buffer")
	fmt.Println("    worstCaseSize := len(src) + (len(src) / 255) + 16")
	fmt.Println("    if dst == nil || len(dst) < worstCaseSize {")
	fmt.Println("        dst = make([]byte, worstCaseSize)")
	fmt.Println("    }")
	fmt.Println("    ")
	fmt.Println("    // Actually use the block object for compression")
	fmt.Println("    // This could be a method like:")
	fmt.Println("    return block.CompressToBuffer(dst)")
	fmt.Println("}")
	fmt.Println("```")
}
