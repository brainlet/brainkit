// "compiler" module — WASM compilation access from .ts code.
export async function compile(source, opts) {
  var raw = await (typeof __go_brainkit_request_async === "function"
    ? __go_brainkit_request_async("wasm.compile", JSON.stringify({ source: source, options: opts || {} }))
    : __go_brainkit_request("wasm.compile", JSON.stringify({ source: source, options: opts || {} })));
  var result = JSON.parse(raw);
  if (result && result.error) throw new Error("compiler: " + result.error);
  return result;
}
