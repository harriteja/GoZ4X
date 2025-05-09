// Package v04 provides the version 0.4 implementation of the GoZ4X compression library
// with SIMD optimizations.
package v04

import (
	"errors"
	"runtime"

	"github.com/harriteja/GoZ4X/compress"
	v03 "github.com/harriteja/GoZ4X/v03"
	"github.com/harriteja/GoZ4X/v04/simd"
)

// MaxCompressedSize returns the maximum size required for compressing data of length sourceSize
func MaxCompressedSize(sourceSize int) int {
	// LZ4 worst case size formula:
	// Maximum output = input + (input / 255) + 16
	return sourceSize + (sourceSize / 255) + 16
}

// Version constants
const (
	// Version of this implementation
	Version = "0.4.0"
)

// CompressionLevel defines the compression level
type CompressionLevel int

// Compression levels
const (
	DefaultLevel CompressionLevel = 6 // Default compression level

	// Level ranges
	MinLevel CompressionLevel = 1  // Fastest compression
	MaxLevel CompressionLevel = 12 // Best compression
)

// Configuration options for v0.4 compression
type Options struct {
	// Compression level (1-12)
	Level CompressionLevel

	// Use v0.2 compression algorithm (improved match finding)
	UseV2 bool

	// SIMD implementation to use
	SIMDImpl int

	// Block size for compression
	BlockSize int

	// Window size for compression
	WindowSize int

	// Number of worker goroutines for parallel compression
	NumWorkers int
}

// DefaultOptions returns the default options for v0.4 compression
func DefaultOptions() Options {
	return Options{
		Level:      DefaultLevel,
		UseV2:      true,
		SIMDImpl:   simd.BestImplementation(),
		BlockSize:  4 * 1024 * 1024, // 4MB default block size
		WindowSize: 64 * 1024,       // 64KB window size (LZ4 standard)
		NumWorkers: runtime.NumCPU(),
	}
}

// CompressBlock compresses a block using v0.4 implementation with SIMD optimizations.
// It allocates a new destination slice if dst is nil or too small.
func CompressBlock(src []byte, dst []byte) ([]byte, error) {
	return CompressBlockWithOptions(src, dst, DefaultOptions())
}

// CompressBlockLevel compresses a block using v0.4 implementation with the specified level.
// It allocates a new destination slice if dst is nil or too small.
func CompressBlockLevel(src []byte, dst []byte, level int) ([]byte, error) {
	opts := DefaultOptions()
	opts.Level = CompressionLevel(level)
	return CompressBlockWithOptions(src, dst, opts)
}

// CompressBlockWithOptions compresses a block with custom options.
// This is the core function of v0.4 implementation.
func CompressBlockWithOptions(src []byte, dst []byte, opts Options) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("empty source buffer")
	}

	// Validate options
	if opts.Level < MinLevel || opts.Level > MaxLevel {
		opts.Level = DefaultLevel
	}

	// Select the appropriate implementation based on SIMD capabilities
	simdImpl := opts.SIMDImpl
	if simdImpl <= 0 {
		simdImpl = simd.BestImplementation()
	}

	// Use SIMD optimizations when available
	switch simdImpl {
	case simd.ImplSSE41:
		// SSE4.1 optimization
		return compressBlockSSE41(src, dst, opts)
	case simd.ImplAVX2, simd.ImplAVX512:
		// AVX2/AVX512 optimizations
		return compressBlockAVX(src, dst, opts)
	case simd.ImplNEON:
		// ARM NEON optimizations
		return compressBlockNEON(src, dst, opts)
	default:
		// Generic implementation
		if opts.UseV2 {
			return compress.CompressBlockV2Level(src, dst, compress.CompressionLevel(opts.Level))
		}
		return compress.CompressBlockLevel(src, dst, compress.CompressionLevel(opts.Level))
	}
}

// compressBlockSSE41 implements LZ4 block compression with SSE4.1 optimizations
func compressBlockSSE41(src []byte, dst []byte, opts Options) ([]byte, error) {
	// Use the standard implementation for now
	// In the future, this would be fully SIMD optimized
	if opts.UseV2 {
		return compress.CompressBlockV2Level(src, dst, compress.CompressionLevel(opts.Level))
	}
	return compress.CompressBlockLevel(src, dst, compress.CompressionLevel(opts.Level))
}

// compressBlockNEON implements LZ4 block compression with ARM NEON optimizations
func compressBlockNEON(src []byte, dst []byte, opts Options) ([]byte, error) {
	// Use the standard implementation for now
	// In the future, this would be fully SIMD optimized
	if opts.UseV2 {
		return compress.CompressBlockV2Level(src, dst, compress.CompressionLevel(opts.Level))
	}
	return compress.CompressBlockLevel(src, dst, compress.CompressionLevel(opts.Level))
}

// compressBlockAVX implements LZ4 block compression with AVX/AVX2 optimizations
func compressBlockAVX(src []byte, dst []byte, opts Options) ([]byte, error) {
	// For now, fall back to SSE4.1 implementation
	// In the future, we'll add AVX-specific optimizations
	return compressBlockSSE41(src, dst, opts)
}

// CompressBlockParallel compresses a block using multiple goroutines with default options.
// This provides better performance on multicore systems for large inputs.
func CompressBlockParallel(src []byte, dst []byte) ([]byte, error) {
	opts := DefaultOptions()
	return CompressBlockParallelWithOptions(src, dst, opts)
}

// CompressBlockParallelWithOptions compresses a block using multiple goroutines with custom options.
func CompressBlockParallelWithOptions(src []byte, dst []byte, opts Options) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("empty source buffer")
	}

	// For now, all implementations use the v0.3 parallel framework with our SIMD acceleration
	// In the future, we'll have SIMD-specific parallel implementations
	if opts.UseV2 {
		return v03.CompressBlockV2ParallelLevel(src, dst, int(opts.Level))
	}

	return v03.CompressBlockParallelLevel(src, dst, int(opts.Level))
}
