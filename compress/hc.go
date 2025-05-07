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
}

// NewHCMatcher creates a new high-compression matcher
func NewHCMatcher(level CompressionLevel) *HCMatcher {
	// Determine search parameters based on level
	maxAttempts := 0
	switch {
	case level <= 3:
		maxAttempts = 4
	case level <= 6:
		maxAttempts = 8
	case level <= 9:
		maxAttempts = 16
	default:
		maxAttempts = 32
	}

	return &HCMatcher{
		hashTable:   make([]int, HashTableSize),
		chainTable:  nil, // Lazily initialized
		maxAttempts: maxAttempts,
		windowSize:  MaxDistance,
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
	return ((v * 2654435761) >> (32 - HashLog)) & HashMask
}

// InsertHash inserts the current position into the hash table
func (hc *HCMatcher) InsertHash(pos int) {
	h := hc.hash4(pos)
	hc.chainTable[pos] = hc.hashTable[h]
	hc.hashTable[h] = pos
}

// FindBestMatch finds the best match at the current position
func (hc *HCMatcher) FindBestMatch() (offset, length int) {
	if hc.pos+MinMatch > hc.end {
		return 0, 0
	}

	h := hc.hash4(hc.pos)
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

	for current > limit && attempts > 0 {
		attempts--

		// Check match length
		length := 0
		maxLength := hc.end - hc.pos

		// Compare bytes
		for length < maxLength && hc.buf[hc.pos+length] == hc.buf[current+length] {
			length++
		}

		// Update best match
		if length > bestLength {
			bestLength = length
			bestOffset = hc.pos - current

			// Early exit if we found a "good enough" match
			if length >= MaxMatch {
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

	// Save current position
	currentPos := hc.pos

	// Try next position
	hc.pos++
	nextOffset, nextLength := hc.FindBestMatch()

	// Restore position
	hc.pos = currentPos

	// If next position gives better compression, use it
	if nextLength > length+1 {
		return nextOffset, nextLength, 2 // Skip current and use next
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
