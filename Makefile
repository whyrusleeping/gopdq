# PDQ Hash Implementation Makefile
# Provides tools for building, testing, and comparing PDQ hash implementations

.PHONY: help setup build test benchmark compare clean install-deps check-deps

# Default target
help:
	@echo "PDQ Hash Implementation - Available Commands:"
	@echo ""
	@echo "Setup and Dependencies:"
	@echo "  make setup          - Clone and build Facebook's PDQ implementation"
	@echo "  make install-deps   - Install required system dependencies"
	@echo "  make check-deps     - Check if all dependencies are installed"
	@echo ""
	@echo "Building and Testing:"
	@echo "  make build          - Build the Go PDQ implementation"
	@echo "  make test           - Run Go unit tests"
	@echo "  make benchmark      - Run synthetic image benchmark"
	@echo ""
	@echo "Comparison:"
	@echo "  make compare IMAGE=<path>  - Compare Go vs Facebook implementation"
	@echo "  make compare-dir DIR=<path> - Compare on directory of images"
	@echo "  make validate       - Run validation on test images"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean          - Clean build artifacts and temporary files"
	@echo ""
	@echo "Examples:"
	@echo "  make setup"
	@echo "  make compare IMAGE=test.jpg"
	@echo "  make compare-dir DIR=./test_images/"

# Variables
FACEBOOK_PDQ_DIR = facebook-pdq
FACEBOOK_HASHER = $(FACEBOOK_PDQ_DIR)/pdq/cpp/pdq-photo-hasher
GO_MODULE = github.com/why/gopdq
BENCHMARK_DIR = benchmark

# Check if required dependencies are installed
check-deps:
	@echo "Checking dependencies..."
	@which git >/dev/null || (echo "❌ git not found. Please install git." && exit 1)
	@which go >/dev/null || (echo "❌ Go not found. Please install Go 1.19+." && exit 1)
	@which g++ >/dev/null || (echo "❌ g++ not found. Please install build-essential." && exit 1)
	@which make >/dev/null || (echo "❌ make not found. Please install build-essential." && exit 1)
	@echo "✅ All dependencies found"

# Install system dependencies (Ubuntu/Debian)
install-deps:
	@echo "Installing system dependencies..."
	sudo apt-get update
	sudo apt-get install -y build-essential git imagemagick
	@echo "✅ System dependencies installed"
	@echo "Note: Go must be installed separately. Visit https://golang.org/dl/"

# Setup: Clone and build Facebook's PDQ implementation
setup: check-deps
	@echo "Setting up Facebook's PDQ implementation..."
	@if [ ! -d "$(FACEBOOK_PDQ_DIR)" ]; then \
		echo "Cloning Facebook's ThreatExchange repository..."; \
		git clone https://github.com/facebook/ThreatExchange.git $(FACEBOOK_PDQ_DIR); \
	else \
		echo "Facebook PDQ repository already exists"; \
	fi
	@echo "Building Facebook's PDQ hasher..."
	@cd $(FACEBOOK_PDQ_DIR)/pdq/cpp && make pdq-photo-hasher
	@echo "✅ Facebook PDQ implementation ready"

# Build the Go implementation
build:
	@echo "Building Go PDQ implementation..."
	@go build -o $(BENCHMARK_DIR)/pdq-benchmark $(BENCHMARK_DIR)/main.go
	@go build -o $(BENCHMARK_DIR)/pdq-compare cmd/compare/main.go
	@echo "✅ Go implementation built"

# Run Go unit tests
test:
	@echo "Running Go unit tests..."
	@go test -v ./...
	@echo "✅ Tests completed"

# Run synthetic benchmark
benchmark: build
	@echo "Running synthetic benchmark..."
	@cd $(BENCHMARK_DIR) && ./pdq-benchmark
	@echo "✅ Benchmark completed"

# Compare single image
compare: 
	@if [ -z "$(IMAGE)" ]; then \
		echo "❌ Please specify IMAGE=<path>"; \
		echo "Example: make compare IMAGE=test.jpg"; \
		exit 1; \
	fi
	@if [ ! -f "$(FACEBOOK_HASHER)" ]; then \
		echo "❌ Facebook hasher not found. Run 'make setup' first."; \
		exit 1; \
	fi
	@echo "Comparing implementations on: $(IMAGE)"
	@cd $(BENCHMARK_DIR) && go run ../cmd/compare/main.go "$(IMAGE)"

# Compare directory of images
compare-dir:
	@if [ -z "$(DIR)" ]; then \
		echo "❌ Please specify DIR=<path>"; \
		echo "Example: make compare-dir DIR=./test_images/"; \
		exit 1; \
	fi
	@if [ ! -f "$(FACEBOOK_HASHER)" ]; then \
		echo "❌ Facebook hasher not found. Run 'make setup' first."; \
		exit 1; \
	fi
	@echo "Comparing implementations on directory: $(DIR)"
	@cd $(BENCHMARK_DIR) && go run ../cmd/compare/main.go "$(DIR)"

# Validate with known test images from Facebook's test suite
validate:
	@if [ ! -f "$(FACEBOOK_HASHER)" ]; then \
		echo "❌ Facebook hasher not found. Run 'make setup' first."; \
		exit 1; \
	fi
	@if [ -d "$(FACEBOOK_PDQ_DIR)/pdq/data/reg-test-input" ]; then \
		echo "Running validation on Facebook's test images..."; \
		cd $(BENCHMARK_DIR) && go run ../cmd/compare/main.go "../$(FACEBOOK_PDQ_DIR)/pdq/data/reg-test-input"; \
	else \
		echo "⚠️  Facebook test images not found. Using synthetic benchmark..."; \
		$(MAKE) benchmark; \
	fi

# Clean build artifacts and temporary files
clean:
	@echo "Cleaning up..."
	@rm -f $(BENCHMARK_DIR)/pdq-benchmark $(BENCHMARK_DIR)/pdq-compare
	@rm -rf $(BENCHMARK_DIR)/benchmark_images/
	@rm -f $(BENCHMARK_DIR)/*.png $(BENCHMARK_DIR)/*.jpg
	@go clean
	@echo "✅ Cleanup completed"

# Deep clean including Facebook's PDQ
clean-all: clean
	@echo "Removing Facebook PDQ implementation..."
	@rm -rf $(FACEBOOK_PDQ_DIR)
	@echo "✅ Deep cleanup completed"

# Development helpers
fmt:
	@go fmt ./...

vet:
	@go vet ./...

mod-tidy:
	@go mod tidy

# Show current status
status:
	@echo "PDQ Implementation Status:"
	@echo "========================="
	@echo -n "Go module: "
	@if go list $(GO_MODULE) >/dev/null 2>&1; then echo "✅ Ready"; else echo "❌ Not found"; fi
	@echo -n "Facebook hasher: "
	@if [ -f "$(FACEBOOK_HASHER)" ]; then echo "✅ Ready"; else echo "❌ Not built (run 'make setup')"; fi
	@echo -n "ImageMagick: "
	@if which convert >/dev/null 2>&1; then echo "✅ Ready"; else echo "❌ Not installed"; fi