// Test: os.release returns real kernel version, os.platform/arch/type work
import { output } from "kit";

output({
  release: os.release(),
  releaseNotStub: os.release() !== "0.0.0",
  platform: os.platform(),
  arch: os.arch(),
  type: os.type(),
  hostname: os.hostname(),
  hasCpus: os.cpus().length > 0,
  hasEOL: typeof os.EOL === "string",
});
