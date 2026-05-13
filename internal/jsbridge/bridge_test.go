package jsbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
	val, err := b.Eval("test.js", code)
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

func TestBridgeCloseContextRetriesSingleCloseWaiter(t *testing.T) {
	b := newTestBridge(t)

	started := make(chan struct{})
	release := make(chan struct{})
	b.Go(func(context.Context) {
		close(started)
		<-release
	})
	<-started

	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := b.CloseContext(closeCtx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want %v", err, context.DeadlineExceeded)
	}
	snap := b.DebugSnapshot()
	if !snap.Closing || snap.Closed || snap.ActiveGoroutines != 1 {
		t.Fatalf("snapshot after timed-out close = %+v, want closing with one active goroutine", snap)
	}

	ran := make(chan struct{})
	b.Go(func(context.Context) { close(ran) })
	select {
	case <-ran:
		t.Fatal("Bridge.Go started work after close began")
	case <-time.After(20 * time.Millisecond):
	}

	retryDeadlineCtx, retryDeadlineCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err = b.CloseContext(retryDeadlineCtx)
	retryDeadlineCancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second CloseContext error = %v, want %v", err, context.DeadlineExceeded)
	}

	close(release)
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := b.CloseContext(retryCtx); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	snap = b.DebugSnapshot()
	if snap.Closing || !snap.Closed || snap.ActiveGoroutines != 0 {
		t.Fatalf("snapshot after successful retry = %+v, want closed with no active goroutines", snap)
	}
}

func TestBridgeCloseContextCancelsExecProcess(t *testing.T) {
	b := newTestBridge(t, Exec())
	startedPath := filepath.Join(t.TempDir(), "exec-started")
	command := fmt.Sprintf("sh -c 'echo started > %s; sleep 10'", startedPath)

	done := make(chan error, 1)
	go func() {
		val, err := b.EvalAsync("exec-close.js", fmt.Sprintf(`(async () => {
			await child_process.exec(%q);
			return "done";
		})()`, command))
		if val != nil {
			val.Free()
		}
		done <- err
	}()

	waitForFile(t, startedPath)
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v snapshot=%+v", err, b.DebugSnapshot())
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("EvalAsync did not return after bridge close canceled exec")
	}
}

