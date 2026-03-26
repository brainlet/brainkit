// Test: Buffer.poolSize, Buffer.isEncoding, Buffer.byteLength, Buffer.compare
import { output } from "kit";

output({
  poolSize: Buffer.poolSize,
  isEncodingUtf8: Buffer.isEncoding("utf8"),
  isEncodingHex: Buffer.isEncoding("hex"),
  isEncodingFake: !Buffer.isEncoding("fake-encoding"),
  byteLengthStr: Buffer.byteLength("hello"),
  byteLengthBase64: Buffer.byteLength("aGVsbG8=", "base64"),
  compareEqual: Buffer.compare(Buffer.from("abc"), Buffer.from("abc")) === 0,
  compareLess: Buffer.compare(Buffer.from("abc"), Buffer.from("def")) < 0,
});
