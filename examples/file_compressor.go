// Example file compressor using GoZ4X
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/harriteja/GoZ4X/compress"
)

const (
	// Default file extension for compressed files
	defaultExtension = ".lz4"

	// Buffer size for reading/writing
	bufferSize = 4 * 1024 * 1024 // 4MB
)

// Command line flags
var (
	decompress  bool
	level       int
	outputFile  string
	threads     int
	showVersion bool
)

func init() {
	// Parse command line flags
	flag.BoolVar(&decompress, "d", false, "Decompress mode")
	flag.IntVar(&level, "l", int(compress.DefaultLevel), "Compression level (1-12)")
	flag.StringVar(&outputFile, "o", "", "Output file (default: automatic)")
	flag.IntVar(&threads, "t", runtime.GOMAXPROCS(0), "Number of threads")
	flag.BoolVar(&showVersion, "v", false, "Show version information")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GoZ4X File Compressor\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] file\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main3() {
	// Parse flags
	flag.Parse()

	// Show version if requested
	if showVersion {
		fmt.Println("GoZ4X File Compressor v0.1")
		fmt.Println("Go version:", runtime.Version())
		fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0))
		os.Exit(0)
	}

	// Check we have exactly one input file
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Get input file
	inputFile := flag.Arg(0)

	// Check input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file %s does not exist\n", inputFile)
		os.Exit(1)
	}

	// Determine output file if not specified
	if outputFile == "" {
		if decompress {
			// For decompression, remove .lz4 extension if present
			if strings.HasSuffix(inputFile, defaultExtension) {
				outputFile = inputFile[0 : len(inputFile)-len(defaultExtension)]
			} else {
				outputFile = inputFile + ".out"
			}
		} else {
			// For compression, add .lz4 extension
			outputFile = inputFile + defaultExtension
		}
	}

	// Validate compression level
	if level < 1 || level > 12 {
		fmt.Fprintf(os.Stderr, "Error: Compression level must be between 1 and 12\n")
		os.Exit(1)
	}

	// Process the file
	var err error
	if decompress {
		err = decompressFile(inputFile, outputFile)
	} else {
		err = compressFile(inputFile, outputFile, compress.CompressionLevel(level))
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// compressFile compresses a file
func compressFile(inputPath, outputPath string, level compress.CompressionLevel) error {
	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %v", err)
	}
	defer inputFile.Close()

	// Get file info for progress reporting
	fileInfo, err := inputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat input file: %v", err)
	}
	fileSize := fileInfo.Size()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Create LZ4 writer
	lz4Writer := compress.NewWriterLevel(outputFile, level)
	defer lz4Writer.Close()

	// Copy data
	buffer := make([]byte, bufferSize)
	totalRead := int64(0)
	startTime := time.Now()

	for {
		// Read a chunk
		n, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %v", err)
		}

		if n == 0 {
			break
		}

		// Write compressed chunk
		_, err = lz4Writer.Write(buffer[:n])
		if err != nil {
			return fmt.Errorf("write error: %v", err)
		}

		// Update progress
		totalRead += int64(n)
		if fileSize > 0 {
			progress := float64(totalRead) / float64(fileSize) * 100
			fmt.Printf("\rCompressing: %.1f%% complete", progress)
		} else {
			fmt.Printf("\rCompressing: %d bytes", totalRead)
		}
	}

	// Finalize compression
	err = lz4Writer.Close()
	if err != nil {
		return fmt.Errorf("close error: %v", err)
	}

	// Get compressed file size
	compressedInfo, err := outputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}
	compressedSize := compressedInfo.Size()

	// Calculate ratio and speed
	ratio := float64(compressedSize) / float64(fileSize) * 100
	duration := time.Since(startTime)
	speed := float64(fileSize) / duration.Seconds() / (1024 * 1024)

	fmt.Printf("\rCompressed %s to %s: %d -> %d bytes (%.2f%%), %.2f MB/s\n",
		inputPath, outputPath, fileSize, compressedSize, ratio, speed)

	return nil
}

// decompressFile decompresses a file
func decompressFile(inputPath, outputPath string) error {
	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %v", err)
	}
	defer inputFile.Close()

	// Get file info for progress reporting
	fileInfo, err := inputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat input file: %v", err)
	}
	fileSize := fileInfo.Size()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Create LZ4 reader
	lz4Reader := compress.NewReader(inputFile)

	// Copy data
	buffer := make([]byte, bufferSize)
	totalWritten := int64(0)
	startTime := time.Now()

	for {
		// Read decompressed chunk
		n, err := lz4Reader.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %v", err)
		}

		if n == 0 {
			break
		}

		// Write to output file
		_, err = outputFile.Write(buffer[:n])
		if err != nil {
			return fmt.Errorf("write error: %v", err)
		}

		// Update progress
		totalWritten += int64(n)
		fmt.Printf("\rDecompressing: %d bytes", totalWritten)
	}

	// Get decompressed file size
	decompressedInfo, err := outputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}
	decompressedSize := decompressedInfo.Size()

	// Calculate ratio and speed
	ratio := float64(fileSize) / float64(decompressedSize) * 100
	duration := time.Since(startTime)
	speed := float64(decompressedSize) / duration.Seconds() / (1024 * 1024)

	fmt.Printf("\rDecompressed %s to %s: %d -> %d bytes (%.2f%%), %.2f MB/s\n",
		inputPath, outputPath, fileSize, decompressedSize, ratio, speed)

	return nil
}
