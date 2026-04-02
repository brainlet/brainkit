#!/bin/bash
# Concatenates runtime/wasm/*.ts files into runtime/wasm_bundle.ts
# Order matters: host.ts first (declares @external), json.ts second (classes used by all),
# types.ts third, then domain files, then shard/state/log, index.ts last.
#
# Import lines (import { ... } from "./...") are stripped since everything
# is in one file after concatenation.

set -e
cd "$(dirname "$0")/.."

WASM_DIR="runtime/wasm"
OUT="runtime/wasm_bundle.ts"

# Infra files in dependency order
INFRA_FILES=(
    "$WASM_DIR/host.ts"
    "$WASM_DIR/json.ts"
    "$WASM_DIR/types.ts"
    "$WASM_DIR/log.ts"
    "$WASM_DIR/state.ts"
    "$WASM_DIR/shard.ts"
)

# Generated domain files — dynamically discovered from codegen output
GENERATED_FILES=($(ls "$WASM_DIR/generated/"*.ts 2>/dev/null | sort))

# Index file last (re-exports everything)
INDEX_FILE="$WASM_DIR/index.ts"

# Combine all in order
FILES=("${INFRA_FILES[@]}" "${GENERATED_FILES[@]}" "$INDEX_FILE")

echo "// AUTO-GENERATED — do not edit. Run scripts/bundle_wasm.sh to regenerate." > "$OUT"
echo "// Source files: ${#FILES[@]} files from runtime/wasm/" >> "$OUT"
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
