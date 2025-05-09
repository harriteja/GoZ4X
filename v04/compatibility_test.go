package v04

import (
	"bytes"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
	v03 "github.com/harriteja/GoZ4X/v03"
)

// TestCompatibility verifies that data compressed by any version
// can be correctly decompressed
func TestCompatibility(t *testing.T) {
	// Use various data sizes for comprehensive testing
	dataSets := []struct {
		name string
		data []byte
	}{
		{"SmallCompressible", generateCompressibleData(smallSize)},
		{"MediumCompressible", generateCompressibleData(mediumSize)},
		{"LargeCompressible", generateCompressibleData(largeSize)},
		{"SmallIncompressible", generateIncompressibleData(smallSize)},
	}

	for _, ds := range dataSets {
		t.Run(ds.name, func(t *testing.T) {
			// Compress with each version
			v1Compressed, err := compress.CompressBlock(ds.data, nil)
			if err != nil {
				t.Fatalf("v0.1 compression failed: %v", err)
			}

			v2Compressed, err := compress.CompressBlockV2(ds.data, nil)
			if err != nil {
				t.Fatalf("v0.2 compression failed: %v", err)
			}

			v3Compressed, err := v03.CompressBlockV2Parallel(ds.data, nil)
			if err != nil {
				t.Fatalf("v0.3 compression failed: %v", err)
			}

			v4Compressed, err := CompressBlock(ds.data, nil)
			if err != nil {
				t.Fatalf("v0.4 compression failed: %v", err)
			}

			v4ParallelCompressed, err := CompressBlockParallel(ds.data, nil)
			if err != nil {
				t.Fatalf("v0.4 parallel compression failed: %v", err)
			}

			// Now decompress each compressed version
			compressedSets := []struct {
				name string
				data []byte
			}{
				{"v0.1", v1Compressed},
				{"v0.2", v2Compressed},
				{"v0.3", v3Compressed},
				{"v0.4", v4Compressed},
				{"v0.4-Parallel", v4ParallelCompressed},
			}

			for _, cs := range compressedSets {
				decompressed, err := compress.DecompressBlock(cs.data, nil, len(ds.data))
				if err != nil {
					t.Fatalf("Failed to decompress %s data: %v", cs.name, err)
				}

				if !bytes.Equal(ds.data, decompressed) {
					t.Fatalf("Data mismatch with %s compression", cs.name)
				}
			}

			// Check that compressed sizes are reasonable
			t.Logf("Original size: %d bytes", len(ds.data))
			t.Logf("v0.1 compressed: %d bytes (%.2f%%)",
				len(v1Compressed), float64(len(v1Compressed))*100/float64(len(ds.data)))
			t.Logf("v0.2 compressed: %d bytes (%.2f%%)",
				len(v2Compressed), float64(len(v2Compressed))*100/float64(len(ds.data)))
			t.Logf("v0.3 compressed: %d bytes (%.2f%%)",
				len(v3Compressed), float64(len(v3Compressed))*100/float64(len(ds.data)))
			t.Logf("v0.4 compressed: %d bytes (%.2f%%)",
				len(v4Compressed), float64(len(v4Compressed))*100/float64(len(ds.data)))
			t.Logf("v0.4-Parallel compressed: %d bytes (%.2f%%)",
				len(v4ParallelCompressed), float64(len(v4ParallelCompressed))*100/float64(len(ds.data)))
		})
	}
}

// TestCustomOptions verifies compression with various options
func TestCustomOptions(t *testing.T) {
	// Use a medium-sized compressible dataset
	data := generateCompressibleData(mediumSize)

	// Test different compression levels
	levels := []int{1, 3, 6, 9, 12}

	for _, level := range levels {
		t.Run("Level-"+string(rune('0'+level)), func(t *testing.T) {
			// Compress with custom level
			opts := DefaultOptions()
			opts.Level = CompressionLevel(level)

			compressed, err := CompressBlockWithOptions(data, nil, opts)
			if err != nil {
				t.Fatalf("Compression with level %d failed: %v", level, err)
			}

			// Verify decompression
			decompressed, err := compress.DecompressBlock(compressed, nil, len(data))
			if err != nil {
				t.Fatalf("Decompression failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Fatalf("Data mismatch with level %d", level)
			}

			// Log compression ratio
			ratio := float64(len(data)) / float64(len(compressed))
			t.Logf("Level %d - Original: %d bytes, Compressed: %d bytes, Ratio: %.2f:1",
				level, len(data), len(compressed), ratio)
		})
	}

	// Test with v2 algorithm enabled/disabled
	t.Run("V2Algorithm", func(t *testing.T) {
		// With v2 algorithm
		optsWithV2 := DefaultOptions()
		optsWithV2.UseV2 = true

		compressedWithV2, err := CompressBlockWithOptions(data, nil, optsWithV2)
		if err != nil {
			t.Fatalf("Compression with V2 failed: %v", err)
		}

		// Without v2 algorithm
		optsWithoutV2 := DefaultOptions()
		optsWithoutV2.UseV2 = false

		compressedWithoutV2, err := CompressBlockWithOptions(data, nil, optsWithoutV2)
		if err != nil {
			t.Fatalf("Compression without V2 failed: %v", err)
		}

		// Verify both decompress correctly
		decompressedWithV2, err := compress.DecompressBlock(compressedWithV2, nil, len(data))
		if err != nil || !bytes.Equal(data, decompressedWithV2) {
			t.Fatalf("Data mismatch with V2 algorithm")
		}

		decompressedWithoutV2, err := compress.DecompressBlock(compressedWithoutV2, nil, len(data))
		if err != nil || !bytes.Equal(data, decompressedWithoutV2) {
			t.Fatalf("Data mismatch without V2 algorithm")
		}

		// Log compression ratios
		ratioWithV2 := float64(len(data)) / float64(len(compressedWithV2))
		ratioWithoutV2 := float64(len(data)) / float64(len(compressedWithoutV2))

		t.Logf("With V2: Compressed size %d bytes, Ratio %.2f:1",
			len(compressedWithV2), ratioWithV2)
		t.Logf("Without V2: Compressed size %d bytes, Ratio %.2f:1",
			len(compressedWithoutV2), ratioWithoutV2)
		t.Logf("V2 improvement: %.2f%%",
			(ratioWithV2-ratioWithoutV2)/ratioWithoutV2*100)
	})
}
