import * as esbuild from "esbuild";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const asRoot = path.join(dirname, "node_modules", "assemblyscript");

// Generate diagnosticMessages.generated.ts from diagnosticMessages.json.
// Mirrors the diagnosticsPlugin in assemblyscript/scripts/build.js.
const diagnostics = {
  name: "diagnostics",
  setup(build) {
    build.onResolve({ filter: /\bdiagnosticMessages\.generated$/ }, (args) => ({
      path: path.join(args.resolveDir, args.path),
      watchFiles: [path.join(asRoot, "src", "diagnosticMessages.json")],
    }));
    build.onLoad({ filter: /\bdiagnosticMessages\.generated$/ }, () => {
      const out = ["// GENERATED FILE. DO NOT EDIT.\n\n"];
      const messages = JSON.parse(
        fs.readFileSync(path.join(asRoot, "src", "diagnosticMessages.json"), "utf8")
      );

      out.push("/** Enum of available diagnostic codes. */\n");
      out.push("export enum DiagnosticCode {\n");

      let first = true;
      for (const text of Object.keys(messages)) {
        const key = text.replace(/[^\w]+/g, "_").replace(/_+$/, "");
        if (!first) out.push(",\n");
        first = false;
        out.push("  " + key + " = " + messages[text]);
      }

      out.push("\n}\n\n");
      out.push("/** Translates a diagnostic code to its respective string. */\n");
      out.push("export function diagnosticCodeToString(code: DiagnosticCode): string {\n  switch (code) {\n");

      for (const text of Object.keys(messages)) {
        out.push("    case " + messages[text] + ": return " + JSON.stringify(text) + ";\n");
      }

      out.push('    default: return "";\n  }\n}\n');
      return { contents: out.join(""), loader: "ts" };
    });
  },
};

// Stub modules that use WebAssembly or Node.js APIs (incompatible with QuickJS).
const stubs = {
  name: "stubs",
  setup(build) {
    // as-float uses WebAssembly.instantiateStreaming — provide pure JS fallback.
    build.onResolve({ filter: /^as-float$/ }, () => ({
      path: "as-float",
      namespace: "stub",
    }));
    build.onLoad({ filter: /^as-float$/, namespace: "stub" }, () => ({
      contents: "export const f64_pow = Math.pow;",
      loader: "js",
    }));

    // Node.js modules not available in QuickJS — empty stubs.
    const emptyModules = ["fs", "module", "crypto"];
    for (const mod of emptyModules) {
      build.onResolve({ filter: new RegExp(`^${mod}$`) }, () => ({
        path: mod,
        namespace: "stub",
      }));
    }
    build.onLoad({ filter: /.*/, namespace: "stub" }, (args) => {
      if (args.path === "as-float") {
        return { contents: "export const f64_pow = Math.pow;", loader: "js" };
      }
      if (args.path === "fs") {
        return { contents: "export default globalThis.fs || {}; export const readFileSync = () => null;", loader: "js" };
      }
      if (args.path === "crypto") {
        return { contents: "export default globalThis.__node_crypto || {};", loader: "js" };
      }
      return { contents: "export default {};", loader: "js" };
    });
  },
};

await esbuild.build({
  entryPoints: ["entry.mjs"],
  bundle: true,
  format: "iife",
  platform: "browser",
  target: "es2020",
  minify: true,
  treeShaking: true,
  external: ["binaryen"],
  plugins: [diagnostics, stubs],
  outfile: "../as_compiler_bundle.js",
});

console.log("Built as_compiler_bundle.js");
