#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${EXAMPLES_SMOKE_LOG_DIR:-/tmp/brainkit-example-logs}"
DOCKER_HOST_VALUE="${DOCKER_HOST:-unix:///tmp/podman/brainkit-api.sock}"

mkdir -p "$LOG_DIR"
cd "$ROOT"

run_logged() {
	local name="$1"
	shift
	local log="$LOG_DIR/${name}.log"
	printf 'RUN %s\n' "$name"
	if "$@" >"$log" 2>&1; then
		printf 'PASS %s\n' "$name"
	else
		local rc=$?
		printf 'FAIL %s exit=%s log=%s\n' "$name" "$rc" "$log"
		tail -160 "$log" || true
		return "$rc"
	fi
}

wait_for_log() {
	local pid="$1"
	local log="$2"
	local pattern="$3"
	local attempts="${4:-120}"
	local sleep_s="${5:-0.5}"
	local i
	for ((i = 0; i < attempts; i++)); do
		if ! kill -0 "$pid" >/dev/null 2>&1; then
			cat "$log" || true
			return 1
		fi
		if grep -Eq "$pattern" "$log"; then
			return 0
		fi
		sleep "$sleep_s"
	done
	cat "$log" || true
	return 1
}

run_compile() {
	run_logged examples-build make examples
	run_logged examples-go-test go test ./examples/... -run '^$' -count=1 -timeout=600s
	run_logged plugin-author-go-test bash -c 'cd examples/plugin-author && go test ./... -run "^$" -count=1 -timeout=300s'
}

run_offline() {
	local examples=(
		hello-embedded
		artifact-runtime
		go-tools
		multi-kit
		cross-kit
		observability
		package-workflow
		plugin-host
		workflows
		schedules
		secrets
		hitl-workflow
		harness-lite
		mcp
	)
	local ex
	for ex in "${examples[@]}"; do
		run_logged "$ex" go run "./examples/$ex"
	done
	run_logged storage-vectors-sqlite env OPENAI_API_KEY= go run ./examples/storage-vectors
}

run_live() {
	run_logged ai-chat go run ./examples/ai-chat -prompt 'Reply with exactly: ok'
	run_logged custom-scorer go run ./examples/custom-scorer
	run_logged guardrails go run ./examples/guardrails
	run_logged hitl-tool-approval go run ./examples/hitl-tool-approval
	run_logged working-memory go run ./examples/working-memory
	run_logged workspace-agent go run ./examples/workspace-agent
	run_logged agent-spawner go run ./examples/agent-spawner \
		-request 'I need an agent that replies with one concise sentence about the topic. Name it tiny-topic-bot.' \
		-ask 'rainy mornings'
	run_logged agent-forge perl -e 'alarm shift @ARGV; exec @ARGV' 420 \
		go run ./examples/agent-forge \
		-request 'Build an agent that replies to any prompt with a concise friendly summary. Name it tiny-summary-bot.' \
		-ask 'Summarize why examples matter in software projects.' \
		-out /tmp/brainkit-forged-agents -keep=false
	run_logged evals perl -e 'alarm shift @ARGV; exec @ARGV' 720 go run ./examples/evals
	run_logged agent-stream bash -c '
		set -euo pipefail
		log="$1/agent-stream-server.log"
		go run ./examples/agent-stream >"$log" 2>&1 &
		pid=$!
		cleanup() { kill "$pid" >/dev/null 2>&1 || true; wait "$pid" >/dev/null 2>&1 || true; }
		trap cleanup EXIT
		addr=""
		for i in {1..240}; do
			kill -0 "$pid" >/dev/null 2>&1 || { cat "$log"; exit 1; }
			addr=$(sed -n "s/.*gateway listening address=\\([^ ]*\\).*/\\1/p" "$log" | tail -1)
			[ -z "$addr" ] && addr=$(sed -n "s/^.*listening on http:\\/\\///p" "$log" | tail -1)
			if [ -n "$addr" ] && grep -Eq "final plan|gateway SSE|Open a second shell|curl -N" "$log"; then
				break
			fi
			sleep 0.5
		done
		[ -n "$addr" ] || { cat "$log"; exit 1; }
		sse=$(curl -fsS -N --max-time 45 "http://$addr/sse/haiku?prompt=write+two+words")
		printf "%s" "$sse" | grep -q "data:"
	' bash "$LOG_DIR"
	run_logged voice-agent go run ./examples/voice-agent -play=false -out /tmp/brainkit-voice-agent-out -question 'Say hello in three words.'
	run_logged voice-broadcast go run ./examples/voice-broadcast
	run_logged voice-chat bash -c 'printf "Say hello in five words.\nexit\n" | go run ./examples/voice-chat'
}

