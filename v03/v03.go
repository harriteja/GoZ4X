package v03

import (
	"github.com/harriteja/GoZ4X/compress"
	"github.com/harriteja/GoZ4X/parallel"
)

// CompressBlockParallel compresses a byte slice using multiple goroutines with default compression level.
// This provides better performance on multicore systems for large inputs.
func CompressBlockParallel(src []byte, dst []byte) ([]byte, error) {
	return CompressBlockParallelLevel(src, dst, int(compress.DefaultLevel))
}

// CompressBlockParallelLevel compresses a byte slice using multiple goroutines with the specified level.
// This provides better performance on multicore systems for large inputs.
func CompressBlockParallelLevel(src []byte, dst []byte, level int) ([]byte, error) {
	dispatcher := parallel.NewDispatcher(0, 0) // Use defaults
	defer dispatcher.Stop()

	if err := dispatcher.Start(); err != nil {
		// Fall back to non-parallel compression
		return compress.CompressBlockLevel(src, dst, compress.CompressionLevel(level))
	}

	return dispatcher.CompressBlocks(src, level)
}

// CompressBlockV2Parallel compresses a byte slice using v0.2 algorithm with multiple goroutines.
// This provides better compression ratio and better performance on multicore systems.
func CompressBlockV2Parallel(src []byte, dst []byte) ([]byte, error) {
	return CompressBlockV2ParallelLevel(src, dst, int(compress.DefaultLevel))
}

// CompressBlockV2ParallelLevel compresses a byte slice using v0.2 algorithm with multiple goroutines.
// This provides better compression ratio and better performance on multicore systems.
func CompressBlockV2ParallelLevel(src []byte, dst []byte, level int) ([]byte, error) {
	dispatcher := parallel.NewDispatcher(0, 0) // Use defaults
	defer dispatcher.Stop()

	if err := dispatcher.Start(); err != nil {
		// Fall back to non-parallel compression
		return compress.CompressBlockV2Level(src, dst, compress.CompressionLevel(level))
	}

	return dispatcher.CompressBlocksV2(src, level)
}
