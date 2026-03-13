.PHONY: deps deps-go deps-npm build test clean

# Install all dependencies (Go + npm)
deps: deps-go deps-npm

# Download Go module dependencies
deps-go:
	go mod download

# Install npm dependencies for all embed packages
deps-npm:
	cd ai-embed/bundle && npm install
	cd agent-embed/bundle && npm install
	cd as-embed/bundle && npm install

# Build all JS bundles
build:
	cd ai-embed/bundle && node build.mjs
	cd agent-embed/bundle && node build.mjs
	cd as-embed/bundle && node build.mjs

# Run all tests
test:
	go test ./jsbridge/... ./ai-embed/... ./agent-embed/... ./as-embed/... -timeout 120s

# Run tests with verbose output
test-v:
	go test -v ./jsbridge/... ./ai-embed/... ./agent-embed/... ./as-embed/... -timeout 120s

# Clean generated bundles and node_modules
clean:
	rm -rf ai-embed/bundle/node_modules agent-embed/bundle/node_modules as-embed/bundle/node_modules
	rm -f ai-embed/bundle/meta.json agent-embed/bundle/meta.json as-embed/bundle/meta.json
