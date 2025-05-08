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

	// LZ4 frame constants
	frameMagic = 0x184D2204

	// Maximum size for frame header
	maxHeaderSize = 20

	// Frame descriptor flags
	flagBlockIndependence = 0x20
	flagBlockChecksum     = 0x10
	flagContentSize       = 0x08
	flagContentChecksum   = 0x04
	flagDictID            = 0x01

	// Maximum block size (corresponds to blockSizeCode 7)
	maxBlockSize = 4 * 1024 * 1024
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
	decompressed   []byte
	bufPos         int
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
	wroteHeader bool
	useV2       bool
	buffer      []byte
	bufferOff   int
}

// frameHeader contains information about the LZ4 frame
type frameHeader struct {
	blockIndependence bool
	blockChecksum     bool
	contentSize       bool
	contentSizeValue  uint64 // Actual content size value
	contentChecksum   bool
	dictID            bool
	dictIDValue       uint32 // Actual dictionary ID value
	blockSizeCode     uint8  // 4-7 (64KB, 256KB, 1MB, 4MB)
}

// WriterOptions provides configuration options for a Writer
type WriterOptions struct {
	// Level sets the compression level
	Level CompressionLevel
	// UseV2 enables the improved v0.2 compression algorithm
	UseV2 bool
	// BlockSize sets the size of compression blocks
	BlockSize int
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

		// Set block size based on header
		switch r.header.blockSizeCode {
		case 4:
			r.blocksizeCache = 64 * 1024
		case 5:
			r.blocksizeCache = 256 * 1024
		case 6:
			r.blocksizeCache = 1 * 1024 * 1024
		case 7:
			r.blocksizeCache = 4 * 1024 * 1024
		default:
			return 0, errors.New("invalid block size code")
		}
	}

	// If we have data in current buffer, return it
	if r.decompressed != nil && r.bufPos < len(r.decompressed) {
		n := copy(p, r.decompressed[r.bufPos:])
		r.bufPos += n

		// If we've consumed all decompressed data, prepare for next block
		if r.bufPos >= len(r.decompressed) {
			r.decompressed = nil
			r.bufPos = 0
		}

		return n, nil
	}

	// Read the next block
	if err := r.readBlock(); err != nil {
		if err == io.EOF {
			r.reachedEof = true
			// Return EOF only if we haven't read anything yet
			if r.decompressed == nil || len(r.decompressed) == 0 {
				return 0, io.EOF
			}
		} else {
			return 0, err
		}
	}

	// If we have no data (empty buffer), return EOF
	if r.decompressed == nil || len(r.decompressed) == 0 {
		r.reachedEof = true
		return 0, io.EOF
	}

	// Now we have data, so read from it
	n := copy(p, r.decompressed)
	r.bufPos = n

	// If we've consumed all decompressed data, prepare for next block
	if r.bufPos >= len(r.decompressed) {
		r.decompressed = nil
		r.bufPos = 0
	}

	return n, nil
}

// readFrameHeader reads and verifies the LZ4 frame header
func (r *Reader) readFrameHeader() error {
	// Read magic number (4 bytes)
	var magic uint32
	if err := binary.Read(r.r, binary.LittleEndian, &magic); err != nil {
		return err
	}

	// Verify magic number
	if magic != frameMagic {
		return errors.New("invalid LZ4 frame magic number")
	}

	// Read FLG byte
	flg := make([]byte, 1)
	if _, err := io.ReadFull(r.r, flg); err != nil {
		return err
	}

	// Parse flags
	r.header.blockIndependence = (flg[0] & flagBlockIndependence) != 0
	r.header.blockChecksum = (flg[0] & flagBlockChecksum) != 0
	r.header.contentSize = (flg[0] & flagContentSize) != 0
	r.header.contentChecksum = (flg[0] & flagContentChecksum) != 0
	r.header.dictID = (flg[0] & flagDictID) != 0

	// Check version - only v1.x supported
	version := (flg[0] >> 6) & 0x3
	if version != 1 {
		return errors.New("unsupported version")
	}

	// Read BD byte
	bd := make([]byte, 1)
	if _, err := io.ReadFull(r.r, bd); err != nil {
		return err
	}

	// Parse block size code
	r.header.blockSizeCode = (bd[0] >> 4) & 0x7

	// Validate block size code (must be 4-7)
	if r.header.blockSizeCode < 4 || r.header.blockSizeCode > 7 {
		return errors.New("invalid block size code")
	}

	// Read HC byte (header checksum) - we don't validate it in v0.1
	hc := make([]byte, 1)
	if _, err := io.ReadFull(r.r, hc); err != nil {
		return err
	}

	// Read optional fields

	// Content size (8 bytes)
	if r.header.contentSize {
		if err := binary.Read(r.r, binary.LittleEndian, &r.header.contentSizeValue); err != nil {
			return err
		}
	}

	// Dictionary ID (4 bytes)
	if r.header.dictID {
		if err := binary.Read(r.r, binary.LittleEndian, &r.header.dictIDValue); err != nil {
			return err
		}
	}

	return nil
}

