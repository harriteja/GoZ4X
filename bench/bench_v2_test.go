package bench

import (
	"bytes"
	"testing"

	goz4x "github.com/harriteja/GoZ4X"
)

// BenchmarkV1vsV2 compares the performance of v0.1 and v0.2 compression
func BenchmarkV1vsV2(b *testing.B) {
	// Sample text data for compression
	textData := bytes.Repeat([]byte("GoZ4X is a pure-Go implementation of the LZ4 compression algorithm. "+
		"It's designed for speed and compatibility with the original LZ4 format."), 50)

	// Sample repetitive data
	repetitiveData := bytes.Repeat([]byte("ABCDEFGHIJ"), 10000)

	tests := []struct {
		name string
		data []byte
	}{
		{"Text", textData},
		{"Repetitive", repetitiveData},
	}

	for _, tt := range tests {
		// Test v0.1 block compression
		b.Run("v0.1_"+tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(len(tt.data)))

			for i := 0; i < b.N; i++ {
				compressed, _ := goz4x.CompressBlock(tt.data, nil)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tt.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})

		// Test v0.2 block compression
		b.Run("v0.2_"+tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(len(tt.data)))

			for i := 0; i < b.N; i++ {
				compressed, _ := goz4x.CompressBlockV2(tt.data, nil)

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tt.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})
	}
}
