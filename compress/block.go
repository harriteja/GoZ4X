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
	// Input data and length
	input := b.input
	inputLen := len(input)

	// Create matcher based on level
	matcher := NewHCMatcher(b.level)
	matcher.Reset(input)

	// Calculate worst-case output size
	worstCaseSize := inputLen + (inputLen / 255) + 16

	// Allocate buffer if needed
	if dst == nil || len(dst) < worstCaseSize {
		dst = make([]byte, worstCaseSize)
	}

	// Initialize positions
	srcPos := 0
	dstPos := 0

	// LastLiteral is the position where the last literal block started
	lastLiteral := 0

	// Main compression loop
	for !matcher.End() {
		// Find the best match at the current position
		offset, matchLen := matcher.FindBestMatch()

		// If no good match, advance and continue
		if matchLen < 4 {
			// Advance the matcher and continue
			matcher.Advance(1)
			srcPos++
			continue
		}

		// We found a match, output the literal sequence since the last match
		literalLen := srcPos - lastLiteral

		// Write token: 4 bits for literal length, 4 bits for match length
		literalLenCode := literalLen
		if literalLenCode > 15 {
			literalLenCode = 15
		}

		matchLenCode := matchLen - 4 // LZ4 stores matchLen as (actual-4)
		if matchLenCode > 15 {
			matchLenCode = 15
		}

		token := byte(literalLenCode<<4 | matchLenCode)
		dst[dstPos] = token
		dstPos++

		// Write extended literal length if needed
		if literalLen >= 15 {
			remaining := literalLen - 15
			for remaining >= 255 {
				dst[dstPos] = 255
				dstPos++
				remaining -= 255
			}
			dst[dstPos] = byte(remaining)
			dstPos++
		}

		// Copy literal data
		copy(dst[dstPos:], input[lastLiteral:srcPos])
		dstPos += literalLen

		// Write match offset (2 bytes, little-endian)
		dst[dstPos] = byte(offset)
		dstPos++
		dst[dstPos] = byte(offset >> 8)
		dstPos++

		// Write extended match length if needed
		if matchLen-4 >= 15 {
			remaining := matchLen - 4 - 15
			for remaining >= 255 {
				dst[dstPos] = 255
				dstPos++
				remaining -= 255
			}
			dst[dstPos] = byte(remaining)
			dstPos++
		}

		// Advance source position and last literal marker
		srcPos += matchLen
		lastLiteral = srcPos

		// Advance the matcher
		matcher.Advance(matchLen)
	}

	// Handle the final literal block
	if lastLiteral < inputLen {
		literalLen := inputLen - lastLiteral

		// Write token: literal only, no match
		literalLenCode := literalLen
		if literalLenCode > 15 {
			literalLenCode = 15
		}

		token := byte(literalLenCode << 4) // No match
		dst[dstPos] = token
		dstPos++

		// Write extended literal length if needed
		if literalLen >= 15 {
			remaining := literalLen - 15
			for remaining >= 255 {
				dst[dstPos] = 255
				dstPos++
				remaining -= 255
			}
			dst[dstPos] = byte(remaining)
			dstPos++
		}

		// Copy the remaining literal data
		copy(dst[dstPos:], input[lastLiteral:])
		dstPos += literalLen
	}

	// Return the filled portion of the buffer
	return dst[:dstPos], nil
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

// max returns the larger of a or b
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DecompressBlock decompresses an LZ4 compressed block.
// If dst is nil or too small, a new buffer will be allocated.
func DecompressBlock(src []byte, dst []byte, maxSize int) ([]byte, error) {
	// Validate input
	if len(src) == 0 {
		return nil, errors.New("empty source buffer")
	}

	if maxSize <= 0 {
		maxSize = 64 * 1024 // Default max size if not specified
	}

	if dst == nil || len(dst) < maxSize {
		dst = make([]byte, maxSize)
	}

	srcPos := 0
	dstPos := 0

	// Process the block byte by byte
	for srcPos < len(src) {
		// Read the token
		if srcPos >= len(src) {
			return nil, errors.New("invalid block: unexpected end of input")
		}

		token := src[srcPos]
		srcPos++

		// Extract literal length from the high 4 bits
		literalLen := int(token >> 4)

		// Handle extended literal length (if the literal length is 15)
		if literalLen == 15 {
			for srcPos < len(src) {
				l := int(src[srcPos])
				srcPos++
				literalLen += l
				if l != 255 {
					break
				}
			}
		}

		// Check if we have enough space for the literal data
		if srcPos+literalLen > len(src) {
			return nil, errors.New("source buffer too small for literal data")
		}

		// Check if we have enough space in the destination buffer
		if dstPos+literalLen > len(dst) {
			// Grow the destination buffer if needed
			newSize := max(len(dst)*2, dstPos+literalLen)
			if newSize > maxSize {
				return nil, errors.New("decompressed data would exceed maxSize")
			}

			newDst := make([]byte, newSize)
			copy(newDst, dst[:dstPos])
			dst = newDst
		}

		// Copy literal data
		copy(dst[dstPos:], src[srcPos:srcPos+literalLen])
		srcPos += literalLen
		dstPos += literalLen

		// If we've reached the end of the block, break
		if srcPos >= len(src) {
			break
		}

		// Extract match offset (2 bytes, little-endian)
		if srcPos+2 > len(src) {
			return nil, errors.New("invalid block: missing match offset")
		}

		offset := int(src[srcPos]) | int(src[srcPos+1])<<8
		srcPos += 2

		// Zero offset is invalid
		if offset == 0 {
			return nil, errors.New("invalid match offset 0")
		}

		// Extract match length from the low 4 bits of the token
		matchLen := int(token & 0x0F)

		// Handle extended match length (if match length is 15)
		if matchLen == 15 {
			for srcPos < len(src) {
				l := int(src[srcPos])
				srcPos++
				matchLen += l
				if l != 255 {
					break
				}
			}
		}

		// LZ4 stores matchLen as (actual-4), add the implicit 4 back
		matchLen += 4

		// Check if the match offset is valid
		if offset > dstPos {
			return nil, errors.New("invalid match: offset beyond current position")
		}

		// Check if we have enough space in the destination buffer
		if dstPos+matchLen > len(dst) {
			// Grow the destination buffer if needed
			newSize := max(len(dst)*2, dstPos+matchLen)
			if newSize > maxSize {
				return nil, errors.New("decompressed data would exceed maxSize")
			}

			newDst := make([]byte, newSize)
			copy(newDst, dst[:dstPos])
			dst = newDst
		}

		// Copy match data (LZ4 allows overlap between match and destination)
		src_idx := dstPos - offset
		for i := 0; i < matchLen; i++ {
			dst[dstPos+i] = dst[src_idx+i]
		}

		dstPos += matchLen
	}

	return dst[:dstPos], nil
}
