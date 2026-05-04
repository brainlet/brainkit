import { bus, output } from "kit";

// msg.send() uses done=false but does NOT have a "type" field in the payload.
// The heartbeat discriminator should NOT start a heartbeat goroutine for it.
// msg.stream.text() DOES have "type" in the payload — heartbeat should start.
//
// We can't observe the heartbeat goroutine directly from JS, but we can
// verify that msg.send() works correctly without interference from heartbeat.

const results: Record<string, any> = {};

bus.on("send-test", function(msg: any) {
  // Use msg.send() (intermediate chunk) then msg.reply() (final)
  msg.send({ chunk: 1 });
  msg.send({ chunk: 2 });
  msg.reply({ final: true, chunks: 2 });
});

// Trigger and wait for the final reply. Go-level stream tests cover chunk
// delivery; this fixture verifies msg.send() does not prevent completion.
const reply: any = await bus.callService("send-no-heartbeat-adv.ts", "send-test", { go: true }, { timeoutMs: 5000 });
results.gotFinalReply = !!reply?.final;
results.chunks = reply?.chunks || 0;
output(results);
