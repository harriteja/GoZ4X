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

### v0.4 Features (Completed)

- SIMD optimizations framework
  - CPU feature detection (SSE4.1, AVX2, AVX512, NEON)
  - Architecture-specific optimizations selection at runtime
  - Improved performance on supported hardware
- Compatibility with all previous versions
- Foundation for hardware-accelerated compression
- Initial implementation of SIMD-based match finding and copy operations

## TODO features

- Complete SIMD implementations for match searching and copy loops
  - AVX2 and AVX512 optimizations for x86_64 architectures
  - Extended NEON support for ARM64 platforms
  - Specialized copy and match finding routines for different instruction sets
- GPU acceleration for supported hardware
  - CUDA/OpenCL backends for NVIDIA/AMD GPUs
  - Runtime detection for GPU availability
  - Fallback mechanisms for non-GPU environments
- Go generics for clean, reusable match-finder and streaming APIs
  - Type-safe match finding algorithms
  - Generic streaming interfaces for different backends
- Pluggable backends (pure-Go, assembly, GPU) selected at runtime
- First-class WASM support for browser & edge functions
  - Browser-specific optimizations
  - TypeScript definitions for JavaScript interoperability
  - Examples for browser integration
- Comprehensive benchmark suite
  - Real-world data type benchmarks
  - Comparison with other LZ4 implementations
  - Performance visualization tools
- Advanced integration options
  - io/fs compatibility for modern Go applications
  - Middleware for common web frameworks
  - Cloud storage adapters (S3, GCS)
- Extended testing capabilities
  - Fuzzing tests for robustness
  - Performance regression testing
  - Edge case validation

## Status

This is version 0.4, with SIMD optimization framework completely implemented. The implementation includes architecture-specific match finding and copy operations for both x86-64 (SSE) and ARM64 (NEON) architectures, providing the groundwork for hardware acceleration while ensuring compatibility with the LZ4 format.

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

### Future Features (Coming Soon)

#### GPU Acceleration

```go
package main

import (
    "bytes"
    "fmt"
    
    "github.com/harriteja/GoZ4X"
    "github.com/harriteja/GoZ4X/gpu"
)

func main() {
    // Create or load some large data
    data := make([]byte, 1024*1024*1024) // 1GB of data
    // ... fill data ...
    
    // Check GPU availability
    gpuInfo := gpu.DetectGPUs()
    fmt.Printf("Available GPUs: %d\n", len(gpuInfo))
    for i, info := range gpuInfo {
        fmt.Printf("GPU %d: %s with %dMB memory\n", i, info.Name, info.MemoryMB)
    }
    
    // Create GPU compression options
    opts := gpu.DefaultOptions()
    opts.PreferredDevice = 0 // Use first GPU
    opts.UsePinnedMemory = true // For faster memory transfers
    
    // Compress with GPU acceleration
    compressedData, _ := gpu.CompressBlock(data, nil, opts)
    
    fmt.Printf("Original size: %d MB\n", len(data)/(1024*1024))
    fmt.Printf("Compressed size: %d MB (%.2f%%)\n", 
        len(compressedData)/(1024*1024), 
        float64(len(compressedData))*100/float64(len(data)))
    
    // Stream compression with GPU acceleration
    var buf bytes.Buffer
    w := gpu.NewWriter(&buf, opts)
    w.Write(data)
    w.Close()
}
```

#### WebAssembly Support

```js
// Browser JavaScript
import { GoZ4X } from '@harriteja/goz4x-wasm';

async function compressFile() {
    const fileInput = document.getElementById('fileInput');
    const file = fileInput.files[0];
    
    // Initialize the WASM module
    const goz4x = await GoZ4X.init();
    
    // Read the file
    const arrayBuffer = await file.arrayBuffer();
    const inputData = new Uint8Array(arrayBuffer);
    
    console.log(`Original size: ${inputData.length} bytes`);
    
    // Compress the data
    const compressedData = goz4x.compressBlock(inputData);
    
    console.log(`Compressed size: ${compressedData.length} bytes`);
    console.log(`Compression ratio: ${(compressedData.length * 100 / inputData.length).toFixed(2)}%`);
    
    // Create a download link
    const blob = new Blob([compressedData], { type: 'application/octet-stream' });
    const url = URL.createObjectURL(blob);
    
    const downloadLink = document.createElement('a');
    downloadLink.href = url;
    downloadLink.download = `${file.name}.lz4`;
    downloadLink.click();
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
- v0.4: SIMD optimizations where possible (completed)
- v0.5: Complete SIMD implementations + GPU acceleration framework
- v0.6: WebAssembly optimization + enhanced benchmarking
- v0.7: Go generics integration + improved API design
- v1.0: Stable release with comprehensive performance optimizations and integrations

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
- **GPU Acceleration**: Offloading compression to graphics hardware
  - **CUDA/OpenCL**: Cross-platform GPU acceleration
  - **Shared Memory**: Efficient data transfer between CPU and GPU
  - **Pipeline Processing**: Simultaneous CPU and GPU operations
- **WebAssembly**:
  - **Browser Optimizations**: Specialized code paths for browser environments
  - **Memory Management**: Efficient handling of browser memory constraints
  - **Worker Threads**: Parallel processing in browser contexts
- **Advanced Integrations**:
  - **io/fs Support**: Full compatibility with modern Go file systems
  - **Middleware Adapters**: Ready-to-use components for web frameworks
  - **Cloud Storage**: Direct compression/decompression with cloud providers

## License

MIT License 