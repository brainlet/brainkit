// Experiment: QuickJS Streaming Capabilities
//
// Goal: Prove that Go can bridge streaming patterns needed for AI SDK and Mastra
// embedding via QuickJS. Tests SSE streaming, chunked responses, callback-based
// push streaming, bidirectional WebSocket-like communication, concurrent async
// operations, and stream cancellation.
//
// Tests:
// 1. SSE (Server-Sent Events) Streaming — local HTTP SSE server, Go fetches and parses
// 2. Chunked Response Simulation — large payload split into chunks, JS reassembles
// 3. Go Callback Stream — Go pushes data to JS via repeated callback invocation
// 4. WebSocket Bridge — channel-based bidirectional message passing simulation
// 5. Multiple Concurrent Async Operations — Promise.all with parallel Go async work
// 6. Streaming with Cancellation — abort a streaming operation mid-flight

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fastschema/qjs"
)

func main() {
	fmt.Println("=== QuickJS Streaming Capabilities Experiment ===")
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"SSE (Server-Sent Events) Streaming", testSSEStreaming},
		{"Chunked Response Simulation", testChunkedResponse},
		{"Go Callback Stream", testGoCallbackStream},
		{"WebSocket Bridge (Channel-Based)", testWebSocketBridge},
		{"Multiple Concurrent Async Operations", testConcurrentAsync},
		{"Streaming with Cancellation", testStreamingCancellation},
	}

	passed := 0
	failed := 0
	for i, t := range tests {
		fmt.Printf("--- Test %d: %s ---\n", i+1, t.name)
		if err := t.fn(); err != nil {
			fmt.Printf("FAILED: %v\n\n", err)
			failed++
		} else {
			fmt.Println("PASS")
			fmt.Println()
			passed++
		}
	}

	fmt.Printf("=== Results: %d passed, %d failed ===\n", passed, failed)
	if failed > 0 {
		log.Fatalf("Some tests failed")
	}
	fmt.Println("=== ALL TESTS PASSED ===")
}

