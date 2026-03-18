import { log, debug, warn, error, logAt } from "wasm";

export function run(): i32 {
  log("info message");
  debug("debug message");
  warn("warn message");
  error("error message");
  logAt("custom level 0", 0);
  logAt("custom level 3", 3);
  return 0;
}
