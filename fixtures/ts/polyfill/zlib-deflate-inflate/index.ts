// Test: zlib deflate/inflate + gzip/gunzip via Go compress/*
import { output } from "kit";

// Sync deflate → inflate round-trip
const original = "Hello brainkit! Testing zlib compression end-to-end.";
const compressed = zlib.deflateSync(Buffer.from(original));
const decompressed = zlib.inflateSync(compressed);

// Gzip round-trip
const gzipped = zlib.gzipSync(Buffer.from(original));
const gunzipped = zlib.gunzipSync(gzipped);

// Async callback
let asyncMatch = false;
zlib.deflate(Buffer.from("async-test"), (err: any, buf: any) => {
  if (!err) {
    zlib.inflate(buf, (err2: any, result: any) => {
      if (!err2) asyncMatch = result.toString("utf8") === "async-test";
    });
  }
});

output({
  deflateMatch: decompressed.toString("utf8") === original,
  gzipMatch: gunzipped.toString("utf8") === original,
  asyncMatch,
  hasConstants: typeof zlib.constants.Z_DEFAULT_COMPRESSION === "number",
});
