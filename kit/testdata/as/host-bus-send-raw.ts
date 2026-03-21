import { bus } from "brainkit";

export function run(): i32 {
  bus.sendRaw("as.test.raw", '{"msg":"raw"}');
  return 0;
}
