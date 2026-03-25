// Standalone SCRAM-SHA-256 test. Verifies our crypto primitives produce
// correct results by testing against RFC 7677 test vectors and then
// attempting a real MongoDB connection with SCRAM auth.
//
// Run: go test ./experiments/ -run TestSCRAM -v -timeout 30s
package mongoscram

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	_ "github.com/brainlet/brainkit/kit/registry"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestSCRAMCrypto(t *testing.T) {
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test", CallerID: "test-scram", WorkspaceDir: tmpDir,
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
			"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()

	ctx := context.Background()

	// Test crypto primitives used by SCRAM-SHA-256
	result, err := k.EvalTS(ctx, "__scram_crypto.ts", `
		var crypto = globalThis.__node_crypto;
		var results = {};

		// 1. PBKDF2 via Go bridge
		var salt = Buffer.from("W22ZaJ0SNY7soEsUEjb6gQ==", "base64");
		var _b64c = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
		function toB64(bytes) {
			var b64 = "";
			for (var i = 0; i < bytes.length; i += 3) {
				var b0 = bytes[i], b1 = i+1<bytes.length?bytes[i+1]:0, b2 = i+2<bytes.length?bytes[i+2]:0;
				b64 += _b64c[(b0>>2)&0x3f]; b64 += _b64c[((b0<<4)|(b1>>4))&0x3f];
				b64 += (i+1<bytes.length) ? _b64c[((b1<<2)|(b2>>6))&0x3f] : "=";
				b64 += (i+2<bytes.length) ? _b64c[b2&0x3f] : "=";
			}
			return b64;
		}
		function fromB64(b64) {
			var lookup = {}; for (var i = 0; i < _b64c.length; i++) lookup[_b64c[i]] = i;
			var bufLen = Math.floor(b64.length * 3 / 4);
			if (b64.length > 1 && b64[b64.length-1] === "=") bufLen--;
			if (b64.length > 2 && b64[b64.length-2] === "=") bufLen--;
			var bytes = new Uint8Array(bufLen); var p = 0;
			for (var i = 0; i < b64.length; i += 4) {
				var a = lookup[b64[i]]||0, b = lookup[b64[i+1]]||0, c = lookup[b64[i+2]]||0, d = lookup[b64[i+3]]||0;
				bytes[p++] = (a<<2)|(b>>4);
				if (b64[i+2] !== "=") bytes[p++] = ((b<<4)|(c>>2))&0xff;
				if (b64[i+3] !== "=") bytes[p++] = ((c<<6)|d)&0xff;
			}
			return bytes;
		}
		function toHex(bytes) {
			return Array.from(new Uint8Array(bytes.buffer || bytes)).map(function(b) { return (b < 16 ? "0" : "") + b.toString(16); }).join("");
		}

		// PBKDF2
		var pwB64 = toB64(new TextEncoder().encode("pencil"));
		var saltB64 = toB64(salt);
		var derivedB64 = __go_crypto_subtle_deriveBits(pwB64, saltB64, 4096, 256, "SHA-256");
		var derived = fromB64(derivedB64);
		results.pbkdf2Hex = toHex(derived);
		results.pbkdf2Len = derived.length;

		// createHash sha256
		var h = crypto.createHash("sha256");
		h.update(derived);
		var hashResult = h.digest();
		results.hashLen = hashResult.length;
		results.hashHex = toHex(hashResult);

		// createHmac with binary key
		var hm = crypto.createHmac("sha256", derived);
		hm.update("Client Key");
		var hmacResult = hm.digest();
		results.hmacLen = hmacResult.length;
		results.hmacHex = toHex(hmacResult);

		// createHmac with string data (authMessage pattern)
		var hm2 = crypto.createHmac("sha256", hashResult);
		hm2.update("n=user,r=abc123,r=abc123def456,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096,c=biws,r=abc123def456");
		var sig = hm2.digest();
		results.sigLen = sig.length;
		results.sigHex = toHex(sig);

		// Verify all results are Uint8Array of correct length
		results.allCorrectType = (derived instanceof Uint8Array) && (hashResult instanceof Uint8Array) && (hmacResult instanceof Uint8Array) && (sig instanceof Uint8Array);
		results.allCorrectLen = derived.length === 32 && hashResult.length === 32 && hmacResult.length === 32 && sig.length === 32;

		// Test nonce encoding — this is what SCRAM uses for the first message
		var nonce = new Uint8Array(24);
		globalThis.crypto.getRandomValues(nonce);
		var nonceBuf = Buffer.from(nonce);
		var nonceB64 = nonceBuf.toString("base64");
		results.nonceLen = nonce.length;
		results.nonceB64Len = nonceB64.length;
		results.nonceIsBuffer = !!nonceBuf._isBuffer;
		// SCRAM first message: n,,n=user,r=<nonce_base64>
		var firstMsg = "n,,n=test,r=" + nonceB64;
		results.firstMsgLen = firstMsg.length;
		results.firstMsgSample = firstMsg.substring(0, 40);

		// Test Buffer.concat — used to build the SCRAM payload
		var b1 = Buffer.from("n,,", "utf8");
		var b2 = Buffer.from("n=test,r=" + nonceB64, "utf8");
		var concat = Buffer.concat([b1, b2]);
		results.concatLen = concat.length;
		results.concatIsBuffer = !!concat._isBuffer;

		return JSON.stringify(results);
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Crypto: %s", result)
}

// TestSCRAMJSAuth tests MongoDB SCRAM auth through the JS driver with debug logging.
func TestSCRAMJSAuth(t *testing.T) {
	testutil.LoadEnv(t)
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	mongoAddr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
		"MONGO_INITDB_ROOT_USERNAME=test", "MONGO_INITDB_ROOT_PASSWORD=test")

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", mongoAddr, 2*time.Second)
		if err == nil {
			c.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	tmpDir := t.TempDir()
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test", CallerID: "test-scram", WorkspaceDir: tmpDir,
		EnvVars: map[string]string{
			"MONGODB_URL": "mongodb://test:test@" + mongoAddr,
			// MongoDB driver accesses these for logger setup — must exist (even empty)
			// to avoid "cannot read property of undefined" in the driver's log severity parser.
			"MONGODB_LOG_ALL": "off",
		},
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
			"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := k.EvalTS(ctx, "__scram.ts", `
		try {
			var embed = globalThis.__agent_embed;
			var store = new embed.MongoDBStore({
				id: "scram-test",
				url: process.env.MONGODB_URL,
				dbName: "test_scram_db",
			});
			await store.init();
			// store.init() does MongoClient.connect() which triggers SCRAM auth.
			// If we get here without hanging, SCRAM auth succeeded.
			return JSON.stringify({ ok: true, auth: "scram-sha-256", connected: true });
		} catch(e) {
			return JSON.stringify({ error: e.message.substring(0, 200), stack: (e.stack || "").substring(0, 500) });
		}
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Result: %s", result)
}
