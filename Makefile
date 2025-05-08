# GoZ4X Makefile

# Go commands
GO = go
GOFLAGS = -v
GOBUILD = $(GO) build $(GOFLAGS)
GOTEST = $(GO) test $(GOFLAGS)
GOMOD = $(GO) mod
GOBENCH = $(GO) test -bench=. -benchmem -count=5
GOPROF = $(GO) test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof

# Go build flags
BUILD_FLAGS = -trimpath -ldflags "-s -w"

# Output directories
BIN_DIR = bin
BUILD_DIR = build
PROFILE_DIR = profiles

# Binary names
BINARY_NAME = goz4x
FILE_COMP_BINARY = file_compressor

# Paths
EXAMPLES_DIR = examples
BENCH_DIR = bench
TEST_DIRS = ./compress/... ./matcher/... ./parallel/...

# Default target
.PHONY: all
all: build

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(BUILD_DIR) $(PROFILE_DIR) *.prof
	rm -f $(BINARY_NAME) $(FILE_COMP_BINARY)

# Init directories
.PHONY: init
init:
	mkdir -p $(BIN_DIR) $(BUILD_DIR) $(PROFILE_DIR)

# Build file compressor example
.PHONY: build-example
build-example: init
	$(GOBUILD) -o $(BIN_DIR)/$(FILE_COMP_BINARY) $(EXAMPLES_DIR)/file_compressor.go

# Run all tests
.PHONY: test
test:
	$(GOTEST) $(TEST_DIRS)

# Run basic benchmarks
.PHONY: bench
bench:
	$(GOBENCH) $(BENCH_DIR)/...

# Run detailed benchmarks with profiling
.PHONY: bench-profile
bench-profile: init
	mkdir -p $(PROFILE_DIR)
	$(GOPROF) $(BENCH_DIR)/...
	$(GO) tool pprof -pdf cpu.prof > $(PROFILE_DIR)/cpu.pdf
	$(GO) tool pprof -pdf mem.prof > $(PROFILE_DIR)/mem.pdf
	mv *.prof $(PROFILE_DIR)/

# Format all Go code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Run Go vet
.PHONY: vet
vet:
	$(GO) vet ./...

# Run all quality checks
.PHONY: check
check: fmt vet test

# Tidy go.mod and go.sum
.PHONY: tidy
tidy:
	$(GOMOD) tidy

# Build examples for all architectures (Linux, macOS, Windows)
.PHONY: release
release: init
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(FILE_COMP_BINARY)-linux-amd64 $(EXAMPLES_DIR)/file_compressor.go
	# Linux arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(FILE_COMP_BINARY)-linux-arm64 $(EXAMPLES_DIR)/file_compressor.go
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(FILE_COMP_BINARY)-darwin-amd64 $(EXAMPLES_DIR)/file_compressor.go
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(FILE_COMP_BINARY)-darwin-arm64 $(EXAMPLES_DIR)/file_compressor.go
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(FILE_COMP_BINARY)-windows-amd64.exe $(EXAMPLES_DIR)/file_compressor.go

# Default build target
.PHONY: build
build: build-example

# Help target
.PHONY: help
help:
	@echo "GoZ4X Makefile targets:"
	@echo "  all          - Default target, same as 'build'"
	@echo "  build        - Build the file compressor example"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run all tests"
	@echo "  bench        - Run benchmarks"
	@echo "  bench-profile - Run benchmarks with CPU and memory profiling"
	@echo "  fmt          - Format all Go code"
	@echo "  vet          - Run Go vet"
	@echo "  check        - Run all quality checks (fmt, vet, test)"
	@echo "  tidy         - Tidy go.mod and go.sum"
	@echo "  release      - Build examples for all architectures"
	@echo "  help         - Show this help message"

local_test:
	$(GO) test ./... -v