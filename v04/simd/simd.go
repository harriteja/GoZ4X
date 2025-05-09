// Package simd provides SIMD-accelerated implementations of LZ4 compression algorithms.
package simd

import (
	"runtime"
	"sync"
)

// Match represents a found match during compression
type Match struct {
	Offset int // Offset from current position
	Length int // Length of the match
}

// CPU architecture and feature detection
var (
	// Architecture flags
	isAMD64 = runtime.GOARCH == "amd64"
	isARM64 = runtime.GOARCH == "arm64"

	// Feature flags
	hasSSE2   bool
	hasSSE41  bool
	hasAVX2   bool
	hasAVX512 bool
	hasNEON   bool

	// Initialization
	detectOnce sync.Once
)

// Implementation types
const (
	ImplGeneric = iota // Pure Go implementation
	ImplSSE41          // SSE4.1 implementation
	ImplAVX2           // AVX2 implementation
	ImplAVX512         // AVX512 implementation
	ImplNEON           // ARM NEON implementation
)

// Features represents CPU feature flags
type Features struct {
	HasSSE2   bool
	HasSSE41  bool
	HasAVX2   bool
	HasAVX512 bool
	HasNEON   bool
}

// DetectFeatures initializes CPU feature detection
func DetectFeatures() Features {
	detectOnce.Do(func() {
		detectCPUFeatures()
	})

	return Features{
		HasSSE2:   hasSSE2,
		HasSSE41:  hasSSE41,
		HasAVX2:   hasAVX2,
		HasAVX512: hasAVX512,
		HasNEON:   hasNEON,
	}
}

// detectCPUFeatures performs CPU feature detection
func detectCPUFeatures() {
	// Default values based on architecture
	if isAMD64 {
		// x86-64 always has SSE2
		hasSSE2 = true

		// For now, make conservative assumptions about other features
		// The actual runtime detection is implemented in CPU-specific files
		hasSSE41 = true   // Assume SSE4.1 support for x86-64
		hasAVX2 = false   // Don't assume AVX2 by default
		hasAVX512 = false // Don't assume AVX512 by default
	}

	if isARM64 {
		// ARM64 always has NEON
		hasNEON = true
	}

	// Call architecture-specific detection
	// This function is implemented in CPU-specific files with build tags
	detectCPUFeaturesImpl()
}

// BestImplementation returns the best SIMD implementation available on this CPU
func BestImplementation() int {
	// Ensure features are detected
	DetectFeatures()

	// Check for best available implementation
	if isAMD64 {
		if hasAVX512 {
			return ImplAVX512
		}
		if hasAVX2 {
			return ImplAVX2
		}
		if hasSSE41 {
			return ImplSSE41
		}
	}

	if isARM64 && hasNEON {
		return ImplNEON
	}

	// Fallback to generic implementation
	return ImplGeneric
}

// ImplementationName returns a string name for the implementation type
func ImplementationName(impl int) string {
	switch impl {
	case ImplGeneric:
		return "Generic"
	case ImplSSE41:
		return "SSE4.1"
	case ImplAVX2:
		return "AVX2"
	case ImplAVX512:
		return "AVX512"
	case ImplNEON:
		return "NEON"
	default:
		return "Unknown"
	}
}
