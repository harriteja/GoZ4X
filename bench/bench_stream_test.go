package bench

import (
	"bytes"
	"io"
	"testing"

	"github.com/harriteja/GoZ4X/compress"
)

// Benchmark streaming compression
func BenchmarkStreamCompress(b *testing.B) {
	// For each test data size (skip huge size)
	for _, size := range []int{smallSize, mediumSize, largeSize} {
		// For each compressibility level
		for _, comp := range []float64{0.0, 0.5, 0.9} {
			data := generateData(size, comp)
			name := ""
			switch {
			case size == smallSize:
				name = "Small"
			case size == mediumSize:
				name = "Medium"
			case size == largeSize:
				name = "Large"
			}

			compStr := ""
			switch {
			case comp == 0.0:
				compStr = "Random"
			case comp == 0.5:
				compStr = "Mixed"
			case comp == 0.9:
				compStr = "Compressible"
			}

			// For each compression level
			for _, level := range benchLevels {
				b.Run(name+"_"+compStr+"_Level"+string(rune('0'+int(level))), func(b *testing.B) {
					// Reset benchmark timer
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						// Reset timer during buffer allocation
						b.StopTimer()
						var buf bytes.Buffer
						b.StartTimer()

						// Create writer with specified level
						w := compress.NewWriterLevel(&buf, level)

						// Write data
						_, err := w.Write(data)
						if err != nil {
							b.Fatal(err)
						}

						// Close to flush any remaining data
						err = w.Close()
						if err != nil {
							b.Fatal(err)
						}

						// Report compression ratio
						b.ReportMetric(float64(buf.Len())/float64(len(data)), "ratio")
					}

					b.SetBytes(int64(len(data)))
				})
			}
		}
	}
}

// Benchmark streaming decompression
func BenchmarkStreamDecompress(b *testing.B) {
	// For each test data size (skip huge size)
	for _, size := range []int{smallSize, mediumSize, largeSize} {
		// For each compressibility level
		for _, comp := range []float64{0.0, 0.5, 0.9} {
			data := generateData(size, comp)

			// Compress data first
			var buf bytes.Buffer
			w := compress.NewWriter(&buf)
			_, err := w.Write(data)
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}

			// Save compressed data
			compressed := buf.Bytes()

			name := ""
			switch {
			case size == smallSize:
				name = "Small"
			case size == mediumSize:
				name = "Medium"
			case size == largeSize:
				name = "Large"
			}

			compStr := ""
			switch {
			case comp == 0.0:
				compStr = "Random"
			case comp == 0.5:
				compStr = "Mixed"
			case comp == 0.9:
				compStr = "Compressible"
			}

			b.Run(name+"_"+compStr, func(b *testing.B) {
				// Reset benchmark timer
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// Reset timer during buffer allocation
					b.StopTimer()
					reader := bytes.NewReader(compressed)
					b.StartTimer()

					// Create reader
					r := compress.NewReader(reader)

					// Read and discard all data
					result := new(bytes.Buffer)
					_, err := io.Copy(result, r)
					if err != nil {
						b.Fatal(err)
					}

					// Verify decompression (only on first iteration to avoid overhead)
					if i == 0 && !bytes.Equal(result.Bytes(), data) {
						b.Fatal("decompression failed")
					}
				}

				b.SetBytes(int64(len(data)))
			})
		}
	}
}