func TestBridgeCloseContextCancelsSpawnProcessAndTrackedWaiters(t *testing.T) {
	b := newTestBridge(t, Exec())

	val, err := b.EvalAsync("spawn-close.js", `(async () => {
		const proc = child_process.spawn("sh", ["-c", "echo ready; sleep 10"]);
		const line = await proc.readLine();
		return line;
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync spawn: %v", err)
	}
	if got := val.String(); got != "ready" {
		t.Fatalf("spawn readLine = %q, want ready", got)
	}
	val.Free()
	if got := resourceCount(b, "exec.spawnedProcesses"); got != 1 {
		t.Fatalf("spawned process resources = %d, want 1 snapshot=%+v", got, b.DebugSnapshot())
	}

	waitForActiveBridgeGoroutines(t, b)
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	if snap := b.DebugSnapshot(); snap.ActiveGoroutines != 0 || !snap.Closed {
		t.Fatalf("snapshot after spawn close = %+v, want closed with no active goroutines", snap)
	}
}

func TestBridgeDebugSnapshotTracksGenericResources(t *testing.T) {
	b := newTestBridge(t)

	release := b.TrackResource("test.resource")
	if got := resourceCount(b, "test.resource"); got != 1 {
		t.Fatalf("resource count after track = %d, want 1", got)
	}
	release()
	release()
	if got := resourceCount(b, "test.resource"); got != 0 {
		t.Fatalf("resource count after release = %d, want 0", got)
	}
}

func TestBridgeDebugSnapshotTracksTimerResourcesThroughClose(t *testing.T) {
	b := newTestBridge(t, Timers())

	val, err := b.Eval("timer-resource.js", `setTimeout(function(){}, 10000); "scheduled";`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	val.Free()

	waitForBridgeResource(t, b, "timers.timeouts", 1)
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v snapshot=%+v", err, b.DebugSnapshot())
	}
	if got := resourceCount(b, "timers.timeouts"); got != 0 {
		t.Fatalf("timer resources after close = %d, want 0 snapshot=%+v", got, b.DebugSnapshot())
	}
}

func TestBridgeDebugSnapshotTracksFSHandlesThroughClose(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "open.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	b := newTestBridge(t, Encoding(), Buffer(), Events(), NodeStreams(), FS(root))

	val, err := b.EvalAsync("fs-resource.js", `(async () => {
		globalThis.__openHandle = await fs.promises.open("open.txt", "r");
		return "opened";
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync open: %v", err)
	}
	val.Free()

	if got := resourceCount(b, "fs.fileHandles"); got != 1 {
		t.Fatalf("fs file handle resources = %d, want 1 snapshot=%+v", got, b.DebugSnapshot())
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v snapshot=%+v", err, b.DebugSnapshot())
	}
	if got := resourceCount(b, "fs.fileHandles"); got != 0 {
		t.Fatalf("fs file handle resources after close = %d, want 0 snapshot=%+v", got, b.DebugSnapshot())
	}
}

func TestBridgeDebugSnapshotTracksFetchRequestResources(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	closeRelease := func() {
		releaseOnce.Do(func() { close(release) })
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(func() {
		closeRelease()
		srv.Close()
	})

	b := newTestBridge(t, Encoding(), Fetch())
	done := make(chan error, 1)
	go func() {
		val, err := b.EvalAsync("fetch-resource.js", fmt.Sprintf(`(async () => {
			const res = await fetch(%q);
			return await res.text();
		})()`, srv.URL))
		if val != nil {
			val.Free()
		}
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("server did not receive fetch")
	}
	waitForBridgeResource(t, b, "fetch.requests", 1)

	closeRelease()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EvalAsync fetch: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fetch")
	}
}

func TestBridgeCloseContextCancelsPendingFetchRequest(t *testing.T) {
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	var enteredOnce sync.Once
	var cancelledOnce sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enteredOnce.Do(func() { close(entered) })
		<-r.Context().Done()
		cancelledOnce.Do(func() { close(cancelled) })
	}))
	t.Cleanup(srv.Close)

	b := newTestBridge(t, Encoding(), Fetch())
	done := make(chan error, 1)
	go func() {
		val, err := b.EvalAsync("fetch-close-cancel.js", fmt.Sprintf(`(async () => {
			const res = await fetch(%q);
			return await res.text();
		})()`, srv.URL))
		if val != nil {
			val.Free()
		}
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("server did not receive fetch")
	}
	waitForBridgeResource(t, b, "fetch.requests", 1)

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v snapshot=%+v", err, b.DebugSnapshot())
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("fetch request context was not cancelled by bridge close")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("EvalAsync fetch did not return after bridge close")
	}
	snapshot := b.DebugSnapshot()
	if snapshot.ActiveGoroutines != 0 || !snapshot.Closed {
		t.Fatalf("snapshot after pending fetch close = %+v, want closed with no active goroutines", snapshot)
	}
	if got := resourceCount(b, "fetch.requests"); got != 0 {
		t.Fatalf("fetch request resources after close = %d, want 0 snapshot=%+v", got, snapshot)
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForBridgeResource(t *testing.T, b *Bridge, key string, wantAtLeast int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if got := resourceCount(b, key); got >= wantAtLeast {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for resource %s >= %d; snapshot=%+v", key, wantAtLeast, b.DebugSnapshot())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func resourceCount(b *Bridge, key string) int {
	if b == nil {
		return 0
	}
	return b.DebugSnapshot().Resources[key]
}

func waitForActiveBridgeGoroutines(t *testing.T, b *Bridge) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if snap := b.DebugSnapshot(); snap.ActiveGoroutines > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for active bridge goroutines")
		}
		time.Sleep(10 * time.Millisecond)
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
	result := evalString(t, b, `crypto.createHash('sha256').update('hello world').digest('hex')`)
	// Known SHA-256 of "hello world"
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if result != want {
		t.Errorf("sha256 = %q, want %q", result, want)
	}
}

