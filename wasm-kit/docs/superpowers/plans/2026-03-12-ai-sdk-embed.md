# AI SDK Embedding via QuickJS — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bundle the Vercel AI SDK (core + OpenAI provider) into a single JS file, embed it in Go via jsbridge/QuickJS, and call `generateText` end-to-end from Go — proving the full architecture with a real library.

**Architecture:** esbuild bundles AI SDK into a single IIFE that exposes `generateText` + `createOpenAI` on `globalThis.__ai_sdk`. Go loads the bundle via `//go:embed`, initializes jsbridge with polyfills, and provides a typed Go API. The jsbridge polyfills (fetch, encoding, console, etc.) provide the Web APIs the SDK needs. No streaming in v1 — only `generateText` (non-streaming `POST` → JSON response).

**Tech Stack:** esbuild, `ai@7.x`, `@ai-sdk/openai`, `jsbridge` (fastschema/qjs), `//go:embed`

---

## File Structure

```
brainkit/ai-embed/
├── bundle/
│   ├── package.json         # esbuild + AI SDK npm deps
│   ├── entry.mjs            # Custom entry: exports generateText + createOpenAI to globalThis
│   └── build.mjs            # esbuild build script
├── ai_sdk_bundle.js         # Generated bundle (go:embed target, committed)
├── embed.go                 # go:embed directive + bundle loader
├── client.go                # Client type with GenerateText method
└── client_test.go           # Tests with mock OpenAI HTTP server
```

**What each file does:**
- `bundle/package.json` — npm workspace with esbuild + AI SDK as deps. Only used at build time.
- `bundle/entry.mjs` — Narrow import of just `generateText` + `createOpenAI`. Assigns to `globalThis.__ai_sdk`. This is what esbuild tree-shakes from.
- `bundle/build.mjs` — esbuild config: IIFE format, browser platform, ES2020 target, minified.
- `ai_sdk_bundle.js` — Output of esbuild. Committed to repo so `go:embed` works without npm at build time.
- `embed.go` — `//go:embed ai_sdk_bundle.js` + `LoadBundle(b *jsbridge.Bridge)` to eval the bundle into a bridge.
- `client.go` — `Client` struct with `GenerateText(model, prompt string) (*Result, error)`. Wraps JS calls.
- `client_test.go` — Uses `httptest.NewServer` to mock OpenAI's `/v1/chat/completions` endpoint.

---

## Chunk 1: Bundle the AI SDK

### Task 1: Create bundle workspace

**Files:**
- Create: `brainkit/ai-embed/bundle/package.json`

- [ ] **Step 1: Create the directory**

```bash
mkdir -p /Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/bundle
```

- [ ] **Step 2: Create package.json**

```json
{
  "name": "ai-sdk-bundle",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "node build.mjs"
  },
  "dependencies": {
    "ai": "^4.0.0",
    "@ai-sdk/openai": "^1.0.0",
    "@ai-sdk/provider": "^1.0.0",
    "@ai-sdk/provider-utils": "^2.0.0"
  },
  "devDependencies": {
    "esbuild": "^0.24.0"
  }
}
```

Note: Use published npm versions (not the local v7-beta clone). The v4.x `ai` package is stable and has the same `generateText` API. If v4 doesn't resolve, try `"ai": "latest"` and pin after install.

- [ ] **Step 3: Install dependencies**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/bundle && npm install
```

Verify: `node_modules/ai/` and `node_modules/@ai-sdk/openai/` exist.

- [ ] **Step 4: Commit**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit
git add ai-embed/bundle/package.json ai-embed/bundle/package-lock.json
git commit -m "feat(ai-embed): create bundle workspace with AI SDK deps"
```

---

### Task 2: Create bundle entry point

**Files:**
- Create: `brainkit/ai-embed/bundle/entry.mjs`

- [ ] **Step 1: Create entry.mjs**

This file is the narrow entry point that esbuild tree-shakes from. Only import what we need.

