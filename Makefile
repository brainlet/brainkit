.PHONY: all brainkit install deps deps-go deps-npm build generate test test-v bench bench-stable bench-runtime bench-save bench-check evals-save evals-check docs-bus-topics examples clean podman-init podman-up podman-down podman-status podman-reset podman-ensure

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
test: podman-ensure
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

# Capture the gated runtime benches to test/bench/latest.json — the
# artifact bench-check compares against baseline.json. `2>/dev/null`
# drops interleaved log lines from Kit SES init so the parser sees
# clean stdout.
bench-save:
	@mkdir -p test/bench
	go test -run='^$$' -benchmem -benchtime=5x \
		-bench='BenchmarkCall$$|BenchmarkCallParallel$$|BenchmarkEnvelopeEncode$$|BenchmarkEnvelopeDecode$$|BenchmarkEnvelopeRoundTrip$$|BenchmarkKitNew$$' \
		./test/bench/ 2>/dev/null > test/bench/latest.json
	@echo "Wrote test/bench/latest.json"

# Compare test/bench/latest.json against test/bench/baseline.json and
# fail if any non-skipped metric regresses beyond the baseline's
# tolerance_percent. Run `make bench-save && make bench-check`.
bench-check:
	go run scripts/bench-compare.go test/bench/baseline.json test/bench/latest.json

# Eval regression gate. `evals-save` captures latest averages
# into examples/evals/latest.json; `evals-check` compares to
# baseline.json and exits non-zero on a regression beyond
# tolerance_percent. Requires OPENAI_API_KEY.
evals-save:
	go run ./examples/evals -save

evals-check:
	go run ./examples/evals -check

# Regenerate docs/bus-topics.md from sdk/*_messages.go.
docs-bus-topics:
	go run scripts/gen-bus-topics.go

# Smoke-check every example builds. All binaries land under bin/
# so the repo root stays clean. Per-example modules (e.g.
# plugin-author) are handled via a subshell so their go.mod is
# picked up as a nested module; their binary also lands in
# ../../bin/ via a relative -o path.
examples:
	@mkdir -p bin
	go build -o bin/agent-forge     ./examples/agent-forge
	go build -o bin/agent-spawner   ./examples/agent-spawner
	go build -o bin/agent-stream    ./examples/agent-stream
	go build -o bin/ai-chat         ./examples/ai-chat
	go build -o bin/cross-kit       ./examples/cross-kit
	go build -o bin/custom-scorer   ./examples/custom-scorer
	go build -o bin/evals           ./examples/evals
	go build -o bin/hello-embedded  ./examples/hello-embedded
	go build -o bin/hello-server    ./examples/hello-server
	go build -o bin/multi-kit       ./examples/multi-kit
	go build -o bin/observability   ./examples/observability
	go build -o bin/package-workflow ./examples/package-workflow
	go build -o bin/gateway-routes  ./examples/gateway-routes
	go build -o bin/go-tools        ./examples/go-tools
	go build -o bin/guardrails      ./examples/guardrails
	go build -o bin/harness-lite    ./examples/harness-lite
	go build -o bin/hitl-tool-approval ./examples/hitl-tool-approval
	go build -o bin/hitl-workflow   ./examples/hitl-workflow
	go build -o bin/mcp             ./examples/mcp
	go build -o bin/plugin-host     ./examples/plugin-host
	go build -o bin/rag-pipeline    ./examples/rag-pipeline
	go build -o bin/schedules       ./examples/schedules
	go build -o bin/secrets         ./examples/secrets
	go build -o bin/storage-vectors ./examples/storage-vectors
	go build -o bin/streaming       ./examples/streaming
	go build -o bin/voice-agent     ./examples/voice-agent
	go build -o bin/voice-broadcast ./examples/voice-broadcast
	go build -o bin/voice-chat      ./examples/voice-chat
	go build -o bin/voice-realtime  ./examples/voice-realtime
	go build -o bin/workflows       ./examples/workflows
	go build -o bin/working-memory  ./examples/working-memory
	go build -o bin/workspace-agent ./examples/workspace-agent
	cd examples/plugin-author && go build -o ../../bin/plugin-author .
	@echo "All examples built into bin/"

# Clean generated bundles, node_modules, and binaries
clean:
	rm -rf bin/
	rm -rf internal/embed/ai/bundle/node_modules internal/embed/agent/bundle/node_modules internal/embed/compiler/bundle/node_modules
	rm -f internal/embed/ai/bundle/meta.json internal/embed/agent/bundle/meta.json internal/embed/compiler/bundle/meta.json

# ---------------------------------------------------------------------------
# Podman machine lifecycle — dedicated brainkit VM (4 CPU / 8 GiB / 60 GB)
# ---------------------------------------------------------------------------

podman-init:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@if podman machine list --format '{{.Name}}' | sed 's/\*$$//' | grep -q '^brainkit$$'; then \
		echo "brainkit machine already exists (skipping init)"; \
	else \
		echo "Initializing brainkit podman machine (4 CPU / 8 GiB / 60 GB)..."; \
		podman machine init --cpus 4 --memory 8192 --disk-size 60 brainkit; \
		echo "brainkit machine initialized."; \
	fi

podman-up:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@state=$$(podman machine list --format '{{.Name}} {{.Running}}' | sed 's/\*//' | awk '$$1 == "brainkit" {print $$2}'); \
	if [ "$$state" = "true" ]; then \
		echo "brainkit machine already Running."; \
	else \
		other=$$(podman machine list --format '{{.Name}} {{.Running}}' | awk '$$2 == "true" {print $$1}' | sed 's/\*$$//'); \
		if [ -n "$$other" ]; then \
			echo "Stopping currently running machine '$$other' so brainkit can start..."; \
			podman machine stop "$$other"; \
		fi; \
		echo "Starting brainkit machine..."; \
		podman machine start brainkit; \
	fi
	podman system connection default brainkit
	@podman --connection brainkit info >/dev/null 2>&1 || { echo "ERROR: brainkit socket unreachable"; exit 1; }
	@echo "brainkit machine ready (default connection = brainkit)."

podman-down:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@state=$$(podman machine list --format '{{.Name}} {{.Running}}' | sed 's/\*//' | awk '$$1 == "brainkit" {print $$2}'); \
	if [ "$$state" = "true" ]; then \
		echo "Stopping brainkit machine..."; \
		podman machine stop brainkit; \
	else \
		echo "brainkit machine not running (no-op)."; \
	fi

podman-status:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@echo "=== brainkit machine ==="
	@podman machine ls | grep -E 'NAME|^brainkit' || true
	@echo "=== default connection ==="
	@podman system connection list --format '{{.Name}} {{.Default}}' | grep -E 'Name|brainkit' || true

podman-reset:
	@if [ "$${CONFIRM}" != "1" ]; then \
		echo "ERROR: podman-reset requires CONFIRM=1 (this will destroy the brainkit machine)"; \
		exit 1; \
	fi
	$(MAKE) podman-down
	podman machine rm -f brainkit
	$(MAKE) podman-init
	$(MAKE) podman-up

podman-ensure: podman-init podman-up

# Type-check gate for fixtures under fixtures/ts/** against internal/engine/runtime/*.d.ts.
# Uses the typescript@5.9.x pinned by the repo-root package.json (node_modules/.bin/tsc).
.PHONY: type-check
type-check: ## Run tsc --noEmit on all fixtures
	node_modules/.bin/tsc --noEmit -p fixtures/tsconfig.base.json
