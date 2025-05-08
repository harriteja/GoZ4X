// Package goz4x provides a fast, pure-Go implementation of the LZ4 compression algorithm.
package goz4x

import (
	"io"

	"github.com/harriteja/GoZ4X/compress"
	v03 "github.com/harriteja/GoZ4X/v03"
)

// Version constants
const (
	// Version of the library
	Version = "0.3.0"
	// VersionMajor is the major version number
	VersionMajor = 0
	// VersionMinor is the minor version number
	VersionMinor = 3
	// VersionPatch is the patch version number
	VersionPatch = 0
)

// CompressBlock compresses a byte slice using the default compression level.
// It allocates a new destination slice if dst is nil or too small.
// Returns the compressed data slice.
func CompressBlock(src []byte, dst []byte) ([]byte, error) {
	return compress.CompressBlock(src, dst)
}

// CompressBlockLevel compresses a byte slice with the specified compression level.
// Levels range from 1 (fastest) to 12 (best compression).
// It allocates a new destination slice if dst is nil or too small.
func CompressBlockLevel(src []byte, dst []byte, level int) ([]byte, error) {
	return compress.CompressBlockLevel(src, dst, compress.CompressionLevel(level))
}

// DecompressBlock decompresses an LZ4-compressed block.
// It allocates a new destination slice if dst is nil or too small.
// The maxSize parameter limits the maximum size of the decompressed data.
func DecompressBlock(src []byte, dst []byte, maxSize int) ([]byte, error) {
	return compress.DecompressBlock(src, dst, maxSize)
}

// V2 API functions with improved compression

// CompressBlockV2 compresses a byte slice using the v0.2 algorithm with default compression level.
// It offers better compression ratios than CompressBlock.
func CompressBlockV2(src []byte, dst []byte) ([]byte, error) {
	return compress.CompressBlockV2(src, dst)
}

// CompressBlockV2Level compresses a byte slice with the v0.2 algorithm and specified compression level.
// It offers better compression ratios than CompressBlockLevel.
func CompressBlockV2Level(src []byte, dst []byte, level int) ([]byte, error) {
	return compress.CompressBlockV2Level(src, dst, compress.CompressionLevel(level))
}

// V3 API functions with parallel compression

// CompressBlockParallel compresses a byte slice using multiple goroutines with default compression level.
// This provides better performance on multicore systems for large inputs.
func CompressBlockParallel(src []byte, dst []byte) ([]byte, error) {
	return v03.CompressBlockParallel(src, dst)
}

// CompressBlockParallelLevel compresses a byte slice using multiple goroutines with the specified level.
// This provides better performance on multicore systems for large inputs.
func CompressBlockParallelLevel(src []byte, dst []byte, level int) ([]byte, error) {
	return v03.CompressBlockParallelLevel(src, dst, level)
}

// CompressBlockV2Parallel compresses a byte slice using v0.2 algorithm with multiple goroutines.
// This provides better compression ratio and better performance on multicore systems.
func CompressBlockV2Parallel(src []byte, dst []byte) ([]byte, error) {
	return v03.CompressBlockV2Parallel(src, dst)
}

// CompressBlockV2ParallelLevel compresses a byte slice using v0.2 algorithm and multiple goroutines.
// This provides better compression ratio and better performance on multicore systems.
func CompressBlockV2ParallelLevel(src []byte, dst []byte, level int) ([]byte, error) {
	return v03.CompressBlockV2ParallelLevel(src, dst, level)
}

// Reader is an io.Reader that decompresses data from an LZ4 stream.
type Reader struct {
	r *compress.Reader
}

// NewReader creates a new Reader that decompresses from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: compress.NewReader(r)}
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

// Writer is an io.WriteCloser that compresses data to an LZ4 stream.
type Writer struct {
	w *compress.Writer
}

// NewWriter creates a new Writer that compresses to w using the default compression level.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: compress.NewWriter(w)}
}

// NewWriterLevel creates a new Writer that compresses to w using the specified compression level.
// Levels range from 1 (fastest) to 12 (best compression).
func NewWriterLevel(w io.Writer, level int) *Writer {
	return &Writer{w: compress.NewWriterLevel(w, compress.CompressionLevel(level))}
}

// NewWriterV2 creates a new Writer that compresses to w using the v0.2 algorithm with default level.
// It offers better compression than NewWriter.
func NewWriterV2(w io.Writer) *Writer {
	return &Writer{w: compress.NewWriterWithOptions(w, compress.WriterOptions{
		Level: compress.DefaultLevel,
		UseV2: true,
	})}
}

// NewWriterV2Level creates a new Writer that compresses to w using the v0.2 algorithm with specified level.
// It offers better compression than NewWriterLevel.
func NewWriterV2Level(w io.Writer, level int) *Writer {
	return &Writer{w: compress.NewWriterWithOptions(w, compress.WriterOptions{
		Level: compress.CompressionLevel(level),
		UseV2: true,
	})}
}

// ParallelWriter is an io.WriteCloser that compresses data in parallel for better performance.
type ParallelWriter struct {
	w *v03.ParallelWriter
}

// NewParallelWriter creates a new parallel writer with default options.
func NewParallelWriter(w io.Writer) *ParallelWriter {
	return &ParallelWriter{w: v03.NewParallelWriter(w)}
}

// NewParallelWriterLevel creates a new parallel writer with custom compression level.
func NewParallelWriterLevel(w io.Writer, level int) *ParallelWriter {
	return &ParallelWriter{w: v03.NewParallelWriterLevel(w, level)}
}

// NewParallelWriterV2 creates a new parallel writer using v0.2 algorithm with default options.
func NewParallelWriterV2(w io.Writer) *ParallelWriter {
	return &ParallelWriter{w: v03.NewParallelWriterWithOptions(w, v03.ParallelWriterOptions{
		Level: int(compress.DefaultLevel),
		UseV2: true,
	})}
}

// NewParallelWriterV2Level creates a new parallel writer using v0.2 algorithm with custom level.
func NewParallelWriterV2Level(w io.Writer, level int) *ParallelWriter {
	return &ParallelWriter{w: v03.NewParallelWriterWithOptions(w, v03.ParallelWriterOptions{
		Level: level,
		UseV2: true,
	})}
}

// Write implements io.Writer.
func (w *Writer) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

// Close implements io.Closer.
func (w *Writer) Close() error {
	return w.w.Close()
}

// Reset resets the Writer to write to dst.
func (w *Writer) Reset(dst io.Writer) {
	w.w.Reset(dst)
}

// Write implements io.Writer.
func (pw *ParallelWriter) Write(p []byte) (int, error) {
	return pw.w.Write(p)
}

// Close implements io.Closer.
func (pw *ParallelWriter) Close() error {
	return pw.w.Close()
}

// Reset resets the ParallelWriter to write to dst.
func (pw *ParallelWriter) Reset(dst io.Writer) {
	pw.w.Reset(dst)
}

// SetNumWorkers sets the number of worker goroutines used for compression.
// A value of 0 means use GOMAXPROCS.
func (pw *ParallelWriter) SetNumWorkers(n int) {
	pw.w.SetNumWorkers(n)
}

// SetChunkSize sets the chunk size used for parallel compression.
// A value of 0 means use default chunk size.
func (pw *ParallelWriter) SetChunkSize(size int) {
	pw.w.SetChunkSize(size)
}
