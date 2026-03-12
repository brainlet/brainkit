// Experiment: QuickJS Web API Polyfills
//
// Goal: Prove that all common Web API polyfills needed for embedding JS
// libraries in Go via QuickJS can be implemented by bridging Go functions.
//
// Tests:
// 1.  TextEncoder/TextDecoder — UTF-8 encode/decode via Go
// 2.  crypto.randomUUID() — v4 UUID from Go's crypto/rand
// 3.  crypto.createHash(alg) — SHA-256 hex digest via Go
// 4.  crypto.createHmac(alg, key) — HMAC-SHA256 via Go
// 5.  console.log/warn/error — Captured output with levels
// 6.  setTimeout/clearTimeout — Timer mechanism via Go
// 7.  URL/URLSearchParams — URL parsing via Go's net/url
// 8.  btoa/atob — Base64 encode/decode via Go
// 9.  AbortController/AbortSignal — Abort mechanism via Go
// 10. EventEmitter (pure JS) — No Go bridge needed

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/fastschema/qjs"
)

func main() {
	fmt.Println("=== QuickJS Web API Polyfills Experiment ===")
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"TextEncoder/TextDecoder", testTextEncoderDecoder},
		{"crypto.randomUUID()", testCryptoRandomUUID},
		{"crypto.createHash(alg)", testCryptoCreateHash},
		{"crypto.createHmac(alg, key)", testCryptoCreateHmac},
		{"console.log/warn/error", testConsole},
		{"setTimeout/clearTimeout", testSetTimeout},
		{"URL/URLSearchParams", testURLParsing},
		{"btoa/atob", testBtoaAtob},
		{"AbortController/AbortSignal", testAbortController},
		{"EventEmitter (pure JS)", testEventEmitter},
	}

	passed := 0
	for i, t := range tests {
		fmt.Printf("--- Test %d: %s ---\n", i+1, t.name)
		if err := t.fn(); err != nil {
			log.Fatalf("FAILED: %v\n", err)
		}
		fmt.Println("PASS")
		fmt.Println()
		passed++
	}

	fmt.Printf("=== ALL %d TESTS PASSED ===\n", passed)
}

// ---------------------------------------------------------------------------
// Test 1: TextEncoder/TextDecoder
// ---------------------------------------------------------------------------

