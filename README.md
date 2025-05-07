# GoZ4X

A pure-Go, ultra-fast, high-compression LZ4 library for modern workloads—server, edge, WASM, GPU/accelerator offload.

## Features in v0.1

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

## TODO features

- Go generics for clean, reusable match-finder and streaming APIs
- SIMD/assembly for match searching and copy loops (SSE4.1, AVX2, NEON)
- Parallel block compression to saturate all cores
- Pluggable backends (pure-Go, assembly, GPU) selected at runtime
- First-class WASM support for browser & edge functions
- Comprehensive benchmark suite


## Status

This is version 0.1, providing basic functionality with a focus on the streaming API. Block-level compression and advanced features are still in development. In this initial version, compression ratio is not optimized yet, but the foundation is in place for future improvements.

## Usage

### Streaming API

```go
package main

import (
    "bytes"
    "fmt"
    "io"
    "strings"
    
    "github.com/harriteja/GoZ4X/compress"
)

func main() {
    // Create sample data
    data := "Hello, GoZ4X!"
    
    // Compress
    var buf bytes.Buffer
    w := compress.NewWriter(&buf)
    w.Write([]byte(data))
    w.Close()
    
    fmt.Printf("Original size: %d bytes\n", len(data))
    fmt.Printf("Compressed size: %d bytes\n", buf.Len())
    
    // Decompress
    r := compress.NewReader(bytes.NewReader(buf.Bytes()))
    result, _ := io.ReadAll(r)
    
    fmt.Printf("Decompressed: %s\n", string(result))
}
```

## Installation

```
go get github.com/harriteja/GoZ4X
```

## Roadmap

- v0.1: Pure-Go implementation with streaming API (completed)
- v0.2: Improved block compression with proper match finding
- v0.3: Parallelism + HC levels refinement
- v0.4: SIMD optimizations where possible
- v1.0: Stable release with optimized performance

## License

MIT License 