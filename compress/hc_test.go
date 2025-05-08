package compress

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

// Test creating a new HCMatcher with different compression levels
func TestNewHCMatcher(t *testing.T) {
	tests := []struct {
		name  string
		level CompressionLevel
	}{
		{"Fast level", 1},
		{"Default level", 6},
		{"High level", 9},
		{"Max level", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewHCMatcher(tt.level)

			if matcher == nil {
				t.Fatalf("NewHCMatcher(%v) = nil, expected non-nil", tt.level)
			}

			// Check that the matcher has been initialized properly
			if matcher.hashTable == nil {
				t.Errorf("hashTable is nil")
			}

			// Check hash table size based on level
			expectedSize := HashTableSize
			if tt.level > 9 {
				expectedSize = HashTableSizeHC
			}

			if len(matcher.hashTable) != expectedSize {
				t.Errorf("hashTable size = %v, want %v", len(matcher.hashTable), expectedSize)
			}

			// Check that maxAttempts is set based on level
			if matcher.maxAttempts <= 0 {
				t.Errorf("maxAttempts = %v, expected > 0", matcher.maxAttempts)
			}

			// Higher levels should have more max attempts
			if tt.level > 6 && matcher.maxAttempts <= 8 {
				t.Errorf("maxAttempts for level %v = %v, expected > 8", tt.level, matcher.maxAttempts)
			}
		})
	}
}

// Test Reset functionality
func TestHCMatcherReset(t *testing.T) {
	tests := []struct {
		name      string
		inputSize int
	}{
		{"Small input", 1024},
		{"Medium input", 64 * 1024},
		{"Large input", 1 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewHCMatcher(DefaultLevel)

			// Generate test data
			input := generateCompressibleData(tt.inputSize)

			// Reset with new input
			matcher.Reset(input)

			// Check that state has been reset properly
			if matcher.buf == nil {
				t.Errorf("buf is nil after Reset")
			}

			if len(matcher.buf) != tt.inputSize {
				t.Errorf("buf length = %v, want %v", len(matcher.buf), tt.inputSize)
			}

			if matcher.pos != 0 {
				t.Errorf("pos = %v, want 0", matcher.pos)
			}

			if matcher.end != tt.inputSize {
				t.Errorf("end = %v, want %v", matcher.end, tt.inputSize)
			}

			if matcher.chainTable == nil {
				t.Errorf("chainTable is nil after Reset")
			}

			if len(matcher.chainTable) != tt.inputSize {
				t.Errorf("chainTable length = %v, want %v", len(matcher.chainTable), tt.inputSize)
			}

			// Reset with larger input to test resizing
			largerInput := generateCompressibleData(tt.inputSize * 2)
			matcher.Reset(largerInput)

			if len(matcher.chainTable) != len(largerInput) {
				t.Errorf("chainTable length after resize = %v, want %v", len(matcher.chainTable), len(largerInput))
			}
		})
	}
}

// Test hash4 function
func TestHCMatcherHash4(t *testing.T) {
	matcher := NewHCMatcher(DefaultLevel)

	// Generate test data with a repeatable pattern
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 20)
	matcher.Reset(data)

	// Test hash at different positions
	pos1 := 0
	h1 := matcher.hash4(pos1)

	// Same pattern at a different position should hash to the same value
	pos2 := 26 // One pattern length later
	h2 := matcher.hash4(pos2)

	if h1 != h2 {
		t.Errorf("hash4(%d) = %v, hash4(%d) = %v; expected equal hashes for identical patterns",
			pos1, h1, pos2, h2)
	}

	// Different pattern should hash to a different value
	differentData := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 20)
	matcher.Reset(differentData)
	h3 := matcher.hash4(pos1)

	if h1 == h3 {
		t.Logf("hash4 collision between different patterns, this is possible but rare")
	}

	// Test hashing near the end of the buffer
	matcher.Reset(data)
	endPos := len(data) - 3 // Too close to the end for a full hash
	h4 := matcher.hash4(endPos)

	if h4 != 0 {
		t.Errorf("hash4 near end of buffer = %v, want 0", h4)
	}
}

