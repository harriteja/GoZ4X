// Package matcher provides improved match-finding algorithms for LZ4 compression.
package matcher

// LZ4XMatcher is an improved match finder for LZ4X that provides
// better compression ratios through optimized match finding strategies.
type LZ4XMatcher struct {
	// Input buffer
	buf []byte

	// Primary hash table for 4-byte matches
	hashTable []int

	// Chain table for linked matches
	chainTable []int

	// Current position in buffer
	pos int

	// End position in buffer
	end int

	// Window size for search
	windowSize int

	// Hash configuration
	hashLog  uint
	hashMask int

	// Search parameters
	maxAttempts  int
	skipStrength int
}

// LZ4XConfig defines the configuration for the LZ4X matcher
type LZ4XConfig struct {
	// HashLog determines hash table size (1 << HashLog)
	HashLog uint
	// WindowSize defines how far back we can search
	WindowSize int
	// MaxAttempts limits search depth
	MaxAttempts int
	// SkipStrength controls how many positions to skip when searching
	SkipStrength int
}

// DefaultLZ4XConfig returns an optimized default configuration
func DefaultLZ4XConfig() LZ4XConfig {
	return LZ4XConfig{
		HashLog:      16,
		WindowSize:   65535,
		MaxAttempts:  16,
		SkipStrength: 3,
	}
}

// NewLZ4XMatcher creates a new LZ4X matcher with the given configuration
func NewLZ4XMatcher(config LZ4XConfig) *LZ4XMatcher {
	hashSize := 1 << config.HashLog

	return &LZ4XMatcher{
		hashTable:    make([]int, hashSize),
		chainTable:   nil, // Will be initialized in Reset
		pos:          0,
		end:          0,
		windowSize:   config.WindowSize,
		hashLog:      config.HashLog,
		hashMask:     hashSize - 1,
		maxAttempts:  config.MaxAttempts,
		skipStrength: config.SkipStrength,
	}
}

// Reset prepares the matcher for new input
func (m *LZ4XMatcher) Reset(input []byte) {
	m.buf = input
	m.end = len(input)
	m.pos = 0

	// Initialize or resize chain table if needed
	if m.chainTable == nil || cap(m.chainTable) < len(input) {
		m.chainTable = make([]int, len(input))
	} else {
		m.chainTable = m.chainTable[:len(input)]
	}

	// Reset hash table
	for i := range m.hashTable {
		m.hashTable[i] = 0
	}
}

// hash4 computes a 4-byte hash at the given position
func (m *LZ4XMatcher) hash4(pos int) int {
	if pos+4 > m.end {
		return 0
	}

	// Combine 4 bytes into a 32-bit value
	v := uint32(m.buf[pos]) | uint32(m.buf[pos+1])<<8 |
		uint32(m.buf[pos+2])<<16 | uint32(m.buf[pos+3])<<24

	// Use multiply-shift hashing
	return int(((v * 2654435761) >> (32 - m.hashLog)) & uint32(m.hashMask))
}

// InsertHash inserts the current position into the hash table
func (m *LZ4XMatcher) InsertHash(pos int) {
	h4 := m.hash4(pos)
	if h4 != 0 {
		m.chainTable[pos] = m.hashTable[h4]
		m.hashTable[h4] = pos
	}
}

// FindBestMatch finds the best match at the current position
func (m *LZ4XMatcher) FindBestMatch() (offset, length int) {
	const MinMatch = 4 // Minimum match length for LZ4

	if m.pos+MinMatch > m.end {
		m.InsertHash(m.pos)
		return 0, 0
	}

	h4 := m.hash4(m.pos)
	current := m.hashTable[h4]

	// No match found
	if current <= 0 || current <= m.pos-m.windowSize || current >= m.pos {
		m.InsertHash(m.pos)
		return 0, 0
	}

	// Find the best match
	bestLength := 0
	bestOffset := 0
	limit := m.pos - m.windowSize
	attempts := m.maxAttempts

	// Check 4-byte hash matches
	for current > limit && attempts > 0 && current < m.pos {
		attempts--

		// Skip positions based on skip strength
		if m.skipStrength > 1 && attempts%m.skipStrength != 0 && current != 0 {
			current = m.chainTable[current]
			continue
		}

		// Calculate potential offset
		offset := m.pos - current

		// Only consider valid offsets for LZ4 (1-65535)
		if offset <= 0 || offset > 65535 {
			current = m.chainTable[current]
			continue
		}

		// Check match length
		length := 0
		maxLen := min(m.end-m.pos, 255+MinMatch) // LZ4 max match length

		// Compare bytes
		for length < maxLen &&
			current+length < m.end &&
			m.buf[m.pos+length] == m.buf[current+length] {
			length++
		}

		// Update best match if better than current
		if length >= MinMatch && length > bestLength {
			bestLength = length
			bestOffset = offset

			// Early exit if we found a very long match
			if length >= 64 {
				break
			}
		}

		// Move to next position in chain
		current = m.chainTable[current]
	}

	// Insert current position
	m.InsertHash(m.pos)

	// Return results
	if bestLength >= MinMatch {
		return bestOffset, bestLength
	}

	return 0, 0
}

// AdvanceHashOnly updates hash tables without checking for matches
func (m *LZ4XMatcher) AdvanceHashOnly(steps int) {
	for i := 0; i < steps; i++ {
		if m.pos+i < m.end-4 {
			m.InsertHash(m.pos + i)
		}
	}
	m.pos += steps
}

// Advance moves the current position forward
func (m *LZ4XMatcher) Advance(steps int) {
	m.pos += steps
}

// Current returns the current position
func (m *LZ4XMatcher) Current() int {
	return m.pos
}

// End returns true if we've reached the end of the input
func (m *LZ4XMatcher) End() bool {
	const MinMatch = 4
	return m.pos >= m.end-MinMatch
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
