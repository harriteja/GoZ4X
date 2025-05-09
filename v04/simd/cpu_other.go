//go:build !amd64 && !arm64
// +build !amd64,!arm64

package simd

// detectCPUFeaturesImpl is a fallback implementation
// for unsupported architectures
func detectCPUFeaturesImpl() {
	// No SIMD features on unsupported platforms
}