// Test InsertHash function
func TestHCMatcherInsertHash(t *testing.T) {
	matcher := NewHCMatcher(DefaultLevel)

	// Generate test data with a fixed pattern to ensure predictable hashing
	data := bytes.Repeat([]byte("abcdefgh"), 128)
	matcher.Reset(data)

	// Insert hash at position 0
	matcher.InsertHash(0)

	// Calculate the hash value
	h0 := matcher.hash4(0)

	// Verify hash table entry
	if matcher.hashTable[h0] != 0 {
		t.Errorf("hashTable[%v] = %v, want 0", h0, matcher.hashTable[h0])
	}

	// Set up a second insert at position 8
	// Using the same pattern creates the same hash
	pos2 := 8

	// Verify this is the same pattern (same hash)
	h1 := matcher.hash4(pos2)
	if h1 != h0 {
		t.Skipf("Test requires same hash for both positions")
	}

	// Insert hash at second position
	matcher.InsertHash(pos2)

	// Now hashTable should have been updated to the new position
	if matcher.hashTable[h0] != pos2 {
		t.Errorf("hashTable[%v] = %v, want %v", h0, matcher.hashTable[h0], pos2)
	}

	// And chainTable should link to the previous position
	if matcher.chainTable[pos2] != 0 {
		t.Errorf("chainTable[%v] = %v, want 0", pos2, matcher.chainTable[pos2])
	}
}

// Test FindBestMatch functionality
func TestHCMatcherFindBestMatch(t *testing.T) {
	matcher := NewHCMatcher(DefaultLevel)

	// Case 1: No matches (random data)
	t.Run("No matches", func(t *testing.T) {
		data := generateRandomData(1024)
		matcher.Reset(data)

		// Try to find a match at the beginning (should be none)
		offset, length := matcher.FindBestMatch()

		if offset != 0 || length != 0 {
			t.Errorf("FindBestMatch() = (%v, %v), want (0, 0) for random data", offset, length)
		}
	})

	// Case 2: Repeating pattern (should find matches)
	t.Run("Repeating pattern", func(t *testing.T) {
		// Create data with a repeating pattern
		pattern := []byte("abcdefghijklmnopqrstuvwxyz")
		data := bytes.Repeat(pattern, 10)
		matcher.Reset(data)

		// First, insert some pattern positions
		for i := 0; i < 50; i += len(pattern) {
			matcher.InsertHash(i)
		}

		// Now set position to the start of a repeat and search
		matcher.pos = len(pattern) * 2 // Third repeat

		offset, length := matcher.FindBestMatch()

		if offset == 0 || length == 0 {
			t.Errorf("FindBestMatch() = (%v, %v), expected non-zero match for repeating pattern", offset, length)
		}

		if offset != len(pattern) {
			t.Errorf("offset = %v, want %v (pattern length)", offset, len(pattern))
		}

		if length < MinMatch {
			t.Errorf("length = %v, want >= %v", length, MinMatch)
		}
	})

	// Case 3: Position near end of buffer
	t.Run("Near end", func(t *testing.T) {
		data := generateCompressibleData(1024)
		matcher.Reset(data)

		// Set position very close to the end
		matcher.pos = len(data) - 3 // Less than MinMatch from end

		offset, length := matcher.FindBestMatch()

		if offset != 0 || length != 0 {
			t.Errorf("FindBestMatch() near end = (%v, %v), want (0, 0)", offset, length)
		}
	})
}

// Test Advance and End functionality
func TestHCMatcherAdvanceAndEnd(t *testing.T) {
	matcher := NewHCMatcher(DefaultLevel)

	// Generate test data
	data := generateCompressibleData(1024)
	matcher.Reset(data)

	// Initial position should be 0
	if matcher.pos != 0 {
		t.Errorf("Initial pos = %v, want 0", matcher.pos)
	}

	// After initialization, End() should be false
	if matcher.End() {
		t.Errorf("End() = true, want false initially")
	}

	// Advance by some steps
	steps := 10
	matcher.Advance(steps)

	if matcher.pos != steps {
		t.Errorf("pos after Advance(%d) = %v, want %v", steps, matcher.pos, steps)
	}

	// Advance to near the end
	matcher.Advance(len(data) - steps - MinMatch - 1)

	if matcher.End() {
		t.Errorf("End() = true, but should still be false near end")
	}

	// Advance to the end
	matcher.Advance(MinMatch + 1)

	if !matcher.End() {
		t.Errorf("End() = false, want true at end")
	}
}

