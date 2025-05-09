//go:build amd64
// +build amd64

package simd

import (
	"runtime"
)

// detectCPUFeaturesImpl provides x86-64 specific CPU feature detection
func detectCPUFeaturesImpl() {
	// In a real implementation, this would use CPUID instructions
	// to detect CPU features.

	// For now, we'll use a conservative approach - most modern CPUs
	// support at least SSE4.1, so we'll enable it by default
	hasSSE41 = true

	// We only check runtime.GOARCH here since we're already
	// behind an amd64 build tag. No need to check again.
	// Simple version check to enable AVX2 only on go1.18+
	if hasGoVersion("go1.18") {
		hasAVX2 = true
	}
}

// hasGoVersion checks if the current Go runtime is at least the given version
func hasGoVersion(version string) bool {
	return runtime.Version() >= version
}
