// Test: process.emitWarning, getuid/getgid, hrtime, nextTick
import { output } from "kit";

// emitWarning should not throw
process.emitWarning("test warning — should be silently ignored");

// POSIX
const uid = process.getuid();
const gid = process.getgid();

// hrtime
const hr = process.hrtime();

// nextTick
let ticked = false;
process.nextTick(() => { ticked = true; });
// nextTick uses queueMicrotask — runs before next await
await new Promise(r => setTimeout(r, 10));

output({
  emitWarningExists: typeof process.emitWarning === "function",
  uid: typeof uid === "number",
  gid: typeof gid === "number",
  hrtimeIsArray: Array.isArray(hr) && hr.length === 2,
  ticked,
  version: process.version,
  platform: process.platform,
});