// Test LazyMatch functionality
func TestHCMatcherLazyMatch(t *testing.T) {
	matcher := NewHCMatcher(DefaultLevel)

	// Create data with patterns that would benefit from lazy matching
	// We'll create data where position i+1 has a better match than position i
	data := bytes.Repeat([]byte("abcdefg"), 50)
	data = append(data, []byte("abXdefgabcdefgabcdefg")...) // Insert a small change

	matcher.Reset(data)

	// Build hash table
	for i := 0; i < len(data)-MinMatch; i++ {
		matcher.InsertHash(i)
	}

	// Set position to the spot right before our modified pattern
	matcher.pos = len(data) - 25

	// First, find a match at the current position
	offset1, length1 := matcher.FindBestMatch()

	// Now use lazy matching to see if next position is better
	_, newLength, advance := matcher.LazyMatch(offset1, length1)

	if advance <= 0 {
		t.Errorf("LazyMatch advance = %v, want > 0", advance)
	}

	// In our contrived example, we expect the new match to be better
	// But this depends on the exact data pattern and hash collisions
	t.Logf("Original match: offset=%v, length=%v", offset1, length1)
	t.Logf("Lazy match: length=%v, advance=%v", newLength, advance)

	// Test the case where the current match is very short (should try lazy)
	_, shortLength, advance := matcher.LazyMatch(10, 1)
	if advance <= 0 {
		t.Errorf("LazyMatch advance for short match = %v, want > 0", advance)
	}
	t.Logf("Lazy match for short match: length=%v, advance=%v",
		shortLength, advance)
}

// Test UpdateTables functionality
func TestHCMatcherUpdateTables(t *testing.T) {
	matcher := NewHCMatcher(DefaultLevel)

	// Generate test data
	data := generateCompressibleData(1024)
	matcher.Reset(data)

	// Initially, hash table should be zeroed
	for i := 0; i < 10; i++ {
		h := matcher.hash4(i)
		if matcher.hashTable[h] != 0 {
			t.Errorf("hashTable[%v] = %v, want 0 before UpdateTables", h, matcher.hashTable[h])
		}
	}

	// Update tables for a range
	start := 0
	end := 20
	matcher.UpdateTables(start, end)

	// Check that hash table entries have been updated
	updated := false
	for i := start; i < end; i++ {
		h := matcher.hash4(i)
		if matcher.hashTable[h] != 0 {
			updated = true
			break
		}
	}

	if !updated {
		t.Errorf("No hash table entries were updated by UpdateTables")
	}
}

// TestHCMatcherLevels tests the HC matcher with different compression levels
func TestHCMatcherLevels(t *testing.T) {
	// Test all compression levels to ensure proper behavior
	for level := CompressionLevel(1); level <= 12; level++ {
		t.Run("Level-"+string(rune('0'+level)), func(t *testing.T) {
			// Create HC matcher with this level
			matcher := NewHCMatcher(level)

			// Check configuration based on level
			switch {
			case level <= 3:
				if matcher.maxAttempts != 4 {
					t.Errorf("Expected maxAttempts to be 4 for level %d, got %d", level, matcher.maxAttempts)
				}
				if matcher.windowSize != 16*1024 {
					t.Errorf("Expected windowSize to be %d for level %d, got %d", 16*1024, level, matcher.windowSize)
				}
				if matcher.useEnhancedHC {
					t.Errorf("Expected useEnhancedHC to be false for level %d", level)
				}
			case level <= 6:
				if matcher.maxAttempts != 8 {
					t.Errorf("Expected maxAttempts to be 8 for level %d, got %d", level, matcher.maxAttempts)
				}
				if matcher.windowSize != 32*1024 {
					t.Errorf("Expected windowSize to be %d for level %d, got %d", 32*1024, level, matcher.windowSize)
				}
				if matcher.useEnhancedHC {
					t.Errorf("Expected useEnhancedHC to be false for level %d", level)
				}
			case level <= 9:
				if matcher.maxAttempts != 16 {
					t.Errorf("Expected maxAttempts to be 16 for level %d, got %d", level, matcher.maxAttempts)
				}
				if matcher.windowSize != 64*1024 {
					t.Errorf("Expected windowSize to be %d for level %d, got %d", 64*1024, level, matcher.windowSize)
				}
				if !matcher.useEnhancedHC {
					t.Errorf("Expected useEnhancedHC to be true for level %d", level)
				}
			default: // level > 9
				if matcher.maxAttempts != 32 {
					t.Errorf("Expected maxAttempts to be 32 for level %d, got %d", level, matcher.maxAttempts)
				}
				if matcher.windowSize != MaxDistance {
					t.Errorf("Expected windowSize to be %d for level %d, got %d", MaxDistance, level, matcher.windowSize)
				}
				if !matcher.useEnhancedHC {
					t.Errorf("Expected useEnhancedHC to be true for level %d", level)
				}
				if matcher.hashLog != HashLogHC {
					t.Errorf("Expected hashLog to be %d for level %d, got %d", HashLogHC, level, matcher.hashLog)
				}
			}
		})
	}
}

