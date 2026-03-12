import * as esbuild from "esbuild";

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
      return {
        contents: "export default {}; export const readFileSync = () => null;",
        loader: "js",
      };
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
  plugins: [stubs],
  outfile: "../as_compiler_bundle.js",
});

console.log("Built as_compiler_bundle.js");
