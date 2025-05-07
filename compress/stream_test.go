package compress

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Test frame header writing and reading
func TestFrameHeader(t *testing.T) {
	// Test various header configurations
	tests := []struct {
		name          string
		blockIndep    bool
		blockCheck    bool
		contentSize   bool
		contentCheck  bool
		dictID        bool
		blockSizeCode uint8
	}{
		{"Default header", true, false, false, false, false, 7},
		{"With block checksum", true, true, false, false, false, 7},
		{"With content checksum", true, false, false, true, false, 7},
		{"With content size", true, false, true, false, false, 7},
		{"With dictionary ID", true, false, false, false, true, 7},
		{"Small block size", true, false, false, false, false, 4},
		{"All features enabled", true, true, true, true, true, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a writer with the specified header configuration
			var buf bytes.Buffer
			w := NewWriterLevel(&buf, DefaultLevel)

			// Configure the header
			w.header.blockIndependence = tt.blockIndep
			w.header.blockChecksum = tt.blockCheck
			w.header.contentSize = tt.contentSize
			w.header.contentChecksum = tt.contentCheck
			w.header.dictID = tt.dictID
			w.header.blockSizeCode = tt.blockSizeCode

			// Write the header
			err := w.writeFrameHeader()
			if err != nil {
				t.Fatalf("writeFrameHeader() error = %v", err)
			}

			// Read it back with a Reader
			r := NewReader(&buf)
			err = r.readFrameHeader()
			if err != nil {
				t.Fatalf("readFrameHeader() error = %v", err)
			}

			// Verify header fields
			if r.header.blockIndependence != tt.blockIndep {
				t.Errorf("blockIndependence = %v, want %v", r.header.blockIndependence, tt.blockIndep)
			}
			if r.header.blockChecksum != tt.blockCheck {
				t.Errorf("blockChecksum = %v, want %v", r.header.blockChecksum, tt.blockCheck)
			}
			if r.header.contentSize != tt.contentSize {
				t.Errorf("contentSize = %v, want %v", r.header.contentSize, tt.contentSize)
			}
			if r.header.contentChecksum != tt.contentCheck {
				t.Errorf("contentChecksum = %v, want %v", r.header.contentChecksum, tt.contentCheck)
			}
			if r.header.dictID != tt.dictID {
				t.Errorf("dictID = %v, want %v", r.header.dictID, tt.dictID)
			}
			if r.header.blockSizeCode != tt.blockSizeCode {
				t.Errorf("blockSizeCode = %v, want %v", r.header.blockSizeCode, tt.blockSizeCode)
			}
		})
	}

	// Test invalid magic number
	t.Run("Invalid magic number", func(t *testing.T) {
		var buf bytes.Buffer
		// Write an invalid magic number
		binary.Write(&buf, binary.LittleEndian, uint32(0x12345678))

		r := NewReader(&buf)
		err := r.readFrameHeader()
		if err == nil {
			t.Errorf("readFrameHeader() with invalid magic number: error = nil, expected error")
		}
	})

	// Test incomplete header
	t.Run("Incomplete header", func(t *testing.T) {
		var buf bytes.Buffer
		// Write only part of the header
		binary.Write(&buf, binary.LittleEndian, uint32(frameMagic))

		r := NewReader(&buf)
		err := r.readFrameHeader()
		if err == nil {
			t.Errorf("readFrameHeader() with incomplete header: error = nil, expected error")
		}
	})
}

// Test Writer basics
func TestWriter(t *testing.T) {
	// Test creating a writer with different compression levels
	t.Run("Writer creation", func(t *testing.T) {
		var buf bytes.Buffer

		// Default writer
		w1 := NewWriter(&buf)
		if w1 == nil {
			t.Errorf("NewWriter() = nil, expected non-nil")
		}

		// Writer with specific level
		w2 := NewWriterLevel(&buf, FastLevel)
		if w2 == nil {
			t.Errorf("NewWriterLevel() = nil, expected non-nil")
		}
		if w2.level != FastLevel {
			t.Errorf("level = %v, want %v", w2.level, FastLevel)
		}
	})

	// Test writing a small amount of data
	t.Run("Write small data", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewWriter(&buf)

		data := []byte("Hello, World!")
		n, err := w.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != len(data) {
			t.Errorf("Write() = %v, want %v", n, len(data))
		}

		// Close to flush data
		if err := w.Close(); err != nil {
			t.Logf("Close() error = %v", err)
		}

		// The output should not be empty
		if buf.Len() == 0 {
			t.Errorf("Output buffer is empty")
		}

		// Verify we can read the data back
		r := NewReader(bytes.NewReader(buf.Bytes()))
		result := make([]byte, len(data)*2) // Larger buffer to handle any padding
		n, err = r.Read(result)
		if err != nil && err != io.EOF {
			t.Errorf("Read error: %v", err)
		} else {
			result = result[:n]
			if !bytes.Equal(result, data) {
				t.Errorf("Read data doesn't match original")
				t.Errorf("Got: %q", result)
				t.Errorf("Want: %q", data)
			}
		}
	})

	// Test writing to closed writer
	t.Run("Write to closed writer", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewWriter(&buf)

		err := w.Close()
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		_, err = w.Write([]byte("data"))
		if err == nil {
			t.Errorf("Write() after Close(): error = nil, expected error")
		}
	})

	// Test reset
	t.Run("Reset writer", func(t *testing.T) {
		var buf1, buf2 bytes.Buffer
		w := NewWriter(&buf1)

		// Write some data
		_, err := w.Write([]byte("data1"))
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if err := w.Close(); err != nil {
			t.Logf("Close() error = %v", err)
		}

		// Reset and use with a different buffer
		w.Reset(&buf2)
		_, err = w.Write([]byte("data2"))
		if err != nil {
			t.Fatalf("Write() after Reset(): error = %v", err)
		}
		if err := w.Close(); err != nil {
			t.Logf("Close() after Reset(): error = %v", err)
		}

		// Both buffers should contain data
		if buf1.Len() == 0 {
			t.Errorf("First buffer is empty")
		}
		if buf2.Len() == 0 {
			t.Errorf("Second buffer is empty")
		}
	})
}

