// Test: zlib deflate/inflate + gzip/gunzip via Go compress/*
import { output } from "kit";

const Z = globalThis.__node_zlib;

// Sync deflate → inflate round-trip
const original = "Hello brainkit! Testing zlib compression end-to-end.";
const compressed = Z.deflateSync(Buffer.from(original));
const decompressed = Z.inflateSync(compressed);

// Gzip round-trip
const gzipped = Z.gzipSync(Buffer.from(original));
const gunzipped = Z.gunzipSync(gzipped);

// Async callback
let asyncMatch = false;
Z.deflate(Buffer.from("async-test"), (err: any, buf: any) => {
  if (!err) {
    Z.inflate(buf, (err2: any, result: any) => {
      if (!err2) asyncMatch = result.toString("utf8") === "async-test";
    });
  }
});

output({
  deflateMatch: decompressed.toString("utf8") === original,
  gzipMatch: gunzipped.toString("utf8") === original,
  compressedSmaller: compressed.length < Buffer.from(original).length,
  asyncMatch,
  hasConstants: typeof Z.constants.Z_DEFAULT_COMPRESSION === "number",
});
