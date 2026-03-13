// Entry point for the AssemblyScript compiler embedded bundle.
// Imports directly from the AS source tree (not dist/ which uses WebAssembly).
// Binaryen is marked external — our Go bridge provides it.
// Dependencies: as-float, long (needed by glue/js/ float and i64 modules).

// The AS source entry: loads JS glue, then re-exports compiler C-like API.
import "/Users/davidroman/Documents/code/clones/assemblyscript/src/index-js.ts";
import {
  newOptions,
  setTarget,
  setRuntime,
  setFeature,
  setOptimizeLevelHints,
  setSourceMap,
  setDebugInfo,
  setStackSize,
  DEFAULT_STACK_SIZE,
  newProgram,
  parse,
  nextFile,
  getDependee,
  initializeProgram,
  compile,
  validate,
  optimize,
  getBinaryenModuleRef,
  nextDiagnostic,
  getDiagnosticCode,
  getDiagnosticCategory,
  getDiagnosticMessage,
  formatDiagnostic,
  isError,
  isWarning,
  isInfo,
  FEATURES_DEFAULT,
  FEATURES_ALL,
} from "/Users/davidroman/Documents/code/clones/assemblyscript/src/index-wasm.ts";

// Target and Runtime enums live in std/assembly/shared/, not index-wasm.ts
import { Target } from "/Users/davidroman/Documents/code/clones/assemblyscript/std/assembly/shared/target.ts";
import { Runtime } from "/Users/davidroman/Documents/code/clones/assemblyscript/std/assembly/shared/runtime.ts";

globalThis.__as_compiler = {
  newOptions,
  setTarget,
  setRuntime,
  setFeature,
  setOptimizeLevelHints,
  setSourceMap,
  setDebugInfo,
  setStackSize,
  DEFAULT_STACK_SIZE,
  newProgram,
  parse,
  nextFile,
  getDependee,
  initializeProgram,
  compile,
  validate,
  optimize,
  getBinaryenModuleRef,
  nextDiagnostic,
  getDiagnosticCode,
  getDiagnosticCategory,
  getDiagnosticMessage,
  formatDiagnostic,
  isError,
  isWarning,
  isInfo,
  Target,
  Runtime,
  FEATURES_DEFAULT,
  FEATURES_ALL,
};
