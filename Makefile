.PHONY: all brainkit install deps deps-go deps-npm build generate test test-v bench bench-stable bench-runtime docs-bus-topics examples clean

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

# Regenerate SDK wrappers (sdk/typed_gen.go + root call_gen.go).
# Run after adding / renaming anything in sdk/*_messages.go.
generate:
	go run ./cmd/sdkgen -messages ./sdk -out ./sdk/typed_gen.go -call-out ./call_gen.go

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

# Run runtime benchmarks (Kit construction, Call round trip, deploy, eval, bus).
bench-runtime:
	go test -run='^$$' -bench=. -benchmem -benchtime=1x ./test/bench/...

# Regenerate docs/bus-topics.md from sdk/*_messages.go.
docs-bus-topics:
	go run scripts/gen-bus-topics.go

# Smoke-check every example builds. Per-example modules (e.g.
# plugin-author) are handled via a subshell so their go.mod is
# picked up as a nested module.
examples:
	go build ./examples/ai-chat
	go build ./examples/cross-kit
	go build ./examples/hello-embedded
	go build ./examples/hello-server
	go build ./examples/multi-kit
	go build ./examples/observability
	go build ./examples/gateway-routes
	go build ./examples/go-tools
	go build ./examples/harness-lite
	go build ./examples/mcp
	go build ./examples/plugin-host
	go build ./examples/schedules
	go build ./examples/secrets
	go build ./examples/storage-vectors
	go build ./examples/streaming
	go build ./examples/workflows
	cd examples/plugin-author && go build .
	@echo "All examples build."

# Clean generated bundles, node_modules, and binaries
clean:
	rm -rf bin/
	rm -rf internal/embed/ai/bundle/node_modules internal/embed/agent/bundle/node_modules internal/embed/compiler/bundle/node_modules
	rm -f internal/embed/ai/bundle/meta.json internal/embed/agent/bundle/meta.json internal/embed/compiler/bundle/meta.json