func TestCryptoHmac(t *testing.T) {
	b := newTestBridge(t, Crypto())
	result := evalString(t, b, `crypto.createHmac('sha256', 'secret').update('hello').digest('hex')`)
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

func TestNodeURLHelpers(t *testing.T) {
	b := newTestBridge(t, URL())
	result := evalString(t, b, `
		JSON.stringify({
			filePath: node_url.fileURLToPath("file:///tmp/x.txt"),
			fileURL: node_url.pathToFileURL("/tmp/x.txt").toString(),
			formatted: node_url.format({ protocol: "https:", host: "example.com", pathname: "/x" }),
			parsed: node_url.parse("https://example.com/x?q=1").path,
			resolved: node_url.resolve("https://example.com/a/", "../b")
		});
	`)
	expected := `{"filePath":"/tmp/x.txt","fileURL":"file:///tmp/x.txt","formatted":"https://example.com/x","parsed":"/x?q=1","resolved":"https://example.com/b"}`
	if result != expected {
		t.Errorf("node_url helpers = %s", result)
	}
}

func TestTimers(t *testing.T) {
	timers := Timers()
	b := newTestBridge(t, timers)

	// Use EvalAsync with async IIFE so microtasks (queueMicrotask) get processed via Await
	val, err := b.EvalAsync("timer-test.js", `(async () => {
		globalThis._results = [];
		setTimeout(() => _results.push("a"), 0);
		setTimeout(() => _results.push("b"), 0);
		const id = setTimeout(() => _results.push("cancelled"), 0);
		clearTimeout(id);
		// Give microtasks a chance to run
		await new Promise(r => setTimeout(r, 0));
		return JSON.stringify(_results);
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	result := val.String()
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

	// Use EvalAsync — Go-level Await that processes ctx.Schedule'd work
	val, err := b.EvalAsync("test.js", fmt.Sprintf(`(async () => {
		const resp = await fetch("%s/api");
		const data = await resp.json();
		return JSON.stringify({
			ok: resp.ok,
			status: resp.status,
			contentType: resp.headers.get("content-type"),
			custom: resp.headers.get("x-custom"),
			message: data.message,
			method: data.method,
		});
	})()`, srv.URL))
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
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

	val, err := b.EvalAsync("test.js", fmt.Sprintf(`(async () => {
		const resp = await fetch("%s/api", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ key: "value" }),
		});
		const data = await resp.json();
		return JSON.stringify(data);
	})()`, srv.URL))
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
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

func TestFetchConcurrent(t *testing.T) {
	// Test that multiple fetch calls within one EvalAsync run concurrently
	// (not sequentially) by using Promise.all with a slow server.
	callCount := int32(0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		// Small delay to ensure requests overlap
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"n":%d}`, n)
	}))
	defer srv.Close()

	b := newTestBridge(t, Encoding(), Fetch(FetchClient(srv.Client())))

	start := time.Now()
	val, err := b.EvalAsync("test.js", fmt.Sprintf(`(async () => {
		// Launch 3 fetches concurrently via Promise.all
		const results = await Promise.all([
			fetch("%[1]s/1").then(r => r.json()),
			fetch("%[1]s/2").then(r => r.json()),
			fetch("%[1]s/3").then(r => r.json()),
		]);
		return JSON.stringify(results.map(r => r.n));
	})()`, srv.URL))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var nums []int
	if err := json.Unmarshal([]byte(val.String()), &nums); err != nil {
		t.Fatalf("parse: %v (raw: %s)", err, val.String())
	}

	if len(nums) != 3 {
		t.Errorf("expected 3 results, got %d", len(nums))
	}
	t.Logf("Concurrent fetch results: %v in %s", nums, elapsed.Round(time.Millisecond))

	// If truly concurrent: ~50ms (parallel). If sequential: ~150ms (3 * 50ms).
	// Allow generous margin but verify it's faster than sequential.
	if elapsed > 140*time.Millisecond {
		t.Errorf("expected concurrent execution (~50ms), took %s — fetches may be sequential", elapsed)
	}
}

