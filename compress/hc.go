package compress

const (
	// MinMatch is the minimum match length
	MinMatch = 4
	// MaxMatch is the maximum match length
	MaxMatch = 65535

	// MaxDistance is the maximum distance for a match
	MaxDistance = 65535

	// HashLog determines the size of the hash table
	HashLog = 16
	// HashTableSize is the size of the hash table
	HashTableSize = 1 << HashLog
	// HashMask is used to mask hash values
	HashMask = HashTableSize - 1

	// HashLogHC is for high compression levels (9-12)
	HashLogHC = 17
	// HashTableSizeHC is the size of high compression hash table
	HashTableSizeHC = 1 << HashLogHC
	// HashMaskHC is used to mask high compression hash values
	HashMaskHC = HashTableSizeHC - 1
)

// HCMatcher implements a high-compression match finder for LZ4HC
type HCMatcher struct {
	// Input buffer
	buf []byte

	// Hash tables - chain of positions with same hash
	hashTable  []int
	chainTable []int

	// Search depth based on compression level
	maxAttempts int

	// Window size controls how far back we can search
	windowSize int

	// Current position in the buffer
	pos int

	// End of buffer
	end int

	// Hash table size (may vary by level)
	hashLog       int
	hashSize      int
	hashMask      int
	useEnhancedHC bool
}

// NewHCMatcher creates a new high-compression matcher
func NewHCMatcher(level CompressionLevel) *HCMatcher {
	// Determine search parameters based on level
	maxAttempts := 0
	hashLog := HashLog
	windowSize := MaxDistance
	useEnhancedHC := false

	// Improved HC levels for v0.3
	switch {
	case level <= 3:
		maxAttempts = 4
		windowSize = 16 * 1024 // 16KB window for lowest levels
	case level <= 6:
		maxAttempts = 8
		windowSize = 32 * 1024 // 32KB window for medium levels
	case level <= 9:
		maxAttempts = 16
		windowSize = 64 * 1024 // 64KB window for high levels
		useEnhancedHC = true
	default:
		maxAttempts = 32
		windowSize = MaxDistance
		hashLog = HashLogHC // Use larger hash table for highest levels
		useEnhancedHC = true
	}

	hashSize := 1 << hashLog
	hashMask := hashSize - 1

	return &HCMatcher{
		hashTable:     make([]int, hashSize),
		chainTable:    nil, // Lazily initialized
		maxAttempts:   maxAttempts,
		windowSize:    windowSize,
		hashLog:       hashLog,
		hashSize:      hashSize,
		hashMask:      hashMask,
		useEnhancedHC: useEnhancedHC,
	}
}

// Reset prepares the matcher for a new input
func (hc *HCMatcher) Reset(input []byte) {
	hc.buf = input
	hc.end = len(input)
	hc.pos = 0

	// Initialize or resize chain table if needed
	if cap(hc.chainTable) < len(input) {
		hc.chainTable = make([]int, len(input))
	} else {
		hc.chainTable = hc.chainTable[:len(input)]
	}

	// Reset hash table
	for i := range hc.hashTable {
		hc.hashTable[i] = 0
	}
}

// hash4 computes a 4-byte hash
func (hc *HCMatcher) hash4(pos int) uint32 {
	if pos+4 > hc.end {
		return 0
	}

	v := uint32(hc.buf[pos]) | uint32(hc.buf[pos+1])<<8 | uint32(hc.buf[pos+2])<<16 | uint32(hc.buf[pos+3])<<24
	return ((v * 2654435761) >> (32 - hc.hashLog)) & uint32(hc.hashMask)
}

// hash5 computes a 5-byte hash for enhanced compression levels
func (hc *HCMatcher) hash5(pos int) uint32 {
	if pos+5 > hc.end {
		return 0
	}

	v := uint32(hc.buf[pos]) | uint32(hc.buf[pos+1])<<8 | uint32(hc.buf[pos+2])<<16 | uint32(hc.buf[pos+3])<<24
	v = v*2654435761 + uint32(hc.buf[pos+4])
	return ((v * 2654435761) >> (32 - hc.hashLog)) & uint32(hc.hashMask)
}

