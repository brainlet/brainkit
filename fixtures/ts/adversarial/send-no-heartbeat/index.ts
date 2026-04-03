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

// Trigger and wait for reply
const pr = bus.sendTo("send-no-heartbeat.ts", "send-test", { go: true });
const replySub = bus.subscribe(pr.replyTo, function(reply: any) {
  // We should get the intermediate chunks and the final reply
  if (reply.payload && reply.payload.final) {
    results.gotFinalReply = true;
    results.chunks = reply.payload.chunks;
    bus.unsubscribe(replySub);
    output(results);
  }
});
