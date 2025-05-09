//go:build amd64
// +build amd64

package simd

import (
	"golang.org/x/sys/cpu"
)

// detectCPUFeaturesImpl is the architecture-specific implementation
// of CPU feature detection for AMD64
func detectCPUFeaturesImpl() {
	// Update feature flags based on what's actually available
	hasSSE2 = cpu.X86.HasSSE2 // Should always be true on amd64
	hasSSE41 = cpu.X86.HasSSE41
	hasAVX2 = cpu.X86.HasAVX2
	hasAVX512 = cpu.X86.HasAVX512F && cpu.X86.HasAVX512BW
}
