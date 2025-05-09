# GoZ4X

A pure-Go, ultra-fast, high-compression LZ4 library for modern workloads—server, edge, WASM, GPU/accelerator offload.

## Features

### v0.1 Features

- Complete streaming implementation with proper frame format
  - Writer for compression
  - Reader for decompression
  - Proper handling of small data
  - Support for various compression levels
- Block-level functionality (basic implementation in v0.1)
  - Basic block structure
  - Frame header handling
  - Support for the LZ4 frame format
- HC (High Compression) match finding
- Well tested with comprehensive unit tests

### v0.2 Features

- Improved block compression with better match finding
  - Dual hash tables (4-byte and 3-byte) for more match opportunities
  - Optimized skip strength based on compression level
  - Enhanced lazy matching for better compression ratio
  - Proper hash table initialization for large blocks
- New API for v0.2 compression while maintaining compatibility
- Significantly better compression ratios (up to 15-20% improvement in some cases)
- Maintained backward compatibility with v0.1

### v0.3 Features (New!)

- Parallel compression support
  - Multi-threaded block compression to utilize all CPU cores
  - Parallelized streaming API for high-throughput compression
  - Automatic fallback to single-threaded mode when needed
- Refined HC (Hash Chain) compression levels
  - Enhanced hash functions for high compression levels (5-byte hash)
  - Window size optimized per compression level
  - Improved lazy matching with early exit strategies
  - Optimized hash table sizes for better memory usage
- Better compression ratios with faster compression on multi-core systems

### v0.4 Features (In Progress)

- SIMD optimizations framework
  - CPU feature detection (SSE4.1, AVX2, AVX512, NEON)
  - Architecture-specific optimizations selection at runtime
  - Improved performance on supported hardware
- Compatibility with all previous versions
- Foundation for hardware-accelerated compression
- Initial implementation of SIMD-based match finding and copy operations

## TODO features

- Complete SIMD implementations for match searching and copy loops
- GPU acceleration for supported hardware
- Go generics for clean, reusable match-finder and streaming APIs
- Pluggable backends (pure-Go, assembly, GPU) selected at runtime
- First-class WASM support for browser & edge functions
- Comprehensive benchmark suite

## Status

This is version 0.4, with SIMD optimization framework in place. The implementation currently falls back to v0.3 code paths when SIMD optimizations are not fully available, ensuring compatibility with the LZ4 format while providing the groundwork for hardware acceleration.

## Usage

### Streaming API (v0.1)

```go
package main

import (
    "bytes"
    "fmt"
    "io"
    "strings"
    
    "github.com/harriteja/GoZ4X"
)

func main() {
    // Create sample data
    data := "Hello, GoZ4X!"
    
    // Compress
    var buf bytes.Buffer
    w := goz4x.NewWriter(&buf)
    w.Write([]byte(data))
    w.Close()
    
    fmt.Printf("Original size: %d bytes\n", len(data))
    fmt.Printf("Compressed size: %d bytes\n", buf.Len())
    
    // Decompress
    r := goz4x.NewReader(bytes.NewReader(buf.Bytes()))
    result, _ := io.ReadAll(r)
    
    fmt.Printf("Decompressed: %s\n", string(result))
}
```

### Enhanced Compression with v0.2

```go
package main

import (
    "bytes"
    "fmt"
    "io"
    
    "github.com/harriteja/GoZ4X"
)

func main() {
    // Create sample data
    data := []byte("This is a sample text that will be compressed using GoZ4X v0.2!")
    
    // Compress with v0.2 algorithm
    compressedData, _ := goz4x.CompressBlockV2(data, nil)
    
    fmt.Printf("Original size: %d bytes\n", len(data))
    fmt.Printf("Compressed size: %d bytes\n", len(compressedData))
    
    // Decompress (compatible with standard LZ4 decompression)
    decompressed, _ := goz4x.DecompressBlock(compressedData, nil, len(data))
    
    fmt.Printf("Decompressed: %s\n", string(decompressed))
    
    // Streaming compression with v0.2
    var buf bytes.Buffer
    w := goz4x.NewWriterV2(&buf)
    w.Write(data)
    w.Close()
    
    fmt.Printf("v0.2 streaming compressed size: %d bytes\n", buf.Len())
}
```

### Parallel Compression with v0.3