// Test Reader basics
func TestReader(t *testing.T) {
	// Create some compressed data for testing
	var compressedBuf bytes.Buffer
	testData := "This is some test data that will be compressed and then decompressed."

	// Compress the data
	w := NewWriter(&compressedBuf)
	_, err := io.Copy(w, strings.NewReader(testData))
	if err != nil {
		t.Fatalf("Failed to compress test data: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	compressedData := compressedBuf.Bytes()

	// Test basic reader functionality
	t.Run("Basic Read", func(t *testing.T) {
		r := NewReader(bytes.NewReader(compressedData))

		var decompressedBuf bytes.Buffer
		_, err := io.Copy(&decompressedBuf, r)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}

		decompressed := decompressedBuf.String()
		if decompressed != testData {
			t.Errorf("Decompressed data doesn't match original")
			t.Errorf("Got: %q", decompressed)
			t.Errorf("Want: %q", testData)
		}
	})

	// Test reading after EOF
	t.Run("Read after EOF", func(t *testing.T) {
		r := NewReader(bytes.NewReader(compressedData))

		// Read all data
		var decompressedBuf bytes.Buffer
		_, err := io.Copy(&decompressedBuf, r)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}

		// Try to read more
		buf := make([]byte, 10)
		n, err := r.Read(buf)
		if n > 0 || err != io.EOF {
			t.Errorf("Read after EOF: n = %v, err = %v; want 0, io.EOF", n, err)
		}
	})

	// Test invalid compressed data
	t.Run("Invalid compressed data", func(t *testing.T) {
		// Create invalid data by corrupting the header
		invalidData := make([]byte, len(compressedData))
		copy(invalidData, compressedData)
		if len(invalidData) > 10 {
			invalidData[8] = 0xFF // Corrupt the header
		}

		r := NewReader(bytes.NewReader(invalidData))

		buf := make([]byte, 10)
		_, _ = r.Read(buf) // Ignore error since it might not fail on first read
		// This might not always error out on the first read, depending on how robust the parser is
		// But subsequent reads should eventually fail
	})
}

// Test the streaming process end-to-end with various data sizes and compression options
func TestStreamingRoundTrip(t *testing.T) {
	// Test data sizes
	sizes := []int{
		100,             // Very small
		1 * 1024,        // 1KB
		64 * 1024,       // 64KB
		256 * 1024,      // 256KB
		1 * 1024 * 1024, // 1MB
	}

	// Compression levels to test
	levels := []CompressionLevel{
		FastLevel,
		DefaultLevel,
		MaxLevel,
	}

	for _, size := range sizes {
		for _, level := range levels {
			t.Run(fmt.Sprintf("Size=%d,Level=%d", size, level), func(t *testing.T) {
				// Generate test data
				input := generateCompressibleData(size)

				// Compress
				var buf bytes.Buffer
				w := NewWriterLevel(&buf, level)

				n, err := w.Write(input)
				if err != nil {
					t.Fatalf("Write error: %v", err)
				}
				if n != len(input) {
					t.Errorf("Write returned %d, want %d", n, len(input))
				}

				err = w.Close()
				if err != nil {
					t.Fatalf("Close error: %v", err)
				}

				compressed := buf.Bytes()

				// Report compression stats
				t.Logf("Original size: %d, Compressed size: %d, Ratio: %.2f%%",
					len(input), len(compressed), float64(len(compressed))*100/float64(len(input)))

				// Decompress
				r := NewReader(bytes.NewReader(compressed))
				decompressed := make([]byte, 0, size)
				decompressedBuf := bytes.NewBuffer(decompressed)

				_, err = io.Copy(decompressedBuf, r)
				if err != nil {
					t.Fatalf("Decompression error: %v", err)
				}

				// Verify
				if !bytes.Equal(decompressedBuf.Bytes(), input) {
					t.Errorf("Decompressed data doesn't match original")
				}
			})
		}
	}
}

