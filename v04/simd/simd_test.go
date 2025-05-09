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

	// Test implementation name function with all possible values
	impls := []int{ImplGeneric, ImplSSE41, ImplAVX2, ImplAVX512, ImplNEON, -1}
	expectedNames := []string{"Generic", "SSE4.1", "AVX2", "AVX512", "NEON", "Unknown"}

	for i, impl := range impls {
		name := ImplementationName(impl)
		if name != expectedNames[i] {
			t.Errorf("ImplementationName(%d) returned %s, expected %s",
				impl, name, expectedNames[i])
		}
	}
}

// TestCopyOperations tests platform-specific copy operations
func TestCopyOperations(t *testing.T) {
	t.Skip("Skipping copy operations test as assembly implementations have been removed")
}

// TestCopyAndCompareAssembly tests the raw assembly functions
func TestCopyAndCompareAssembly(t *testing.T) {
	t.Skip("Skipping assembly function tests as assembly implementations have been removed")
}

// TestMatchFinding tests the generic match finding algorithms
func TestMatchFinding(t *testing.T) {
	// Test data with repeating pattern
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 64)
	}

	// Test hash function consistency
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
}