// TestHCHash5 tests the 5-byte hash function
func TestHCHash5(t *testing.T) {
	// Create test data with a known pattern
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Create matcher
	matcher := NewHCMatcher(12) // Use highest level to ensure 5-byte hash is used
	matcher.Reset(data)

	// Get hash values at different positions
	hashes := make(map[uint32]bool)
	for i := 0; i < len(data)-5; i++ {
		hash := matcher.hash5(i)
		hashes[hash] = true
	}

	// Verify that we get a good distribution of hash values
	// For 95 positions, we should have a decent number of unique hashes
	// but fewer than 95 due to the hash function's properties
	if len(hashes) < 20 || len(hashes) > 95 {
		t.Errorf("Expected a reasonable number of unique hashes, got %d", len(hashes))
	}

	// Test hash at end of buffer
	hash := matcher.hash5(len(data) - 4)
	if hash != 0 {
		t.Errorf("Expected hash at end of buffer to be 0, got %d", hash)
	}
}

// TestHCMatcherWindowSizes tests the window size behavior
func TestHCMatcherWindowSizes(t *testing.T) {
	// Create test data with repeated patterns
	data := createRepeatedData(1024 * 1024) // 1MB

	// Test with different levels to check window size behavior
	levels := []CompressionLevel{3, 6, 9, 12}
	for _, level := range levels {
		t.Run("Level-"+string(rune('0'+level)), func(t *testing.T) {
			testHCMatcherWithLevel(t, data, level)
		})
	}
}

// Helper function to test a specific level
func testHCMatcherWithLevel(t *testing.T, data []byte, level CompressionLevel) {
	matcher := NewHCMatcher(level)
	matcher.Reset(data)

	// Initialize hash tables
	for i := 0; i < 1024; i += 4 {
		matcher.InsertHash(i)
	}

	// Get matches at different positions
	foundMatchCount := 0
	for i := 1024; i < 1024*5; i += 8 {
		matcher.pos = i
		offset, length := matcher.FindBestMatch()
		if length >= MinMatch {
			foundMatchCount++

			// Check that offset is within window size
			if offset > matcher.windowSize {
				t.Errorf("Match offset %d exceeds window size %d at level %d",
					offset, matcher.windowSize, level)
			}
		}
	}

	// Make sure we found some matches
	if foundMatchCount == 0 {
		t.Errorf("No matches found for level %d", level)
	}
}

// TestLazyMatching tests the lazy matching improvements
func TestLazyMatching(t *testing.T) {
	// Create test data with repeating patterns and some variations
	data := createRepeatedData(256 * 1024) // 256KB

	// Test standard vs. enhanced lazy matching
	levels := []CompressionLevel{6, 12} // Level 6 (standard) vs Level 12 (enhanced)

	for _, level := range levels {
		t.Run("Level-"+string(rune('0'+level)), func(t *testing.T) {
			matcher := NewHCMatcher(level)
			matcher.Reset(data)

			// Initialize hash tables
			for i := 0; i < 1024; i += 4 {
				matcher.InsertHash(i)
			}

			// Test lazy matching
			lazyUpgrades := 0
			standardMatches := 0

			for i := 1024; i < 10000; i += 7 {
				matcher.pos = i
				offset, length := matcher.FindBestMatch()

				if length >= MinMatch {
					// Try lazy matching
					_, newLength, advance := matcher.LazyMatch(offset, length)

					if advance > 1 {
						lazyUpgrades++
						// Verify that the new match is better
						if newLength <= length {
							t.Errorf("Lazy matching didn't improve match: %d -> %d at level %d",
								length, newLength, level)
						}
					} else {
						standardMatches++
					}
				}
			}

			// Ensure we have some standard matches and some lazy upgrades
			if standardMatches == 0 || lazyUpgrades == 0 {
				t.Errorf("Expected both standard matches and lazy upgrades for level %d, got %d standard, %d lazy",
					level, standardMatches, lazyUpgrades)
			}

			// For enhanced HC, check early exit for long matches
			if matcher.useEnhancedHC {
				// Create a very long match
				matcher.pos = 5000
				longOffset, longLength := 100, 64 // A match better than the early exit threshold
				_, _, advance := matcher.LazyMatch(longOffset, longLength)
				if advance != 1 {
					t.Errorf("Long match (%d bytes) should trigger early exit but didn't for level %d",
						longLength, level)
				}
			}
		})
	}
}

