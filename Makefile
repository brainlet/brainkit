.PHONY: all brainkit install deps deps-go deps-npm build generate test test-v test-compile test-suite test-full bench bench-stable bench-runtime bench-save bench-check evals-save evals-check docs-bus-topics examples clean podman-init podman-start podman-up podman-launchd-up podman-launchd-down podman-link-socket podman-verify podman-down podman-status podman-reset podman-nuke podman-ensure type-check

PODMAN_MACHINE ?= brainkit
PODMAN_CPUS ?= 4
PODMAN_MEMORY ?= 8192
PODMAN_DISK ?= 60
PODMAN_SOCKET ?= /tmp/podman/$(PODMAN_MACHINE)-api.sock
PODMAN_LAUNCHD_LABEL ?= com.brainkit.podman.keepalive
PODMAN_LAUNCHD_LOG ?= /tmp/podman/$(PODMAN_MACHINE)-launchd-start.out

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

# Regenerate typed wrappers for SDK and module-owned message packages.
# Run after adding / renaming anything in sdk/*_messages.go or module message files.
generate:
	go run ./cmd/sdkgen -messages ./sdk -out ./sdk/typed_gen.go
	go run ./cmd/sdkgen -messages ./sdk/systemmsg -out ./sdk/systemmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/agents/agentmsg -out ./modules/agents/agentmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/audit/auditmsg -out ./modules/audit/auditmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/control -out ./modules/control/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/eval/evalmsg -out ./modules/eval/evalmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/gateway/gatewaymsg -out ./modules/gateway/gatewaymsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/health -out ./modules/health/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/messaging -out ./modules/messaging/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/metrics -out ./modules/metrics/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/mcp/mcpmsg -out ./modules/mcp/mcpmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/packages/packagemsg -out ./modules/packages/packagemsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/plugins/pluginmsg -out ./modules/plugins/pluginmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/reference/referencemsg -out ./modules/reference/referencemsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/registry/registrymsg -out ./modules/registry/registrymsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/schedules/schedulemsg -out ./modules/schedules/schedulemsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/secrets/secretmsg -out ./modules/secrets/secretmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/testing/testingmsg -out ./modules/testing/testingmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/topology -out ./modules/topology/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/tools/toolmsg -out ./modules/tools/toolmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/tracing/tracingmsg -out ./modules/tracing/tracingmsg/typed_gen.go
	go run ./cmd/sdkgen -messages ./modules/workflow/workflowmsg -out ./modules/workflow/workflowmsg/typed_gen.go

# Run the main behavior suite.
test: test-suite

test-suite: podman-ensure
	go test ./test/suite/... -count=1 -timeout=900s

test-compile:
	go test ./... -run '^$$' -timeout=600s

test-full: podman-ensure test-compile test-suite

# Run tests with verbose output
test-v: podman-ensure
	go test -v ./test/suite/... -count=1 -timeout=900s

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
# Podman machine lifecycle — dedicated brainkit VM.
#
# AppleHV helper processes can be reaped when started from short-lived agent
# shells. `podman-up` first tries the normal foreground start, then falls back
# to a launchd keepalive job that owns the VM process group outside the shell.
# ---------------------------------------------------------------------------

podman-init:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@if podman machine list --format '{{.Name}}' | sed 's/\*$$//' | grep -q '^$(PODMAN_MACHINE)$$'; then \
		echo "$(PODMAN_MACHINE) machine already exists (skipping init)"; \
	else \
		echo "Initializing $(PODMAN_MACHINE) podman machine ($(PODMAN_CPUS) CPU / $(PODMAN_MEMORY) MiB / $(PODMAN_DISK) GiB)..."; \
		podman machine init --cpus $(PODMAN_CPUS) --memory $(PODMAN_MEMORY) --disk-size $(PODMAN_DISK) $(PODMAN_MACHINE); \
		echo "$(PODMAN_MACHINE) machine initialized."; \
	fi