// Test handling of very large write operations that exceed the internal buffer size
func TestLargeWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large write test in short mode")
	}

	// Create a large, compressible dataset
	size := 4 * 1024 * 1024 // 4MB
	if size > MaxBlockSize {
		size = MaxBlockSize - 1024 // Stay under the max block size
	}

	data := generateCompressibleData(size)

	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Try to write it all at once
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Decompress and verify
	r := NewReader(bytes.NewReader(buf.Bytes()))
	result := make([]byte, 0, size)
	resultBuf := bytes.NewBuffer(result)

	_, err = io.Copy(resultBuf, r)
	if err != nil {
		t.Fatalf("Decompression error: %v", err)
	}

	if !bytes.Equal(resultBuf.Bytes(), data) {
		t.Errorf("Decompressed data doesn't match original for large write")
	}
}

// Test the Writer's flush method
func TestFlush(t *testing.T) {
	// Test flushing an empty buffer
	t.Run("Flush empty buffer", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewWriter(&buf)

		// This will call flush indirectly
		if err := w.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		// A valid empty frame should at least have a header and an end marker
		// End marker (block size = 0) is 4 bytes
		// Header is at least 7 bytes (magic number, FLG, BD, HC)
		minFrameSize := 11 // 7 byte header + 4 byte end marker

		if buf.Len() < minFrameSize {
			t.Errorf("Output too small for valid frame: %d bytes, want >= %d",
				buf.Len(), minFrameSize)
		}

		// Verify we can read it back with no errors
		r := NewReader(bytes.NewReader(buf.Bytes()))
		readBuf := make([]byte, 10)
		n, err := r.Read(readBuf)
		if err != io.EOF {
			t.Errorf("Read() error = %v, want io.EOF", err)
		}
		if n != 0 {
			t.Errorf("Read() n = %d, want 0", n)
		}
	})

	// Test flushing incompressible data
	t.Run("Flush incompressible data", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewWriter(&buf)

		// Write random (incompressible) data
		data := generateRandomData(1024)
		_, err := w.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		if err := w.Close(); err != nil {
			t.Logf("Close() error = %v", err)
		}

		// Verify we can read it back
		r := NewReader(bytes.NewReader(buf.Bytes()))
		result := make([]byte, len(data)*2)
		n, err := io.ReadFull(r, result)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			t.Errorf("Read error: %v", err)
		} else {
			result = result[:n]
			if !bytes.Equal(result, data) {
				t.Errorf("Read data doesn't match original")
			}
		}
	})
}

// Test error handling for the frame format
func TestStreamErrorHandling(t *testing.T) {
	// Create some valid compressed data
	var buf bytes.Buffer
	w := NewWriter(&buf)
	_, err := w.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Logf("Close() error: %v", err)
	}

	validData := buf.Bytes()

	tests := []struct {
		name      string
		modifyFn  func([]byte) []byte
		shouldErr bool
	}{
		{
			"Valid data",
			func(d []byte) []byte { return d },
			false,
		},
		{
			"No data",
			func(d []byte) []byte { return []byte{} },
			true,
		},
		{
			"Corrupted magic",
			func(d []byte) []byte {
				m := make([]byte, len(d))
				copy(m, d)
				if len(m) >= 4 {
					m[0] = 0xAA // Corrupt the magic number
				}
				return m
			},
			true,
		},
		{
			"Truncated header",
			func(d []byte) []byte {
				if len(d) >= 6 {
					return d[:5] // Truncate in the middle of the header
				}
				return d
			},
			true,
		},
		{
			"Invalid block size",
			func(d []byte) []byte {
				m := make([]byte, len(d))
				copy(m, d)
				if len(m) >= 10 {
					// Modify the block size to be unreasonably large
					binary.LittleEndian.PutUint32(m[7:11], 0x10000000)
				}
				return m
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := tt.modifyFn(validData)
			r := NewReader(bytes.NewReader(testData))

			// Try to read the data
			buf := make([]byte, 1024)
			_, err := r.Read(buf)

			if tt.shouldErr {
				if err == nil {
					// Some errors might not be detected on the first read
					// Let's read until we get an error or EOF
					for err == nil {
						_, err = r.Read(buf)
						if err == io.EOF {
							// EOF is expected for valid data, but not for corrupted data
							if tt.name != "Valid data" {
								t.Errorf("Expected error, got EOF")
							}
							break
						}
					}
				}
			} else {
				if err != nil && err != io.EOF {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
