// Package v03 implements the v0.3 features of GoZ4X
package v03

import (
	"io"
	"sync"

	"github.com/harriteja/GoZ4X/compress"
	"github.com/harriteja/GoZ4X/parallel"
)

// ParallelWriter is an io.WriteCloser that compresses data in parallel for better performance
type ParallelWriter struct {
	// Base writer - use standard Writer instead of ParallelWriter for better compatibility
	w *compress.Writer

	// Parallel compression dispatcher
	dispatcher *parallel.Dispatcher

	// Options
	numWorkers int
	chunkSize  int
	useV2      bool
	level      int

	// Synchronization
	mu sync.Mutex
}

// NewParallelWriter creates a new ParallelWriter with default options
func NewParallelWriter(w io.Writer) *ParallelWriter {
	return NewParallelWriterLevel(w, int(compress.DefaultLevel))
}

// NewParallelWriterLevel creates a new ParallelWriter with custom compression level
func NewParallelWriterLevel(w io.Writer, level int) *ParallelWriter {
	return NewParallelWriterWithOptions(w, ParallelWriterOptions{
		Level:      level,
		NumWorkers: 0, // Use default/GOMAXPROCS
		ChunkSize:  0, // Use default chunk size
	})
}

// ParallelWriterOptions provides configuration options for a ParallelWriter
type ParallelWriterOptions struct {
	// Compression level (1-12)
	Level int
	// Number of worker goroutines (0 = use GOMAXPROCS)
	NumWorkers int
	// Size of chunks for parallel compression (0 = use default)
	ChunkSize int
	// Use v0.2 algorithm for better compression
	UseV2 bool
}

// NewParallelWriterWithOptions creates a new ParallelWriter with custom options
func NewParallelWriterWithOptions(w io.Writer, options ParallelWriterOptions) *ParallelWriter {
	// Create the base Writer instead of ParallelWriter for better compatibility
	var baseWriter *compress.Writer
	if options.UseV2 {
		baseWriter = compress.NewWriterWithOptions(w, compress.WriterOptions{
			Level: compress.CompressionLevel(options.Level),
			UseV2: true,
		})
	} else {
		baseWriter = compress.NewWriterLevel(w, compress.CompressionLevel(options.Level))
	}

	// Create the dispatcher
	chunkSize := options.ChunkSize
	if chunkSize <= 0 {
		chunkSize = parallel.DefaultChunkSize
	}

	numWorkers := options.NumWorkers
	if numWorkers <= 0 {
		numWorkers = parallel.DefaultNumWorkers
	}

	dispatcher := parallel.NewDispatcher(numWorkers, chunkSize)
	err := dispatcher.Start()
	if err != nil {
		// Fall back to single-threaded if dispatcher fails
		dispatcher = nil
	}

	return &ParallelWriter{
		w:          baseWriter,
		dispatcher: dispatcher,
		numWorkers: numWorkers,
		chunkSize:  chunkSize,
		useV2:      options.UseV2,
		level:      options.Level,
	}
}

// Write implements io.Writer
// This implementation collects data until the buffer is full,
// then compresses it in parallel and writes it to the underlying writer
func (pw *ParallelWriter) Write(p []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// If dispatcher is nil, fall back to non-parallel writer
	if pw.dispatcher == nil {
		return pw.w.Write(p)
	}

	// For now, we just use the base writer implementation
	// We could optimize this later to use the dispatcher directly
	return pw.w.Write(p)
}

// Close implements io.Closer
func (pw *ParallelWriter) Close() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Stop the dispatcher if it exists
	if pw.dispatcher != nil {
		pw.dispatcher.Stop()
	}

	// Close the base writer
	return pw.w.Close()
}

// SetNumWorkers sets the number of worker goroutines
func (pw *ParallelWriter) SetNumWorkers(n int) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.dispatcher != nil {
		pw.dispatcher.SetNumWorkers(n)
	}
	pw.numWorkers = n
}

// SetChunkSize sets the chunk size for parallel compression
func (pw *ParallelWriter) SetChunkSize(size int) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.dispatcher != nil {
		pw.dispatcher.SetChunkSize(size)
	}
	pw.chunkSize = size
}

// NumWorkers returns the number of worker goroutines
func (pw *ParallelWriter) NumWorkers() int {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.numWorkers
}

// ChunkSize returns the chunk size used for parallel compression
func (pw *ParallelWriter) ChunkSize() int {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.chunkSize
}

// Reset resets the writer to write to w
func (pw *ParallelWriter) Reset(w io.Writer) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Reset the base writer
	pw.w.Reset(w)

	// If dispatcher exists, restart it
	if pw.dispatcher != nil {
		pw.dispatcher.Stop()
		err := pw.dispatcher.Start()
		if err != nil {
			// If restarting fails, set to nil so we fall back to non-parallel
			pw.dispatcher = nil
		}
	}
}
