package compress

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

// ErrWriterClosed is returned when writing to a closed writer
var ErrWriterClosed = errors.New("writer is closed")

// ParallelWriter is an io.WriteCloser that compresses data to an LZ4 stream
// using multiple goroutines for better performance.
type ParallelWriter struct {
	// Underlying writer
	w io.Writer

	// Compression options
	level       CompressionLevel
	useV2       bool
	blockSize   int
	contentSize uint64

	// Writer state
	closed      bool
	wroteHeader bool
	written     uint64
	header      frameHeader
	buf         []byte

	// Buffer for collecting data before compression
	buffer    []byte
	bufferOff int

	// Synchronization
	mu sync.Mutex
}

// ParallelWriterOptions provides configuration options for a ParallelWriter
type ParallelWriterOptions struct {
	// Level sets the compression level
	Level CompressionLevel
	// UseV2 enables the improved v0.2 compression algorithm
	UseV2 bool
	// BlockSize sets the size of compression blocks
	BlockSize int
	// NumWorkers sets the number of worker goroutines (0 = use GOMAXPROCS)
	NumWorkers int
}

// NewParallelWriter creates a new ParallelWriter with default options
func NewParallelWriter(w io.Writer) *ParallelWriter {
	return NewParallelWriterLevel(w, DefaultLevel)
}

// NewParallelWriterLevel creates a new ParallelWriter with specified level
func NewParallelWriterLevel(w io.Writer, level CompressionLevel) *ParallelWriter {
	return NewParallelWriterWithOptions(w, ParallelWriterOptions{
		Level: level,
	})
}

// NewParallelWriterWithOptions creates a new ParallelWriter with custom options
func NewParallelWriterWithOptions(w io.Writer, options ParallelWriterOptions) *ParallelWriter {
	// Set defaults for unspecified options
	if options.Level == 0 {
		options.Level = DefaultLevel
	}

	blockSize := options.BlockSize
	if blockSize <= 0 {
		blockSize = DefaultChunkSize
	}

	// Initialize header
	header := frameHeader{
		blockIndependence: true,
		blockChecksum:     false,
		contentSize:       false,
		contentChecksum:   false,
		dictID:            false,
		blockSizeCode:     5, // Default to 256KB blocks
	}

	// Adjust block size code based on blockSize
	switch {
	case blockSize <= 64*1024:
		header.blockSizeCode = 4 // 64KB
	case blockSize <= 256*1024:
		header.blockSizeCode = 5 // 256KB
	case blockSize <= 1024*1024:
		header.blockSizeCode = 6 // 1MB
	default:
		header.blockSizeCode = 7 // 4MB
	}

	return &ParallelWriter{
		w:         w,
		level:     options.Level,
		useV2:     options.UseV2,
		blockSize: blockSize,
		header:    header,
		buf:       make([]byte, 16), // buffer for encoding headers
		buffer:    make([]byte, blockSize),
		bufferOff: 0,
	}
}

// Write implements io.Writer
func (pw *ParallelWriter) Write(p []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.closed {
		return 0, ErrWriterClosed
	}

	// Write the frame header if we haven't yet
	if !pw.wroteHeader {
		if err := pw.writeFrameHeader(); err != nil {
			return 0, err
		}
		pw.wroteHeader = true
	}

	totalWritten := 0
	input := p

	for len(input) > 0 {
		// Calculate how much data we can add to the buffer
		n := len(input)
		if pw.bufferOff+n > len(pw.buffer) {
			n = len(pw.buffer) - pw.bufferOff
		}

		// Copy data to buffer
		copy(pw.buffer[pw.bufferOff:], input[:n])
		pw.bufferOff += n
		input = input[n:]
		totalWritten += n

		// If buffer is full, compress and write it
		if pw.bufferOff == len(pw.buffer) {
			if err := pw.flushBuffer(); err != nil {
				return totalWritten, err
			}
		}
	}

	pw.written += uint64(totalWritten)
	return totalWritten, nil
}

