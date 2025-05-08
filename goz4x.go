// Package goz4x provides a fast, pure-Go implementation of the LZ4 compression algorithm.
package goz4x

import (
	"io"

	"github.com/harriteja/GoZ4X/compress"
)

// Version constants
const (
	// Version of the library
	Version = "0.2.0"
	// VersionMajor is the major version number
	VersionMajor = 0
	// VersionMinor is the minor version number
	VersionMinor = 2
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
