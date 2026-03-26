// Test: dns.lookup resolves hostnames via Go net.LookupHost
import { output } from "kit";

const dns = globalThis.__node_dns;

// Sync lookup
let syncResult: any = null;
dns.lookup("localhost", (err: any, addr: string, family: number) => {
  syncResult = { addr, family, err: err ? err.message : null };
});

// Promises lookup
const asyncResult = await dns.promises.lookup("localhost");

output({
  syncHasAddr: typeof syncResult?.addr === "string" && syncResult.addr.length > 0,
  syncFamily: syncResult?.family,
  asyncHasAddr: typeof asyncResult?.address === "string" && asyncResult.address.length > 0,
  asyncFamily: asyncResult?.family,
});
