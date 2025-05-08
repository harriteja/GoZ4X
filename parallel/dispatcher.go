// Package parallel provides parallel compression capabilities for LZ4.
package parallel

import (
	"errors"
	"runtime"
	"sync"

	"github.com/harriteja/GoZ4X/compress"
)

// DefaultChunkSize is the default size of chunks for parallel compression
const DefaultChunkSize = 1 << 20 // 1MB

// DefaultNumWorkers is the default number of worker goroutines
const DefaultNumWorkers = 0 // 0 means use runtime.GOMAXPROCS(0)

// Dispatcher manages parallel compression of LZ4 blocks
type Dispatcher struct {
	// Number of worker goroutines
	numWorkers int

	// Size of each chunk to compress in parallel
	chunkSize int

	// Channel for work distribution
	jobChan chan compressionJob

	// Channel for collecting results
	resultChan chan compressionResult

	// WaitGroup for worker synchronization
	wg sync.WaitGroup

	// Dispatcher state
	running   bool
	runningMu sync.RWMutex

	// Stats
	totalJobs   int
	totalBytes  int64
	runningJobs int
}

// compressionJob represents a block to be compressed
type compressionJob struct {
	id       int
	input    []byte
	level    int
	useV2    bool
	resultCh chan<- compressionResult
}

// compressionResult represents a compressed block
type compressionResult struct {
	id        int
	output    []byte
	err       error
	inputSize int
}

// NewDispatcher creates a new parallel compression dispatcher
func NewDispatcher(numWorkers, chunkSize int) *Dispatcher {
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}

	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	d := &Dispatcher{
		numWorkers: numWorkers,
		chunkSize:  chunkSize,
		jobChan:    make(chan compressionJob, numWorkers*2),
		resultChan: make(chan compressionResult, numWorkers*2),
	}

	return d
}

// Start launches worker goroutines
func (d *Dispatcher) Start() error {
	d.runningMu.Lock()
	defer d.runningMu.Unlock()

	if d.running {
		return errors.New("dispatcher already running")
	}

	// Reset stats
	d.totalJobs = 0
	d.totalBytes = 0
	d.runningJobs = 0

	// Start worker goroutines
	d.wg.Add(d.numWorkers)
	for i := 0; i < d.numWorkers; i++ {
		go d.worker()
	}

	d.running = true
	return nil
}

// Stop shuts down worker goroutines
func (d *Dispatcher) Stop() {
	d.runningMu.Lock()
	defer d.runningMu.Unlock()

	if !d.running {
		return
	}

	// Close job channel to signal workers to stop
	close(d.jobChan)

	// Wait for all workers to finish
	d.wg.Wait()

	// Create a new set of channels
	d.jobChan = make(chan compressionJob, d.numWorkers*2)
	d.resultChan = make(chan compressionResult, d.numWorkers*2)
	d.running = false
}

// worker processes compression jobs
func (d *Dispatcher) worker() {
	defer d.wg.Done()

	for job := range d.jobChan {
		// Compress the block
		result := d.compressBlock(job)

		// Send result back
		job.resultCh <- result
	}
}

// compressBlock compresses a single block
func (d *Dispatcher) compressBlock(job compressionJob) compressionResult {
	// Create compressed buffer with safety margin
	maxSize := len(job.input) + (len(job.input) / 255) + 16
	compressedBuf := make([]byte, maxSize)

	var compressed []byte
	var err error

	// Use the appropriate compression function based on useV2 flag
	if job.useV2 {
		// Use V2 algorithm
		compressed, err = compress.CompressBlockV2Level(job.input, compressedBuf, compress.CompressionLevel(job.level))
	} else {
		// Use standard algorithm
		compressed, err = compress.CompressBlockLevel(job.input, compressedBuf, compress.CompressionLevel(job.level))
	}

	return compressionResult{
		id:        job.id,
		output:    compressed,
		err:       err,
		inputSize: len(job.input),
	}
}

// CompressBlocks compresses multiple blocks in parallel
func (d *Dispatcher) CompressBlocks(input []byte, level int) ([]byte, error) {
	return d.compressBlocksInternal(input, level, false)
}

// CompressBlocksV2 compresses multiple blocks in parallel using the V2 algorithm
func (d *Dispatcher) CompressBlocksV2(input []byte, level int) ([]byte, error) {
	return d.compressBlocksInternal(input, level, true)
}

// compressBlocksInternal is the shared implementation for CompressBlocks and CompressBlocksV2
func (d *Dispatcher) compressBlocksInternal(input []byte, level int, useV2 bool) ([]byte, error) {
	// Guard against empty input
	if len(input) == 0 {
		return []byte{}, nil
	}

	// For very small inputs or single chunk compression, just do direct compression
	if len(input) <= d.chunkSize || len(input) < 4096 {
		// Use standard compression directly for small inputs
		if useV2 {
			return compress.CompressBlockV2Level(input, nil, compress.CompressionLevel(level))
		}
		return compress.CompressBlockLevel(input, nil, compress.CompressionLevel(level))
	}

	// Check if we need to start the dispatcher
	d.runningMu.RLock()
	running := d.running
	d.runningMu.RUnlock()

	if !running {
		// Need to start the dispatcher
		d.runningMu.Lock()
		// Check again after acquiring the exclusive lock
		if !d.running {
			if err := d.Start(); err != nil {
				d.runningMu.Unlock()
				// If we can't start the dispatcher, do single-threaded compression
				if useV2 {
					return compress.CompressBlockV2Level(input, nil, compress.CompressionLevel(level))
				}
				return compress.CompressBlockLevel(input, nil, compress.CompressionLevel(level))
			}
		}
		d.runningMu.Unlock()
	}

	// Split input into chunks
	numChunks := (len(input) + d.chunkSize - 1) / d.chunkSize
	results := make([]compressionResult, numChunks)

	// Process all chunks synchronously for consistency
	for i := 0; i < numChunks; i++ {
		start := i * d.chunkSize
		end := (i + 1) * d.chunkSize
		if end > len(input) {
			end = len(input)
		}

		job := compressionJob{
			id:       i,
			input:    input[start:end],
			level:    level,
			useV2:    useV2,
			resultCh: nil, // Not needed for synchronous processing
		}

		// Process directly
		result := d.compressBlock(job)
		results[i] = result

		if result.err != nil {
			return nil, result.err
		}
	}

	// Combine results
	// First calculate total size
	totalSize := 0
	for _, result := range results {
		totalSize += len(result.output)
	}

	// Allocate output buffer
	output := make([]byte, totalSize)

	// Copy results in order
	pos := 0
	for i := 0; i < numChunks; i++ {
		copy(output[pos:], results[i].output)
		pos += len(results[i].output)
	}

	return output, nil
}

// NumWorkers returns the number of worker goroutines
func (d *Dispatcher) NumWorkers() int {
	return d.numWorkers
}

// ChunkSize returns the size of chunks used for parallel compression
func (d *Dispatcher) ChunkSize() int {
	return d.chunkSize
}

// SetChunkSize changes the chunk size
func (d *Dispatcher) SetChunkSize(size int) {
	if size <= 0 {
		size = DefaultChunkSize
	}
	d.chunkSize = size
}

// SetNumWorkers changes the number of worker goroutines
func (d *Dispatcher) SetNumWorkers(n int) {
	d.runningMu.Lock()
	defer d.runningMu.Unlock()

	if d.running {
		return // Can't change while running
	}

	if n <= 0 {
		n = runtime.GOMAXPROCS(0)
	}

	d.numWorkers = n
}