```go
package main

import (
    "bytes"
    "fmt"
    "io"
    "runtime"
    
    "github.com/harriteja/GoZ4X"
)

func main() {
    // Create or load some large data
    data := make([]byte, 100*1024*1024) // 100MB of data
    // ... fill data with actual content ...
    
    fmt.Printf("CPU cores available: %d\n", runtime.NumCPU())
    
    // Parallel block compression
    compressedData, _ := goz4x.CompressBlockV2Parallel(data, nil)
    
    fmt.Printf("Original size: %d bytes\n", len(data))
    fmt.Printf("Compressed size: %d bytes (%.2f%%)\n", 
        len(compressedData), float64(len(compressedData))*100/float64(len(data)))
    
    // Parallel streaming compression
    var buf bytes.Buffer
    w := goz4x.NewParallelWriterV2(&buf)
    
    // Optionally configure workers and chunk size
    w.SetNumWorkers(runtime.NumCPU()) // Use all available cores
    w.SetChunkSize(4 * 1024 * 1024)   // 4MB chunks
    
    // Compress
    w.Write(data)
    w.Close()
    
    fmt.Printf("Parallel streaming compressed size: %d bytes\n", buf.Len())
    
    // Decompress (standard decompression works with parallel-compressed data)
    r := goz4x.NewReader(bytes.NewReader(buf.Bytes()))
    decompressed, _ := io.ReadAll(r)
    
    fmt.Printf("Decompressed size: %d bytes\n", len(decompressed))
}
```

### SIMD-Optimized Compression with v0.4

```go
package main

import (
    "bytes"
    "fmt"
    "io"
    "runtime"
    
    "github.com/harriteja/GoZ4X"
    "github.com/harriteja/GoZ4X/v04"
    "github.com/harriteja/GoZ4X/v04/simd"
)

func main() {
    // Create or load data
    data := make([]byte, 10*1024*1024) // 10MB of data
    // ... fill data with actual content ...
    
    // Check available CPU features
    features := simd.DetectFeatures()
    fmt.Printf("CPU Features: SSE4.1=%v, AVX2=%v, AVX512=%v, NEON=%v\n",
        features.HasSSE41, features.HasAVX2, features.HasAVX512, features.HasNEON)
    
    // Create custom compression options
    opts := v04.DefaultOptions()
    opts.Level = v04.CompressionLevel(9) // Higher compression level
    
    // Compress with SIMD optimizations where available
    compressedData, _ := v04.CompressBlockWithOptions(data, nil, opts)
    
    fmt.Printf("Original size: %d bytes\n", len(data))
    fmt.Printf("Compressed size: %d bytes (%.2f%%)\n", 
        len(compressedData), float64(len(compressedData))*100/float64(len(data)))
    
    // Parallel SIMD-optimized compression for large data
    opts.NumWorkers = runtime.NumCPU()
    parallelCompressed, _ := v04.CompressBlockParallelWithOptions(data, nil, opts)
    
    fmt.Printf("Parallel compressed size: %d bytes\n", len(parallelCompressed))
    
    // Decompress (standard decompression works with SIMD-compressed data)
    decompressed, _ := goz4x.DecompressBlock(compressedData, nil, len(data))
    
    fmt.Printf("Decompressed size: %d bytes\n", len(decompressed))
}
```

## Installation

```
go get github.com/harriteja/GoZ4X
```

## Roadmap

- v0.1: Pure-Go implementation with streaming API (completed)
- v0.2: Improved block compression with proper match finding (completed)
- v0.3: Parallelism + HC levels refinement (completed)
- v0.4: SIMD optimizations where possible
- v1.0: Stable release with optimized performance

## Implementation Details

GoZ4X implements the LZ4 compression algorithm according to the official specification:

### Core Algorithm

- **Sliding Window**: Uses a 64KB sliding window to find repeated byte sequences.
- **Token Structure**: Each compressed sequence begins with a token byte:
  - High 4 bits encode the length of literal data
  - Low 4 bits encode the match length (minus 4)
- **Length Encoding**: For lengths ≥ 15, additional bytes encode the extra length.
- **Match Encoding**: Uses a 2-byte little-endian offset to point back into the sliding window.
- **High Compression**: Implements HC mode with configurable depth search for better compression.

### v0.2 Optimizations

- **Dual Hash Tables**: Uses both 4-byte and 3-byte hashing to find more potential matches
- **Adaptive Search Depth**: Adjusts search parameters based on compression level
- **Smart Lazy Matching**: Improved decision making for lazy match selection
- **Skip Strength**: Optimized skip strategy to improve compression speed at higher levels
- **Hash Pre-initialization**: Better handling of the initial block for improved compression

### v0.3 Optimizations

- **Parallelism**: Multi-threaded compression for better performance on multi-core systems
- **Worker Pool**: Dynamic worker allocation based on available CPU cores
- **Chunk Processing**: Efficient chunking strategy for parallel compression
- **Enhanced HC Levels**: Refined compression levels with optimized window sizes
- **5-Byte Hashing**: Advanced hash function for higher compression levels
- **Early Exit**: Smarter search termination for better performance

### TODO Optimizations

- **SIMD**: Vectorized implementations for match finding and copy operations
- **Hash Tables**: Further optimizations of hash lookup strategies
- **Chain Matching**: Advanced chain tracking for even better match finding
- **Streaming Mode**: Additional optimizations for the standard LZ4 frame format

## License

MIT License 