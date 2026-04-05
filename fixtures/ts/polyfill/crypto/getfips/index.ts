// Test: crypto.getFips, getHashes, getCiphers, timingSafeEqual
import { output } from "kit";

output({
  fips: crypto.getFips(),
  hashes: crypto.getHashes(),
  ciphersEmpty: crypto.getCiphers().length === 0,
  timingSafe: crypto.timingSafeEqual(Buffer.from("hello"), Buffer.from("hello")),
  timingSafeFalse: !crypto.timingSafeEqual(Buffer.from("hello"), Buffer.from("world")),
});
