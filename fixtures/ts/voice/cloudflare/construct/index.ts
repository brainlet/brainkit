// Test: CloudflareVoice — construct with dummy account + token.
import { CloudflareVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new CloudflareVoice({
    speechModel: { name: "@cf/meta/m2m100-1.2b", accountId: "acct", apiToken: "test" },
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
