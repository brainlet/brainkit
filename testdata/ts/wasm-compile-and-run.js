// Test: compile AssemblyScript to WASM and execute it
import { wasm, output } from "kit";

// Compile
const compiled = await wasm.compile(
  'export function run(): i32 { return 42; }'
);

// Run
const wasmResult = await wasm.run(compiled, {});

output({
  moduleId: compiled.moduleId,
  exitCode: wasmResult.exitCode,
});