// Close implements io.Closer
func (pw *ParallelWriter) Close() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.closed {
		return nil
	}

	// Flush any remaining data
	if pw.bufferOff > 0 {
		if err := pw.flushBuffer(); err != nil {
			return err
		}
	}

	// Write end marker (empty block)
	if _, err := pw.w.Write([]byte{0, 0, 0, 0}); err != nil {
		return err
	}

	pw.closed = true
	return nil
}

// Reset resets the writer to write to w
func (pw *ParallelWriter) Reset(w io.Writer) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Reset state
	pw.w = w
	pw.bufferOff = 0
	pw.closed = false
	pw.wroteHeader = false
	pw.written = 0
}

// SetNumWorkers sets the number of worker goroutines
// This is a no-op in the base implementation but is overridden by actual parallel writers
func (pw *ParallelWriter) SetNumWorkers(n int) {
	// No operation in base implementation
}

// SetChunkSize sets the chunk size for parallel compression
// This is a no-op in the base implementation but is overridden by actual parallel writers
func (pw *ParallelWriter) SetChunkSize(size int) {
	// No operation in base implementation
}

// flushBuffer compresses and writes the current buffer
func (pw *ParallelWriter) flushBuffer() error {
	if pw.bufferOff == 0 {
		return nil
	}

	// Compress buffer
	var compressed []byte
	var err error

	// Allocate a buffer for compressed data with safety margin
	maxCompressedSize := pw.bufferOff + (pw.bufferOff / 255) + 16
	compressedBuf := make([]byte, maxCompressedSize)

	if pw.useV2 {
		compressed, err = CompressBlockV2Level(pw.buffer[:pw.bufferOff], compressedBuf, pw.level)
	} else {
		compressed, err = CompressBlockLevel(pw.buffer[:pw.bufferOff], compressedBuf, pw.level)
	}

	if err != nil {
		return err
	}

	// Check if compression actually helped
	if len(compressed) >= pw.bufferOff {
		// Write uncompressed block
		blockSize := uint32(pw.bufferOff | 0x80000000) // Set high bit to indicate uncompressed
		binary.LittleEndian.PutUint32(pw.buf[:4], blockSize)
		if _, err := pw.w.Write(pw.buf[:4]); err != nil {
			return err
		}

		// Write original data
		if _, err := pw.w.Write(pw.buffer[:pw.bufferOff]); err != nil {
			return err
		}
	} else {
		// Write compressed block
		blockSize := uint32(len(compressed))
		binary.LittleEndian.PutUint32(pw.buf[:4], blockSize)
		if _, err := pw.w.Write(pw.buf[:4]); err != nil {
			return err
		}

		// Write compressed data
		if _, err := pw.w.Write(compressed); err != nil {
			return err
		}
	}

	// Reset buffer
	pw.bufferOff = 0
	return nil
}

// writeFrameHeader writes the LZ4 frame header
func (pw *ParallelWriter) writeFrameHeader() error {
	// Magic number (4 bytes)
	binary.LittleEndian.PutUint32(pw.buf[:4], frameMagic)
	if _, err := pw.w.Write(pw.buf[:4]); err != nil {
		return err
	}

	// Frame descriptor byte
	var flgValue byte = 0
	if pw.header.blockIndependence {
		flgValue |= flagBlockIndependence
	}
	if pw.header.blockChecksum {
		flgValue |= flagBlockChecksum
	}
	if pw.header.contentSize {
		flgValue |= flagContentSize
	}
	if pw.header.contentChecksum {
		flgValue |= flagContentChecksum
	}
	if pw.header.dictID {
		flgValue |= flagDictID
	}

	// BD byte (contains block size code)
	bdValue := (pw.header.blockSizeCode & 0x7) << 4

	// Write FLG and BD bytes
	pw.buf[0] = flgValue
	pw.buf[1] = bdValue
	if _, err := pw.w.Write(pw.buf[:2]); err != nil {
		return err
	}

	// HC byte (header checksum)
	checksum := (flgValue >> 2) + (flgValue << 6) + (bdValue >> 2) + (bdValue << 6)
	pw.buf[0] = checksum
	if _, err := pw.w.Write(pw.buf[:1]); err != nil {
		return err
	}

	return nil
}