podman-start:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@state=$$(podman machine list --format '{{.Name}} {{.Running}}' | sed 's/\*//' | awk '$$1 == "$(PODMAN_MACHINE)" {print $$2}'); \
	if [ "$$state" = "true" ]; then \
		echo "$(PODMAN_MACHINE) machine already Running."; \
	else \
		other=$$(podman machine list --format '{{.Name}} {{.Running}}' | awk '$$2 == "true" {print $$1}' | sed 's/\*$$//'); \
		if [ -n "$$other" ]; then \
			echo "Stopping currently running machine '$$other' so $(PODMAN_MACHINE) can start..."; \
			podman machine stop "$$other"; \
		fi; \
		echo "Starting $(PODMAN_MACHINE) machine..."; \
		podman machine start $(PODMAN_MACHINE); \
	fi
	podman system connection default $(PODMAN_MACHINE)
	@podman --connection $(PODMAN_MACHINE) info >/dev/null 2>&1 || { echo "ERROR: $(PODMAN_MACHINE) socket unreachable after foreground start"; exit 1; }

podman-up: podman-init
	@$(MAKE) --no-print-directory podman-launchd-up || $(MAKE) --no-print-directory podman-start
	@$(MAKE) --no-print-directory podman-link-socket
	@$(MAKE) --no-print-directory podman-verify

podman-launchd-up:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@command -v launchctl >/dev/null 2>&1 || { echo "ERROR: launchctl binary not found"; exit 1; }
	@mkdir -p /tmp/podman
	@if launchctl list | grep -q '$(PODMAN_LAUNCHD_LABEL)' && podman --connection $(PODMAN_MACHINE) info >/dev/null 2>&1; then \
		echo "$(PODMAN_MACHINE) launchd machine already ready."; \
		podman system connection default $(PODMAN_MACHINE); \
		exit 0; \
	fi; \
	launchctl remove $(PODMAN_LAUNCHD_LABEL) >/dev/null 2>&1 || true; \
	other=$$(podman machine list --format '{{.Name}} {{.Running}}' | awk '$$2 == "true" {print $$1}' | sed 's/\*$$//'); \
	if [ -n "$$other" ]; then \
		for machine in $$other; do \
			echo "Stopping currently running machine '$$machine' so $(PODMAN_MACHINE) can start under launchd..."; \
			podman machine stop "$$machine" || true; \
		done; \
	fi; \
	echo "Starting $(PODMAN_MACHINE) under launchd keepalive $(PODMAN_LAUNCHD_LABEL)..."; \
	launchctl submit -l $(PODMAN_LAUNCHD_LABEL) -- /bin/zsh -lc 'podman machine start $(PODMAN_MACHINE) >$(PODMAN_LAUNCHD_LOG) 2>&1 || true; sleep 86400'; \
	i=0; \
	while [ $$i -lt 90 ]; do \
		if podman machine list --format '{{.Name}} {{.Running}}' | sed 's/\*//' | awk '$$1 == "$(PODMAN_MACHINE)" && $$2 == "true" {found=1} END {exit !found}'; then \
			if podman --connection $(PODMAN_MACHINE) info >/dev/null 2>&1; then \
				podman system connection default $(PODMAN_MACHINE); \
				echo "$(PODMAN_MACHINE) launchd machine ready."; \
				exit 0; \
			fi; \
		fi; \
		i=$$((i + 1)); \
		sleep 2; \
	done; \
	echo "ERROR: $(PODMAN_MACHINE) launchd start did not become reachable"; \
	tail -n 80 $(PODMAN_LAUNCHD_LOG) 2>/dev/null || true; \
	exit 1

podman-launchd-down:
	@command -v launchctl >/dev/null 2>&1 || { echo "ERROR: launchctl binary not found"; exit 1; }
	@launchctl remove $(PODMAN_LAUNCHD_LABEL) >/dev/null 2>&1 || true
	@echo "Removed launchd keepalive $(PODMAN_LAUNCHD_LABEL) if it existed."

