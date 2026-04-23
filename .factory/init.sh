#!/usr/bin/env bash
# Idempotent environment setup for brainkit type-alignment mission workers.
#
# Runs at the start of each worker session. Safe to re-run.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

log() { printf '[init.sh] %s\n' "$*"; }

# 1. Verify toolchain presence. Abort loudly if any is missing.
for tool in go node npm pnpm tsc git; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    log "MISSING TOOL: $tool"
    exit 1
  fi
done

log "go=$(go version | awk '{print $3}')  node=$(node --version)  npm=$(npm --version)  pnpm=$(pnpm --version)  tsc=$(tsc --version | awk '{print $2}')"

# 2. Ensure the two canonical clones exist. M0 worker is responsible for the initial
#    fetch + tag checkout; later workers just verify presence.
if [ ! -d /Users/davidroman/Documents/code/clones/mastra/.git ]; then
  log "FATAL: clones/mastra not a git repo"
  exit 1
fi
if [ ! -d /Users/davidroman/Documents/code/clones/ai/.git ]; then
  log "FATAL: clones/ai not a git repo"
  exit 1
fi

# 3. Ensure .env exists (workers use OPENAI_API_KEY for runtime fixtures).
if [ ! -f "$REPO_ROOT/.env" ]; then
  log "WARN: $REPO_ROOT/.env missing; AI-gated fixtures will skip"
fi

# 4. Ensure the bundle node_modules has been installed (M0 does this once).
#    Later workers may rely on the existing install; we do NOT run pnpm install
#    on every worker because it is slow and the mission is sequential.
BUNDLE_DIR="$REPO_ROOT/internal/embed/agent/bundle"
if [ -d "$BUNDLE_DIR/node_modules/@mastra/core" ]; then
  log "bundle node_modules: present"
else
  log "bundle node_modules: MISSING (expected to be created by M0 foundation worker)"
fi

# 5. Ensure root package.json (for tsc pinning) exists once M0 has landed.
if [ -f "$REPO_ROOT/package.json" ]; then
  log "root package.json: present"
else
  log "root package.json: MISSING (expected to be created by M0 foundation worker)"
fi

# 6. Quick Podman sanity check so AI/container fixture runs don't surprise us later.
#    Per user directive: keep this WARN-only (non-fatal).
if podman machine list --format '{{.Name}} {{.Running}}' 2>/dev/null | grep -q '^brainkit.*true'; then
  log "brainkit podman machine: running"
else
  log "WARN: brainkit podman machine not running; run make podman-ensure"
fi

log "init complete"