```javascript
import { generateText } from 'ai';
import { createOpenAI } from '@ai-sdk/openai';

globalThis.__ai_sdk = {
  generateText,
  createOpenAI,
};
```

- [ ] **Step 2: Verify imports resolve**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/bundle
node -e "import('ai').then(m => console.log('ai:', Object.keys(m).length, 'exports')).catch(e => console.error(e.message))"
```

Expected: prints export count, no errors.

- [ ] **Step 3: Commit**

```bash
git add ai-embed/bundle/entry.mjs
git commit -m "feat(ai-embed): add bundle entry point with generateText + createOpenAI"
```

---

### Task 3: Create esbuild config and build

**Files:**
- Create: `brainkit/ai-embed/bundle/build.mjs`
- Create: `brainkit/ai-embed/ai_sdk_bundle.js` (generated)

- [ ] **Step 1: Create build.mjs**

```javascript
import { build } from 'esbuild';

const result = await build({
  entryPoints: ['entry.mjs'],
  bundle: true,
  format: 'iife',
  target: 'es2020',
  platform: 'browser',
  outfile: '../ai_sdk_bundle.js',
  minify: true,
  define: {
    'process.env.NODE_ENV': '"production"',
    'process.env': '{}',
  },
  logLevel: 'info',
});

// Report size
import { statSync } from 'node:fs';
const stats = statSync('../ai_sdk_bundle.js');
console.log(`Bundle size: ${(stats.size / 1024).toFixed(1)} KB`);
```

- [ ] **Step 2: Run the build**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/bundle && npm run build
```

Expected: builds successfully, prints bundle size. Target is under 500KB.

If esbuild errors on unresolved Node.js builtins (e.g., `node:crypto`, `node:stream`), add to build.mjs:

```javascript
// Add inside the build() call:
external: ['node:*'],
```

Or if specific browser polyfills are needed:

```javascript
alias: {
  'node:crypto': './shims/empty.mjs',
  'node:stream': './shims/empty.mjs',
},
```

With `shims/empty.mjs` = `export default {};`

Iterate until the build succeeds. The AI SDK is designed to be browser-compatible, so minimal shimming should be needed.

**IMPORTANT:** `ai_sdk_bundle.js` must exist before proceeding to Task 4. If this step fails, Task 4 will fail at compile time (`//go:embed` requires the file to exist).

- [ ] **Step 3: Verify bundle is valid JS**

```bash
node -e "require('/Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/ai_sdk_bundle.js'); console.log('OK:', typeof globalThis.__ai_sdk)"
```

Expected: `OK: object`

If IIFE format doesn't work with `require`, try:

```bash
node --input-type=module -e "await import('/Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/ai_sdk_bundle.js'); console.log('OK:', typeof globalThis.__ai_sdk)"
```

- [ ] **Step 4: Add bundle to .gitignore for node_modules but commit the bundle**

Create `brainkit/ai-embed/bundle/.gitignore`:

```
node_modules/
```

- [ ] **Step 5: Commit**

```bash
git add ai-embed/bundle/build.mjs ai-embed/bundle/.gitignore ai-embed/ai_sdk_bundle.js
git commit -m "feat(ai-embed): esbuild config and initial AI SDK bundle"
```

---

### Task 4: Smoke test — bundle loads in QuickJS

**Files:**
- Create: `brainkit/ai-embed/client_test.go`
- Create: `brainkit/ai-embed/embed.go`

- [ ] **Step 1: Write failing test — bundle loads without errors**

Create `brainkit/ai-embed/client_test.go` with the complete import block (all tests in this file share these imports):

```go
package aiembed

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fastschema/qjs"
)

func TestBundleLoads(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	// Verify __ai_sdk is defined
	val, err := c.bridge.Eval("test.js", qjs.Code(`typeof globalThis.__ai_sdk`))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	if val.String() != "object" {
		t.Errorf("__ai_sdk type = %q, want 'object'", val.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit && go test ./ai-embed/ -run TestBundleLoads -v
```