// TestCompressionQuality tests overall compression quality
func TestCompressionQuality(t *testing.T) {
	// Generate test data with good compressibility
	data := createRepeatedData(1024 * 1024) // 1MB

	// Test compression with different levels
	for level := CompressionLevel(1); level <= 12; level += 3 {
		// Only test a few levels to save time
		t.Run("Level-"+string(rune('0'+level)), func(t *testing.T) {
			compressedV1, err := CompressBlockLevel(data, nil, level)
			if err != nil {
				t.Fatalf("CompressBlockLevel error: %v", err)
			}

			compressedV2, err := CompressBlockV2Level(data, nil, level)
			if err != nil {
				t.Fatalf("CompressBlockV2Level error: %v", err)
			}

			// Verify both compressed outputs
			decompressedV1, err := DecompressBlock(compressedV1, nil, len(data))
			if err != nil {
				t.Fatalf("DecompressBlock error: %v", err)
			}

			decompressedV2, err := DecompressBlock(compressedV2, nil, len(data))
			if err != nil {
				t.Fatalf("DecompressBlock error: %v", err)
			}

			if !bytes.Equal(data, decompressedV1) || !bytes.Equal(data, decompressedV2) {
				t.Fatalf("Decompressed data doesn't match original")
			}

			// For higher levels, V2 should generally give better compression
			if level >= 9 && len(compressedV2) >= len(compressedV1) {
				t.Logf("Warning: V2 compression (%d bytes) not better than V1 (%d bytes) at level %d",
					len(compressedV2), len(compressedV1), level)
			}

			// Log compression ratios
			ratioV1 := float64(len(data)) / float64(len(compressedV1))
			ratioV2 := float64(len(data)) / float64(len(compressedV2))
			t.Logf("Level %d: V1 ratio: %.2fx, V2 ratio: %.2fx", level, ratioV1, ratioV2)
		})
	}
}

// Helper function to create test data with repeated patterns
func createRepeatedData(size int) []byte {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, size)

	// Create several patterns
	patternCount := 5
	patterns := make([][]byte, patternCount)
	for i := 0; i < patternCount; i++ {
		patterns[i] = make([]byte, 128)
		for j := 0; j < 128; j++ {
			patterns[i][j] = byte(rand.Intn(256))
		}
	}

	// Fill data with patterns
	pos := 0
	for pos < size {
		// Pick a random pattern
		pattern := patterns[rand.Intn(patternCount)]

		// Determine length of this pattern (with some variation)
		repeatCount := rand.Intn(64) + 1
		for i := 0; i < repeatCount && pos < size; i++ {
			// Copy the pattern
			copyLen := min(len(pattern), size-pos)
			copy(data[pos:], pattern[:copyLen])

			// Maybe modify a few bytes to create some variations
			if rand.Float32() < 0.2 {
				// Modify 1-3 bytes
				modCount := rand.Intn(3) + 1
				for j := 0; j < modCount && pos+j < size; j++ {
					modPos := pos + rand.Intn(copyLen)
					if modPos < size {
						data[modPos] = byte(rand.Intn(256))
					}
				}
			}

			pos += copyLen
		}
	}

	return data
}

// BenchmarkHCMatcher benchmarks the matcher with different levels
func BenchmarkHCMatcher(b *testing.B) {
	// Create test data
	data := createRepeatedData(1024 * 1024) // 1MB

	// Test all compression levels
	for level := CompressionLevel(1); level <= 12; level++ {
		b.Run("Level-"+string(rune('0'+level)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a new matcher for each iteration to ensure fair comparison
				matcher := NewHCMatcher(level)
				matcher.Reset(data)

				// Find matches throughout the data
				matches := 0
				pos := 0
				for pos < len(data)-MinMatch {
					matcher.pos = pos
					_, length := matcher.FindBestMatch()
					if length >= MinMatch {
						matches++
						pos += length
					} else {
						pos++
					}
				}

				// Use matches to prevent compiler optimization
				if matches == 0 {
					b.Fatalf("No matches found in test data for level %d", level)
				}
			}
		})
	}
}
