package jsbridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastschema/qjs"
)

func newTestBridge(t *testing.T, polyfills ...Polyfill) *Bridge {
	t.Helper()
	b, err := New(Config{}, polyfills...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(b.Close)
	return b
}

func evalString(t *testing.T, b *Bridge, code string) string {
	t.Helper()
	val, err := b.Eval("test.js", qjs.Code(code))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()
	return val.String()
}

func TestBridgeBasic(t *testing.T) {
	b := newTestBridge(t)
	result := evalString(t, b, `1 + 2`)
	if result != "3" {
		t.Errorf("got %q, want %q", result, "3")
	}
}

func TestConsole(t *testing.T) {
	var stdout, stderr bytes.Buffer
	con := Console(ConsoleStdout(&stdout), ConsoleStderr(&stderr))
	b := newTestBridge(t, con)

	evalString(t, b, `
		console.log("hello", "world");
		console.warn("warning");
		console.error("err");
		console.info("info");
		console.debug("dbg");
	`)

	msgs := con.Messages()
	if len(msgs) != 5 {
		t.Fatalf("got %d messages, want 5", len(msgs))
	}

	expected := []struct{ level, msg string }{
		{"log", "hello world"},
		{"warn", "warning"},
		{"error", "err"},
		{"info", "info"},
		{"debug", "dbg"},
	}
	for i, e := range expected {
		if msgs[i].Level != e.level || msgs[i].Message != e.msg {
			t.Errorf("msg[%d] = {%q, %q}, want {%q, %q}", i, msgs[i].Level, msgs[i].Message, e.level, e.msg)
		}
	}

	if !strings.Contains(stdout.String(), "hello world") {
		t.Errorf("stdout missing 'hello world': %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning") {
		t.Errorf("stderr missing 'warning': %q", stderr.String())
	}
}

func TestCryptoRandomUUID(t *testing.T) {
	b := newTestBridge(t, Crypto())
	result := evalString(t, b, `crypto.randomUUID()`)
	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	if len(result) != 36 || result[14] != '4' {
		t.Errorf("invalid UUID: %q", result)
	}

	// Uniqueness
	result2 := evalString(t, b, `crypto.randomUUID()`)
	if result == result2 {
		t.Error("two UUIDs should be different")
	}
}

func TestCryptoHash(t *testing.T) {
	b := newTestBridge(t, Crypto())
	result := evalString(t, b, `__node_crypto.createHash('sha256').update('hello world').digest('hex')`)
	// Known SHA-256 of "hello world"
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if result != want {
		t.Errorf("sha256 = %q, want %q", result, want)
	}
}

func TestCryptoHmac(t *testing.T) {
	b := newTestBridge(t, Crypto())
	result := evalString(t, b, `__node_crypto.createHmac('sha256', 'secret').update('hello').digest('hex')`)
	// Known HMAC-SHA256 of "hello" with key "secret"
	want := "88aab3ede8d3adf94d26ab90d3bafd4a2083070c3bcce9c014ee04a443847c0b"
	if result != want {
		t.Errorf("hmac = %q, want %q", result, want)
	}
}

func TestTextEncoder(t *testing.T) {
	b := newTestBridge(t, Encoding())
	result := evalString(t, b, `
		const enc = new TextEncoder();
		const buf = enc.encode("Hello");
		buf.length.toString();
	`)
	if result != "5" {
		t.Errorf("TextEncoder length = %q, want %q", result, "5")
	}
}

func TestTextDecoder(t *testing.T) {
	b := newTestBridge(t, Encoding())
	result := evalString(t, b, `
		const enc = new TextEncoder();
		const dec = new TextDecoder();
		dec.decode(enc.encode("Hello 🌍"));
	`)
	if result != "Hello 🌍" {
		t.Errorf("TextDecoder = %q, want %q", result, "Hello 🌍")
	}
}

func TestBtoaAtob(t *testing.T) {
	b := newTestBridge(t, Encoding())
	result := evalString(t, b, `atob(btoa("Hello World"))`)
	if result != "Hello World" {
		t.Errorf("btoa/atob = %q, want %q", result, "Hello World")
	}
}

func TestURL(t *testing.T) {
	b := newTestBridge(t, URL())
	result := evalString(t, b, `
		const u = new URL("https://example.com:8080/path?q=1&r=2#hash");
		JSON.stringify({
			protocol: u.protocol,
			hostname: u.hostname,
			port: u.port,
			pathname: u.pathname,
			search: u.search,
			hash: u.hash,
		});
	`)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	checks := map[string]string{
		"protocol": "https:",
		"hostname": "example.com",
		"port":     "8080",
		"pathname": "/path",
		"search":   "?q=1&r=2",
		"hash":     "#hash",
	}
	for k, want := range checks {
		if parsed[k] != want {
			t.Errorf("URL.%s = %q, want %q", k, parsed[k], want)
		}
	}
}

func TestURLSearchParams(t *testing.T) {
	b := newTestBridge(t, URL())
	result := evalString(t, b, `
		const p = new URLSearchParams("a=1&b=2&a=3");
		JSON.stringify({ a: p.get("a"), b: p.get("b"), all: p.getAll("a") });
	`)
	var parsed struct {
		A   string   `json:"a"`
		B   string   `json:"b"`
		All []string `json:"all"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.A != "1" {
		t.Errorf("a = %q, want %q", parsed.A, "1")
	}
	if parsed.B != "2" {
		t.Errorf("b = %q, want %q", parsed.B, "2")
	}
	if len(parsed.All) != 2 {
		t.Errorf("getAll('a') len = %d, want 2", len(parsed.All))
	}
}

func TestTimers(t *testing.T) {
	timers := Timers()
	b := newTestBridge(t, timers)

	evalString(t, b, `
		globalThis._results = [];
		setTimeout(() => _results.push("a"), 0);
		setTimeout(() => _results.push("b"), 0);
		const id = setTimeout(() => _results.push("cancelled"), 0);
		clearTimeout(id);
	`)

	if err := timers.Drain(b.Context()); err != nil {
		t.Fatalf("Drain: %v", err)
	}

	result := evalString(t, b, `JSON.stringify(_results)`)
	if result != `["a","b"]` {
		t.Errorf("timers result = %s, want %s", result, `["a","b"]`)
	}
}

func TestAbortController(t *testing.T) {
	b := newTestBridge(t, Abort())
	result := evalString(t, b, `
		const ctrl = new AbortController();
		const before = ctrl.signal.aborted;
		ctrl.abort();
		const after = ctrl.signal.aborted;
		JSON.stringify({ before, after });
	`)
	var parsed map[string]bool
	json.Unmarshal([]byte(result), &parsed)
	if parsed["before"] != false {
		t.Error("signal should not be aborted before abort()")
	}
	if parsed["after"] != true {
		t.Error("signal should be aborted after abort()")
	}
}

func TestAbortSignalListener(t *testing.T) {
	b := newTestBridge(t, Abort())
	result := evalString(t, b, `
		const ctrl = new AbortController();
		let fired = false;
		ctrl.signal.addEventListener('abort', () => { fired = true; });
		ctrl.abort();
		String(fired);
	`)
	if result != "true" {
		t.Errorf("abort listener fired = %q, want %q", result, "true")
	}
}

func TestEventEmitter(t *testing.T) {
	b := newTestBridge(t, Events())
	result := evalString(t, b, `
		const ee = new EventEmitter();
		const results = [];
		ee.on('data', (x) => results.push(x));
		ee.emit('data', 'hello');
		ee.emit('data', 'world');
		JSON.stringify(results);
	`)
	if result != `["hello","world"]` {
		t.Errorf("EventEmitter = %s, want %s", result, `["hello","world"]`)
	}
}

func TestEventEmitterOnce(t *testing.T) {
	b := newTestBridge(t, Events())
	result := evalString(t, b, `
		const ee = new EventEmitter();
		let count = 0;
		ee.once('x', () => count++);
		ee.emit('x');
		ee.emit('x');
		String(count);
	`)
	if result != "1" {
		t.Errorf("once count = %q, want %q", result, "1")
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom", "test-value")
		fmt.Fprintf(w, `{"message":"hello from server","method":"%s"}`, r.Method)
	}))
	defer srv.Close()

	b := newTestBridge(t, Encoding(), Fetch(FetchClient(srv.Client())))

	val, err := b.Eval("test.js", qjs.Code(fmt.Sprintf(`
		const resp = await fetch("%s/api");
		const data = await resp.json();
		JSON.stringify({
			ok: resp.ok,
			status: resp.status,
			contentType: resp.headers.get("content-type"),
			custom: resp.headers.get("x-custom"),
			message: data.message,
			method: data.method,
		});
	`, srv.URL)), qjs.FlagAsync())
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var result struct {
		OK          bool   `json:"ok"`
		Status      int    `json:"status"`
		ContentType string `json:"contentType"`
		Custom      string `json:"custom"`
		Message     string `json:"message"`
		Method      string `json:"method"`
	}
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		t.Fatalf("parse result %q: %v", val.String(), err)
	}

	if !result.OK {
		t.Error("expected ok=true")
	}
	if result.Status != 200 {
		t.Errorf("status = %d, want 200", result.Status)
	}
	if result.Message != "hello from server" {
		t.Errorf("message = %q", result.Message)
	}
	if result.Method != "GET" {
		t.Errorf("method = %q, want GET", result.Method)
	}
	if result.Custom != "test-value" {
		t.Errorf("x-custom = %q, want 'test-value'", result.Custom)
	}
}

func TestFetchPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"method":"%s","body":%s,"ct":"%s"}`,
			r.Method, string(body), r.Header.Get("Content-Type"))
	}))
	defer srv.Close()

	b := newTestBridge(t, Encoding(), Fetch(FetchClient(srv.Client())))

	val, err := b.Eval("test.js", qjs.Code(fmt.Sprintf(`
		const resp = await fetch("%s/api", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ key: "value" }),
		});
		const data = await resp.json();
		JSON.stringify(data);
	`, srv.URL)), qjs.FlagAsync())
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var result struct {
		Method string `json:"method"`
		Body   struct {
			Key string `json:"key"`
		} `json:"body"`
		CT string `json:"ct"`
	}
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Method != "POST" {
		t.Errorf("method = %q, want POST", result.Method)
	}
	if result.Body.Key != "value" {
		t.Errorf("body.key = %q, want 'value'", result.Body.Key)
	}
	if result.CT != "application/json" {
		t.Errorf("content-type = %q", result.CT)
	}
}

