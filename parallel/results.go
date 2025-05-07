package parallel

import (
	"errors"
	"sync"
)

// ResultsCollector manages ordered collection of compression results
type ResultsCollector struct {
	// Results buffer
	results []BlockResult

	// Current state
	numBlocks      int
	nextBlockIndex int
	complete       bool

	// Synchronization
	mu        sync.Mutex
	completed sync.Cond
}

// BlockResult represents a compressed block and its metadata
type BlockResult struct {
	// Block index in original sequence
	Index int

	// Compressed data
	Data []byte

	// Original size
	OriginalSize int

	// Checksum if enabled
	Checksum uint32
}

// NewResultsCollector creates a new results collector for the specified number of blocks
func NewResultsCollector(numBlocks int) *ResultsCollector {
	if numBlocks <= 0 {
		numBlocks = 1
	}

	rc := &ResultsCollector{
		results:        make([]BlockResult, numBlocks),
		numBlocks:      numBlocks,
		nextBlockIndex: 0,
	}
	rc.completed.L = &rc.mu

	return rc
}

// AddResult adds a compressed block result
func (rc *ResultsCollector) AddResult(result BlockResult) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.complete {
		return errors.New("collector is already complete")
	}

	if result.Index < 0 || result.Index >= rc.numBlocks {
		return errors.New("block index out of range")
	}

	// Store the result
	rc.results[result.Index] = result

	// Check if collection is complete
	complete := true
	for i := 0; i < rc.numBlocks; i++ {
		if rc.results[i].Data == nil {
			complete = false
			break
		}
	}

	if complete {
		rc.complete = true
		rc.completed.Broadcast()
	}

	return nil
}

// WaitForCompletion waits until all blocks have been collected
func (rc *ResultsCollector) WaitForCompletion() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for !rc.complete {
		rc.completed.Wait()
	}
}

// IsComplete returns true if all results have been collected
func (rc *ResultsCollector) IsComplete() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	return rc.complete
}

// GetResult gets a specific block result
func (rc *ResultsCollector) GetResult(index int) (BlockResult, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if index < 0 || index >= rc.numBlocks {
		return BlockResult{}, errors.New("block index out of range")
	}

	result := rc.results[index]
	if result.Data == nil {
		return BlockResult{}, errors.New("block not available")
	}

	return result, nil
}

// GetAllResults gets all block results in order
func (rc *ResultsCollector) GetAllResults() ([]BlockResult, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if !rc.complete {
		return nil, errors.New("collection not complete")
	}

	// Make a copy to avoid race conditions
	results := make([]BlockResult, len(rc.results))
	copy(results, rc.results)

	return results, nil
}

// GetNextResult gets the next sequential block result, waiting if necessary
func (rc *ResultsCollector) GetNextResult() (BlockResult, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Wait for next block to become available
	for rc.nextBlockIndex < rc.numBlocks {
		if rc.results[rc.nextBlockIndex].Data != nil {
			result := rc.results[rc.nextBlockIndex]
			rc.nextBlockIndex++
			return result, nil
		}

		// Wait for more results
		rc.completed.Wait()
	}

	return BlockResult{}, errors.New("no more blocks")
}

// CombineResults combines all compressed blocks into a single buffer
func (rc *ResultsCollector) CombineResults() ([]byte, error) {
	results, err := rc.GetAllResults()
	if err != nil {
		return nil, err
	}

	// Calculate total size
	totalSize := 0
	for _, result := range results {
		totalSize += len(result.Data)
	}

	// Combine results
	combined := make([]byte, totalSize)
	offset := 0

	for _, result := range results {
		copy(combined[offset:], result.Data)
		offset += len(result.Data)
	}

	return combined, nil
}

// Reset prepares the collector for reuse
func (rc *ResultsCollector) Reset(numBlocks int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if numBlocks <= 0 {
		numBlocks = 1
	}

	rc.results = make([]BlockResult, numBlocks)
	rc.numBlocks = numBlocks
	rc.nextBlockIndex = 0
	rc.complete = false
}

// BlockResultHeap implements a priority queue for out-of-order block results
type BlockResultHeap []BlockResult

// Len implements sort.Interface
func (h BlockResultHeap) Len() int {
	return len(h)
}

// Less implements sort.Interface
func (h BlockResultHeap) Less(i, j int) bool {
	return h[i].Index < h[j].Index
}

// Swap implements sort.Interface
func (h BlockResultHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push implements heap.Interface
func (h *BlockResultHeap) Push(x interface{}) {
	*h = append(*h, x.(BlockResult))
}

// Pop implements heap.Interface
func (h *BlockResultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	result := old[n-1]
	*h = old[0 : n-1]
	return result
}
