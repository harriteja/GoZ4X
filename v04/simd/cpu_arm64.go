//go:build arm64
// +build arm64

package simd

// detectCPUFeaturesImpl provides ARM64 specific CPU feature detection
func detectCPUFeaturesImpl() {
	// ARM64 always has NEON, so we enable it unconditionally
	hasNEON = true
}