Expected: FAIL — `NewClient` and `embed.go` don't exist yet.

- [ ] **Step 3: Create embed.go with go:embed and bundle loader**

```go
package aiembed

import (
	_ "embed"

	"fmt"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

//go:embed ai_sdk_bundle.js
var bundleSource string

// LoadBundle evaluates the AI SDK bundle into a jsbridge.Bridge.
// After loading, globalThis.__ai_sdk is available with generateText and createOpenAI.
func LoadBundle(b *jsbridge.Bridge) error {
	val, err := b.Eval("ai-sdk-bundle.js", qjs.Code(bundleSource))
	if err != nil {
		return fmt.Errorf("ai-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}
```

- [ ] **Step 4: Create client.go with NewClient**

```go
package aiembed

import (
	"fmt"
	"net/http"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

// ClientConfig configures an AI SDK client.
type ClientConfig struct {
	HTTPClient *http.Client // optional; defaults to http.DefaultClient
}

// Client wraps a jsbridge.Bridge with a loaded AI SDK bundle.
type Client struct {
	bridge *jsbridge.Bridge
}

// NewClient creates a Client with all polyfills and the AI SDK bundle loaded.
func NewClient(cfg ClientConfig) (*Client, error) {
	fetchOpts := []jsbridge.FetchOption{}
	if cfg.HTTPClient != nil {
		fetchOpts = append(fetchOpts, jsbridge.FetchClient(cfg.HTTPClient))
	}

	b, err := jsbridge.New(jsbridge.Config{},
		jsbridge.Console(),
		jsbridge.Encoding(),
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.Fetch(fetchOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: create bridge: %w", err)
	}

	if err := LoadBundle(b); err != nil {
		b.Close()
		return nil, err
	}

	return &Client{bridge: b}, nil
}

// Close shuts down the client and frees all resources.
func (c *Client) Close() {
	if c.bridge != nil {
		c.bridge.Close()
	}
}

// Bridge returns the underlying jsbridge.Bridge for advanced use.
func (c *Client) Bridge() *jsbridge.Bridge { return c.bridge }
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit && go test ./ai-embed/ -run TestBundleLoads -v
```

Expected: PASS — bundle loads, `__ai_sdk` is type `"object"`.

**If the bundle fails to load**, the error will show what's missing. Common issues:
- Missing `ReadableStream` — add a stub: `globalThis.ReadableStream = class ReadableStream { constructor() {} };` before loading the bundle (only needed to avoid reference errors, not for actual streaming)
- Missing `process` — the `define` in esbuild should handle this, but if not, add `globalThis.process = { env: {} };` before the bundle
- Missing `navigator` — add `globalThis.navigator = {};` if needed

Add any required stubs to `embed.go`'s `LoadBundle` function (eval them before the bundle).

- [ ] **Step 6: Commit**

```bash
git add ai-embed/embed.go ai-embed/client.go ai-embed/client_test.go
git commit -m "feat(ai-embed): bundle loads in QuickJS with smoke test"
```

---

## Chunk 2: generateText End-to-End

### Task 5: Write failing test — generateText with mock server

**Files:**
- Modify: `brainkit/ai-embed/client_test.go`

- [ ] **Step 1: Write the mock OpenAI server and generateText test**

Add to `client_test.go`:

```go
func TestGenerateText(t *testing.T) {
	// Mock OpenAI /v1/chat/completions endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", 404)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}

		// Verify request has correct structure
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if req["model"] != "gpt-4" {
			t.Errorf("model = %v, want gpt-4", req["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-test-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello from mock!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 4,
				"total_tokens":      14,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		BaseURL: srv.URL + "/v1",
		APIKey:  "test-key",
		Model:   "gpt-4",
		Prompt:  "Say hello",
	})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	if result.Text != "Hello from mock!" {
		t.Errorf("text = %q, want %q", result.Text, "Hello from mock!")
	}
}
```

