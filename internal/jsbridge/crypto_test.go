package jsbridge

import "testing"

func TestCrypto_Pbkdf2Sync(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto())
	result := evalString(t, b, `
		var derived = globalThis.crypto.pbkdf2Sync(
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
			globalThis.crypto.pbkdf2("pencil", Buffer.from("salt"), 1000, 32, "sha256", function(err, key) {
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
			eq: globalThis.crypto.timingSafeEqual(a, b),
			neq: globalThis.crypto.timingSafeEqual(a, c),
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
		var sync = globalThis.crypto.randomBytes(16);
		var cbResult = null;
		globalThis.crypto.randomBytes(16, function(err, buf) { cbResult = buf; });
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
		var c = globalThis.crypto;
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
			saltedPwHex: Buffer.from(saltedPw).toString("hex"),
			clientKeyLen: clientKey.length,
			clientKeyHex: Buffer.from(clientKey).toString("hex"),
			storedKeyLen: storedKey.length,
			storedKeyHex: Buffer.from(storedKey).toString("hex"),
			sigLen: sig.length,
			sigHex: Buffer.from(sig).toString("hex"),
			allCorrectLen: saltedPw.length === 32 && clientKey.length === 32 && storedKey.length === 32 && sig.length === 32,
		});
	`)
	expected := `{"saltedPwLen":32,"saltedPwHex":"c4a49510323ab4f952cac1fa99441939e78ea74d6be81ddf7096e87513dc615d","clientKeyLen":32,"clientKeyHex":"a60fc923d67e8644a92d16b96eda5ef4656b0c725c484374be25535576996e8b","storedKeyLen":32,"storedKeyHex":"586e5df283e6dceb5c3e791d8b8528ec191e664045ce971792e2e6b5bb13e2a6","sigLen":32,"sigHex":"8e4aca3ab701264c1675ac645b75f39b5329532fd05cab9633d332bea5bd28a0","allCorrectLen":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestCrypto_WebCryptoSCRAMSequence(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Buffer(), Crypto())
	val, err := b.EvalAsync("test.js", `(async () => {
		function hex(v) { return Buffer.from(v).toString("hex"); }
		const password = "pencil";
		const salt = Buffer.from("W22ZaJ0SNY7soEsUEjb6gQ==", "base64");
		const key = await crypto.subtle.importKey("raw", new TextEncoder().encode(password), { name: "PBKDF2" }, false, ["deriveBits"]);
		const saltedPw = new Uint8Array(await crypto.subtle.deriveBits({
			name: "PBKDF2",
			salt,
			iterations: 4096,
			hash: { name: "SHA-256" },
		}, key, 256));
		const hmacKey = await crypto.subtle.importKey("raw", saltedPw, { name: "HMAC", hash: { name: "SHA-256" } }, false, ["sign"]);
		const clientKey = new Uint8Array(await crypto.subtle.sign("HMAC", hmacKey, new TextEncoder().encode("Client Key")));
		const storedKey = new Uint8Array(await crypto.subtle.digest("SHA-256", clientKey));
		const sigKey = await crypto.subtle.importKey("raw", storedKey, { name: "HMAC", hash: { name: "SHA-256" } }, false, ["sign"]);
		const sig = new Uint8Array(await crypto.subtle.sign("HMAC", sigKey, new TextEncoder().encode("test-auth-message")));
		return JSON.stringify({
			saltedPwHex: hex(saltedPw),
			clientKeyHex: hex(clientKey),
			storedKeyHex: hex(storedKey),
			sigHex: hex(sig),
		});
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	expected := `{"saltedPwHex":"c4a49510323ab4f952cac1fa99441939e78ea74d6be81ddf7096e87513dc615d","clientKeyHex":"a60fc923d67e8644a92d16b96eda5ef4656b0c725c484374be25535576996e8b","storedKeyHex":"586e5df283e6dceb5c3e791d8b8528ec191e664045ce971792e2e6b5bb13e2a6","sigHex":"8e4aca3ab701264c1675ac645b75f39b5329532fd05cab9633d332bea5bd28a0"}`
	if val.String() != expected {
		t.Fatalf("got %s", val.String())
	}
}

func TestCrypto_GetHashes(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Crypto())
	result := evalString(t, b, `
		var h = globalThis.crypto.getHashes();
		JSON.stringify(h.includes("sha256") && h.includes("sha512") && h.includes("md5"));
	`)
	if result != "true" {
		t.Errorf("got %s", result)
	}
}
