// Test: compile AssemblyScript to WASM and execute it
import { compile } from "compiler";
import { output } from "kit";

// Compile
const compiled = await compile(
  'export function run(): i32 { return 42; }'
);

// Run
const wasmResult = await compiled.run({});

output({
  moduleId: compiled.moduleId,
  exitCode: wasmResult.exitCode,
});
