// Test: crypto.getFips, getHashes, getCiphers, timingSafeEqual
import { output } from "kit";

const C = globalThis.__node_crypto;

output({
  fips: C.getFips(),
  hashes: C.getHashes(),
  ciphersEmpty: C.getCiphers().length === 0,
  timingSafe: C.timingSafeEqual(
    Buffer.from("hello"),
    Buffer.from("hello")
  ),
  timingSafeFalse: !C.timingSafeEqual(
    Buffer.from("hello"),
    Buffer.from("world")
  ),
});
