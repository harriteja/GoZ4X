# GoZ4X

A pure-Go, ultra-fast, high-compression LZ4 library for modern workloads—server, edge, WASM, GPU/accelerator offload.

## Features

- Go generics for clean, reusable match-finder and streaming APIs
- SIMD/assembly for match searching and copy loops (SSE4.1, AVX2, NEON)
- Parallel block compression to saturate all cores
- Pluggable backends (pure-Go, assembly, GPU) selected at runtime
- First-class WASM support for browser & edge functions
- Comprehensive benchmark suite

## Usage

```go
import "github.com/harriteja/GoZ4X"

// Compress a byte slice
compressed := goz4x.CompressBlock(data, nil)

// Use the streaming API
r, w := io.Pipe()
go func() {
    zw := goz4x.NewWriter(w)
    io.Copy(zw, sourceReader)
    zw.Close()
    w.Close()
}()
io.Copy(destination, goz4x.NewReader(r))
```

## Installation

```
go get github.com/harriteja/GoZ4X
```

## Roadmap

- v0.1: Pure-Go generics-based block & stream API
- v0.2: Parallelism + HC levels
- v0.3: SIMD backend
- v1.0: Stable release
- v2.0: Next-gen features

## License

MIT License 