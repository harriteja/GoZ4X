//go:build arm64
// +build arm64

package simd

import (
	"unsafe"
)

// NEONMatchFinder implements match finding using ARM NEON instructions
type NEONMatchFinder struct {
	// Configuration
	minMatch   int
	maxOffset  int
	maxMatches int

	// Hash table for position lookup
	hashTable []int

	// Input data
	data []byte
}

// NewNEONMatchFinder creates a new NEON-accelerated match finder
func NewNEONMatchFinder(windowSize int, minMatch int) *NEONMatchFinder {
	if minMatch < 4 {
		minMatch = 4 // LZ4 minimum match length
	}

	// Create hash table based on window size
	hashBits := 16
	if windowSize > 65536 {
		hashBits = 18
	}
	hashSize := 1 << hashBits

	return &NEONMatchFinder{
		minMatch:   minMatch,
		maxOffset:  windowSize,
		maxMatches: 64, // Maximum matches to consider
		hashTable:  make([]int, hashSize),
	}
}

// Reset prepares the match finder for a new input
func (m *NEONMatchFinder) Reset(data []byte) {
	m.data = data

	// Clear hash table
	for i := range m.hashTable {
		m.hashTable[i] = 0
	}
}

// Hash4 computes a 4-byte hash at position p
func (m *NEONMatchFinder) Hash4(p int) int {
	// ARM-optimized 4-byte hash function
	if p+4 > len(m.data) {
		return 0
	}

	h := uint32(m.data[p]) | (uint32(m.data[p+1]) << 8) |
		(uint32(m.data[p+2]) << 16) | (uint32(m.data[p+3]) << 24)

	// FNV-1a hash with ARM-friendly operations
	h *= 2654435761
	h ^= h >> 17
	return int(h) & (len(m.hashTable) - 1)
}

// Stub implementations for testing purposes

//go:linkname compareNEON github.com/harriteja/GoZ4X/v04/simd.compareNEON
func compareNEON(a, b unsafe.Pointer, length int) int {
	// Simple Go implementation for testing
	aSlice := unsafe.Slice((*byte)(a), length)
	bSlice := unsafe.Slice((*byte)(b), length)

	for i := 0; i < length; i++ {
		if aSlice[i] != bSlice[i] {
			return i
		}
	}
	return length
}

//go:linkname countMatchingBytesNEON github.com/harriteja/GoZ4X/v04/simd.countMatchingBytesNEON
func countMatchingBytesNEON(a, b unsafe.Pointer, limit int) int {
	// Simple Go implementation for testing
	aSlice := unsafe.Slice((*byte)(a), limit)
	bSlice := unsafe.Slice((*byte)(b), limit)

	for i := 0; i < limit; i++ {
		if aSlice[i] != bSlice[i] {
			return i
		}
	}
	return limit
}

// FindMatchNEON uses NEON instructions to find the longest match at position p
func (m *NEONMatchFinder) FindMatchNEON(p int) (offset, length int) {
	// Ensure we have enough bytes
	if p+m.minMatch > len(m.data) {
		return 0, 0
	}

	// Get hash for current position
	h := m.Hash4(p)

	// Get the previous position with the same hash
	prev := m.hashTable[h]

	// Update hash table
	m.hashTable[h] = p

	// If no previous match or too far back, return no match
	if prev == 0 || p-prev > m.maxOffset || p-prev < 4 {
		return 0, 0
	}

	// Calculate maximum match length
	maxLen := len(m.data) - p
	if maxLen > 65535 { // LZ4 limit for match length
		maxLen = 65535
	}

	// First check if 4 bytes match (minimum match length)
	if *((*uint32)(unsafe.Pointer(&m.data[prev]))) != *((*uint32)(unsafe.Pointer(&m.data[p]))) {
		return 0, 0
	}

	// Use NEON to find match length beyond the first 4 bytes
	matchLen := 4
	if maxLen > 4 {
		// Use NEON-optimized function to count matching bytes
		a := unsafe.Pointer(&m.data[prev+4])
		b := unsafe.Pointer(&m.data[p+4])
		matchLen += countMatchingBytesNEON(a, b, maxLen-4)
	}

	// Return match if long enough
	if matchLen >= m.minMatch {
		return p - prev, matchLen
	}

	return 0, 0
}

// FindMatches finds all matches at position p and returns them in order of length
func (m *NEONMatchFinder) FindMatches(p int) []Match {
	offset, length := m.FindMatchNEON(p)
	if length == 0 {
		return nil
	}

	return []Match{
		{Offset: offset, Length: length},
	}
}

// In a complete implementation, we would have the following assembly functions:
// - compareNEON: Use NEON for comparing 16 bytes at a time
// - countMatchingBytesNEON: Use NEON to count matching byte prefix
// - findLongestMatchNEON: Main NEON accelerated match finder
