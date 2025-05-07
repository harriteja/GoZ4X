package compress

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

const (
	// DefaultChunkSize is the default size of chunks for streaming
	DefaultChunkSize = 256 * 1024 // 256KB

	// Magic number for LZ4 frame detection
	frameMagic uint32 = 0x184D2204

	// Maximum size for frame header
	maxHeaderSize = 20
)

var (
	// ErrInvalidFrame indicates an invalid frame format
	ErrInvalidFrame = errors.New("invalid LZ4 frame format")
)

// Reader is an io.Reader that decompresses from an LZ4 stream
type Reader struct {
	r              io.Reader
	buf            []byte
	current        []byte
	header         frameHeader
	readHeader     bool
	reachedEof     bool
	blocksizeCache int
	mu             sync.Mutex
}

// Writer is an io.WriteCloser that compresses to an LZ4 stream
type Writer struct {
	w           io.Writer
	level       CompressionLevel
	blockSize   int
	contentSize uint64
	closed      bool
	compress    *Block[[]byte]
	header      frameHeader
	buf         []byte
	bufUsed     int
	written     uint64
	mu          sync.Mutex
}

// frameHeader contains LZ4 frame information
type frameHeader struct {
	blockIndependence bool
	blockChecksum     bool
	contentSize       bool
	contentChecksum   bool
	dictID            bool
	blockSizeCode     uint8
}

// NewReader returns a new Reader that decompresses from r
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:   r,
		buf: make([]byte, 8192),
	}
}

// Read implements io.Reader
func (r *Reader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reachedEof {
		return 0, io.EOF
	}

	// Read the frame header if we haven't yet
	if !r.readHeader {
		if err := r.readFrameHeader(); err != nil {
			return 0, err
		}
		r.readHeader = true
	}

	// If we have data in current buffer, return it
	if len(r.current) > 0 {
		n := copy(p, r.current)
		r.current = r.current[n:]
		return n, nil
	}

	// Read the next block
	// Placeholder implementation
	n, err := r.r.Read(p)
	if err != nil {
		if err == io.EOF {
			r.reachedEof = true
		}
	}

	return n, err
}

// readFrameHeader reads and validates the LZ4 frame header
func (r *Reader) readFrameHeader() error {
	// Read magic number
	_, err := io.ReadFull(r.r, r.buf[:4])
	if err != nil {
		return err
	}

	magic := binary.LittleEndian.Uint32(r.buf[:4])
	if magic != frameMagic {
		return ErrInvalidFrame
	}

	// Read frame descriptor
	_, err = io.ReadFull(r.r, r.buf[:2])
	if err != nil {
		return err
	}

	flg := r.buf[0]
	bd := r.buf[1]

	// Parse frame descriptor
	r.header.blockIndependence = (flg & 0x20) != 0
	r.header.blockChecksum = (flg & 0x10) != 0
	r.header.contentSize = (flg & 0x08) != 0
	r.header.contentChecksum = (flg & 0x04) != 0
	r.header.dictID = (flg & 0x01) != 0
	r.header.blockSizeCode = (bd >> 4) & 0x7

	// TODO: Handle content size, dictID, and header checksum

	return nil
}

// NewWriter returns a new Writer that compresses to w
func NewWriter(w io.Writer) *Writer {
	return NewWriterLevel(w, DefaultLevel)
}

// NewWriterLevel returns a new Writer that compresses to w with the given level
func NewWriterLevel(w io.Writer, level CompressionLevel) *Writer {
	return &Writer{
		w:         w,
		level:     level,
		blockSize: DefaultChunkSize,
		buf:       make([]byte, DefaultChunkSize),
		header: frameHeader{
			blockIndependence: true,
			blockSizeCode:     7, // 4MB
		},
	}
}

// Write implements io.Writer
func (z *Writer) Write(p []byte) (int, error) {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.closed {
		return 0, errors.New("write to closed writer")
	}

	// If this is the first write, write the frame header
	if z.written == 0 {
		if err := z.writeFrameHeader(); err != nil {
			return 0, err
		}
	}

	total := 0
	for len(p) > 0 {
		// If the buffer is full, flush it
		if z.bufUsed == len(z.buf) {
			if err := z.flush(); err != nil {
				return total, err
			}
		}

		// Copy data to the buffer
		n := copy(z.buf[z.bufUsed:], p)
		z.bufUsed += n
		total += n
		p = p[n:]
	}

	return total, nil
}

// writeFrameHeader writes the LZ4 frame header
func (z *Writer) writeFrameHeader() error {
	// Magic number
	binary.LittleEndian.PutUint32(z.buf[:4], frameMagic)

	// Frame descriptor
	var flg byte
	if z.header.blockIndependence {
		flg |= 0x20
	}
	if z.header.blockChecksum {
		flg |= 0x10
	}
	if z.header.contentSize {
		flg |= 0x08
	}
	if z.header.contentChecksum {
		flg |= 0x04
	}
	if z.header.dictID {
		flg |= 0x01
	}
	z.buf[4] = flg

	// Block descriptor
	z.buf[5] = z.header.blockSizeCode << 4

	// Write the header
	_, err := z.w.Write(z.buf[:6])
	if err != nil {
		return err
	}

	// TODO: Write content size, dictID, and header checksum

	return nil
}

// flush compresses and writes a block
func (z *Writer) flush() error {
	if z.bufUsed == 0 {
		return nil
	}

	// Placeholder implementation
	compressed := make([]byte, z.bufUsed)
	copy(compressed, z.buf[:z.bufUsed])

	// Write block size
	binary.LittleEndian.PutUint32(z.buf[:4], uint32(len(compressed)))

	// Write block size marker
	_, err := z.w.Write(z.buf[:4])
	if err != nil {
		return err
	}

	// Write compressed data
	_, err = z.w.Write(compressed)
	if err != nil {
		return err
	}

	z.written += uint64(z.bufUsed)
	z.bufUsed = 0

	return nil
}

// Close closes the Writer, flushing any unwritten data to the underlying io.Writer
func (z *Writer) Close() error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.closed {
		return nil
	}

	z.closed = true

	// Flush any remaining data
	if err := z.flush(); err != nil {
		return err
	}

	// Write end marker (empty block)
	z.buf[0] = 0
	z.buf[1] = 0
	z.buf[2] = 0
	z.buf[3] = 0
	_, err := z.w.Write(z.buf[:4])

	return err
}

// Reset discards the Writer's state and makes it equivalent to the result of NewWriter
func (z *Writer) Reset(w io.Writer) {
	z.mu.Lock()
	defer z.mu.Unlock()

	z.w = w
	z.bufUsed = 0
	z.written = 0
	z.closed = false
}
