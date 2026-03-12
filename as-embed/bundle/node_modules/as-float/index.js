import { MathWasmBase64 } from "./build/math.js";

export const {f64_pow} = (await WebAssembly.instantiateStreaming(fetch(MathWasmBase64))).instance.exports;
