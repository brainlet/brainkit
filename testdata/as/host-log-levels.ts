import { logAt, debug, warn, error } from "wasm";

export function run(): i32 {
  // logAt with explicit levels 0-3
  logAt("level 0 debug msg", 0);
  logAt("level 1 info msg", 1);
  logAt("level 2 warn msg", 2);
  logAt("level 3 error msg", 3);

  // Convenience functions
  debug("convenience debug");
  warn("convenience warn");
  error("convenience error");

  return 0;
}
