import { emit } from "brainkit";

export function run(): i32 {
  emit("as.test.raw", '{"msg":"raw"}');
  return 0;
}