run_server() {
	run_logged gateway-routes bash -c '
		set -euo pipefail
		log="$1/gateway-routes-server.log"
		go run ./examples/gateway-routes >"$log" 2>&1 &
		pid=$!
		cleanup() { kill "$pid" >/dev/null 2>&1 || true; wait "$pid" >/dev/null 2>&1 || true; }
		trap cleanup EXIT
		addr=""
		for i in {1..80}; do
			kill -0 "$pid" >/dev/null 2>&1 || { cat "$log"; exit 1; }
			addr=$(sed -n "s/^listening on http:\\/\\///p" "$log" | tail -1)
			[ -n "$addr" ] && break
			sleep 0.25
		done
		[ -n "$addr" ] || { cat "$log"; exit 1; }
		resp=$(curl -fsS --max-time 10 "http://$addr/hello?name=world")
		printf "%s" "$resp" | grep -q "\"greeting\":\"hello, world\""
	' bash "$LOG_DIR"

	run_logged streaming bash -c '
		set -euo pipefail
		log="$1/streaming-server.log"
		go run ./examples/streaming >"$log" 2>&1 &
		pid=$!
		cleanup() { kill "$pid" >/dev/null 2>&1 || true; wait "$pid" >/dev/null 2>&1 || true; }
		trap cleanup EXIT
		addr=""
		for i in {1..100}; do
			kill -0 "$pid" >/dev/null 2>&1 || { cat "$log"; exit 1; }
			addr=$(sed -n "s/^listening on http:\\/\\///p" "$log" | tail -1)
			if [ -n "$addr" ] && grep -q "terminal: done=true total=5" "$log"; then
				break
			fi
			sleep 0.25
		done
		[ -n "$addr" ] || { cat "$log"; exit 1; }
		sse=$(curl -fsS -N --max-time 10 "http://$addr/sse/count?n=2")
		webhook=$(curl -fsS --max-time 10 -X POST "http://$addr/webhook/log" -d "{\"msg\":\"hi\"}")
		printf "%s" "$sse" | grep -q "tick"
		printf "%s" "$webhook" | grep -q "\"ok\":true"
	' bash "$LOG_DIR"

	run_logged hello-server bash -c '
		set -euo pipefail
		log="$1/hello-server.log"
		cfg=$(mktemp)
		sed "s/listen: :8080/listen: 127.0.0.1:18080/" examples/hello-server/brainkit.yaml >"$cfg"
		go run ./examples/hello-server -config "$cfg" >"$log" 2>&1 &
		pid=$!
		cleanup() { kill "$pid" >/dev/null 2>&1 || true; wait "$pid" >/dev/null 2>&1 || true; rm -f "$cfg"; }
		trap cleanup EXIT
		for i in {1..100}; do
			kill -0 "$pid" >/dev/null 2>&1 || { cat "$log"; exit 1; }
			grep -q "gateway listening" "$log" && break
			sleep 0.25
		done
		health=$(curl -fsS --max-time 10 "http://127.0.0.1:18080/health")
		printf "%s" "$health" | grep -q "\"healthy\":true"
	' bash "$LOG_DIR"

	run_logged voice-realtime bash -c '
		set -euo pipefail
		log="$1/voice-realtime-server.log"
		go run ./examples/voice-realtime -addr 127.0.0.1:18787 >"$log" 2>&1 &
		pid=$!
		cleanup() { kill "$pid" >/dev/null 2>&1 || true; wait "$pid" >/dev/null 2>&1 || true; }
		trap cleanup EXIT
		for i in {1..120}; do
			kill -0 "$pid" >/dev/null 2>&1 || { cat "$log"; exit 1; }
			grep -q "open http://127.0.0.1:18787" "$log" && break
			sleep 0.5
		done
		page=$(curl -fsS --max-time 10 "http://127.0.0.1:18787/")
		printf "%s" "$page" | grep -Eq "voice|mic|ws/voice"
	' bash "$LOG_DIR"
}

compose_up() {
	local file="$1"
	DOCKER_HOST="$DOCKER_HOST_VALUE" podman compose -f "$file" up -d
}

compose_down() {
	local file="$1"
	DOCKER_HOST="$DOCKER_HOST_VALUE" podman compose -f "$file" down -v
}

wait_container_healthy() {
	local name="$1"
	local i
	for ((i = 0; i < 60; i++)); do
		local health_state
		health_state="$(DOCKER_HOST="$DOCKER_HOST_VALUE" podman inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
		if [ "$health_state" = "healthy" ]; then
			return 0
		fi
		sleep 2
	done
	DOCKER_HOST="$DOCKER_HOST_VALUE" podman logs "$name" --tail 80 || true
	return 1
}

run_external() {
	command -v podman >/dev/null 2>&1 || { echo "podman is required for external examples"; exit 1; }

	compose_up examples/rag-pipeline/docker-compose.yml
	trap 'compose_down examples/rag-pipeline/docker-compose.yml >/dev/null 2>&1 || true' EXIT
	wait_container_healthy brainkit-pgvector-rag
	run_logged rag-pipeline env PGVECTOR_URL='postgres://brainkit:brainkit@127.0.0.1:5434/brainkit?sslmode=disable' go run ./examples/rag-pipeline
	compose_down examples/rag-pipeline/docker-compose.yml
	trap - EXIT

	compose_up examples/storage-vectors/docker-compose.yml
	trap 'compose_down examples/storage-vectors/docker-compose.yml >/dev/null 2>&1 || true' EXIT
	wait_container_healthy brainkit-pgvector-demo
	run_logged storage-vectors-pgvector env PGVECTOR_URL='postgres://brainkit:brainkit@127.0.0.1:5433/brainkit?sslmode=disable' go run ./examples/storage-vectors
	compose_down examples/storage-vectors/docker-compose.yml
	trap - EXIT
}

usage() {
	cat <<'USAGE'
Usage: scripts/examples-smoke.sh <tier> [<tier>...]

Tiers:
  compile   build all examples and compile-test example packages
  offline   run examples that do not require provider credentials
  live      run OpenAI/provider-backed examples using .env or process env
  server    run long-running server examples with HTTP probes
  external  run pgvector-backed examples with temporary Podman compose stacks
  default   compile + offline + server
  all       compile + offline + live + server + external
USAGE
}

if [ "$#" -eq 0 ]; then
	set -- default
fi

for tier in "$@"; do
	case "$tier" in
		compile) run_compile ;;
		offline) run_offline ;;
		live) run_live ;;
		server) run_server ;;
		external) run_external ;;
		default)
			run_compile
			run_offline
			run_server
			;;
		all)
			run_compile
			run_offline
			run_live
			run_server
			run_external
			;;
		-h|--help|help) usage ;;
		*) usage; exit 2 ;;
	esac
done