// readBlock reads and decompresses the next LZ4 block
func (r *Reader) readBlock() error {
	// Read block size (4 bytes)
	var blockSize uint32
	if err := binary.Read(r.r, binary.LittleEndian, &blockSize); err != nil {
		return err
	}

	// Check for end marker
	if blockSize == 0 {
		return io.EOF
	}

	// Check if block is compressed
	isCompressed := true
	if (blockSize & 0x80000000) != 0 {
		// High bit set indicates uncompressed data
		isCompressed = false
		blockSize &= 0x7FFFFFFF // Clear the high bit
	}

	// Handle empty uncompressed block (which might be generated for small data)
	if blockSize == 0 && !isCompressed {
		r.decompressed = []byte{}
		return nil
	}

	// Validate block size
	if blockSize > uint32(r.blocksizeCache) {
		return errors.New("block size too large")
	}

	// Read block data
	blockData := make([]byte, blockSize)
	if _, err := io.ReadFull(r.r, blockData); err != nil {
		return err
	}

	// Skip block checksum if present
	if r.header.blockChecksum {
		checksum := make([]byte, 4)
		if _, err := io.ReadFull(r.r, checksum); err != nil {
			return err
		}
	}

	// If block is uncompressed, just use it
	if !isCompressed {
		r.decompressed = blockData
		return nil
	}

	// Decompress block
	decompressed, err := DecompressBlock(blockData, nil, r.blocksizeCache)
	if err != nil {
		return err
	}

	r.decompressed = decompressed
	return nil
}

// NewWriter creates a new LZ4 writer with default compression level
func NewWriter(w io.Writer) *Writer {
	return NewWriterLevel(w, DefaultLevel)
}

// NewWriterLevel creates a new LZ4 writer with specified compression level
func NewWriterLevel(w io.Writer, level CompressionLevel) *Writer {
	// Ensure we have a valid compression level
	if level < 1 || level > MaxLevel {
		level = DefaultLevel
	}

	// Determine block size based on blockSizeCode 7 (4MB default)
	blockSize := 4 * 1024 * 1024

	z := &Writer{
		w:     w,
		level: level,
		header: frameHeader{
			blockIndependence: true,
			blockSizeCode:     7, // 4MB blocks by default
		},
		blockSize: blockSize,
		// Allocate a buffer large enough for the block plus header/footer overhead
		buf: make([]byte, maxBlockSize+maxHeaderSize+16),
	}

	return z
}

// Reset resets the Writer to write to w
func (z *Writer) Reset(w io.Writer) {
	z.w = w
	z.bufUsed = 0
	z.closed = false
	z.wroteHeader = false
	z.written = 0

	// Re-initialize the block size based on the header block size code
	switch z.header.blockSizeCode {
	case 4:
		z.blockSize = 64 * 1024
	case 5:
		z.blockSize = 256 * 1024
	case 6:
		z.blockSize = 1 * 1024 * 1024
	case 7:
		z.blockSize = 4 * 1024 * 1024
	default:
		// Default to max block size
		z.blockSize = 4 * 1024 * 1024
		z.header.blockSizeCode = 7
	}
}

// Write implements io.Writer
func (z *Writer) Write(p []byte) (int, error) {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.closed {
		return 0, errors.New("write to closed stream")
	}

	// Write the frame header if this is the first write
	if !z.wroteHeader {
		err := z.writeFrameHeader()
		if err != nil {
			return 0, err
		}
		z.wroteHeader = true
	}

	var written int
	for len(p) > 0 {
		// Check if we need to flush the current block
		remaining := z.blockSize - z.bufUsed
		if remaining == 0 {
			// Flush current block
			err := z.flush()
			if err != nil {
				return written, err
			}
			remaining = z.blockSize
		}

		// Copy data to buffer
		n := copy(z.buf[z.bufUsed:z.bufUsed+remaining], p)
		z.bufUsed += n
		p = p[n:]
		written += n
	}

	return written, nil
}

