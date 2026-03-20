#!/bin/bash
# Downloads and builds binaryen into as-embed/deps/binaryen/.
# Parallel to `npm install` for the JS bundle dependencies.
#
# Usage: ./scripts/setup-binaryen.sh
set -e

BINARYEN_VERSION="version_123"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPS_DIR="$SCRIPT_DIR/../deps"
INSTALL_DIR="$DEPS_DIR/binaryen"

if [ -f "$INSTALL_DIR/lib/libbinaryen.a" ]; then
  echo "binaryen already built at $INSTALL_DIR"
  exit 0
fi

echo "Building binaryen $BINARYEN_VERSION..."
mkdir -p "$DEPS_DIR"

# Clone (shallow) into a temp build directory
BUILD_DIR="$DEPS_DIR/binaryen-build"
rm -rf "$BUILD_DIR"
git clone --depth 1 --branch "$BINARYEN_VERSION" \
  https://github.com/WebAssembly/binaryen.git "$BUILD_DIR"

# Build and install to deps/binaryen/
cd "$BUILD_DIR"
cmake -B build \
  -DCMAKE_INSTALL_PREFIX="$INSTALL_DIR" \
  -DBUILD_TESTS=OFF \
  -DBUILD_TOOLS=OFF \
  -DBUILD_STATIC_LIB=ON \
  -DCMAKE_BUILD_TYPE=Release
cmake --build build -j"$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)"
cmake --install build

# Clean up build directory
rm -rf "$BUILD_DIR"

echo "binaryen installed to $INSTALL_DIR"
echo "  headers: $INSTALL_DIR/include/binaryen-c.h"
echo "  library: $INSTALL_DIR/lib/libbinaryen.a"
