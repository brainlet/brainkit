import { busSendRaw } from "wasm";

export function run(): i32 {
  busSendRaw("as.test.raw", '{"msg":"raw"}');
  return 0;
}
