package compress

import (
	"github.com/harriteja/GoZ4X/matcher"
)

const (
	// Maximum size of a literal sequence
	maxLiteralLength = 0xFFFFFF
	// Maximum size of a match sequence
	maxMatchLength = 0xFFFF
)

// V2Block represents an improved LZ4 block with better compression capabilities
type V2Block struct {
	// Input data
	src []byte
	// Compression level
	level CompressionLevel
	// LZ4X matcher
	matcher *matcher.LZ4XMatcher
	// Options
	options BlockOptions
}

// NewV2Block creates a new V2Block with improved compression
func NewV2Block(src []byte, level CompressionLevel, options BlockOptions) (*V2Block, error) {
	if len(src) < MinBlockSize || len(src) > MaxBlockSize {
		return nil, ErrInvalidBlockSize
	}

	if level < 0 || level > MaxLevel {
		return nil, ErrInvalidCompressionLevel
	}

	// Create configuration based on compression level
	config := matcher.LZ4XConfig{
		HashLog:      16,
		WindowSize:   65535,
		MaxAttempts:  8,
		SkipStrength: 1,
	}

	// Adjust settings based on compression level for better performance
	// Higher levels do more thorough searching for matches
	switch {
	case level <= 3:
		config.MaxAttempts = 4
		config.SkipStrength = 1
	case level <= 6:
		config.MaxAttempts = 8
		config.SkipStrength = 2
	case level <= 9:
		config.MaxAttempts = 16
		config.SkipStrength = 2
	default:
		config.MaxAttempts = 32
		config.SkipStrength = 3
	}

	// Create matcher
	lz4xMatcher := matcher.NewLZ4XMatcher(config)
	lz4xMatcher.Reset(src)

	return &V2Block{
		src:     src,
		level:   level,
		matcher: lz4xMatcher,
		options: options,
	}, nil
}

// CompressToBuffer compresses the block data to the provided buffer
func (b *V2Block) CompressToBuffer(dst []byte) ([]byte, error) {
	// Input data and length
	inputLen := len(b.src)

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

	// Pre-initialize hash table for better compression
	if b.level >= 4 {
		// Initialize more of the hash table for higher levels
		limit := min(inputLen-4, 512)
		if b.level >= 8 {
			limit = min(inputLen-4, 1024)
		}

		// Initialize by stepping
		step := 4
		for i := 0; i < limit; i += step {
			b.matcher.InsertHash(i)
		}
		b.matcher.Advance(0) // Keep the position at 0
	}

	// Main compression loop
	for !b.matcher.End() {
		// Find the best match at the current position
		offset, matchLen := b.matcher.FindBestMatch()

		// If no good match, advance and continue
		if matchLen < 4 {
			// No good match found, just advance one byte
			b.matcher.Advance(1)
			srcPos++
			continue
		}

		// We found a match, output the literal sequence since the last match
		literalLen := srcPos - lastLiteral

		// Write token: 4 bits for literal length, 4 bits for match length
		literalLenCode := min(literalLen, 15)
		matchLenCode := min(matchLen-4, 15) // LZ4 stores matchLen as (actual-4)

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
		copy(dst[dstPos:], b.src[lastLiteral:srcPos])
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
		b.matcher.Advance(matchLen)
	}

	// Handle the final literal block
	if lastLiteral < inputLen {
		literalLen := inputLen - lastLiteral

		// Write token: literal only, no match
		literalLenCode := min(literalLen, 15)
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
		copy(dst[dstPos:], b.src[lastLiteral:])
		dstPos += literalLen
	}

	// Return the filled portion of the buffer
	return dst[:dstPos], nil
}

// CompressBlockV2 compresses the src data using the improved LZ4X algorithm
// with default compression level. It returns the compressed data.
func CompressBlockV2(src []byte, dst []byte) ([]byte, error) {
	return CompressBlockV2Level(src, dst, DefaultLevel)
}

// CompressBlockV2Level compresses the src data using the improved LZ4X algorithm
// with specified compression level. It returns the compressed data.
func CompressBlockV2Level(src []byte, dst []byte, level CompressionLevel) ([]byte, error) {
	// Create a V2Block
	block, err := NewV2Block(src, level, BlockOptions{})
	if err != nil {
		return nil, err
	}

	// Compress the data
	return block.CompressToBuffer(dst)
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DecompressBlockV2 decompresses a block of data compressed with LZ4X v0.2
// The implementation is compatible with regular LZ4 decompression
func DecompressBlockV2(src []byte, dst []byte, maxSize int) ([]byte, error) {
	// Reuse the existing decompression code since the format is compatible
	return DecompressBlock(src, dst, maxSize)
}
