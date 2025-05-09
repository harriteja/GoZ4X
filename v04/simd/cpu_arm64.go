//go:build arm64
// +build arm64

package simd

// detectCPUFeaturesImpl is the architecture-specific implementation
// of CPU feature detection for ARM64
func detectCPUFeaturesImpl() {
	// All ARM64 platforms have NEON, already set in the main detection
	hasNEON = true
}
