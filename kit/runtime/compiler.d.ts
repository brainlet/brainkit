/**
 * "compiler" module — AssemblyScript WASM compilation from .ts code.
 *
 * @example
 * ```ts
 * import { compile } from "compiler";
 *
 * const result = await compile(`
 *   export function run(): i32 { return 42; }
 * `, { name: "my-module" });
 * console.log(result.moduleId, result.size, result.exports);
 * ```
 */
declare module "compiler" {
  /**
   * Compile AssemblyScript source to WASM.
   *
   * @param source - AssemblyScript source code
   * @param opts - Compilation options
   * @returns Compilation result with module ID and metadata
   */
  export function compile(source: string, opts?: CompileOptions): Promise<CompileResult>;

  interface CompileOptions {
    /** Module name (auto-generated if empty). */
    name?: string;
    /** AssemblyScript runtime type. */
    runtime?: string;
  }

  interface CompileResult {
    /** Module ID (same as name). */
    moduleId: string;
    /** Module name. */
    name: string;
    /** Compilation output text (warnings, etc). */
    text?: string;
    /** Binary size in bytes. */
    size: number;
    /** Exported function names. */
    exports: string[];
  }
}
