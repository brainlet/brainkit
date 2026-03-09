// Package compiler implements the AssemblyScript-to-WebAssembly compiler.
//
// This is a faithful 1:1 port of assemblyscript/src/compiler.ts (10,687 lines).
//
// # Architecture
//
// The compiler package bridges the program package (AST elements, type resolution)
// and the module package (Binaryen IR construction). It walks the typed AST and
// emits WebAssembly instructions via the module's expression builder methods.
//
// The Compiler struct embeds DiagnosticEmitter for error reporting and holds
// compilation state (current flow, memory layout, function table, etc.).
//
// # Key Types
//
//   - Options: Compiler configuration (target, features, memory settings, etc.)
//   - Compiler: The main compiler that drives compilation from Program to Module.
//
// # Compilation Flow
//
//  1. Compiler.Compile(program) creates a new Compiler and calls compile()
//  2. compile() initializes the module, compiles entry files, resolves lazy
//     functions and override stubs, finalizes memory/table/start function
//  3. Individual elements are compiled via compileGlobal, compileEnum,
//     compileFunction, compileStatement, compileExpression, etc.
//
// # ensureType / prepareType
//
// The ensureType and prepareType bridge functions (TS lines 3696-4009) that
// convert program types to Binaryen type references are placed here in the
// compiler package (not in module/) to avoid circular dependencies between
// module and program.
package compiler