func TestAllPolyfills(t *testing.T) {
	var stdout bytes.Buffer
	con := Console(ConsoleStdout(&stdout), ConsoleStderr(&stdout))
	timers := Timers()

	b := newTestBridge(t, con, Crypto(), Encoding(), URL(), timers, Abort(), Events())

	result := evalString(t, b, `
		// Test all polyfills together
		const uuid = crypto.randomUUID();
		const hash = __node_crypto.createHash('sha256').update('test').digest('hex');
		const encoded = btoa('hello');
		const decoded = atob(encoded);
		const url = new URL('https://example.com/path');
		const ctrl = new AbortController();
		const ee = new EventEmitter();
		let emitted = false;
		ee.once('test', () => { emitted = true; });
		ee.emit('test');
		console.log('integration test');

		JSON.stringify({
			uuid: uuid.length === 36,
			hash: hash.length === 64,
			base64: decoded === 'hello',
			url: url.hostname === 'example.com',
			abort: !ctrl.signal.aborted,
			events: emitted,
		});
	`)

	var checks map[string]bool
	json.Unmarshal([]byte(result), &checks)
	for k, v := range checks {
		if !v {
			t.Errorf("%s check failed", k)
		}
	}

	if !strings.Contains(stdout.String(), "integration test") {
		t.Error("console output missing")
	}
}
