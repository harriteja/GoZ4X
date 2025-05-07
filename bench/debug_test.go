package bench

import (
	"bytes"
	"io"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
)

// TestStreamRoundTrip tests if data can successfully make a round trip through
// the streaming compression and decompression API
func TestStreamRoundTrip(t *testing.T) {
	sizes := []struct {
		name string
		size int
	}{
		{"small", smallSize},
		{"medium", mediumSize},
	}

	compressibilities := []struct {
		name string
		comp float64
	}{
		{"random", 0.0},
		{"mixed", 0.5},
		{"compressible", 0.9},
	}

	for _, sz := range sizes {
		t.Run(sz.name, func(t *testing.T) {
			for _, comp := range compressibilities {
				t.Run(comp.name, func(t *testing.T) {
					data := generateData(sz.size, comp.comp)

					// Compress
					var buf bytes.Buffer
					w := compress.NewWriter(&buf)

					n, err := w.Write(data)
					if err != nil {
						t.Fatalf("compression write error: %v", err)
					}
					if n != len(data) {
						t.Fatalf("short write: %d != %d", n, len(data))
					}

					err = w.Close()
					if err != nil {
						t.Fatalf("compression close error: %v", err)
					}

					compressed := buf.Bytes()
					t.Logf("Original size: %d, Compressed size: %d, Ratio: %.2f%%",
						len(data), len(compressed), float64(len(compressed))*100/float64(len(data)))

					// Decompress
					r := compress.NewReader(bytes.NewReader(compressed))
					result := new(bytes.Buffer)

					_, err = io.Copy(result, r)
					if err != nil {
						t.Fatalf("decompression error: %v", err)
					}

					// Verify
					if !bytes.Equal(result.Bytes(), data) {
						if len(result.Bytes()) != len(data) {
							t.Fatalf("length mismatch: got %d bytes, want %d bytes",
								len(result.Bytes()), len(data))
						} else {
							// Find the first different byte
							for i := 0; i < len(data); i++ {
								if result.Bytes()[i] != data[i] {
									t.Fatalf("first difference at byte %d: got 0x%02x, want 0x%02x",
										i, result.Bytes()[i], data[i])
								}
							}
						}
					}
				})
			}
		})
	}
}