func TestStructuredClone(t *testing.T) {
	b := newTestBridge(t, StructuredClone())
	result := evalString(t, b, `
		const obj = { a: 1, b: [2, 3], c: { d: "hello" } };
		const cloned = structuredClone(obj);
		cloned.a = 99;
		cloned.b.push(4);
		cloned.c.d = "changed";
		JSON.stringify({ original: obj, cloned: cloned });
	`)
	var parsed struct {
		Original struct {
			A int   `json:"a"`
			B []int `json:"b"`
			C struct {
				D string `json:"d"`
			} `json:"c"`
		} `json:"original"`
		Cloned struct {
			A int   `json:"a"`
			B []int `json:"b"`
			C struct {
				D string `json:"d"`
			} `json:"c"`
		} `json:"cloned"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Original should be unchanged
	if parsed.Original.A != 1 {
		t.Errorf("original.a = %d, want 1", parsed.Original.A)
	}
	if len(parsed.Original.B) != 2 {
		t.Errorf("original.b len = %d, want 2", len(parsed.Original.B))
	}
	if parsed.Original.C.D != "hello" {
		t.Errorf("original.c.d = %q, want %q", parsed.Original.C.D, "hello")
	}
	// Cloned should have mutations
	if parsed.Cloned.A != 99 {
		t.Errorf("cloned.a = %d, want 99", parsed.Cloned.A)
	}
	if len(parsed.Cloned.B) != 3 {
		t.Errorf("cloned.b len = %d, want 3", len(parsed.Cloned.B))
	}
	if parsed.Cloned.C.D != "changed" {
		t.Errorf("cloned.c.d = %q, want %q", parsed.Cloned.C.D, "changed")
	}
}

// --- Phase 2: fs ---

func TestFsReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	b := newTestBridge(t, FS(tmpDir))

	content := "Hello from JS via Go bridge!"

	val, err := b.EvalAsync("test.js", fmt.Sprintf(`(function() {
		fs.writeFileSync("test.txt", %q);
		return fs.readFileSync("test.txt", "utf8");
	})()`, content))
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	if val.String() != content {
		t.Errorf("readFileSync = %q, want %q", val.String(), content)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if string(data) != content {
		t.Errorf("Go read = %q, want %q", string(data), content)
	}
}

func TestFsReaddir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "alpha.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "beta.txt"), []byte("b"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	b := newTestBridge(t, FS(tmpDir))

	val, err := b.EvalAsync("test.js", `(function() {
		var names = fs.readdirSync(".");
		return JSON.stringify({ count: names.length });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var parsed struct {
		Count int `json:"count"`
	}
	json.Unmarshal([]byte(val.String()), &parsed)
	if parsed.Count != 3 {
		t.Errorf("count = %d, want 3", parsed.Count)
	}
}

func TestFsStat(t *testing.T) {
	tmpDir := t.TempDir()
	content := "stat test content"
	os.WriteFile(filepath.Join(tmpDir, "statfile.txt"), []byte(content), 0644)

	b := newTestBridge(t, FS(tmpDir))

	val, err := b.EvalAsync("test.js", `(function() {
		var s = fs.statSync("statfile.txt");
		return JSON.stringify({ size: s.size, isFile: s.isFile(), isDir: s.isDirectory() });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var parsed struct {
		Size   int  `json:"size"`
		IsFile bool `json:"isFile"`
		IsDir  bool `json:"isDir"`
	}
	json.Unmarshal([]byte(val.String()), &parsed)
	if parsed.Size != len(content) {
		t.Errorf("size = %d, want %d", parsed.Size, len(content))
	}
	if !parsed.IsFile {
		t.Error("expected isFile=true")
	}
	if parsed.IsDir {
		t.Error("expected isDirectory=false")
	}
}

func TestFsMkdirAndRm(t *testing.T) {
	tmpDir := t.TempDir()
	b := newTestBridge(t, FS(tmpDir))

	val, err := b.EvalAsync("test.js", `(function() {
		fs.mkdirSync("a/b/c", {recursive: true});
		fs.rmSync("a", {recursive: true});
		return "done";
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	if _, err := os.Stat(filepath.Join(tmpDir, "a")); !os.IsNotExist(err) {
		t.Error("expected directory to be removed")
	}
}

func TestFsReadBinary(t *testing.T) {
	tmpDir := t.TempDir()
	// Write binary data with null bytes — would corrupt if treated as string
	binData := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x0D, 0x0A, 0x1A, 0x0A, 0xFF, 0xFE}
	os.WriteFile(filepath.Join(tmpDir, "test.bin"), binData, 0644)

	b := newTestBridge(t, Encoding(), Buffer(), FS(tmpDir))

	// readFileSync without encoding should return a Buffer, not a string
	val, err := b.EvalAsync("test.js", `(function() {
		var buf = fs.readFileSync("test.bin");
		if (typeof buf === "string") return "GOT_STRING";
		if (!buf._isBuffer && !Buffer.isBuffer(buf)) return "NOT_BUFFER: " + typeof buf;
		return "OK:len=" + buf.length;
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	result := val.String()
	if result != fmt.Sprintf("OK:len=%d", len(binData)) {
		t.Errorf("readFileSync binary = %q, want OK:len=%d", result, len(binData))
	}
}

func TestFsReadBinaryPreservesBytes(t *testing.T) {
	tmpDir := t.TempDir()
	// Write known bytes, read back, verify round-trip
	os.WriteFile(filepath.Join(tmpDir, "data.bin"), []byte{0x01, 0x02, 0x03, 0xFE, 0xFF}, 0644)

	b := newTestBridge(t, Encoding(), Buffer(), FS(tmpDir))

	val, err := b.EvalAsync("test.js", `(function() {
		var buf = fs.readFileSync("data.bin");
		var bytes = [];
		for (var i = 0; i < buf.length; i++) bytes.push(buf[i]);
		return JSON.stringify(bytes);
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	expected := "[1,2,3,254,255]"
	if val.String() != expected {
		t.Errorf("binary roundtrip = %q, want %q", val.String(), expected)
	}
}

func TestFsWriteBinaryRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	b := newTestBridge(t, Encoding(), Buffer(), FS(tmpDir))

	// Write binary data via Buffer, read it back, verify bytes preserved
	val, err := b.EvalAsync("test.js", `(function() {
		var buf = Buffer.from([0x89, 0x50, 0x4E, 0x47, 0x00, 0xFF, 0xFE]);
		fs.writeFileSync("binary.bin", buf);
		var readBack = fs.readFileSync("binary.bin");
		var bytes = [];
		for (var i = 0; i < readBack.length; i++) bytes.push(readBack[i]);
		return JSON.stringify(bytes);
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	expected := "[137,80,78,71,0,255,254]"
	if val.String() != expected {
		t.Errorf("binary write roundtrip = %q, want %q", val.String(), expected)
	}

	// Also verify on disk
	data, _ := os.ReadFile(filepath.Join(tmpDir, "binary.bin"))
	if len(data) != 7 || data[0] != 0x89 || data[4] != 0x00 || data[6] != 0xFE {
		t.Errorf("on-disk bytes wrong: %v", data)
	}
}

func TestFsReadWithEncodingReturnsString(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "text.txt"), []byte("hello"), 0644)

	b := newTestBridge(t, Encoding(), Buffer(), FS(tmpDir))

	val, err := b.EvalAsync("test.js", `(function() {
		var result = fs.readFileSync("text.txt", "utf8");
		return typeof result + ":" + result;
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	if val.String() != "string:hello" {
		t.Errorf("readFileSync utf8 = %q, want 'string:hello'", val.String())
	}
}

func TestFsCreateReadStreamPipe(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "stream-src.txt"), []byte("stream content here"), 0644)

	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), FS(tmpDir))

	// createReadStream should return a Readable with pipe(), emit 'open', track bytesRead
	val, err := b.EvalAsync("test.js", `(async function() {
		var rs = fs.createReadStream("stream-src.txt");
		var hasPipe = typeof rs.pipe === "function";
		var hasPending = rs.pending === true;
		var gotOpen = false;
		var chunks = [];
		return new Promise(function(resolve) {
			rs.on("open", function() { gotOpen = true; });
			rs.on("data", function(chunk) { chunks.push(chunk); });
			rs.on("end", function() {
				// Chunks are Uint8Array (binary-safe); concat then decode.
				var total = 0;
				for (var i = 0; i < chunks.length; i++) total += chunks[i].byteLength;
				var out = new Uint8Array(total);
				var off = 0;
				for (var j = 0; j < chunks.length; j++) {
					out.set(chunks[j], off);
					off += chunks[j].byteLength;
				}
				resolve(JSON.stringify({
					hasPipe: hasPipe, hasPending: hasPending, gotOpen: gotOpen,
					data: new TextDecoder().decode(out), bytesRead: rs.bytesRead
				}));
			});
			rs.on("error", function(e) { resolve("ERROR:" + e.message); });
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	result := val.String()
	if strings.HasPrefix(result, "ERROR:") {
		t.Fatalf("createReadStream error: %s", result)
	}
	var parsed struct {
		HasPipe    bool   `json:"hasPipe"`
		HasPending bool   `json:"hasPending"`
		GotOpen    bool   `json:"gotOpen"`
		Data       string `json:"data"`
		BytesRead  int    `json:"bytesRead"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if !parsed.HasPipe {
		t.Error("createReadStream should return a Readable with pipe()")
	}
	if !parsed.HasPending {
		t.Error("createReadStream should start with pending=true")
	}
	if !parsed.GotOpen {
		t.Error("createReadStream should emit 'open' event")
	}
	if parsed.Data != "stream content here" {
		t.Errorf("stream data = %q, want 'stream content here'", parsed.Data)
	}
	if parsed.BytesRead != 19 {
		t.Errorf("bytesRead = %d, want 19", parsed.BytesRead)
	}
}

func TestFsCreateWriteStreamAsyncOpen(t *testing.T) {
	tmpDir := t.TempDir()

	b := newTestBridge(t, Events(), NodeStreams(), FS(tmpDir))

	// createWriteStream opens file async, emits 'open', then flushes buffered writes
	val, err := b.EvalAsync("test.js", `(async function() {
		var ws = fs.createWriteStream("async-ws.txt");
		var hasWrite = typeof ws.write === "function";
		var hasEnd = typeof ws.end === "function";
		var hasPending = ws.pending === true;
		ws.write("hello ");
		ws.write("world");
		return new Promise(function(resolve) {
			ws.on("open", function(fd) {
				ws.end(function() {
					resolve(JSON.stringify({hasWrite: hasWrite, hasEnd: hasEnd, hasPending: hasPending, gotOpen: true, fd: fd}));
				});
			});
			ws.on("error", function(e) { resolve("ERROR:" + e.message); });
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	result := val.String()
	if strings.HasPrefix(result, "ERROR:") {
		t.Fatalf("createWriteStream error: %s", result)
	}
	var parsed struct {
		HasWrite   bool `json:"hasWrite"`
		HasEnd     bool `json:"hasEnd"`
		HasPending bool `json:"hasPending"`
		GotOpen    bool `json:"gotOpen"`
		FD         int  `json:"fd"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if !parsed.HasWrite {
		t.Error("createWriteStream should have write()")
	}
	if !parsed.HasEnd {
		t.Error("createWriteStream should have end()")
	}
	if !parsed.HasPending {
		t.Error("createWriteStream should start with pending=true")
	}
	if !parsed.GotOpen {
		t.Error("createWriteStream should emit 'open' event")
	}
	if parsed.FD <= 0 {
		t.Errorf("createWriteStream 'open' fd = %d, want >0", parsed.FD)
	}

	// Verify file was written (writes buffered until open, then flushed)
	data, err := os.ReadFile(filepath.Join(tmpDir, "async-ws.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("file content = %q, want 'hello world'", string(data))
	}
}

// --- Phase 2: path ---

func TestPathJoin(t *testing.T) {
	b := newTestBridge(t, Path())

	tests := []struct {
		code string
		want string
	}{
		{`path.join("/usr", "local", "bin")`, "/usr/local/bin"},
		{`path.join("foo", "bar", "baz.txt")`, "foo/bar/baz.txt"},
		{`path.join("/a", "b", "..", "c")`, "/a/c"},
	}
	for _, tt := range tests {
		result := evalString(t, b, tt.code)
		if result != tt.want {
			t.Errorf("%s = %q, want %q", tt.code, result, tt.want)
		}
	}
}

func TestPathResolve(t *testing.T) {
	b := newTestBridge(t, Path())

	result := evalString(t, b, `path.resolve("/absolute", "path")`)
	if result != filepath.Join("/absolute", "path") {
		t.Errorf("resolve = %q, want %q", result, filepath.Join("/absolute", "path"))
	}

	// Relative should produce absolute path
	result2 := evalString(t, b, `path.resolve(".", "src")`)
	if !filepath.IsAbs(result2) {
		t.Errorf("expected absolute path, got %q", result2)
	}
}

func TestPathComponents(t *testing.T) {
	b := newTestBridge(t, Path())

	result := evalString(t, b, `
		JSON.stringify({
			dir: path.dirname("/usr/local/bin/prog.tar.gz"),
			base: path.basename("/usr/local/bin/prog.tar.gz"),
			ext: path.extname("/usr/local/bin/prog.tar.gz"),
		});
	`)

	var parsed struct {
		Dir  string `json:"dir"`
		Base string `json:"base"`
		Ext  string `json:"ext"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if parsed.Dir != "/usr/local/bin" {
		t.Errorf("dirname = %q, want %q", parsed.Dir, "/usr/local/bin")
	}
	if parsed.Base != "prog.tar.gz" {
		t.Errorf("basename = %q, want %q", parsed.Base, "prog.tar.gz")
	}
	if parsed.Ext != ".gz" {
		t.Errorf("extname = %q, want %q", parsed.Ext, ".gz")
	}
}

// --- Phase 3: child_process ---

func TestExec(t *testing.T) {
	b := newTestBridge(t, Exec())

	val, err := b.EvalAsync("test.js", `(async () => {
		const r = await child_process.exec("echo hello world");
		return JSON.stringify({ stdout: r.stdout.trim(), exit: r.exitCode });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	result := val.String()

	var parsed struct {
		Stdout string `json:"stdout"`
		Exit   int    `json:"exit"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if parsed.Stdout != "hello world" {
		t.Errorf("stdout = %q, want %q", parsed.Stdout, "hello world")
	}
	if parsed.Exit != 0 {
		t.Errorf("exitCode = %d, want 0", parsed.Exit)
	}
}

func TestExecNonZeroExit(t *testing.T) {
	b := newTestBridge(t, Exec())

	val, err := b.EvalAsync("test.js", `(async () => {
		const r = await child_process.exec("exit 42");
		return JSON.stringify({ exit: r.exitCode });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	result := val.String()

	var parsed struct {
		Exit int `json:"exit"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if parsed.Exit != 42 {
		t.Errorf("exitCode = %d, want 42", parsed.Exit)
	}
}

func TestSpawnStreaming(t *testing.T) {
	b := newTestBridge(t, Exec())

	val, err := b.EvalAsync("test.js", `(async () => {
		const proc = child_process.spawn("sh", ["-c", "for i in 1 2 3 4 5; do echo line_$i; done"]);
		const lines = [];
		while (true) {
			const line = await proc.readLine();
			if (line === null) break;
			lines.push(line);
		}
		const exit = await proc.wait();
		return JSON.stringify({ lines, exit, count: lines.length });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	result := val.String()

	var parsed struct {
		Lines []string `json:"lines"`
		Exit  int      `json:"exit"`
		Count int      `json:"count"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if parsed.Count != 5 {
		t.Errorf("count = %d, want 5", parsed.Count)
	}
	if parsed.Exit != 0 {
		t.Errorf("exit = %d, want 0", parsed.Exit)
	}
	for i, line := range parsed.Lines {
		want := fmt.Sprintf("line_%d", i+1)
		if line != want {
			t.Errorf("line[%d] = %q, want %q", i, line, want)
		}
	}
}

// --- Phase 3: process ---

func TestProcessEnv(t *testing.T) {
	b := newTestBridge(t, Process())

	result := evalString(t, b, `
		const pathVal = process.env.PATH;
		process.env.QJS_TEST_VAR = "hello_from_js";
		const custom = process.env.QJS_TEST_VAR;
		const missing = process.env.QJS_NONEXISTENT_12345;
		JSON.stringify({ hasPath: pathVal.length > 0, custom, missing });
	`)
	defer os.Unsetenv("QJS_TEST_VAR")

	var parsed struct {
		HasPath bool   `json:"hasPath"`
		Custom  string `json:"custom"`
		Missing string `json:"missing"`
	}
	json.Unmarshal([]byte(result), &parsed)
	if !parsed.HasPath {
		t.Error("expected PATH to be non-empty")
	}
	if parsed.Custom != "hello_from_js" {
		t.Errorf("custom = %q, want %q", parsed.Custom, "hello_from_js")
	}
	if parsed.Missing != "" {
		t.Errorf("missing = %q, want empty", parsed.Missing)
	}

	// Verify from Go side
	if os.Getenv("QJS_TEST_VAR") != "hello_from_js" {
		t.Error("Go side: QJS_TEST_VAR not set")
	}
}

func TestProcessCwd(t *testing.T) {
	b := newTestBridge(t, Process())

	result := evalString(t, b, `process.cwd()`)
	goCwd, _ := os.Getwd()
	if result != goCwd {
		t.Errorf("cwd = %q, want %q", result, goCwd)
	}
}

// --- Fetch Response.body streaming ---

func TestFetchResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "streaming body test")
	}))
	defer srv.Close()

	b := newTestBridge(t, Encoding(), Streams(), Fetch(FetchClient(srv.Client())))

	val, err := b.EvalAsync("test.js", fmt.Sprintf(`(async () => {
		const resp = await fetch("%s/data");
		const reader = resp.body.getReader();
		const chunks = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			chunks.push(value);
		}
		// value should be Uint8Array, decode it
		const dec = new TextDecoder();
		const text = chunks.map(c => dec.decode(c)).join("");
		return text;
	})()`, srv.URL))
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	if val.String() != "streaming body test" {
		t.Errorf("Response.body = %q, want %q", val.String(), "streaming body test")
	}
}

func TestFetchResponseBodyPipeThrough(t *testing.T) {
	// This is the exact pattern AI SDK uses: response.body.pipeThrough(new TextDecoderStream())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: hello\ndata: world\n")
	}))
	defer srv.Close()

	b := newTestBridge(t, Encoding(), Streams(), Fetch(FetchClient(srv.Client())))

	val, err := b.EvalAsync("test.js", fmt.Sprintf(`(async () => {
		const resp = await fetch("%s/stream");
		const textStream = resp.body.pipeThrough(new TextDecoderStream());
		const reader = textStream.getReader();
		const parts = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			parts.push(value);
		}
		return parts.join("");
	})()`, srv.URL))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	want := "data: hello\ndata: world\n"
	if val.String() != want {
		t.Errorf("pipeThrough = %q, want %q", val.String(), want)
	}
}

// --- Streams ---

func TestReadableStreamBasic(t *testing.T) {
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.EvalAsync("test.js", `(async () => {
		const stream = new ReadableStream({
			start(controller) {
				controller.enqueue("hello");
				controller.enqueue("world");
				controller.close();
			}
		});
		const reader = stream.getReader();
		const chunks = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			chunks.push(value);
		}
		return JSON.stringify(chunks);
	})()`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var chunks []string
	json.Unmarshal([]byte(val.String()), &chunks)
	if len(chunks) != 2 || chunks[0] != "hello" || chunks[1] != "world" {
		t.Errorf("chunks = %v, want [hello, world]", chunks)
	}
}

func TestReadableStreamLocked(t *testing.T) {
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.Eval("test.js", `
		const stream = new ReadableStream({ start(c) { c.close(); } });
		stream.getReader();
		let threw = false;
		try { stream.getReader(); } catch(e) { threw = true; }
		JSON.stringify({ locked: stream.locked, threw });
	`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var result struct {
		Locked bool `json:"locked"`
		Threw  bool `json:"threw"`
	}
	json.Unmarshal([]byte(val.String()), &result)
	if !result.Locked {
		t.Error("expected stream to be locked")
	}
	if !result.Threw {
		t.Error("expected getReader to throw on locked stream")
	}
}

func TestTransformStream(t *testing.T) {
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.EvalAsync("test.js", `(async () => {
		const input = new ReadableStream({
			start(controller) {
				controller.enqueue("hello");
				controller.enqueue("world");
				controller.close();
			}
		});

		const upper = new TransformStream({
			transform(chunk, controller) {
				controller.enqueue(chunk.toUpperCase());
			}
		});

		const output = input.pipeThrough(upper);
		const reader = output.getReader();
		const results = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			results.push(value);
		}
		return JSON.stringify(results);
	})()`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var results []string
	json.Unmarshal([]byte(val.String()), &results)
	if len(results) != 2 || results[0] != "HELLO" || results[1] != "WORLD" {
		t.Errorf("results = %v, want [HELLO, WORLD]", results)
	}
}

func TestWritableStream(t *testing.T) {
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.EvalAsync("test.js", `(async () => {
		const chunks = [];
		const ws = new WritableStream({
			write(chunk) { chunks.push(chunk); },
		});
		const writer = ws.getWriter();
		await writer.write("a");
		await writer.write("b");
		await writer.close();
		return JSON.stringify(chunks);
	})()`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var chunks []string
	json.Unmarshal([]byte(val.String()), &chunks)
	if len(chunks) != 2 || chunks[0] != "a" || chunks[1] != "b" {
		t.Errorf("chunks = %v, want [a, b]", chunks)
	}
}

func TestTextDecoderStream(t *testing.T) {
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.EvalAsync("test.js", `(async () => {
		const encoder = new TextEncoder();
		const bytes = encoder.encode("hello streams");

		const input = new ReadableStream({
			start(controller) {
				controller.enqueue(bytes);
				controller.close();
			}
		});

		const output = input.pipeThrough(new TextDecoderStream());
		const reader = output.getReader();
		const parts = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			parts.push(value);
		}
		return parts.join("");
	})()`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	if val.String() != "hello streams" {
		t.Errorf("TextDecoderStream = %q, want %q", val.String(), "hello streams")
	}
}

func TestStreamPipeToWritable(t *testing.T) {
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.EvalAsync("test.js", `(async () => {
		const collected = [];
		const readable = new ReadableStream({
			start(controller) {
				controller.enqueue("chunk1");
				controller.enqueue("chunk2");
				controller.enqueue("chunk3");
				controller.close();
			}
		});
		const writable = new WritableStream({
			write(chunk) { collected.push(chunk); },
		});
		await readable.pipeTo(writable);
		return JSON.stringify(collected);
	})()`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var collected []string
	json.Unmarshal([]byte(val.String()), &collected)
	if len(collected) != 3 {
		t.Errorf("collected = %v, want 3 chunks", collected)
	}
}

func TestStreamPipeThroughChain(t *testing.T) {
	// Simulates the AI SDK pattern: ReadableStream -> TextDecoderStream -> TransformStream
	b := newTestBridge(t, Encoding(), Streams())

	val, err := b.EvalAsync("test.js", `(async () => {
		const encoder = new TextEncoder();
		const lines = ["data: hello\n", "data: world\n", "data: [DONE]\n"];

		const input = new ReadableStream({
			start(controller) {
				for (const line of lines) {
					controller.enqueue(encoder.encode(line));
				}
				controller.close();
			}
		});

		// Chain: bytes -> text -> parse SSE lines
		const sseParser = new TransformStream({
			transform(chunk, controller) {
				const trimmed = chunk.trim();
				if (trimmed.startsWith("data: ")) {
					const data = trimmed.slice(6);
					if (data !== "[DONE]") {
						controller.enqueue(data);
					}
				}
			}
		});

		const output = input
			.pipeThrough(new TextDecoderStream())
			.pipeThrough(sseParser);

		const reader = output.getReader();
		const tokens = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			tokens.push(value);
		}
		return JSON.stringify(tokens);
	})()`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var tokens []string
	json.Unmarshal([]byte(val.String()), &tokens)
	if len(tokens) != 2 || tokens[0] != "hello" || tokens[1] != "world" {
		t.Errorf("SSE tokens = %v, want [hello, world]", tokens)
	}
}

func TestAllPolyfills(t *testing.T) {
	var stdout bytes.Buffer
	con := Console(ConsoleStdout(&stdout), ConsoleStderr(&stdout))
	timers := Timers()

	b := newTestBridge(t, con, Crypto(), Encoding(), URL(), timers, Abort(), Events(), StructuredClone())

	result := evalString(t, b, `
		// Test all polyfills together
		const uuid = crypto.randomUUID();
		const hash = crypto.createHash('sha256').update('test').digest('hex');
		const encoded = btoa('hello');
		const decoded = atob(encoded);
		const url = new URL('https://example.com/path');
		const ctrl = new AbortController();
		const ee = new EventEmitter();
		let emitted = false;
		ee.once('test', () => { emitted = true; });
		ee.emit('test');
		console.log('integration test');

		const orig = { x: 1 };
		const clone = structuredClone(orig);
		clone.x = 2;

		JSON.stringify({
			uuid: uuid.length === 36,
			hash: hash.length === 64,
			base64: decoded === 'hello',
			url: url.hostname === 'example.com',
			abort: !ctrl.signal.aborted,
			events: emitted,
			structuredClone: orig.x === 1 && clone.x === 2,
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
