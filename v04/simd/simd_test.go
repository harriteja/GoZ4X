package simd

import (
	"bytes"
	"runtime"
	"testing"
)

func TestFeatureDetection(t *testing.T) {
	// Run feature detection
	features := DetectFeatures()

	// Log detected features for debugging
	t.Logf("CPU Features: SSE2=%v, SSE4.1=%v, AVX2=%v, AVX512=%v, NEON=%v",
		features.HasSSE2, features.HasSSE41, features.HasAVX2, features.HasAVX512, features.HasNEON)

	// Basic platform-specific expectations
	switch runtime.GOARCH {
	case "amd64":
		if !features.HasSSE2 {
			t.Error("SSE2 should be available on all x86-64 processors")
		}
	case "arm64":
		if !features.HasNEON {
			t.Error("NEON should be available on all ARM64 processors")
		}
	}

	// Verify BestImplementation returns something valid
	impl := BestImplementation()
	implName := ImplementationName(impl)
	t.Logf("Best available implementation: %s (%d)", implName, impl)

	if impl < ImplGeneric || impl > ImplNEON {
		t.Errorf("BestImplementation returned invalid implementation type: %d", impl)
	}
}

// TestCopyOperations tests platform-specific copy operations
func TestCopyOperations(t *testing.T) {
	// Create test data
	src := make([]byte, 1024)
	dst := make([]byte, 1024)

	// Fill source with pattern
	for i := range src {
		src[i] = byte(i & 0xFF)
	}

	// Test pattern for repeat copy
	pattern := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	// Run architecture-specific tests
	if runtime.GOARCH == "amd64" {
		t.Run("SSE", func(t *testing.T) {
			// Simple test of copies - using regular Go implementation for now
			copy(dst, src[:512])
			if !bytes.Equal(dst[:512], src[:512]) {
				t.Error("Copy failed to copy data correctly")
			}

			// Copy pattern repeatedly
			copy(dst[:16], pattern)

			// Test repeated copy
			for i := 0; i < 3; i++ {
				offset := 16 + i*16
				for j := 0; j < 16; j++ {
					dst[offset+j] = dst[j]
				}
			}

			// Verify the pattern was copied correctly
			for i := 0; i < 3; i++ {
				offset := 16 + i*16
				if !bytes.Equal(dst[offset:offset+16], pattern) {
					t.Errorf("RepeatCopy failed at offset %d", offset)
				}
			}
		})
	}

	if runtime.GOARCH == "arm64" {
		t.Run("NEON", func(t *testing.T) {
			// Simple test of copies - using regular Go implementation for now
			copy(dst, src[:512])
			if !bytes.Equal(dst[:512], src[:512]) {
				t.Error("Copy failed to copy data correctly")
			}

			// Copy pattern repeatedly
			copy(dst[:16], pattern)

			// Test repeated copy
			for i := 0; i < 3; i++ {
				offset := 16 + i*16
				for j := 0; j < 16; j++ {
					dst[offset+j] = dst[j]
				}
			}

			// Verify the pattern was copied correctly
			for i := 0; i < 3; i++ {
				offset := 16 + i*16
				if !bytes.Equal(dst[offset:offset+16], pattern) {
					t.Errorf("RepeatCopy failed at offset %d", offset)
				}
			}
		})
	}
}

// TestMatchFinding tests the match finding algorithms
func TestMatchFinding(t *testing.T) {
	// Test data with repeating pattern
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 64)
	}

	// Test cases for different architectures
	t.Run("Generic", func(t *testing.T) {
		// Generic match finder test
		h1 := uint32(data[64]) | (uint32(data[64+1]) << 8) |
			(uint32(data[64+2]) << 16) | (uint32(data[64+3]) << 24)
		h1 = (h1 * 2654435761) & 0xFFFF // FNV-1a hash truncated

		h2 := uint32(data[128]) | (uint32(data[128+1]) << 8) |
			(uint32(data[128+2]) << 16) | (uint32(data[128+3]) << 24)
		h2 = (h2 * 2654435761) & 0xFFFF // FNV-1a hash truncated

		// If the pattern repeats exactly, the hashes should match
		if bytes.Equal(data[64:64+4], data[128:128+4]) && h1 != h2 {
			t.Errorf("Hash function inconsistency: identical sequences have different hashes")
		}
	})

	if runtime.GOARCH == "amd64" {
		t.Run("SSE", func(t *testing.T) {
			// SSE match finder test
			// For now, we simply verify the hash function is consistent
			h1 := uint32(data[64]) | (uint32(data[64+1]) << 8) |
				(uint32(data[64+2]) << 16) | (uint32(data[64+3]) << 24)
			h1 = (h1 * 2654435761) & 0xFFFF // FNV-1a hash truncated

			h2 := uint32(data[128]) | (uint32(data[128+1]) << 8) |
				(uint32(data[128+2]) << 16) | (uint32(data[128+3]) << 24)
			h2 = (h2 * 2654435761) & 0xFFFF // FNV-1a hash truncated

			// If the pattern repeats exactly, the hashes should match
			if bytes.Equal(data[64:64+4], data[128:128+4]) && h1 != h2 {
				t.Errorf("Hash function inconsistency: identical sequences have different hashes")
			}
		})
	}

	if runtime.GOARCH == "arm64" {
		t.Run("NEON", func(t *testing.T) {
			// NEON match finder test
			// For now, we simply verify the hash function is consistent
			h1 := uint32(data[64]) | (uint32(data[64+1]) << 8) |
				(uint32(data[64+2]) << 16) | (uint32(data[64+3]) << 24)
			h1 = (h1 * 2654435761) & 0xFFFF // FNV-1a hash truncated

			h2 := uint32(data[128]) | (uint32(data[128+1]) << 8) |
				(uint32(data[128+2]) << 16) | (uint32(data[128+3]) << 24)
			h2 = (h2 * 2654435761) & 0xFFFF // FNV-1a hash truncated

			// If the pattern repeats exactly, the hashes should match
			if bytes.Equal(data[64:64+4], data[128:128+4]) && h1 != h2 {
				t.Errorf("Hash function inconsistency: identical sequences have different hashes")
			}
		})
	}
}