Note: All required imports are already in the file from Task 4 Step 1.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit && go test ./ai-embed/ -run TestGenerateText -v
```

Expected: FAIL — `GenerateText` method and types don't exist yet.

---

### Task 6: Implement GenerateText

**Files:**
- Modify: `brainkit/ai-embed/client.go`

- [ ] **Step 1: Add types and GenerateText method**

Add to `client.go`:

```go
// GenerateTextParams configures a generateText call.
type GenerateTextParams struct {
	BaseURL string // e.g., "https://api.openai.com/v1" or mock server URL
	APIKey  string
	Model   string // e.g., "gpt-4"
	Prompt  string
}

// GenerateTextResult holds the result of a generateText call.
type GenerateTextResult struct {
	Text string `json:"text"`
}

// GenerateText calls the AI SDK's generateText function.
func (c *Client) GenerateText(params GenerateTextParams) (*GenerateTextResult, error) {
	js := fmt.Sprintf(`
		const { generateText, createOpenAI } = globalThis.__ai_sdk;
		const openai = createOpenAI({
			apiKey: %q,
			baseURL: %q,
		});
		const result = await generateText({
			model: openai(%q),
			prompt: %q,
		});
		JSON.stringify({ text: result.text });
	`, params.APIKey, params.BaseURL, params.Model, params.Prompt)

	val, err := c.bridge.Eval("generate-text.js", qjs.Code(js), qjs.FlagAsync())
	if err != nil {
		return nil, fmt.Errorf("ai-embed: generateText: %w", err)
	}
	defer val.Free()

	var result GenerateTextResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse result %q: %w", val.String(), err)
	}
	return &result, nil
}
```

Add `"encoding/json"` to the imports in `client.go`.

**CRITICAL: Top-level `await` vs IIFE** — The JS code uses top-level `await` (not wrapped in `async () => { ... }`). This is intentional. `qjs.FlagAsync()` resolves top-level `await` directly in QuickJS v0.0.6. The IIFE pattern does NOT work — it returns `[object Promise]` instead of the resolved value. This was validated in jsbridge experiments. Do NOT wrap in IIFE.

The last expression (`JSON.stringify(...)`) is returned as the eval result. This is how QuickJS works — the value of the last expression is the return value of `Eval()`.

- [ ] **Step 2: Run the test**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit && go test ./ai-embed/ -run TestGenerateText -v
```

Expected: PASS — the full chain works: Go → jsbridge → AI SDK → fetch → mock server → response parsed.

**If it fails**, debug by:
1. Check the error message — it will tell you exactly what Web API is missing
2. Add stubs/polyfills as described in Task 4 Step 5
3. If the AI SDK's fetch call format doesn't match jsbridge's expectations, inspect the request JSON
4. Use `console.log()` in the JS code — jsbridge captures console output via the Console polyfill

- [ ] **Step 3: Commit**

```bash
git add ai-embed/client.go ai-embed/client_test.go
git commit -m "feat(ai-embed): GenerateText end-to-end with mock OpenAI server"
```

---

### Task 7: Add error handling test

**Files:**
- Modify: `brainkit/ai-embed/client_test.go`

- [ ] **Step 1: Write test for API error handling**

```go
func TestGenerateTextAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	_, err = c.GenerateText(GenerateTextParams{
		BaseURL: srv.URL + "/v1",
		APIKey:  "bad-key",
		Model:   "gpt-4",
		Prompt:  "hello",
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	// The AI SDK should surface the error through the JS eval
	t.Logf("Got expected error: %v", err)
}
```

- [ ] **Step 2: Run test**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit && go test ./ai-embed/ -run TestGenerateTextAPIError -v
```

Expected: PASS — error returned, not a panic.

- [ ] **Step 3: Commit**

```bash
git add ai-embed/client_test.go
git commit -m "test(ai-embed): add API error handling test"
```

---

### Task 8: Run all tests together

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/davidroman/Documents/code/brainlet/brainkit && go test ./ai-embed/ -v
```

