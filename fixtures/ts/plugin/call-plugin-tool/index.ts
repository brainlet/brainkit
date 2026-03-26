// Test: call a tool provided by a plugin subprocess
// The test runner starts the testplugin which registers "echo" and "concat" tools.
import { tools, output } from "kit";

try {
  // Call the plugin's echo tool
  const echoResult = await tools.call("echo", { message: "from-fixture" });

  // Call the plugin's concat tool
  const concatResult = await tools.call("concat", { a: "brain", b: "kit" });

  output({
    echoed: (echoResult as any)?.echoed || "",
    concatenated: (concatResult as any)?.result || "",
    hasEcho: echoResult !== null,
    hasConcat: concatResult !== null,
  });
} catch (e: any) {
  output({ error: e.message });
}
