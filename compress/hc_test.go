package compress

import (
	"bytes"
	"testing"
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

			if len(matcher.hashTable) != HashTableSize {
				t.Errorf("hashTable size = %v, want %v", len(matcher.hashTable), HashTableSize)
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
	newOffset, newLength, advance := matcher.LazyMatch(offset1, length1)

	if advance <= 0 {
		t.Errorf("LazyMatch advance = %v, want > 0", advance)
	}

	// In our contrived example, we expect the new match to be better
	// But this depends on the exact data pattern and hash collisions
	t.Logf("Original match: offset=%v, length=%v", offset1, length1)
	t.Logf("Lazy match: offset=%v, length=%v, advance=%v", newOffset, newLength, advance)

	// Test the case where the current match is very short (should try lazy)
	shortOffset, shortLength, advance := matcher.LazyMatch(10, 1)
	if advance <= 0 {
		t.Errorf("LazyMatch advance for short match = %v, want > 0", advance)
	}
	t.Logf("Lazy match for short match: offset=%v, length=%v, advance=%v",
		shortOffset, shortLength, advance)
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
