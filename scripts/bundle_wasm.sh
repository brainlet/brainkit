#!/bin/bash
# Concatenates kit/runtime/wasm/*.ts files into kit/runtime/wasm_bundle.ts
# Order matters: host.ts first (declares @external), json.ts second (classes used by all),
# types.ts third, then domain files, then shard/state/log/bus, index.ts last.
#
# Import lines (import { ... } from "./...") are stripped since everything
# is in one file after concatenation.

set -e
cd "$(dirname "$0")/.."

WASM_DIR="kit/runtime/wasm"
OUT="kit/runtime/wasm_bundle.ts"

# File order (dependency-safe)
# Infra files first (host declares @external, json provides JSONObject/JSONValue),
# then generated domain files (complete, from codegen/wasmgen), index last.
FILES=(
    "$WASM_DIR/host.ts"
    "$WASM_DIR/json.ts"
    "$WASM_DIR/types.ts"
    "$WASM_DIR/log.ts"
    "$WASM_DIR/state.ts"
    "$WASM_DIR/shard.ts"
    "$WASM_DIR/generated/agents.ts"
    "$WASM_DIR/generated/fs.ts"
    "$WASM_DIR/generated/kit.ts"
    "$WASM_DIR/generated/mcp.ts"
    "$WASM_DIR/generated/plugin.ts"
    "$WASM_DIR/generated/registry.ts"
    "$WASM_DIR/generated/tools.ts"
    "$WASM_DIR/generated/wasm.ts"
    "$WASM_DIR/index.ts"
)

echo "// AUTO-GENERATED — do not edit. Run scripts/bundle_wasm.sh to regenerate." > "$OUT"
echo "// Source files: ${#FILES[@]} files from kit/runtime/wasm/" >> "$OUT"
echo "" >> "$OUT"

for f in "${FILES[@]}"; do
    if [ ! -f "$f" ]; then
        echo "ERROR: $f not found" >&2
        exit 1
    fi
    echo "// ════════════════════════════════════════════════════════════" >> "$OUT"
    echo "// Source: $f" >> "$OUT"
    echo "// ════════════════════════════════════════════════════════════" >> "$OUT"
    echo "" >> "$OUT"
    # Strip import lines (import { ... } from "./...")
    grep -v '^import {.*} from "\.\/' "$f" >> "$OUT"
    echo "" >> "$OUT"
done

echo "Bundle generated: $OUT ($(wc -l < "$OUT") lines)"
