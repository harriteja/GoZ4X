//go:build !amd64 && !arm64
// +build !amd64,!arm64

package simd

// detectCPUFeaturesImpl is a no-op implementation for unsupported architectures
func detectCPUFeaturesImpl() {
	// No SIMD features are detected on unsupported architectures
	// All flags remain at their default values (false)
}
