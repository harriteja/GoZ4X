// Package matcher provides generic match-finding algorithms for LZ4 compression.
package matcher

import (
	"runtime"
)

// CPU features
var (
	hasSSE4     bool
	hasAVX2     bool
	hasNEON     bool
	initialized bool
)

func init() {
	// Check CPU features on startup
	initCPUFeatures()
}

// initCPUFeatures detects available CPU instruction sets
func initCPUFeatures() {
	if initialized {
		return
	}

	// Check for SSE4.1
	hasSSE4 = runtime.GOARCH == "amd64" || runtime.GOARCH == "386"

	// Check for AVX2 (runtime detection will be added later)
	hasAVX2 = false // Will be implemented in v0.3

	// Check for NEON (ARM)
	hasNEON = runtime.GOARCH == "arm64"

	initialized = true
}

// HasAcceleration returns true if SIMD acceleration is available
func HasAcceleration() bool {
	return hasSSE4 || hasAVX2 || hasNEON
}

// SIMDMatcher is an interface for SIMD-accelerated matchers
type SIMDMatcher interface {
	// FindLongestMatch finds the longest match at the current position
	// Returns offset and length, or 0,0 if no match found
	FindLongestMatch(buf []byte, pos, end, windowSize int) (offset, length int)

	// SupportsLongMatches returns true if this matcher can find matches longer than 64 bytes
	SupportsLongMatches() bool

	// Name returns the name of the SIMD implementation
	Name() string
}

// NewSIMDMatcher creates a new SIMD-accelerated matcher based on CPU features
func NewSIMDMatcher() SIMDMatcher {
	// This is a placeholder for future SIMD implementations
	// In v0.3, this will return actual SIMD implementations

	if hasAVX2 {
		return &stubSIMDMatcher{name: "AVX2"}
	}

	if hasSSE4 {
		return &stubSIMDMatcher{name: "SSE4"}
	}

	if hasNEON {
		return &stubSIMDMatcher{name: "NEON"}
	}

	// Fallback to stub
	return &stubSIMDMatcher{name: "Stub"}
}

// stubSIMDMatcher is a placeholder implementation
type stubSIMDMatcher struct {
	name string
}

// FindLongestMatch finds the longest match at the current position
func (s *stubSIMDMatcher) FindLongestMatch(buf []byte, pos, end, windowSize int) (offset, length int) {
	// Fallback to a simple match finder for now
	// This will be replaced with actual SIMD implementation in v0.3

	const MinMatch = 4

	if pos+MinMatch > end {
		return 0, 0
	}

	limit := pos - windowSize
	if limit < 0 {
		limit = 0
	}

	bestLength := 0
	bestOffset := 0

	// Simple quadratic search for matches
	for i := pos - 1; i >= limit; i-- {
		// Check if first 4 bytes match
		if buf[i] != buf[pos] ||
			buf[i+1] != buf[pos+1] ||
			buf[i+2] != buf[pos+2] ||
			buf[i+3] != buf[pos+3] {
			continue
		}

		// Count matching bytes
		length := 0
		maxLength := end - pos

		for length < maxLength && buf[i+length] == buf[pos+length] {
			length++
		}

		if length > bestLength {
			bestLength = length
			bestOffset = pos - i

			// Early exit for long matches
			if length >= 64 {
				break
			}
		}
	}

	if bestLength >= MinMatch {
		return bestOffset, bestLength
	}

	return 0, 0
}

// SupportsLongMatches returns true if this matcher can find matches longer than 64 bytes
func (s *stubSIMDMatcher) SupportsLongMatches() bool {
	// The stub implementation supports matches of any length
	return true
}

// Name returns the name of the SIMD implementation
func (s *stubSIMDMatcher) Name() string {
	return s.name
}
