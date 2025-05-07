// Package goz4x provides a fast, pure-Go implementation of the LZ4 compression algorithm.
package goz4x

import (
	"io"

	"github.com/harriteja/GoZ4X/compress"
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