podman-link-socket:
	@expected=$$(podman machine inspect $(PODMAN_MACHINE) --format '{{.ConnectionInfo.PodmanSocket.Path}}' 2>/dev/null || true); \
	if [ -z "$$expected" ]; then \
		echo "ERROR: cannot inspect $(PODMAN_MACHINE) socket path"; \
		exit 1; \
	fi; \
	if [ -S "$$expected" ] && curl --unix-socket "$$expected" -sS http://d/_ping 2>/dev/null | grep -q '^OK$$'; then \
		echo "$(PODMAN_MACHINE) socket ready at $$expected"; \
		exit 0; \
	fi; \
	actual=$$(awk -F"'" '/export DOCKER_HOST=/ {gsub(/^unix:\/\//, "", $$2); print $$2}' $(PODMAN_LAUNCHD_LOG) 2>/dev/null | tail -1); \
	if [ -n "$$actual" ] && [ -S "$$actual" ]; then \
		mkdir -p "$$(dirname "$$expected")"; \
		rm -f "$$expected"; \
		ln -s "$$actual" "$$expected"; \
		echo "Linked $$expected -> $$actual"; \
		exit 0; \
	fi; \
	echo "ERROR: expected socket $$expected is not reachable and no launchd socket was found"; \
	exit 1

podman-verify: podman-link-socket
	@podman system connection default $(PODMAN_MACHINE)
	@podman --connection $(PODMAN_MACHINE) info >/dev/null 2>&1 || { echo "ERROR: $(PODMAN_MACHINE) connection is not reachable"; exit 1; }
	@DOCKER_HOST=unix://$(PODMAN_SOCKET) podman ps >/dev/null 2>&1 || { echo "ERROR: $(PODMAN_SOCKET) does not accept Docker-compatible Podman calls"; exit 1; }
	@curl --unix-socket $(PODMAN_SOCKET) -sS http://d/_ping | grep -q '^OK$$' || { echo "ERROR: $(PODMAN_SOCKET) ping failed"; exit 1; }
	@echo "$(PODMAN_MACHINE) machine ready (default connection = $(PODMAN_MACHINE), socket = $(PODMAN_SOCKET))."

podman-down:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@launchctl remove $(PODMAN_LAUNCHD_LABEL) >/dev/null 2>&1 || true
	@state=$$(podman machine list --format '{{.Name}} {{.Running}}' | sed 's/\*//' | awk '$$1 == "$(PODMAN_MACHINE)" {print $$2}'); \
	if [ "$$state" = "true" ]; then \
		echo "Stopping $(PODMAN_MACHINE) machine..."; \
		podman machine stop $(PODMAN_MACHINE); \
	else \
		echo "$(PODMAN_MACHINE) machine not running (no-op)."; \
	fi
	@if [ -L "$(PODMAN_SOCKET)" ]; then rm -f "$(PODMAN_SOCKET)"; fi

podman-status:
	@command -v podman >/dev/null 2>&1 || { echo "ERROR: podman binary not found"; exit 1; }
	@echo "=== $(PODMAN_MACHINE) machine ==="
	@podman machine ls | grep -E 'NAME|^$(PODMAN_MACHINE)' || true
	@echo "=== default connection ==="
	@podman system connection list --format '{{.Name}} {{.Default}}' | grep -E 'Name|$(PODMAN_MACHINE)' || true
	@echo "=== launchd keepalive ==="
	@launchctl list | grep '$(PODMAN_LAUNCHD_LABEL)' || true
	@echo "=== socket ==="
	@ls -l "$(PODMAN_SOCKET)" 2>/dev/null || true

podman-reset:
	@if [ "$${CONFIRM}" != "1" ]; then \
		echo "ERROR: podman-reset requires CONFIRM=1 (this will destroy the $(PODMAN_MACHINE) machine)"; \
		exit 1; \
	fi
	@$(MAKE) --no-print-directory podman-down || true
	@podman machine rm -f $(PODMAN_MACHINE) || true
	@rm -f "$(PODMAN_SOCKET)" "$(PODMAN_LAUNCHD_LOG)"
	@$(MAKE) --no-print-directory podman-init
	@$(MAKE) --no-print-directory podman-up

podman-nuke: podman-reset

podman-ensure: podman-init podman-up

# Type-check gate for fixtures under fixtures/ts/** against internal/engine/runtime/*.d.ts.
# Uses the typescript@5.9.x pinned by the repo-root package.json (node_modules/.bin/tsc).
type-check: ## Run tsc --noEmit on all fixtures
	node_modules/.bin/tsc --noEmit -p fixtures/tsconfig.base.json
