package compress

// Test helpers to expose internal fields for testing

// RegisterTestHelpers initializes test helpers
func RegisterTestHelpers() {
	// This is just a placeholder to register any test helpers
	// The actual implementation is in the methods below
}

// GetLevel returns the compression level of a Block
func (b *Block[T]) GetLevel() CompressionLevel {
	return b.level
}
