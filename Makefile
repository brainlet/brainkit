.PHONY: all brainkit install deps deps-go deps-npm build test test-v bench bench-stable clean

# Default: build the CLI binary
all: brainkit

# Build the CLI binary to bin/
brainkit:
	@mkdir -p bin
	go build -o bin/brainkit ./cmd/brainkit/
	@echo "Built bin/brainkit"

# Install to /usr/local/bin (requires sudo)
install: brainkit
	sudo cp bin/brainkit /usr/local/bin/brainkit
	@echo "Installed to /usr/local/bin/brainkit"

# Install all dependencies (Go + npm)
deps: deps-go deps-npm

# Download Go module dependencies
deps-go:
	go mod download

# Install npm dependencies for all embed packages
deps-npm:
	cd internal/embed/ai/bundle && npm install
	cd internal/embed/agent/bundle && npm install
	cd internal/embed/compiler/bundle && npm install

# Build all JS bundles
build:
	cd internal/embed/ai/bundle && node build.mjs
	cd internal/embed/agent/bundle && node build.mjs
	cd internal/embed/compiler/bundle && node build.mjs

# Run all tests
test:
	go test ./test/suite/... -timeout 600s

# Run tests with verbose output
test-v:
	go test -v ./test/suite/... -timeout 600s

# Run as-embed benchmarks (compilation performance)
bench:
	cd internal/embed/compiler && go test -run='^$$' -bench=. -benchmem -benchtime=1x -timeout 10m

# Run as-embed benchmarks with 3 iterations for stable numbers
bench-stable:
	cd internal/embed/compiler && go test -run='^$$' -bench=. -benchmem -benchtime=3x -timeout 30m

# Clean generated bundles, node_modules, and binaries
clean:
	rm -rf bin/
	rm -rf internal/embed/ai/bundle/node_modules internal/embed/agent/bundle/node_modules internal/embed/compiler/bundle/node_modules
	rm -f internal/embed/ai/bundle/meta.json internal/embed/agent/bundle/meta.json internal/embed/compiler/bundle/meta.json