func testTextEncoderDecoder() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// __go_text_encode(str) → comma-separated byte values
	ctx.SetFunc("__go_text_encode", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_text_encode requires 1 argument")
		}
		str := args[0].String()
		bytes := []byte(str)
		parts := make([]string, len(bytes))
		for i, b := range bytes {
			parts[i] = fmt.Sprintf("%d", b)
		}
		return this.Context().NewString(strings.Join(parts, ",")), nil
	})

	// __go_text_decode(commaBytes) → string
	ctx.SetFunc("__go_text_decode", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_text_decode requires 1 argument")
		}
		csv := args[0].String()
		parts := strings.Split(csv, ",")
		bytes := make([]byte, len(parts))
		for i, p := range parts {
			var v int
			_, err := fmt.Sscanf(p, "%d", &v)
			if err != nil {
				return nil, fmt.Errorf("invalid byte value %q: %w", p, err)
			}
			bytes[i] = byte(v)
		}
		return this.Context().NewString(string(bytes)), nil
	})

	result, err := ctx.Eval("text_encoder.js", qjs.Code(`
		class TextEncoder {
			encode(str) {
				const csv = __go_text_encode(str);
				const nums = csv.split(",").map(Number);
				return new Uint8Array(nums);
			}
		}

		class TextDecoder {
			decode(uint8arr) {
				const csv = Array.from(uint8arr).join(",");
				return __go_text_decode(csv);
			}
		}

		const encoder = new TextEncoder();
		const decoder = new TextDecoder();

		const original = "Hello 🌍";
		const encoded = encoder.encode(original);
		const decoded = decoder.decode(encoded);

		JSON.stringify({
			original: original,
			encodedLength: encoded.length,
			encodedBytes: Array.from(encoded).join(","),
			decoded: decoded,
			roundTrip: original === decoded
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["roundTrip"] != true {
		return fmt.Errorf("round-trip failed: original=%q decoded=%q", parsed["original"], parsed["decoded"])
	}

	// "Hello 🌍" is 10 bytes in UTF-8 (5 for Hello, 1 for space, 4 for 🌍)
	encodedLen := parsed["encodedLength"].(float64)
	if encodedLen != 10 {
		return fmt.Errorf("expected encoded length 10, got %v", encodedLen)
	}
	fmt.Printf("  Encoded %q to %d bytes, decoded back successfully\n", parsed["original"], int(encodedLen))
	return nil
}

// ---------------------------------------------------------------------------
// Test 2: crypto.randomUUID()
// ---------------------------------------------------------------------------

func testCryptoRandomUUID() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// __go_crypto_randomUUID() → v4 UUID string
	ctx.SetFunc("__go_crypto_randomUUID", func(this *qjs.This) (*qjs.Value, error) {
		uuid := make([]byte, 16)
		if _, err := rand.Read(uuid); err != nil {
			return nil, fmt.Errorf("crypto/rand failed: %w", err)
		}
		// Set version 4
		uuid[6] = (uuid[6] & 0x0f) | 0x40
		// Set variant bits
		uuid[8] = (uuid[8] & 0x3f) | 0x80

		s := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
		return this.Context().NewString(s), nil
	})

	result, err := ctx.Eval("crypto_uuid.js", qjs.Code(`
		const crypto = {
			randomUUID() {
				return __go_crypto_randomUUID();
			}
		};

		const uuid1 = crypto.randomUUID();
		const uuid2 = crypto.randomUUID();

		// UUID v4 pattern: xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
		const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

		JSON.stringify({
			uuid1: uuid1,
			uuid2: uuid2,
			uuid1Valid: uuidPattern.test(uuid1),
			uuid2Valid: uuidPattern.test(uuid2),
			unique: uuid1 !== uuid2
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["uuid1Valid"] != true {
		return fmt.Errorf("uuid1 %q does not match v4 pattern", parsed["uuid1"])
	}
	if parsed["uuid2Valid"] != true {
		return fmt.Errorf("uuid2 %q does not match v4 pattern", parsed["uuid2"])
	}
	if parsed["unique"] != true {
		return fmt.Errorf("two UUIDs are identical: %q", parsed["uuid1"])
	}
	fmt.Printf("  UUID1: %s\n  UUID2: %s\n  Both valid v4, unique\n", parsed["uuid1"], parsed["uuid2"])
	return nil
}

// ---------------------------------------------------------------------------
// Test 3: crypto.createHash(alg)
// ---------------------------------------------------------------------------

func testCryptoCreateHash() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// __go_crypto_createHash(alg, data) → hex digest
	ctx.SetFunc("__go_crypto_createHash", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("__go_crypto_createHash requires 2 arguments (alg, data)")
		}
		alg := args[0].String()
		data := args[1].String()

		switch alg {
		case "sha256":
			h := sha256.Sum256([]byte(data))
			return this.Context().NewString(hex.EncodeToString(h[:])), nil
		default:
			return nil, fmt.Errorf("unsupported hash algorithm: %s", alg)
		}
	})

	// Known SHA-256 of "hello world" = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
	result, err := ctx.Eval("crypto_hash.js", qjs.Code(`
		function createHash(alg) {
			let data = "";
			return {
				update(input) { data += input; return this; },
				digest(encoding) {
					if (encoding === "hex") {
						return __go_crypto_createHash(alg, data);
					}
					throw new Error("unsupported encoding: " + encoding);
				}
			};
		}

		const hash = createHash("sha256").update("hello world").digest("hex");
		const expected = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9";

		JSON.stringify({
			hash: hash,
			expected: expected,
			match: hash === expected
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["match"] != true {
		return fmt.Errorf("hash mismatch: got %q, expected %q", parsed["hash"], parsed["expected"])
	}
	fmt.Printf("  SHA-256(\"hello world\") = %s\n", parsed["hash"])
	return nil
}

// ---------------------------------------------------------------------------
// Test 4: crypto.createHmac(alg, key)
// ---------------------------------------------------------------------------

func testCryptoCreateHmac() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// __go_crypto_createHmac(alg, key, data) → hex digest
	ctx.SetFunc("__go_crypto_createHmac", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 3 {
			return nil, fmt.Errorf("__go_crypto_createHmac requires 3 arguments (alg, key, data)")
		}
		alg := args[0].String()
		key := args[1].String()
		data := args[2].String()

		switch alg {
		case "sha256":
			mac := hmac.New(sha256.New, []byte(key))
			mac.Write([]byte(data))
			return this.Context().NewString(hex.EncodeToString(mac.Sum(nil))), nil
		default:
			return nil, fmt.Errorf("unsupported hmac algorithm: %s", alg)
		}
	})

	// Known HMAC-SHA256 of "hello world" with key "secret":
	// 734cc62f32841568f45f6c1a8eb40c4e4014c059e3b0fbb26a41e8ab199a3c3a
	result, err := ctx.Eval("crypto_hmac.js", qjs.Code(`
		function createHmac(alg, key) {
			let data = "";
			return {
				update(input) { data += input; return this; },
				digest(encoding) {
					if (encoding === "hex") {
						return __go_crypto_createHmac(alg, key, data);
					}
					throw new Error("unsupported encoding: " + encoding);
				}
			};
		}

		const mac = createHmac("sha256", "secret").update("hello world").digest("hex");
		const expected = "734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a";

		JSON.stringify({
			hmac: mac,
			expected: expected,
			match: mac === expected
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["match"] != true {
		return fmt.Errorf("HMAC mismatch: got %q, expected %q", parsed["hmac"], parsed["expected"])
	}
	fmt.Printf("  HMAC-SHA256(\"hello world\", \"secret\") = %s\n", parsed["hmac"])
	return nil
}

// ---------------------------------------------------------------------------
// Test 5: console.log/warn/error
// ---------------------------------------------------------------------------

func testConsole() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Capture console messages
	type consoleMsg struct {
		Level string
		Msg   string
	}
	var mu sync.Mutex
	var captured []consoleMsg

	// __go_console_log(level, msg)
	ctx.SetFunc("__go_console_log", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("__go_console_log requires 2 arguments (level, msg)")
		}
		level := args[0].String()
		msg := args[1].String()

		mu.Lock()
		captured = append(captured, consoleMsg{Level: level, Msg: msg})
		mu.Unlock()

		return this.Context().NewUndefined(), nil
	})

	result, err := ctx.Eval("console.js", qjs.Code(`
		const console = {
			log(...args)   { __go_console_log("log",   args.map(String).join(" ")); },
			warn(...args)  { __go_console_log("warn",  args.map(String).join(" ")); },
			error(...args) { __go_console_log("error", args.map(String).join(" ")); }
		};

		console.log("Hello from JS", 42);
		console.warn("This is a warning");
		console.error("Something went wrong:", "details here");

		"done";
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	mu.Lock()
	msgs := make([]consoleMsg, len(captured))
	copy(msgs, captured)
	mu.Unlock()

	if len(msgs) != 3 {
		return fmt.Errorf("expected 3 console messages, got %d", len(msgs))
	}

	expectations := []consoleMsg{
		{Level: "log", Msg: "Hello from JS 42"},
		{Level: "warn", Msg: "This is a warning"},
		{Level: "error", Msg: "Something went wrong: details here"},
	}

	for i, exp := range expectations {
		if msgs[i].Level != exp.Level {
			return fmt.Errorf("message %d: expected level %q, got %q", i, exp.Level, msgs[i].Level)
		}
		if msgs[i].Msg != exp.Msg {
			return fmt.Errorf("message %d: expected msg %q, got %q", i, exp.Msg, msgs[i].Msg)
		}
		fmt.Printf("  [%s] %s\n", msgs[i].Level, msgs[i].Msg)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 6: setTimeout/clearTimeout
// ---------------------------------------------------------------------------

func testSetTimeout() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Track pending callbacks: id -> JS callback function name
	type pendingTimer struct {
		CallbackID string
		DelayMs    int
		Cancelled  bool
	}
	var mu sync.Mutex
	var timers []pendingTimer
	nextID := 1

	// __go_setTimeout(callbackID, ms) → timerID
	ctx.SetFunc("__go_setTimeout", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("__go_setTimeout requires 2 arguments")
		}
		callbackID := args[0].String()
		delayMs := int(args[1].Int32())

		mu.Lock()
		id := nextID
		nextID++
		timers = append(timers, pendingTimer{
			CallbackID: callbackID,
			DelayMs:    delayMs,
			Cancelled:  false,
		})
		mu.Unlock()

		return this.Context().NewInt32(int32(id)), nil
	})

	// __go_clearTimeout(id) → void
	ctx.SetFunc("__go_clearTimeout", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_clearTimeout requires 1 argument")
		}
		id := int(args[0].Int32())

		mu.Lock()
		if id > 0 && id <= len(timers) {
			timers[id-1].Cancelled = true
		}
		mu.Unlock()

		return this.Context().NewUndefined(), nil
	})

	// __go_drain_timers() → execute all pending non-cancelled timers
	ctx.SetFunc("__go_drain_timers", func(this *qjs.This) (*qjs.Value, error) {
		mu.Lock()
		pending := make([]pendingTimer, len(timers))
		copy(pending, timers)
		mu.Unlock()

		c := this.Context()
		executed := 0
		for _, t := range pending {
			if !t.Cancelled {
				// Call the global callback function by name
				_, err := c.Eval("timer_exec.js", qjs.Code(t.CallbackID+"();"))
				if err != nil {
					return nil, fmt.Errorf("timer callback %q failed: %w", t.CallbackID, err)
				}
				executed++
			}
		}
		return c.NewInt32(int32(executed)), nil
	})

	result, err := ctx.Eval("settimeout.js", qjs.Code(`
		let callbackCounter = 0;
		const callbackRegistry = {};

		function setTimeout(fn, ms) {
			callbackCounter++;
			const cbName = "__timer_cb_" + callbackCounter;
			globalThis[cbName] = fn;
			return __go_setTimeout(cbName, ms);
		}

		function clearTimeout(id) {
			__go_clearTimeout(id);
		}

		// Track which callbacks fired
		const fired = [];

		// Timer 1: should fire
		const t1 = setTimeout(() => { fired.push("timer1"); }, 100);

		// Timer 2: will be cancelled
		const t2 = setTimeout(() => { fired.push("timer2"); }, 200);

		// Timer 3: should fire
		const t3 = setTimeout(() => { fired.push("timer3"); }, 50);

		// Cancel timer 2
		clearTimeout(t2);

		// Drain all pending timers (simulates time passing)
		const executedCount = __go_drain_timers();

		JSON.stringify({
			fired: fired,
			executedCount: executedCount,
			timer2Cancelled: !fired.includes("timer2"),
			timer1Fired: fired.includes("timer1"),
			timer3Fired: fired.includes("timer3")
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["timer1Fired"] != true {
		return fmt.Errorf("timer1 should have fired")
	}
	if parsed["timer3Fired"] != true {
		return fmt.Errorf("timer3 should have fired")
	}
	if parsed["timer2Cancelled"] != true {
		return fmt.Errorf("timer2 should have been cancelled")
	}
	execCount := parsed["executedCount"].(float64)
	if execCount != 2 {
		return fmt.Errorf("expected 2 timers executed, got %v", execCount)
	}

	fmt.Printf("  Timers fired: %v\n", parsed["fired"])
	fmt.Printf("  timer2 cancelled: %v, executed count: %v\n", parsed["timer2Cancelled"], int(execCount))
	return nil
}

// ---------------------------------------------------------------------------
// Test 7: URL/URLSearchParams
// ---------------------------------------------------------------------------

func testURLParsing() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// __go_url_parse(urlStr) → JSON with components
	ctx.SetFunc("__go_url_parse", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_url_parse requires 1 argument")
		}
		rawURL := args[0].String()

		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("url parse failed: %w", err)
		}

		result := map[string]string{
			"protocol": u.Scheme + ":",
			"host":     u.Host,
			"hostname": u.Hostname(),
			"port":     u.Port(),
			"pathname": u.Path,
			"search":   "",
			"hash":     "",
		}
		if u.RawQuery != "" {
			result["search"] = "?" + u.RawQuery
		}
		if u.Fragment != "" {
			result["hash"] = "#" + u.Fragment
		}

		j, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("json marshal failed: %w", err)
		}
		return this.Context().NewString(string(j)), nil
	})

	// __go_url_searchParams(queryStr) → JSON of key-value pairs
	ctx.SetFunc("__go_url_searchParams", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_url_searchParams requires 1 argument")
		}
		queryStr := args[0].String()
		// Strip leading "?" if present
		queryStr = strings.TrimPrefix(queryStr, "?")

		values, err := url.ParseQuery(queryStr)
		if err != nil {
			return nil, fmt.Errorf("query parse failed: %w", err)
		}

		// Convert to array of [key, value] pairs for multiple values
		pairs := [][]string{}
		for k, vs := range values {
			for _, v := range vs {
				pairs = append(pairs, []string{k, v})
			}
		}

		j, err := json.Marshal(pairs)
		if err != nil {
			return nil, fmt.Errorf("json marshal failed: %w", err)
		}
		return this.Context().NewString(string(j)), nil
	})

	result, err := ctx.Eval("url_parse.js", qjs.Code(`
		class URL {
			constructor(urlStr) {
				const parsed = JSON.parse(__go_url_parse(urlStr));
				this.protocol = parsed.protocol;
				this.host = parsed.host;
				this.hostname = parsed.hostname;
				this.port = parsed.port;
				this.pathname = parsed.pathname;
				this.search = parsed.search;
				this.hash = parsed.hash;
				this.href = urlStr;
			}

			get searchParams() {
				return new URLSearchParams(this.search);
			}
		}

		class URLSearchParams {
			constructor(queryStr) {
				this._pairs = JSON.parse(__go_url_searchParams(queryStr || ""));
			}

			get(key) {
				const pair = this._pairs.find(p => p[0] === key);
				return pair ? pair[1] : null;
			}

			getAll(key) {
				return this._pairs.filter(p => p[0] === key).map(p => p[1]);
			}

			has(key) {
				return this._pairs.some(p => p[0] === key);
			}

			toString() {
				return this._pairs.map(p => p[0] + "=" + encodeURIComponent(p[1])).join("&");
			}
		}

		const url = new URL("https://example.com:8080/path/to/resource?foo=bar&baz=qux&foo=second#section1");

		const params = url.searchParams;

		JSON.stringify({
			protocol: url.protocol,
			host: url.host,
			hostname: url.hostname,
			port: url.port,
			pathname: url.pathname,
			search: url.search,
			hash: url.hash,
			paramFoo: params.get("foo"),
			paramBaz: params.get("baz"),
			paramFooAll: params.getAll("foo"),
			hasFoo: params.has("foo"),
			hasMissing: params.has("missing")
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	checks := map[string]interface{}{
		"protocol": "https:",
		"host":     "example.com:8080",
		"hostname": "example.com",
		"port":     "8080",
		"pathname": "/path/to/resource",
		"hash":     "#section1",
		"paramBaz": "qux",
		"hasFoo":   true,
		"hasMissing": false,
	}

	for k, expected := range checks {
		actual := parsed[k]
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return fmt.Errorf("%s: expected %v, got %v", k, expected, actual)
		}
	}

	fmt.Printf("  URL components parsed correctly\n")
	fmt.Printf("  searchParams.get(\"foo\") = %v\n", parsed["paramFoo"])
	fmt.Printf("  searchParams.getAll(\"foo\") = %v\n", parsed["paramFooAll"])
	return nil
}

// ---------------------------------------------------------------------------
// Test 8: btoa/atob
// ---------------------------------------------------------------------------

func testBtoaAtob() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// __go_btoa(str) → base64 encoded string
	ctx.SetFunc("__go_btoa", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_btoa requires 1 argument")
		}
		str := args[0].String()
		encoded := base64.StdEncoding.EncodeToString([]byte(str))
		return this.Context().NewString(encoded), nil
	})

	// __go_atob(encoded) → decoded string
	ctx.SetFunc("__go_atob", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_atob requires 1 argument")
		}
		encoded := args[0].String()
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("base64 decode failed: %w", err)
		}
		return this.Context().NewString(string(decoded)), nil
	})

	result, err := ctx.Eval("btoa_atob.js", qjs.Code(`
		function btoa(str) { return __go_btoa(str); }
		function atob(str) { return __go_atob(str); }

		const testCases = [
			"Hello, World!",
			"Special chars: !@#$%^&*()",
			"Unicode safe ASCII: The quick brown fox",
			"Numbers and symbols: 12345 +-=[]{}",
			"Longer text: The quick brown fox jumps over the lazy dog"
		];

		const results = testCases.map(input => {
			const encoded = btoa(input);
			const decoded = atob(encoded);
			return {
				input: input,
				encoded: encoded,
				decoded: decoded,
				roundTrip: input === decoded
			};
		});

		const allPassed = results.every(r => r.roundTrip);

		JSON.stringify({
			results: results,
			allPassed: allPassed
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["allPassed"] != true {
		return fmt.Errorf("not all btoa/atob round-trips passed")
	}

	results := parsed["results"].([]interface{})
	for _, r := range results {
		rm := r.(map[string]interface{})
		fmt.Printf("  btoa(%q) = %q => roundTrip: %v\n", rm["input"], rm["encoded"], rm["roundTrip"])
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test 9: AbortController/AbortSignal
// ---------------------------------------------------------------------------

func testAbortController() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Track abort controllers in Go
	type abortState struct {
		Aborted bool
		Reason  string
	}
	var mu sync.Mutex
	controllers := map[int]*abortState{}
	nextID := 1

	// __go_abort_create() → controller ID
	ctx.SetFunc("__go_abort_create", func(this *qjs.This) (*qjs.Value, error) {
		mu.Lock()
		id := nextID
		nextID++
		controllers[id] = &abortState{Aborted: false, Reason: ""}
		mu.Unlock()
		return this.Context().NewInt32(int32(id)), nil
	})

	// __go_abort_signal(id) → JSON { aborted, reason }
	ctx.SetFunc("__go_abort_signal", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_abort_signal requires 1 argument")
		}
		id := int(args[0].Int32())

		mu.Lock()
		state := controllers[id]
		mu.Unlock()

		if state == nil {
			return nil, fmt.Errorf("unknown controller ID: %d", id)
		}

		j, _ := json.Marshal(map[string]interface{}{
			"aborted": state.Aborted,
			"reason":  state.Reason,
		})
		return this.Context().NewString(string(j)), nil
	})

	// __go_abort_trigger(id, reason) → void
	ctx.SetFunc("__go_abort_trigger", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_abort_trigger requires 1 argument")
		}
		id := int(args[0].Int32())
		reason := "AbortError"
		if len(args) > 1 {
			reason = args[1].String()
		}

		mu.Lock()
		state := controllers[id]
		if state != nil {
			state.Aborted = true
			state.Reason = reason
		}
		mu.Unlock()

		return this.Context().NewUndefined(), nil
	})

	result, err := ctx.Eval("abort_controller.js", qjs.Code(`
		class AbortSignal {
			constructor(id) {
				this._id = id;
				this._listeners = [];
			}

			get aborted() {
				const state = JSON.parse(__go_abort_signal(this._id));
				return state.aborted;
			}

			get reason() {
				const state = JSON.parse(__go_abort_signal(this._id));
				return state.reason;
			}

			addEventListener(event, fn) {
				if (event === "abort") {
					this._listeners.push(fn);
				}
			}

			_fireAbort() {
				for (const fn of this._listeners) {
					fn();
				}
			}
		}

		class AbortController {
			constructor() {
				this._id = __go_abort_create();
				this.signal = new AbortSignal(this._id);
			}

			abort(reason) {
				__go_abort_trigger(this._id, reason || "AbortError");
				this.signal._fireAbort();
			}
		}

		// Test 1: Basic abort flow
		const ac1 = new AbortController();
		const beforeAbort = ac1.signal.aborted;
		ac1.abort("user cancelled");
		const afterAbort = ac1.signal.aborted;
		const abortReason = ac1.signal.reason;

		// Test 2: Event listener
		const ac2 = new AbortController();
		let listenerFired = false;
		ac2.signal.addEventListener("abort", () => { listenerFired = true; });
		ac2.abort();

		// Test 3: Simulate aborted fetch-like operation
		const ac3 = new AbortController();
		function fakeFetch(url, options) {
			if (options && options.signal && options.signal.aborted) {
				return { error: "AbortError: The operation was aborted" };
			}
			return { data: "success" };
		}

		// Fetch before abort: should succeed
		const result1 = fakeFetch("/api/data", { signal: ac3.signal });

		// Abort, then try to fetch: should fail
		ac3.abort("cancelled");
		const result2 = fakeFetch("/api/data", { signal: ac3.signal });

		JSON.stringify({
			beforeAbort: beforeAbort,
			afterAbort: afterAbort,
			abortReason: abortReason,
			listenerFired: listenerFired,
			fetchBeforeAbort: result1,
			fetchAfterAbort: result2
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	if parsed["beforeAbort"] != false {
		return fmt.Errorf("signal.aborted should be false before abort")
	}
	if parsed["afterAbort"] != true {
		return fmt.Errorf("signal.aborted should be true after abort")
	}
	if parsed["abortReason"] != "user cancelled" {
		return fmt.Errorf("abort reason should be 'user cancelled', got %q", parsed["abortReason"])
	}
	if parsed["listenerFired"] != true {
		return fmt.Errorf("abort event listener should have fired")
	}

	fetchBefore := parsed["fetchBeforeAbort"].(map[string]interface{})
	if fetchBefore["data"] != "success" {
		return fmt.Errorf("fetch before abort should succeed")
	}

	fetchAfter := parsed["fetchAfterAbort"].(map[string]interface{})
	if fetchAfter["error"] == nil {
		return fmt.Errorf("fetch after abort should fail with AbortError")
	}

	fmt.Printf("  Before abort: aborted=%v\n", parsed["beforeAbort"])
	fmt.Printf("  After abort:  aborted=%v, reason=%q\n", parsed["afterAbort"], parsed["abortReason"])
	fmt.Printf("  Event listener fired: %v\n", parsed["listenerFired"])
	fmt.Printf("  Fetch before abort: %v\n", fetchBefore)
	fmt.Printf("  Fetch after abort:  %v\n", fetchAfter)
	return nil
}

// ---------------------------------------------------------------------------
// Test 10: EventEmitter (pure JS)
// ---------------------------------------------------------------------------

func testEventEmitter() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	result, err := ctx.Eval("event_emitter.js", qjs.Code(`
		class EventEmitter {
			constructor() {
				this._listeners = {};
			}

			on(event, fn) {
				if (!this._listeners[event]) {
					this._listeners[event] = [];
				}
				this._listeners[event].push({ fn, once: false });
				return this;
			}

			once(event, fn) {
				if (!this._listeners[event]) {
					this._listeners[event] = [];
				}
				this._listeners[event].push({ fn, once: true });
				return this;
			}

			emit(event, ...args) {
				const listeners = this._listeners[event];
				if (!listeners) return false;

				const toRemove = [];
				for (let i = 0; i < listeners.length; i++) {
					listeners[i].fn(...args);
					if (listeners[i].once) {
						toRemove.push(i);
					}
				}

				// Remove once listeners in reverse order
				for (let i = toRemove.length - 1; i >= 0; i--) {
					listeners.splice(toRemove[i], 1);
				}

				return true;
			}

			removeListener(event, fn) {
				const listeners = this._listeners[event];
				if (!listeners) return this;
				this._listeners[event] = listeners.filter(l => l.fn !== fn);
				return this;
			}

			listenerCount(event) {
				const listeners = this._listeners[event];
				return listeners ? listeners.length : 0;
			}
		}

		// Test suite
		const log = [];
		const emitter = new EventEmitter();

		// Test 1: Basic on() and emit()
		emitter.on("data", (msg) => { log.push("data:" + msg); });
		emitter.emit("data", "hello");
		emitter.emit("data", "world");

		// Test 2: once() fires only once
		emitter.once("connect", () => { log.push("connected"); });
		emitter.emit("connect");
		emitter.emit("connect"); // Should NOT fire again

		// Test 3: Multiple listeners
		emitter.on("multi", (v) => { log.push("multi-a:" + v); });
		emitter.on("multi", (v) => { log.push("multi-b:" + v); });
		emitter.emit("multi", "test");

		// Test 4: removeListener
		const tempHandler = (v) => { log.push("temp:" + v); };
		emitter.on("remove-test", tempHandler);
		emitter.emit("remove-test", "before");
		emitter.removeListener("remove-test", tempHandler);
		emitter.emit("remove-test", "after"); // Should NOT fire

		// Test 5: emit returns false for no listeners
		const emitResult = emitter.emit("nonexistent");

		// Test 6: listenerCount
		const emitter2 = new EventEmitter();
		emitter2.on("x", () => {});
		emitter2.on("x", () => {});
		emitter2.on("y", () => {});
		const countX = emitter2.listenerCount("x");
		const countY = emitter2.listenerCount("y");
		const countZ = emitter2.listenerCount("z");

		JSON.stringify({
			log: log,
			emitNoListeners: emitResult,
			listenerCountX: countX,
			listenerCountY: countY,
			listenerCountZ: countZ,
			expectedLog: [
				"data:hello",
				"data:world",
				"connected",
				"multi-a:test",
				"multi-b:test",
				"temp:before"
			]
		});
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	actualLog := parsed["log"].([]interface{})
	expectedLog := parsed["expectedLog"].([]interface{})

	if len(actualLog) != len(expectedLog) {
		return fmt.Errorf("log length mismatch: got %d entries, expected %d\n  actual:   %v\n  expected: %v",
			len(actualLog), len(expectedLog), actualLog, expectedLog)
	}

	for i, exp := range expectedLog {
		if actualLog[i] != exp {
			return fmt.Errorf("log[%d]: expected %q, got %q", i, exp, actualLog[i])
		}
	}

	if parsed["emitNoListeners"] != false {
		return fmt.Errorf("emit() with no listeners should return false")
	}
	if parsed["listenerCountX"].(float64) != 2 {
		return fmt.Errorf("listenerCount('x') should be 2")
	}
	if parsed["listenerCountY"].(float64) != 1 {
		return fmt.Errorf("listenerCount('y') should be 1")
	}
	if parsed["listenerCountZ"].(float64) != 0 {
		return fmt.Errorf("listenerCount('z') should be 0")
	}

	fmt.Printf("  Event log: %v\n", actualLog)
	fmt.Printf("  on() works: data events fired correctly\n")
	fmt.Printf("  once() works: connect fired once only\n")
	fmt.Printf("  Multiple listeners: multi-a and multi-b both fired\n")
	fmt.Printf("  removeListener() works: temp handler removed\n")
	fmt.Printf("  emit() returns false for no listeners\n")
	fmt.Printf("  listenerCount: x=%v, y=%v, z=%v\n",
		int(parsed["listenerCountX"].(float64)),
		int(parsed["listenerCountY"].(float64)),
		int(parsed["listenerCountZ"].(float64)))
	return nil
}
