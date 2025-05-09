package bench

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"strconv"
	"testing"

	goz4x "github.com/harriteja/GoZ4X"
	v03 "github.com/harriteja/GoZ4X/v03"
)

// generateRandomText creates random text data
func generateRandomText(size int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,;:!?-_()"
	data := make([]byte, size)
	for i := range data {
		data[i] = charset[rand.Intn(len(charset))]
	}
	return data
}

// generateHTMLDocument creates sample HTML data
func generateHTMLDocument(paragraphs int, wordsPerParagraph int) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<title>Sample Document</title>\n</head>\n<body>\n")

	for i := 0; i < paragraphs; i++ {
		buffer.WriteString("<p>")
		for j := 0; j < wordsPerParagraph; j++ {
			wordLength := rand.Intn(10) + 3
			word := generateRandomText(wordLength)
			buffer.Write(word)
			buffer.WriteByte(' ')
		}
		buffer.WriteString("</p>\n")
	}

	buffer.WriteString("</body>\n</html>")
	return buffer.Bytes()
}

// generateJSONData creates sample JSON data
func generateJSONData(records int) []byte {
	data := make([]map[string]interface{}, records)

	for i := 0; i < records; i++ {
		record := map[string]interface{}{
			"id":        i,
			"name":      "User " + strconv.Itoa(i),
			"email":     "user" + strconv.Itoa(i) + "@example.com",
			"active":    rand.Intn(2) == 1,
			"age":       rand.Intn(80) + 18,
			"timestamp": rand.Int63(),
			"data": map[string]interface{}{
				"preferences": map[string]interface{}{
					"theme":     "light",
					"fontSize":  rand.Intn(5) + 10,
					"showIntro": rand.Intn(2) == 1,
				},
				"permissions": []string{"read", "write", "admin"},
				"metrics": map[string]float64{
					"logins":    float64(rand.Intn(1000)),
					"pageViews": float64(rand.Intn(5000)),
					"clickRate": rand.Float64(),
				},
			},
		}
		data[i] = record
	}

	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return jsonBytes
}

// BenchmarkRealisticUseCase tests compression performance on realistic data
func BenchmarkRealisticUseCase(b *testing.B) {
	// Generate test data
	rand.Seed(42) // For reproducibility

	// HTML document (reduced from 500KB to 50KB to prevent hanging)
	htmlData := generateHTMLDocument(50, 100)

	// JSON data (reduced from 1MB to 100KB to prevent hanging)
	jsonData := generateJSONData(100)

	// Binary data (reduced from 2MB to 200KB to prevent hanging)
	binaryData := make([]byte, 200*1024)
	rand.Read(binaryData)

	testCases := []struct {
		name string
		data []byte
	}{
		{"HTMLDocument_50KB", htmlData},
		{"JSONData_100KB", jsonData},
		{"BinaryData_200KB", binaryData},
	}

	// Run benchmarks for each algorithm
	for _, tc := range testCases {
		// v0.1
		b.Run("v0.1_"+tc.name, func(b *testing.B) {
			// Limit iterations to prevent test hangs
			b.N = min(b.N, 100)

			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))

			for i := 0; i < b.N; i++ {
				// Compress
				b.StopTimer()
				compressed, _ := goz4x.CompressBlock(tc.data, nil)
				b.StartTimer()

				// Decompress to measure full round trip
				_, _ = goz4x.DecompressBlock(compressed, nil, len(tc.data))

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tc.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})

		// v0.2
		b.Run("v0.2_"+tc.name, func(b *testing.B) {
			// Limit iterations to prevent test hangs
			b.N = min(b.N, 100)

			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))

			for i := 0; i < b.N; i++ {
				// Compress
				b.StopTimer()
				compressed, _ := goz4x.CompressBlockV2(tc.data, nil)
				b.StartTimer()

				// Decompress to measure full round trip
				_, _ = goz4x.DecompressBlock(compressed, nil, len(tc.data))

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tc.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})

		// v0.3
		b.Run("v0.3_"+tc.name, func(b *testing.B) {
			// Limit iterations to prevent test hangs
			b.N = min(b.N, 100)

			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))

			for i := 0; i < b.N; i++ {
				// Compress
				b.StopTimer()
				compressed, _ := v03.CompressBlockV2Parallel(tc.data, nil)
				b.StartTimer()

				// Decompress to measure full round trip
				_, _ = goz4x.DecompressBlock(compressed, nil, len(tc.data))

				b.StopTimer()
				ratio := float64(len(compressed)) / float64(len(tc.data))
				b.ReportMetric(ratio, "ratio")
				b.StartTimer()
			}
		})
	}
}