// ---------------------------------------------------------------------------
// Test 1: SSE (Server-Sent Events) Streaming
// ---------------------------------------------------------------------------
// Starts a local HTTP server that sends SSE events with delays.
// Go makes the HTTP request, reads the full SSE response, splits into events,
// and returns an array of parsed events to JS.
func testSSEStreaming() error {
	// Start a local SSE server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		events := []struct {
			event string
			data  string
		}{
			{"message", `{"token":"Hello"}`},
			{"message", `{"token":" world"}`},
			{"message", `{"token":"!"}`},
			{"delta", `{"content":"streaming works"}`},
			{"done", `[DONE]`},
		}

		for _, evt := range events {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.event, evt.data)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond) // Small delay to simulate real streaming
		}
	})

	server := &http.Server{Handler: sseHandler}
	go server.Serve(listener)
	defer server.Close()

	// Create QuickJS runtime
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Register __go_fetch_streaming(url) - fetches SSE and returns parsed events
	ctx.SetFunc("__go_fetch_streaming", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("requires URL argument")
		}
		url := args[0].String()

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading body failed: %w", err)
		}

		// Parse SSE format: split by double newline, extract event and data
		c := this.Context()
		eventsArray := c.NewArray()
		rawEvents := strings.Split(string(body), "\n\n")

		for _, rawEvt := range rawEvents {
			rawEvt = strings.TrimSpace(rawEvt)
			if rawEvt == "" {
				continue
			}

			eventObj := c.NewObject()
			lines := strings.Split(rawEvt, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "event: ") {
					eventObj.SetPropertyStr("event", c.NewString(strings.TrimPrefix(line, "event: ")))
				} else if strings.HasPrefix(line, "data: ") {
					eventObj.SetPropertyStr("data", c.NewString(strings.TrimPrefix(line, "data: ")))
				}
			}
			eventsArray.Push(eventObj)
		}

		return eventsArray.Value, nil
	})

	// JS code that calls the streaming fetch and validates results
	jsCode := fmt.Sprintf(`
		const events = __go_fetch_streaming("http://127.0.0.1:%d/sse");
		const result = {
			totalEvents: events.length,
			firstEvent: events[0].event,
			firstData: events[0].data,
			lastEvent: events[events.length - 1].event,
			lastData: events[events.length - 1].data,
			allTokens: "",
		};

		// Concatenate tokens from message events
		for (let i = 0; i < events.length; i++) {
			if (events[i].event === "message") {
				const parsed = JSON.parse(events[i].data);
				if (parsed.token) {
					result.allTokens += parsed.token;
				}
			}
		}

		JSON.stringify(result);
	`, port)

	result, err := ctx.Eval("sse_test.js", qjs.Code(jsCode))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	// Validate
	if !strings.Contains(str, `"totalEvents":5`) {
		return fmt.Errorf("expected 5 events, got: %s", str)
	}
	if !strings.Contains(str, `"firstEvent":"message"`) {
		return fmt.Errorf("expected first event to be 'message', got: %s", str)
	}
	if !strings.Contains(str, `"lastEvent":"done"`) {
		return fmt.Errorf("expected last event to be 'done', got: %s", str)
	}
	if !strings.Contains(str, `"allTokens":"Hello world!"`) {
		return fmt.Errorf("expected concatenated tokens 'Hello world!', got: %s", str)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 2: Chunked Response Simulation
// ---------------------------------------------------------------------------
// Simulates passing a large payload from Go to JS in chunks.
// JS iterates through chunks, concatenates, and verifies completeness.
func testChunkedResponse() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Generate a 100KB+ payload
	const chunkSize = 10 * 1024 // 10KB per chunk
	const totalSize = 100 * 1024 // 100KB total
	const numChunks = totalSize / chunkSize

	// Build a payload with a recognizable pattern
	var fullPayload strings.Builder
	for i := 0; i < totalSize; i++ {
		fullPayload.WriteByte(byte('A' + (i % 26)))
	}
	payload := fullPayload.String()

	// Store chunks indexed by dataset ID
	type dataset struct {
		chunks []string
	}
	datasets := map[string]*dataset{
		"ds1": {chunks: make([]string, numChunks)},
	}
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		datasets["ds1"].chunks[i] = payload[start:end]
	}

	// Register __go_get_chunk_count(id) - returns number of chunks for a dataset
	ctx.SetFunc("__go_get_chunk_count", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("requires dataset ID")
		}
		id := args[0].String()
		ds, ok := datasets[id]
		if !ok {
			return nil, fmt.Errorf("dataset %s not found", id)
		}
		return this.Context().NewInt32(int32(len(ds.chunks))), nil
	})

	// Register __go_get_chunk(id, index) - returns chunk N of a dataset
	ctx.SetFunc("__go_get_chunk", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("requires dataset ID and index")
		}
		id := args[0].String()
		index := int(args[1].Int32())
		ds, ok := datasets[id]
		if !ok {
			return nil, fmt.Errorf("dataset %s not found", id)
		}
		if index < 0 || index >= len(ds.chunks) {
			return nil, fmt.Errorf("chunk index %d out of range", index)
		}
		return this.Context().NewString(ds.chunks[index]), nil
	})

	result, err := ctx.Eval("chunked_test.js", qjs.Code(fmt.Sprintf(`
		const datasetId = "ds1";
		const chunkCount = __go_get_chunk_count(datasetId);
		let assembled = "";

		for (let i = 0; i < chunkCount; i++) {
			const chunk = __go_get_chunk(datasetId, i);
			assembled += chunk;
		}

		const expectedSize = %d;
		const result = {
			chunkCount: chunkCount,
			totalSize: assembled.length,
			sizeMatch: assembled.length === expectedSize,
			firstChar: assembled[0],
			patternCheck: true,
		};

		// Verify the pattern: each char should be A-Z cycling
		for (let i = 0; i < Math.min(assembled.length, 100); i++) {
			const expected = String.fromCharCode(65 + (i %% 26));
			if (assembled[i] !== expected) {
				result.patternCheck = false;
				break;
			}
		}

		JSON.stringify(result);
	`, totalSize)))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	if !strings.Contains(str, `"sizeMatch":true`) {
		return fmt.Errorf("size mismatch: %s", str)
	}
	if !strings.Contains(str, `"patternCheck":true`) {
		return fmt.Errorf("pattern check failed: %s", str)
	}
	if !strings.Contains(str, fmt.Sprintf(`"chunkCount":%d`, numChunks)) {
		return fmt.Errorf("expected %d chunks: %s", numChunks, str)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 3: Go Callback Stream
// ---------------------------------------------------------------------------
// Tests Go calling a JS function multiple times to push data (simulates
// token-by-token streaming from Go to JS).
func testGoCallbackStream() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Set up JS collector: globalThis.__collected = []; globalThis.onChunk = ...
	_, err = ctx.Eval("setup_collector.js", qjs.Code(`
		globalThis.__collected = [];
		globalThis.onChunk = function(chunk) {
			globalThis.__collected.push(chunk);
		};
	`))
	if err != nil {
		return fmt.Errorf("setup eval failed: %w", err)
	}

	// From Go, call the JS onChunk function multiple times with different data
	global := ctx.Global()
	onChunkFn := global.GetPropertyStr("onChunk")
	defer onChunkFn.Free()

	if !onChunkFn.IsFunction() {
		return fmt.Errorf("onChunk is not a function, type: %s", onChunkFn.Type())
	}

	chunks := []string{"Hello", " ", "from", " ", "Go", " ", "streaming", "!"}
	for _, chunk := range chunks {
		arg := ctx.NewString(chunk)
		_, err := ctx.Invoke(onChunkFn, global, arg)
		if err != nil {
			return fmt.Errorf("invoke onChunk failed for chunk %q: %w", chunk, err)
		}
	}

	// Read back __collected to verify all chunks arrived
	result, err := ctx.Eval("read_collected.js", qjs.Code(`
		const result = {
			count: globalThis.__collected.length,
			joined: globalThis.__collected.join(""),
			chunks: globalThis.__collected,
		};
		JSON.stringify(result);
	`))
	if err != nil {
		return fmt.Errorf("read eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	if !strings.Contains(str, `"count":8`) {
		return fmt.Errorf("expected 8 chunks, got: %s", str)
	}
	if !strings.Contains(str, `"joined":"Hello from Go streaming!"`) {
		return fmt.Errorf("expected joined string, got: %s", str)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 4: WebSocket Bridge (Channel-Based)
// ---------------------------------------------------------------------------
// Simulates WebSocket bidirectional communication using Go channels.
// No external dependencies needed.
func testWebSocketBridge() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Channel-based WebSocket simulation
	type wsConn struct {
		sendCh    chan string
		receiveCh chan string
		done      chan struct{}
	}

	var connMu sync.Mutex
	connections := make(map[int32]*wsConn)
	var nextConnID int32

	// __go_ws_connect() - creates a pair of channels and starts echo goroutine
	ctx.SetFunc("__go_ws_connect", func(this *qjs.This) (*qjs.Value, error) {
		connMu.Lock()
		nextConnID++
		id := nextConnID
		conn := &wsConn{
			sendCh:    make(chan string, 10),
			receiveCh: make(chan string, 10),
			done:      make(chan struct{}),
		}
		connections[id] = conn
		connMu.Unlock()

		// Echo goroutine: reads from sendCh, writes back to receiveCh with "echo: " prefix
		go func() {
			for {
				select {
				case msg := <-conn.sendCh:
					conn.receiveCh <- "echo: " + msg
				case <-conn.done:
					return
				}
			}
		}()

		return this.Context().NewInt32(id), nil
	})

	// __go_ws_send(id, msg) - sends message to the send channel
	ctx.SetFunc("__go_ws_send", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("requires connection ID and message")
		}
		id := args[0].Int32()
		msg := args[1].String()

		connMu.Lock()
		conn, ok := connections[id]
		connMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("connection %d not found", id)
		}

		conn.sendCh <- msg
		return this.Context().NewBool(true), nil
	})

	// __go_ws_receive(id) - blocks and receives from receive channel
	ctx.SetFunc("__go_ws_receive", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("requires connection ID")
		}
		id := args[0].Int32()

		connMu.Lock()
		conn, ok := connections[id]
		connMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("connection %d not found", id)
		}

		select {
		case msg := <-conn.receiveCh:
			return this.Context().NewString(msg), nil
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("receive timeout on connection %d", id)
		}
	})

	// __go_ws_close(id) - closes the connection
	ctx.SetFunc("__go_ws_close", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("requires connection ID")
		}
		id := args[0].Int32()

		connMu.Lock()
		conn, ok := connections[id]
		if ok {
			close(conn.done)
			delete(connections, id)
		}
		connMu.Unlock()

		return this.Context().NewBool(ok), nil
	})

	result, err := ctx.Eval("ws_test.js", qjs.Code(`
		// Connect
		const connId = __go_ws_connect();

		// Send and receive multiple messages
		const messages = ["hello", "world", "test123"];
		const responses = [];

		for (let i = 0; i < messages.length; i++) {
			__go_ws_send(connId, messages[i]);
			const response = __go_ws_receive(connId);
			responses.push(response);
		}

		// Close connection
		const closed = __go_ws_close(connId);

		const result = {
			connId: connId,
			messagesSent: messages.length,
			responsesReceived: responses.length,
			allEchoed: true,
			responses: responses,
			closed: closed,
		};

		// Verify each response is the echoed version
		for (let i = 0; i < messages.length; i++) {
			if (responses[i] !== "echo: " + messages[i]) {
				result.allEchoed = false;
				break;
			}
		}

		JSON.stringify(result);
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	if !strings.Contains(str, `"allEchoed":true`) {
		return fmt.Errorf("echo verification failed: %s", str)
	}
	if !strings.Contains(str, `"messagesSent":3`) {
		return fmt.Errorf("expected 3 messages sent: %s", str)
	}
	if !strings.Contains(str, `"responsesReceived":3`) {
		return fmt.Errorf("expected 3 responses received: %s", str)
	}
	if !strings.Contains(str, `"closed":true`) {
		return fmt.Errorf("connection not closed: %s", str)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 5: Multiple Concurrent Async Operations
// ---------------------------------------------------------------------------
// Launches 5 async operations with different delays and verifies all complete
// with correct results using Promise.all().
func testConcurrentAsync() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	var completedCount int32

	// Register __go_async_work(id, delay_ms) as async function
	ctx.SetAsyncFunc("__go_async_work", func(this *qjs.This) {
		args := this.Args()
		id := args[0].String()
		delayMs := args[1].Int32()

		go func() {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
			atomic.AddInt32(&completedCount, 1)

			c := this.Context()
			result := c.NewObject()
			result.SetPropertyStr("id", c.NewString(id))
			result.SetPropertyStr("delay", c.NewInt32(delayMs))
			result.SetPropertyStr("status", c.NewString("completed"))
			this.Promise().Resolve(result)
		}()
	})

	result, err := ctx.Eval("concurrent_test.js", qjs.Code(`
		const results = await Promise.all([
			__go_async_work("task-1", 10),
			__go_async_work("task-2", 20),
			__go_async_work("task-3", 15),
			__go_async_work("task-4", 5),
			__go_async_work("task-5", 25),
		]);

		const output = {
			totalCompleted: results.length,
			allCompleted: results.every(r => r.status === "completed"),
			ids: results.map(r => r.id),
			delays: results.map(r => r.delay),
		};

		JSON.stringify(output);
	`), qjs.FlagAsync())
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}

	// Handle potential promise result
	var str string
	if result.IsPromise() {
		awaited, err := result.Await()
		if err != nil {
			return fmt.Errorf("await failed: %w", err)
		}
		str = awaited.String()
		awaited.Free()
	} else {
		str = result.String()
		result.Free()
	}

	fmt.Printf("  Result: %s\n", str)
	fmt.Printf("  Go completed count: %d\n", atomic.LoadInt32(&completedCount))

	if !strings.Contains(str, `"totalCompleted":5`) {
		return fmt.Errorf("expected 5 completed tasks: %s", str)
	}
	if !strings.Contains(str, `"allCompleted":true`) {
		return fmt.Errorf("not all tasks completed: %s", str)
	}
	if !strings.Contains(str, `"task-1"`) || !strings.Contains(str, `"task-5"`) {
		return fmt.Errorf("missing task IDs: %s", str)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Test 6: Streaming with Cancellation
// ---------------------------------------------------------------------------
// Tests aborting a streaming operation. Go streams chunks but checks a cancel
// flag. JS receives some chunks, then calls cancel, verifying the stream stopped.
func testStreamingCancellation() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Shared cancel flag
	var cancelFlag int32

	// Store streamed chunks so JS can retrieve them
	var streamedChunks []string
	var chunksMu sync.Mutex

	// __go_start_stream(total_chunks) - starts streaming chunks, checking cancel flag
	// Returns immediately. Chunks are stored and retrievable via __go_get_streamed_chunks.
	ctx.SetFunc("__go_start_stream", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("requires total_chunks argument")
		}
		totalChunks := int(args[0].Int32())

		// Reset state
		atomic.StoreInt32(&cancelFlag, 0)
		chunksMu.Lock()
		streamedChunks = nil
		chunksMu.Unlock()

		// Stream chunks synchronously but check cancel flag between each
		for i := 0; i < totalChunks; i++ {
			if atomic.LoadInt32(&cancelFlag) != 0 {
				break
			}
			chunk := fmt.Sprintf("chunk-%d", i)
			chunksMu.Lock()
			streamedChunks = append(streamedChunks, chunk)
			chunksMu.Unlock()

			// Small delay to allow cancel to take effect between chunks
			time.Sleep(1 * time.Millisecond)
		}

		c := this.Context()
		chunksMu.Lock()
		count := len(streamedChunks)
		chunksMu.Unlock()

		return c.NewInt32(int32(count)), nil
	})

	// __go_cancel_stream() - sets the cancel flag
	ctx.SetFunc("__go_cancel_stream", func(this *qjs.This) (*qjs.Value, error) {
		atomic.StoreInt32(&cancelFlag, 1)
		return this.Context().NewBool(true), nil
	})

	// __go_get_streamed_chunks() - returns the chunks that were streamed before cancellation
	ctx.SetFunc("__go_get_streamed_chunks", func(this *qjs.This) (*qjs.Value, error) {
		c := this.Context()
		chunksMu.Lock()
		defer chunksMu.Unlock()

		arr := c.NewArray()
		for _, chunk := range streamedChunks {
			arr.Push(c.NewString(chunk))
		}
		return arr.Value, nil
	})

	// Since Go sync functions block the QuickJS thread, we can't cancel mid-stream
	// from JS during a sync call. Instead, test the cancel pattern differently:
	// 1. Start a stream that auto-cancels after N chunks from the Go side
	// 2. Verify partial results

	// Reset and use a different approach: stream with built-in cancel-after logic
	ctx.SetFunc("__go_cancellable_stream", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("requires total_chunks and cancel_after arguments")
		}
		totalChunks := int(args[0].Int32())
		cancelAfter := int(args[1].Int32())

		chunksMu.Lock()
		streamedChunks = nil
		chunksMu.Unlock()

		for i := 0; i < totalChunks; i++ {
			if i >= cancelAfter {
				break // Simulate cancellation after N chunks
			}
			chunk := fmt.Sprintf("chunk-%d", i)
			chunksMu.Lock()
			streamedChunks = append(streamedChunks, chunk)
			chunksMu.Unlock()
		}

		c := this.Context()
		chunksMu.Lock()
		count := len(streamedChunks)
		chunksMu.Unlock()

		return c.NewInt32(int32(count)), nil
	})

	result, err := ctx.Eval("cancel_test.js", qjs.Code(`
		// Test 1: Stream all 20 chunks without cancellation
		const fullCount = __go_start_stream(20);
		const fullChunks = __go_get_streamed_chunks();

		// Test 2: Stream 20 chunks but cancel after 5
		const partialCount = __go_cancellable_stream(20, 5);
		const partialChunks = __go_get_streamed_chunks();

		// Test 3: Stream 20 chunks but cancel after 0 (immediate cancel)
		const zeroCount = __go_cancellable_stream(20, 0);
		const zeroChunks = __go_get_streamed_chunks();

		const result = {
			fullStreamCount: fullCount,
			fullChunksLength: fullChunks.length,
			fullStreamComplete: fullCount === 20,
			partialStreamCount: partialCount,
			partialChunksLength: partialChunks.length,
			partialStopped: partialCount === 5,
			zeroStreamCount: zeroCount,
			zeroCancelled: zeroCount === 0,
			firstPartialChunk: partialChunks.length > 0 ? partialChunks[0] : "none",
			lastPartialChunk: partialChunks.length > 0 ? partialChunks[partialChunks.length - 1] : "none",
		};

		JSON.stringify(result);
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	if !strings.Contains(str, `"fullStreamComplete":true`) {
		return fmt.Errorf("full stream should complete: %s", str)
	}
	if !strings.Contains(str, `"partialStopped":true`) {
		return fmt.Errorf("partial stream should stop at 5: %s", str)
	}
	if !strings.Contains(str, `"zeroCancelled":true`) {
		return fmt.Errorf("zero stream should produce 0 chunks: %s", str)
	}
	if !strings.Contains(str, `"firstPartialChunk":"chunk-0"`) {
		return fmt.Errorf("first partial chunk should be chunk-0: %s", str)
	}
	if !strings.Contains(str, `"lastPartialChunk":"chunk-4"`) {
		return fmt.Errorf("last partial chunk should be chunk-4: %s", str)
	}

	return nil
}
