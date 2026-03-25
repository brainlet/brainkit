package jsbridge

import "testing"

func TestCrypto_Pbkdf2Sync(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto())
	result := evalString(t, b, `
		var derived = globalThis.__node_crypto.pbkdf2Sync(
			"pencil",
			Buffer.from("W22ZaJ0SNY7soEsUEjb6gQ==", "base64"),
			4096, 32, "sha256"
		);
		JSON.stringify({ len: derived.length, isBuffer: Buffer.isBuffer(derived) });
	`)
	expected := `{"len":32,"isBuffer":true}`
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestCrypto_Pbkdf2Async(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto(), Timers())
	val, err := b.EvalAsync("test.js", `(async () => {
		var result = await new Promise(function(resolve, reject) {
			globalThis.__node_crypto.pbkdf2("pencil", Buffer.from("salt"), 1000, 32, "sha256", function(err, key) {
				if (err) reject(err);
				else resolve(key);
			});
		});
		return JSON.stringify({ len: result.length, isBuffer: Buffer.isBuffer(result) });
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	expected := `{"len":32,"isBuffer":true}`
	if val.String() != expected {
		t.Errorf("got %s", val.String())
	}
}

func TestCrypto_TimingSafeEqual(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto())
	result := evalString(t, b, `
		var a = Buffer.from("hello");
		var b = Buffer.from("hello");
		var c = Buffer.from("world");
		JSON.stringify({
			eq: globalThis.__node_crypto.timingSafeEqual(a, b),
			neq: globalThis.__node_crypto.timingSafeEqual(a, c),
		});
	`)
	expected := `{"eq":true,"neq":false}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestCrypto_RandomBytes(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto())
	result := evalString(t, b, `
		var sync = globalThis.__node_crypto.randomBytes(16);
		var cbResult = null;
		globalThis.__node_crypto.randomBytes(16, function(err, buf) { cbResult = buf; });
		JSON.stringify({
			syncLen: sync.length,
			cbLen: cbResult ? cbResult.length : -1,
		});
	`)
	expected := `{"syncLen":16,"cbLen":16}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestCrypto_SCRAMSequence(t *testing.T) {
	// Full SCRAM-SHA-256 crypto chain as MongoDB driver would call it
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto())
	result := evalString(t, b, `
		var c = globalThis.__node_crypto;
		// 1. pbkdf2Sync (SaltedPassword)
		var saltedPw = c.pbkdf2Sync("pencil", Buffer.from("W22ZaJ0SNY7soEsUEjb6gQ==", "base64"), 4096, 32, "sha256");
		// 2. HMAC (ClientKey)
		var clientKey = c.createHmac("sha256", saltedPw).update("Client Key").digest();
		// 3. Hash (StoredKey)
		var storedKey = c.createHash("sha256").update(clientKey).digest();
		// 4. HMAC (ClientSignature)
		var sig = c.createHmac("sha256", storedKey).update("test-auth-message").digest();
		JSON.stringify({
			saltedPwLen: saltedPw.length,
			clientKeyLen: clientKey.length,
			storedKeyLen: storedKey.length,
			sigLen: sig.length,
			allCorrectLen: saltedPw.length === 32 && clientKey.length === 32 && storedKey.length === 32 && sig.length === 32,
		});
	`)
	expected := `{"saltedPwLen":32,"clientKeyLen":32,"storedKeyLen":32,"sigLen":32,"allCorrectLen":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestCrypto_GetHashes(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Crypto())
	result := evalString(t, b, `
		var h = globalThis.__node_crypto.getHashes();
		JSON.stringify(h.includes("sha256") && h.includes("sha512") && h.includes("md5"));
	`)
	if result != "true" {
		t.Errorf("got %s", result)
	}
}
