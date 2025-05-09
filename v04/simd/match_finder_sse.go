//go:build amd64
// +build amd64

package simd

import (
	"unsafe"
)

// SSEMatchFinder implements match finding using SSE4.1 instructions
type SSEMatchFinder struct {
	// Configuration
	minMatch   int
	maxOffset  int
	maxMatches int

	// Hash table for position lookup
	hashTable []int

	// Input data
	data []byte
}

// NewSSEMatchFinder creates a new SSE-accelerated match finder
func NewSSEMatchFinder(windowSize int, minMatch int) *SSEMatchFinder {
	if minMatch < 4 {
		minMatch = 4 // LZ4 minimum match length
	}

	// Create hash table - size based on window size for better distribution
	hashBits := 16
	if windowSize > 65536 {
		hashBits = 18
	}
	hashSize := 1 << hashBits

	return &SSEMatchFinder{
		minMatch:   minMatch,
		maxOffset:  windowSize,
		maxMatches: 64, // Maximum matches to consider
		hashTable:  make([]int, hashSize),
	}
}

// Reset prepares the match finder for a new input
func (m *SSEMatchFinder) Reset(data []byte) {
	m.data = data

	// Clear hash table
	for i := range m.hashTable {
		m.hashTable[i] = 0
	}
}

// Hash4 computes a 4-byte hash at position p
func (m *SSEMatchFinder) Hash4(p int) int {
	// Fast 4-byte hash function
	if p+4 > len(m.data) {
		return 0
	}

	h := uint32(m.data[p]) | (uint32(m.data[p+1]) << 8) |
		(uint32(m.data[p+2]) << 16) | (uint32(m.data[p+3]) << 24)

	// FNV-1a hash function - good distribution with low collision
	h *= 2654435761
	h ^= h >> 16
	return int(h) & (len(m.hashTable) - 1)
}

// Stub implementations for testing purposes

//go:linkname compareSSE github.com/harriteja/GoZ4X/v04/simd.compareSSE
func compareSSE(a, b unsafe.Pointer, length int) int {
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

//go:linkname countMatchingBytesSSE github.com/harriteja/GoZ4X/v04/simd.countMatchingBytesSSE
func countMatchingBytesSSE(a, b unsafe.Pointer, limit int) int {
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

// FindMatchSSE uses SSE instructions to find the longest match at position p
func (m *SSEMatchFinder) FindMatchSSE(p int) (offset, length int) {
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

	// If longer than 4 bytes, use SSE to find match length
	matchLen := 4
	if maxLen > 4 {
		// Use unsafe pointers for the SSE comparison functions
		a := unsafe.Pointer(&m.data[prev+4])
		b := unsafe.Pointer(&m.data[p+4])

		// Use SSE to count matching bytes
		matchLen += countMatchingBytesSSE(a, b, maxLen-4)
	}

	// Return match if long enough
	if matchLen >= m.minMatch {
		return p - prev, matchLen
	}

	return 0, 0
}

// FindMatches finds all matches at position p and returns them in order of length
func (m *SSEMatchFinder) FindMatches(p int) []Match {
	offset, length := m.FindMatchSSE(p)
	if length == 0 {
		return nil
	}

	return []Match{
		{Offset: offset, Length: length},
	}
}

// In a complete implementation, we would have the following assembly functions:
// - compareSSE: Use SSE4.1 for comparing 16 bytes at a time
// - countMatchingBytesSSE: Use SSE4.1 to count matching byte prefix
// - findLongestMatchSSE: Main SSE accelerated match finder