// writeFrameHeader writes the LZ4 frame header to the output
func (z *Writer) writeFrameHeader() error {
	// Ensure we have enough space for the frame header
	headerSize := 7 // Magic number (4), FLG (1), BD (1), HC (1)
	if z.header.contentSize {
		headerSize += 8
	}
	if z.header.dictID {
		headerSize += 4
	}

	// Write magic number
	binary.LittleEndian.PutUint32(z.buf[0:4], frameMagic)

	// Write FLG byte
	flg := byte(0)
	if z.header.blockIndependence {
		flg |= flagBlockIndependence
	}
	if z.header.blockChecksum {
		flg |= flagBlockChecksum
	}
	if z.header.contentSize {
		flg |= flagContentSize
	}
	if z.header.contentChecksum {
		flg |= flagContentChecksum
	}
	if z.header.dictID {
		flg |= flagDictID
	}
	// Version is always 01 for now
	flg |= (1 << 6)

	z.buf[4] = flg

	// Write BD byte (block descriptor)
	// Block size flag (4-7) in bits 4-6
	bd := byte(0)
	if z.header.blockSizeCode >= 4 && z.header.blockSizeCode <= 7 {
		bd |= (z.header.blockSizeCode & 0x7) << 4
	} else {
		// Default to maximum block size (7 = 4MB)
		bd |= (7 << 4)
	}

	z.buf[5] = bd

	// Write HC byte (header checksum)
	// For v0.1, we just use a fixed value
	hc := byte(0)
	z.buf[6] = hc

	// Write optional fields
	offset := 7

	// Content size (8 bytes)
	if z.header.contentSize {
		binary.LittleEndian.PutUint64(z.buf[offset:offset+8], z.header.contentSizeValue)
		offset += 8
	}

	// Dictionary ID (4 bytes)
	if z.header.dictID {
		binary.LittleEndian.PutUint32(z.buf[offset:offset+4], z.header.dictIDValue)
		offset += 4
	}

	// Write header to output
	_, err := z.w.Write(z.buf[0:offset])

	return err
}

// flush compresses and writes a block
func (z *Writer) flush() error {
	if z.bufUsed == 0 {
		// Nothing to flush - for empty flushes, we'll just write out
		// an uncompressed block with length 0 to satisfy the LZ4 format
		// Write a block size of 0x80000000 (high bit set, zero size)
		sizeBuffer := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuffer, 0x80000000)
		_, err := z.w.Write(sizeBuffer)
		return err
	}

	// Ensure we don't exceed maximum block size
	if z.bufUsed > maxBlockSize {
		return errors.New("block size too large")
	}

	// For very small data, don't try to compress
	if z.bufUsed < 16 { // Minimum viable size for LZ4 compression
		// Just write uncompressed block
		blockSize := uint32(z.bufUsed) | 0x80000000 // Set high bit to indicate uncompressed
		sizeBuffer := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuffer, blockSize)
		_, err := z.w.Write(sizeBuffer)
		if err != nil {
			return err
		}

		// Write the raw data
		_, err = z.w.Write(z.buf[:z.bufUsed])
		z.written += uint64(z.bufUsed)
		z.bufUsed = 0
		return err
	}

	// Create a slice to hold the compressed data
	// For V0.1, we'll use a simple literal block approach for all data
	// Worst case: 4 bytes block header + LZ4 compression overhead + data
	maxCompSize := len(z.buf) + (len(z.buf) / 255) + 16
	compBuf := make([]byte, maxCompSize)

	// Convert input to a Block for simplified LZ4 compression
	inputSlice := z.buf[:z.bufUsed]
	block, err := NewBlock(inputSlice, z.level)
	if err != nil {
		// On error, just store uncompressed
		blockSize := uint32(z.bufUsed) | 0x80000000 // Set high bit to indicate uncompressed
		sizeBuffer := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuffer, blockSize)
		_, err := z.w.Write(sizeBuffer)
		if err != nil {
			return err
		}

		// Write the raw data
		_, err = z.w.Write(z.buf[:z.bufUsed])
		z.written += uint64(z.bufUsed)
		z.bufUsed = 0
		return err
	}

	// Compress the data
	compData, err := block.CompressToBuffer(compBuf[4:]) // Leave space for block size
	if err != nil || len(compData) >= z.bufUsed {
		// Compression failed or didn't save space, use uncompressed
		blockSize := uint32(z.bufUsed) | 0x80000000 // Set high bit to indicate uncompressed
		sizeBuffer := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuffer, blockSize)
		_, err := z.w.Write(sizeBuffer)
		if err != nil {
			return err
		}

		// Write the raw data
		_, err = z.w.Write(z.buf[:z.bufUsed])
		z.written += uint64(z.bufUsed)
		z.bufUsed = 0
		return err
	}

	// Compression succeeded and saved space
	// Write compressed block size
	blockSize := uint32(len(compData))
	binary.LittleEndian.PutUint32(compBuf[:4], blockSize)

	// Add block checksum if enabled
	if z.header.blockChecksum {
		// TODO: Calculate and append checksum
		// For v0.1, this is not implemented
	}

	// Write the block size and compressed data
	_, err = z.w.Write(compBuf[:4])
	if err != nil {
		return err
	}
	_, err = z.w.Write(compData)
	if err != nil {
		return err
	}

	// Update state
	z.written += uint64(z.bufUsed)
	z.bufUsed = 0

	return nil
}

