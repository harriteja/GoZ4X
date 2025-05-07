package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/harriteja/GoZ4X/compress"
)

func main() {
	// Create a sample JSON structure (with more data for better compression)
	testData := map[string]interface{}{
		"name":        "GoZ4X Test",
		"version":     1.0,
		"description": "A pure-Go, ultra-fast, high-compression LZ4 library for modern workloads",
		"features": []string{
			"Go generics for clean, reusable match-finder and streaming APIs",
			"SIMD/assembly for match searching and copy loops (SSE4.1, AVX2, NEON)",
			"Parallel block compression to saturate all cores",
			"Pluggable backends (pure-Go, assembly, GPU) selected at runtime",
			"First-class WASM support for browser & edge functions",
			"Comprehensive benchmark suite",
		},
		"metadata": map[string]interface{}{
			"author":  "Example User",
			"created": "2024-05-10",
			"updated": "2024-05-11",
			"settings": map[string]interface{}{
				"compressionLevel": 6,
				"windowSize":       65535,
				"blockSize":        262144,
				"parallel":         true,
				"numThreads":       8,
			},
		},
		// Add repeated data to demonstrate compression potential
		"repeated": []string{
			"This is a repeated string that should compress well.",
			"This is a repeated string that should compress well.",
			"This is a repeated string that should compress well.",
			"This is a repeated string that should compress well.",
			"This is a repeated string that should compress well.",
		},
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Original JSON size: %d bytes\n", len(jsonData))

	// Before/After Demonstration
	fmt.Println("\n--- Before Fix ---")
	fmt.Println("The original implementation created a Block object but didn't use it:")
	fmt.Println("```go")
	fmt.Println("func CompressBlockLevel(src []byte, dst []byte, level CompressionLevel) ([]byte, error) {")
	fmt.Println("    _, err := NewBlock(src, level)  // Block created but unused!")
	fmt.Println("    if err != nil {")
	fmt.Println("        return nil, err")
	fmt.Println("    }")
	fmt.Println("    ")
	fmt.Println("    // Rest of code didn't use the Block...")
	fmt.Println("    // Instead it copied data directly")
	fmt.Println("}")
	fmt.Println("```")

	fmt.Println("\n--- After Fix ---")
	fmt.Println("The fixed implementation creates and uses the Block object:")
	fmt.Println("```go")
	fmt.Println("func CompressBlockLevel(src []byte, dst []byte, level CompressionLevel) ([]byte, error) {")
	fmt.Println("    block, err := NewBlock(src, level)  // Block created")
	fmt.Println("    if err != nil {")
	fmt.Println("        return nil, err")
	fmt.Println("    }")
	fmt.Println("    ")
	fmt.Println("    // Now actually use the block for compression")
	fmt.Println("    return block.CompressToBuffer(dst)")
	fmt.Println("}")
	fmt.Println("```")

	// Test the fixed implementation
	fmt.Println("\n--- Testing the implementation ---")

	// Compress using different compression levels
	for _, level := range []compress.CompressionLevel{1, 6, 12} {
		// Compress the JSON data
		compressed, err := compress.CompressBlockLevel(jsonData, nil, level)
		if err != nil {
			fmt.Printf("Error compressing at level %d: %v\n", level, err)
			continue
		}

		fmt.Printf("Level %d - Original: %d bytes, Compressed: %d bytes, Ratio: %.2f%%\n",
			level, len(jsonData), len(compressed), float64(len(compressed))/float64(len(jsonData))*100)

		// Note: In the current placeholder implementation, all levels will produce the same result
		// because we're not actually compressing yet, just copying data
		// When real compression is implemented, different levels should produce different results

		// Verify we can decompress
		decompressed, err := compress.DecompressBlock(compressed, nil, len(jsonData))
		if err != nil {
			fmt.Printf("Error decompressing: %v\n", err)
			continue
		}

		if string(decompressed) == string(jsonData) {
			fmt.Printf("Level %d - Decompression successful, data integrity verified\n", level)
		} else {
			fmt.Printf("Level %d - Decompression failed, data mismatch!\n", level)
		}
	}

	fmt.Println("\nNote: The current implementation is still just copying data (placeholder).")
	fmt.Println("When actual LZ4 compression is implemented, we should see better compression ratios.")
	fmt.Println("The key improvement is that we now properly use the Block object that we create.")
}
