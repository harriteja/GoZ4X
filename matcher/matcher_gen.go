// Package matcher provides generic match-finding algorithms for LZ4 compression.
package matcher

// Index represents a type that can be used as an index into a buffer
type Index interface {
	~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64
}

// GenericMatcher is a generic match finder implementation that can work with
// different index types and hash table implementations.
type GenericMatcher[I Index] struct {
	// Input buffer
	buf []byte

	// Hash table
	hashTable []I

	// Chain table for linked matches
	chainTable []I

	// Current position in buffer
	pos I

	// End position in buffer
	end I

	// Window size for search
	windowSize I

	// Hash shift and mask
	hashLog  uint
	hashMask I

	// Search parameters
	maxAttempts int
}

// HashTableConfig defines the configuration for a hash table
type HashTableConfig struct {
	// HashLog determines hash table size (1 << HashLog)
	HashLog uint
	// WindowSize defines how far back we can search
	WindowSize int
	// MaxAttempts limits search depth
	MaxAttempts int
}

// DefaultConfig returns a default hash table configuration
func DefaultConfig() HashTableConfig {
	return HashTableConfig{
		HashLog:     16,
		WindowSize:  65535,
		MaxAttempts: 8,
	}
}

// NewMatcher creates a new generic matcher with the given configuration
func NewMatcher[I Index](config HashTableConfig) *GenericMatcher[I] {
	hashSize := I(1) << config.HashLog

	return &GenericMatcher[I]{
		hashTable:   make([]I, hashSize),
		chainTable:  nil, // Will be initialized in Reset
		pos:         0,
		end:         0,
		windowSize:  I(config.WindowSize),
		hashLog:     config.HashLog,
		hashMask:    hashSize - 1,
		maxAttempts: config.MaxAttempts,
	}
}

// Reset prepares the matcher for new input
func (m *GenericMatcher[I]) Reset(input []byte) {
	m.buf = input
	m.end = I(len(input))
	m.pos = 0

	// Initialize or resize chain table if needed
	if m.chainTable == nil || I(cap(m.chainTable)) < m.end {
		m.chainTable = make([]I, len(input))
	} else {
		m.chainTable = m.chainTable[:len(input)]
	}

	// Reset hash table
	for i := range m.hashTable {
		m.hashTable[i] = 0
	}
}

// hash4 computes a 4-byte hash at the given position
func (m *GenericMatcher[I]) hash4(pos I) I {
	if pos+4 > m.end {
		return 0
	}

	// Combine 4 bytes into a 32-bit value
	v := uint32(m.buf[pos]) | uint32(m.buf[pos+1])<<8 |
		uint32(m.buf[pos+2])<<16 | uint32(m.buf[pos+3])<<24

	// Use multiply-shift hashing (FNV-1a variant)
	return I(((v * 2654435761) >> (32 - m.hashLog)) & uint32(m.hashMask))
}

// InsertHash inserts the current position into the hash table
func (m *GenericMatcher[I]) InsertHash(pos I) {
	h := m.hash4(pos)
	m.chainTable[pos] = m.hashTable[h]
	m.hashTable[h] = pos
}

// FindBestMatch finds the best match at the current position
func (m *GenericMatcher[I]) FindBestMatch() (offset I, length I) {
	const MinMatch = 4 // Minimum match length for LZ4

	if m.pos+MinMatch > m.end {
		return 0, 0
	}

	h := m.hash4(m.pos)
	current := m.hashTable[h]

	// No match found
	if current <= 0 || current <= m.pos-m.windowSize {
		m.InsertHash(m.pos)
		return 0, 0
	}

	// Find the best match
	var bestLength I = 0
	var bestOffset I = 0
	limit := m.pos - m.windowSize
	attempts := m.maxAttempts

	for current > limit && attempts > 0 {
		attempts--

		// Check match length
		var length I = 0
		maxLen := m.end - m.pos

		// Compare bytes
		for length < maxLen && m.buf[m.pos+length] == m.buf[current+length] {
			length++
		}

		// Update best match
		if length > bestLength {
			bestLength = length
			bestOffset = m.pos - current

			// Early exit if we found a very long match
			if length >= I(258) { // Common maximum in compression algorithms
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

// Advance moves the current position forward
func (m *GenericMatcher[I]) Advance(steps I) {
	m.pos += steps
}

// Current returns the current position
func (m *GenericMatcher[I]) Current() I {
	return m.pos
}

// End returns true if we've reached the end of the input
func (m *GenericMatcher[I]) End() bool {
	const MinMatch = 4
	return m.pos >= m.end-MinMatch
}

// DictionaryMatcher adds dictionary support to the generic matcher
type DictionaryMatcher[I Index] struct {
	*GenericMatcher[I]
	dictEnd I
}

// NewDictionaryMatcher creates a new matcher with dictionary support
func NewDictionaryMatcher[I Index](config HashTableConfig) *DictionaryMatcher[I] {
	return &DictionaryMatcher[I]{
		GenericMatcher: NewMatcher[I](config),
		dictEnd:        0,
	}
}

// LoadDictionary loads a dictionary into the matcher
func (dm *DictionaryMatcher[I]) LoadDictionary(dict []byte) {
	// Reset with dictionary buffer
	dm.Reset(dict)

	// Build hash table for dictionary
	for pos := I(0); pos < I(len(dict))-4; pos++ {
		dm.InsertHash(pos)
	}

	// Mark end of dictionary
	dm.dictEnd = I(len(dict))
}

// LoadInput loads the input data after dictionary
func (dm *DictionaryMatcher[I]) LoadInput(input []byte) {
	// Create a combined buffer
	combined := make([]byte, dm.dictEnd+I(len(input)))
	copy(combined, dm.buf[:dm.dictEnd])
	copy(combined[dm.dictEnd:], input)

	// Reset with combined buffer but keep hash table
	oldChainTable := dm.chainTable
	dm.buf = combined
	dm.end = I(len(combined))
	dm.pos = dm.dictEnd

	// Resize chain table if needed
	if I(cap(oldChainTable)) < dm.end {
		dm.chainTable = make([]I, len(combined))
		copy(dm.chainTable[:dm.dictEnd], oldChainTable[:dm.dictEnd])
	} else {
		dm.chainTable = oldChainTable[:len(combined)]
	}
}