// Close implements io.Closer
func (z *Writer) Close() error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.closed {
		return nil
	}

	var err error

	// Make sure we've written the header
	if !z.wroteHeader {
		err = z.writeFrameHeader()
		if err != nil {
			return err
		}
		z.wroteHeader = true
	}

	// Flush any remaining data
	if z.bufUsed > 0 {
		err = z.flush()
		if err != nil {
			return err
		}
	} else {
		// If there's no data at all, write an empty block
		// This is necessary for valid LZ4 frames to have at least one block
		sizeBuffer := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuffer, 0x80000000) // High bit set, zero size
		_, err = z.w.Write(sizeBuffer)
		if err != nil {
			return err
		}
	}

	// Write end marker (block size = 0)
	endMarker := make([]byte, 4)
	_, err = z.w.Write(endMarker) // All zeros for end marker
	if err != nil {
		return err
	}

	// Write content checksum if enabled
	if z.header.contentChecksum {
		// Placeholder for content checksum calculation
		// In a real implementation, we would have tracked a running checksum
		// For v0.1, we'll just write zeros
		zeroChecksum := make([]byte, 4)
		_, err = z.w.Write(zeroChecksum)
		if err != nil {
			return err
		}
	}

	z.closed = true
	return nil
}

// NewWriterWithOptions creates a new Writer with custom options
func NewWriterWithOptions(w io.Writer, options WriterOptions) *Writer {
	writer := &Writer{
		w:           w,
		level:       options.Level,
		useV2:       options.UseV2,
		blockSize:   maxBlockSize,
		closed:      false,
		buf:         make([]byte, 0),
		wroteHeader: false,
	}

	// Use specified block size if provided
	if options.BlockSize > 0 {
		writer.blockSize = options.BlockSize
	}

	// Allocate buffer
	writer.buf = make([]byte, writer.blockSize)
	writer.bufUsed = 0

	return writer
}

// write compresses and writes a block of data
func (w *Writer) write(block []byte) error {
	var compressed []byte
	var err error

	if w.useV2 {
		// Use v0.2 algorithm if enabled
		compressed, err = CompressBlockV2Level(block, nil, w.level)
	} else {
		// Use original algorithm
		compressed, err = CompressBlockLevel(block, nil, w.level)
	}

	if err != nil {
		return err
	}

	// Check if compression actually helped
	if len(compressed) >= len(block) {
		// Write uncompressed block with appropriate flag
		// Block size (4 bytes)
		binary.LittleEndian.PutUint32(w.buf[:4], uint32(len(block)|0x80000000))
		if _, err := w.w.Write(w.buf[:4]); err != nil {
			return err
		}

		// Write original data
		if _, err := w.w.Write(block); err != nil {
			return err
		}
	} else {
		// Write compressed block
		// Block size (4 bytes)
		binary.LittleEndian.PutUint32(w.buf[:4], uint32(len(compressed)))
		if _, err := w.w.Write(w.buf[:4]); err != nil {
			return err
		}

		// Write compressed data
		if _, err := w.w.Write(compressed); err != nil {
			return err
		}
	}

	return nil
}
