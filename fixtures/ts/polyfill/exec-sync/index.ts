// Test: child_process.execSync/execFileSync/spawnSync via Go os/exec
import { output } from "kit";

const cp = globalThis.child_process;

// execSync
const echoResult = cp.execSync("echo hello-brainkit");
const echoStr = typeof echoResult === "string" ? echoResult : echoResult.toString("utf8");

// execFileSync
const fileResult = cp.execFileSync("echo", ["file-sync-test"]);
const fileStr = typeof fileResult === "string" ? fileResult : fileResult.toString("utf8");

// spawnSync
const spawnResult = cp.spawnSync("echo", ["spawn-sync-test"]);

// execSync with non-zero exit
let caughtError = false;
try {
  cp.execSync("exit 1");
} catch (e: any) {
  caughtError = true;
}

output({
  execSync: echoStr.trim() === "hello-brainkit",
  execFileSync: fileStr.trim() === "file-sync-test",
  spawnSyncStdout: spawnResult.stdout.trim() === "spawn-sync-test",
  spawnSyncStatus: spawnResult.status === 0,
  errorCaught: caughtError,
});