Expected: All 3 tests pass (TestBundleLoads, TestGenerateText, TestGenerateTextAPIError).

- [ ] **Step 2: Check bundle size**

```bash
wc -c /Users/davidroman/Documents/code/brainlet/brainkit/ai-embed/ai_sdk_bundle.js | awk '{printf "%.1f KB\n", $1/1024}'
```

Document the bundle size — target is under 500KB.

- [ ] **Step 3: Final commit**

```bash
git add -A ai-embed/
git commit -m "feat(ai-embed): complete AI SDK embedding with generateText end-to-end"
```

---

## Future Possibilities

These are not part of this plan but should be kept in mind for subsequent work.

### 2. Migrate Phase 2/3 polyfills into jsbridge

The experiment code at `brainkit/experiments/quickjs-system-bridge/` has working implementations for:
- `fs.readFile/writeFile/readdir/stat/mkdir/unlink/rm` → Go `os` package
- `path.join/resolve/dirname/basename/extname` → Go `filepath` package
- `child_process.exec` → Go `os/exec`
- `child_process.spawn` (streaming) → Go `os/exec` + goroutine pipe drain
- `process.env` get/set → Go `os.Getenv/Setenv`
- `process.cwd` → Go `os.Getwd`

These follow the same `Polyfill` interface pattern. Migration is mechanical — extract from experiment into `jsbridge/fs.go`, `jsbridge/path.go`, `jsbridge/process.go`. Unlocks Mastra stores, voice, agent-builder, and workspace features.

### 3. Add streaming support (streamText)

Requires:
- **ReadableStream/TransformStream/WritableStream polyfill** in jsbridge — the AI SDK's `streamText` and `createEventSourceResponseHandler` depend on `response.body` (a ReadableStream). Options: pure-JS polyfill (e.g., port web-streams-polyfill to a jsbridge polyfill) or Go-bridged stream with `ctx.Invoke()` callbacks.
- **`ctx.Invoke()` for Go→JS callbacks** — proven in experiment `quickjs-streaming`. Go pushes chunks into a JS callback function, enabling token-by-token streaming.
- **eventsource-parser** — already bundled by esbuild if streamText is imported. Parses SSE `data: {json}\n\n` format.
- Adds `StreamText` method to `Client` that returns a Go channel or iterator of tokens.

### 4. Bytecode precompilation pipeline

The AI SDK bundle can be precompiled to QuickJS bytecode at build time:
```go
bytecode, _ := bridge.Compile("ai-sdk.js", qjs.Code(bundleSource), qjs.FlagCompileOnly())
// Embed bytecode instead of source
bridge.Eval("ai-sdk.js", qjs.Bytecode(bytecode))
```
Experiment showed **4.24x speedup** (100KB: 11ms source vs 3ms bytecode). For a ~400KB bundle, this would cut load time from ~30ms to ~7ms. Add a build step that compiles JS → bytecode and embeds via `//go:embed`.

### 5. Binaryen bridge productionization

The experiment at `brainkit/experiments/quickjs-binaryen-bridge/` proved the three-layer architecture:
```
JS (QuickJS) → Go LinearMemory (64MB bump allocator) → CGo libbinaryen
```
Productionizing means:
- Move `LinearMemory` + mutex into a `binaryen-bridge` package
- Auto-generate the ~900 `_Binaryen*()` JS shims from AssemblyScript's `src/glue/binaryen.js`
- Connect to existing `wasm-kit/pkg/binaryen/` CGo bindings (6,519 lines, 95%+ API coverage)
- Performance budget: ~8μs/call, 100K calls ≈ 800ms (acceptable for compile tool)
- This unlocks running the AssemblyScript compiler's backend through QuickJS
