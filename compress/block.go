// Package compress provides LZ4HC compression algorithms.
package compress

import (
	_ "encoding/binary"
	"errors"
	_ "math/bits"
)

const (
	// MinBlockSize is the minimum size of a block
	MinBlockSize = 16
	// MaxBlockSize is the maximum size of a block
	MaxBlockSize = 4 << 20 // 4MB
)

// CompressionLevel defines how much effort to spend on compression
type CompressionLevel int

const (
	// DefaultLevel is the default compression level (6)
	DefaultLevel CompressionLevel = 6
	// FastLevel optimizes for speed over compression ratio
	FastLevel CompressionLevel = 3
	// MaxLevel provides the highest compression at the cost of speed
	MaxLevel CompressionLevel = 12
)

var (
	// ErrInvalidBlockSize indicates the block is too small or too large
	ErrInvalidBlockSize = errors.New("invalid block size")
	// ErrInvalidCompressionLevel indicates the compression level is outside valid range
	ErrInvalidCompressionLevel = errors.New("invalid compression level")
)

// Block represents a compressible data block with a specific compression level
type Block[T ~[]byte] struct {
	input   T
	level   CompressionLevel
	options BlockOptions
}

// BlockOptions provides configuration for block compression
type BlockOptions struct {
	// PreallocateBuffer preallocates an output buffer of a given size
	PreallocateBuffer int
	// SkipChecksums skips calculating checksums
	SkipChecksums bool
}

// NewBlock creates a new block from input with default options
func NewBlock[T ~[]byte](input T, level CompressionLevel) (*Block[T], error) {
	return NewBlockWithOptions(input, level, BlockOptions{})
}

// NewBlockWithOptions creates a new block with specific options
func NewBlockWithOptions[T ~[]byte](input T, level CompressionLevel, options BlockOptions) (*Block[T], error) {
	if len(input) < MinBlockSize || len(input) > MaxBlockSize {
		return nil, ErrInvalidBlockSize
	}

	if level < 0 || level > MaxLevel {
		return nil, ErrInvalidCompressionLevel
	}

	return &Block[T]{
		input:   input,
		level:   level,
		options: options,
	}, nil
}

// CompressToBuffer compresses the block data to the provided buffer
// This is a new method that will be used by CompressBlockLevel
func (b *Block[T]) CompressToBuffer(dst []byte) ([]byte, error) {
	// Worst case estimate for LZ4 is input size + (input size / 255) + 16
	worstCaseSize := len(b.input) + (len(b.input) / 255) + 16

	if dst == nil || len(dst) < worstCaseSize {
		dst = make([]byte, worstCaseSize)
	}

	// This is a placeholder. The actual implementation will be done in v0.1
	// Here we just copy the input to the output for now
	// In a real implementation, this would use the HCMatcher and LZ4 algorithm
	copied := copy(dst, b.input)

	return dst[:copied], nil
}

// CompressBlock compresses input using LZ4HC algorithm with default compression level.
// If dst is nil or too small, a new buffer will be allocated.
func CompressBlock(src []byte, dst []byte) ([]byte, error) {
	return CompressBlockLevel(src, dst, DefaultLevel)
}

// CompressBlockLevel compresses input with specified compression level.
// If dst is nil or too small, a new buffer will be allocated.
func CompressBlockLevel(src []byte, dst []byte, level CompressionLevel) ([]byte, error) {
	block, err := NewBlock(src, level)
	if err != nil {
		return nil, err
	}

	// Now actually use the block object for compression
	return block.CompressToBuffer(dst)
}

// DecompressBlock decompresses an LZ4 compressed block.
// If dst is nil or too small, a new buffer will be allocated.
func DecompressBlock(src []byte, dst []byte, maxSize int) ([]byte, error) {
	// Placeholder implementation for decompression
	// Will be implemented in v0.1
	if len(src) == 0 {
		return nil, errors.New("empty source buffer")
	}

	if maxSize <= 0 {
		maxSize = 64 * 1024 // Default max size if not specified
	}

	if dst == nil || len(dst) < maxSize {
		dst = make([]byte, maxSize)
	}

	// This is a placeholder. The actual implementation will be done in v0.1
	copy(dst, src)

	return dst[:len(src)], nil
}
