package jsbridge

import (
	"testing"
)

func TestZlib_DeflateInflate(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Zlib())
	val, err := b.Eval("test.js", `
		var Z = globalThis.__node_zlib;
		var original = "Hello, World! This is a test of zlib compression.";
		var compressed = Z.deflateSync(Buffer.from(original));
		var decompressed = Z.inflateSync(compressed);
		JSON.stringify({
			compressedLen: compressed.length,
			decompressedLen: decompressed.length,
			match: decompressed.toString("utf8") === original,
		});
	`)
	if err != nil {
		t.Fatalf("zlib deflate/inflate: %v", err)
	}
	defer val.Free()
	t.Logf("zlib: %s", val.String())
	if val.String() == "" {
		t.Fatal("empty result")
	}
}

func TestZlib_GzipGunzip(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Zlib())
	val, err := b.Eval("test.js", `
		var Z = globalThis.__node_zlib;
		var original = "Gzip test data 1234567890";
		var compressed = Z.gzipSync(Buffer.from(original));
		var decompressed = Z.gunzipSync(compressed);
		JSON.stringify({
			match: decompressed.toString("utf8") === original,
		});
	`)
	if err != nil {
		t.Fatalf("zlib gzip/gunzip: %v", err)
	}
	defer val.Free()
	t.Logf("gzip: %s", val.String())
}

func TestZlib_AsyncCallback(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Zlib())
	val, err := b.Eval("test.js", `
		var Z = globalThis.__node_zlib;
		var result = null;
		Z.deflate(Buffer.from("async test"), function(err, compressed) {
			if (err) throw err;
			Z.inflate(compressed, function(err2, decompressed) {
				if (err2) throw err2;
				result = decompressed.toString("utf8");
			});
		});
		JSON.stringify({ match: result === "async test" });
	`)
	if err != nil {
		t.Fatalf("zlib async: %v", err)
	}
	defer val.Free()
	t.Logf("zlib async: %s", val.String())
}