// InsertHash inserts the current position at the given index into the hash table
// and maintains the linked list in the chain table
func (hc *HCMatcher) InsertHash(pos int) {
	// Calculate hash for this position
	var h uint32
	if hc.useEnhancedHC {
		h = hc.hash5(pos)
	} else {
		h = hc.hash4(pos)
	}

	if h == 0 {
		return // Position at end of buffer, can't hash properly
	}

	// Update chainTable to point to the previous occurrence of this hash
	// Save current hash entry in chainTable
	hc.chainTable[pos] = hc.hashTable[h]

	// Update hash table to point to current position
	hc.hashTable[h] = pos
}

// FindBestMatch finds the best match at the current position
func (hc *HCMatcher) FindBestMatch() (offset, length int) {
	if hc.pos+MinMatch > hc.end {
		return 0, 0
	}

	// Calculate hash
	var h uint32
	if hc.useEnhancedHC {
		h = hc.hash5(hc.pos)
	} else {
		h = hc.hash4(hc.pos)
	}

	current := hc.hashTable[h]

	// No match
	if current <= 0 || current <= hc.pos-hc.windowSize {
		hc.InsertHash(hc.pos)
		return 0, 0
	}

	// Find the best match
	bestLength := 0
	bestOffset := 0
	limit := hc.pos - hc.windowSize
	attempts := hc.maxAttempts

	// Enhanced search algorithm for v0.3
	for current > limit && attempts > 0 {
		attempts--

		// Check match length
		length := 0
		maxLength := hc.end - hc.pos

		// Quickly check if the first 4 bytes match to filter bad matches
		if hc.buf[current] == hc.buf[hc.pos] &&
			hc.buf[current+1] == hc.buf[hc.pos+1] &&
			hc.buf[current+2] == hc.buf[hc.pos+2] &&
			hc.buf[current+3] == hc.buf[hc.pos+3] {

			// Compare bytes starting from 4th position (we already checked first 4)
			length = 4
			for length < maxLength && hc.buf[hc.pos+length] == hc.buf[current+length] {
				length++
			}
		}

		// Update best match
		if length > bestLength {
			bestLength = length
			bestOffset = hc.pos - current

			// Early exit if we found a "good enough" match
			if length >= MaxMatch {
				break
			}

			// Early exit strategy for higher levels
			if hc.useEnhancedHC && length >= 64 {
				break
			}
		}

		// Move to next position in chain
		current = hc.chainTable[current]
	}

	// Insert current position
	hc.InsertHash(hc.pos)

	// Return results
	if bestLength >= MinMatch {
		return bestOffset, bestLength
	}

	return 0, 0
}

// UpdateTables updates all hash tables with positions from start to end
func (hc *HCMatcher) UpdateTables(start, end int) {
	for pos := start; pos < end; pos++ {
		hc.InsertHash(pos)
	}
}

// LazyMatch performs lazy matching (check if next position gives better match)
func (hc *HCMatcher) LazyMatch(offset, length int) (newOffset, newLength int, advance int) {
	// Don't try lazy matching for short matches or near the end
	if length <= 1 || hc.pos+1 >= hc.end-MinMatch {
		return offset, length, 1
	}

	// Enhanced lazy evaluation for v0.3
	// If we already have a good long match, don't bother with lazy matching
	if hc.useEnhancedHC && length >= 32 {
		return offset, length, 1
	}

	// Save current position
	currentPos := hc.pos

	// Try next position
	hc.pos++
	nextOffset, nextLength := hc.FindBestMatch()

	// Restore position
	hc.pos = currentPos

	// For high compression levels, use a more aggressive lazy matching strategy
	if hc.useEnhancedHC {
		// If next position gives better compression, use it
		if nextLength > length {
			return nextOffset, nextLength, 2 // Skip current and use next
		}
	} else {
		// Standard strategy: next must be better by at least 2 bytes to be worth it
		if nextLength > length+1 {
			return nextOffset, nextLength, 2 // Skip current and use next
		}
	}

	// Otherwise use current match
	return offset, length, 1
}

// Advance moves the current position forward
func (hc *HCMatcher) Advance(steps int) {
	hc.pos += steps
}

// End returns true if we've reached the end of the input
func (hc *HCMatcher) End() bool {
	return hc.pos >= hc.end-MinMatch
}
