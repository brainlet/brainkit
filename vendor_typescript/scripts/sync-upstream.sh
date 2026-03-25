#!/usr/bin/env bash
# Re-sync vendored typescript-go from upstream.
# Usage: ./scripts/sync-upstream.sh /path/to/typescript-go-checkout
#
# This script copies the transpilation-relevant internal packages from
# microsoft/typescript-go, rewrites imports, and removes test files.
# After running, verify with: go mod tidy && go build ./internal/...
set -euo pipefail

SRC="${1:?Usage: $0 /path/to/typescript-go}"
DEST="$(cd "$(dirname "$0")/.." && pwd)/internal"

echo "Source: $SRC/internal"
echo "Dest:   $DEST"

# Remove old vendored code
rm -rf "$DEST"
mkdir -p "$DEST"

# Copy all internal packages
rsync -a "$SRC/internal/" "$DEST/"

# Copy locale data for diagnostics
rsync -a "$SRC/internal/diagnostics/loc/" "$DEST/diagnostics/loc/" 2>/dev/null || true

# Remove packages we don't need (IDE, LSP, project system, test infra, CLI)
for pkg in api bundled compiler execute format fourslash jsonrpc ls lsp project pprof repo testrunner testutil diagnosticwriter; do
  find "$DEST/$pkg" -type f -delete 2>/dev/null || true
  find "$DEST/$pkg" -type d -empty -delete 2>/dev/null || true
done

# Remove unneeded vfs sub-packages
for sub in vfsmock vfstest wrapvfs cachedvfs osvfs; do
  find "$DEST/vfs/$sub" -type f -delete 2>/dev/null || true
  find "$DEST/vfs/$sub" -type d -empty -delete 2>/dev/null || true
done

# Remove tsoptions test subpackage
find "$DEST/tsoptions/tsoptionstest" -type f -delete 2>/dev/null || true
find "$DEST/tsoptions/tsoptionstest" -type d -empty -delete 2>/dev/null || true

# Remove all test files
find "$DEST" -name '*_test.go' -delete

# Rewrite imports
find "$DEST" -name '*.go' -exec sed -i '' \
  's|github.com/microsoft/typescript-go/internal/|github.com/brainlet/brainkit/vendor_typescript/internal/|g' {} +

echo ""
echo "Sync complete."
echo "Files: $(find "$DEST" -name '*.go' | wc -l | tr -d ' ')"
echo ""
echo "Next steps:"
echo "  cd vendor_typescript && go mod tidy && go build ./internal/..."
