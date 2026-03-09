// Package module wraps the binaryen package with AssemblyScript-specific
// conveniences: size-aware unary/binary op dispatch, shadow-stack integration,
// and custom optimization pass ordering.
//
// This is a faithful 1:1 port of assemblyscript/src/module.ts (4,009 lines).
//
// # Architecture
//
// The module package sits between the compiler and the low-level binaryen
// CGo bindings (pkg/binaryen). No CGo is used in this package; all Binaryen
// C API calls are delegated to the binaryen package.
//
// # Intentionally Omitted from the TS Source
//
// The following items from module.ts are intentionally NOT ported because
// they are TypeScript/WASM-specific implementation details that have no
// equivalent or are unnecessary in Go:
//
//   - allocStringCached / readStringCached (TS lines 2861-2882):
//     The TS source caches string pointers allocated in WASM linear memory
//     to avoid repeated allocation/free cycles when passing strings to
//     Binaryen. In Go, the binaryen package's StringPool handles C string
//     caching internally, and Go's garbage collector manages string lifetime.
//
//   - lit buffer (TS line 1326-1329):
//     The TS Module constructor allocates a literal buffer in WASM memory
//     via _BinaryenSizeofLiteral() for passing literal values to Binaryen
//     const constructors. In Go, the binaryen package calls the C API
//     directly (e.g., BinaryenConst) and handles literal construction
//     internally without exposing a raw memory buffer.
//
//   - Internal WASM memory helpers (TS lines 3523-3694):
//     allocU8Array, allocI32Array, allocU32Array, allocPtrArray,
//     stringLengthUTF8, allocString, readBuffer, readString, and the
//     BinaryModule class for raw binary output — all TS-specific helpers
//     for managing data in the AssemblyScript WASM heap. In Go, the CGo
//     bridge in pkg/binaryen handles all C memory allocation and
//     conversion directly.
//
//   - ensureType, tryEnsureBasicType, determinePackedType, prepareType
//     (TS lines 3696-4009):
//     These type builder bridge functions require program.ElementKind,
//     program.PropertyPrototype, and other compiler types. They will be
//     placed in the compiler package when it is ported, not in the module
//     package, to avoid circular dependencies between module and program.
package module
