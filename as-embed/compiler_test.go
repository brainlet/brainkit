package asembed

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

func newTestBridge(t *testing.T) *jsbridge.Bridge {
	t.Helper()
	b, err := jsbridge.New(jsbridge.Config{},
		jsbridge.Console(),
		jsbridge.Encoding(),
		jsbridge.Streams(),
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.StructuredClone(),
		jsbridge.Fetch(),
	)
	if err != nil {
		t.Fatalf("jsbridge.New: %v", err)
	}
	t.Cleanup(func() { b.Close() })
	return b
}

func TestBundleLoads(t *testing.T) {
	b := newTestBridge(t)

	if err := LoadBundle(b); err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}

	val, err := b.Eval("test.js", qjs.Code(`typeof globalThis.__as_compiler`))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	if val.String() != "object" {
		t.Errorf("__as_compiler type = %q, want 'object'", val.String())
	}
}

func TestParserWorks(t *testing.T) {
	b := newTestBridge(t)
	if err := LoadBundle(b); err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}

	val, err := b.Eval("test-parse.js", qjs.Code(`
		const asc = globalThis.__as_compiler;
		const options = asc.newOptions();
		const program = asc.newProgram(options);

		asc.parse(program, "export function add(a: i32, b: i32): i32 { return a + b; }", "input.ts", true);

		let fileCount = 0;
		let file;
		while ((file = asc.nextFile(program)) !== null) {
			asc.parse(program, null, file, false);
			fileCount++;
		}

		let errors = [];
		let diag;
		while ((diag = asc.nextDiagnostic(program)) !== null) {
			if (asc.isError(diag)) {
				errors.push(asc.formatDiagnostic(diag, false, false));
			}
		}

		JSON.stringify({
			parsed: true,
			fileCount: fileCount,
			errorCount: errors.length,
			errors: errors.slice(0, 3),
		});
	`))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	t.Logf("Parse result: %s", val.String())
}

func TestMemoryBridgeFromJS(t *testing.T) {
	b := newTestBridge(t)
	lm := NewLinearMemory()
	RegisterMemoryBridge(b.Context(), lm)

	val, err := b.Eval("test-mem.js", qjs.Code(`
		var ptr = _malloc(16);
		__i32_store(ptr, 42);
		var loaded = __i32_load(ptr);
		var sPtr = _malloc(6);
		var str = "hello";
		for (var i = 0; i < str.length; i++) {
			__i32_store8(sPtr + i, str.charCodeAt(i));
		}
		__i32_store8(sPtr + str.length, 0);
		var fPtr = _malloc(16);
		__f64_store(fPtr, 3.14159);
		var fLoaded = __f64_load(fPtr);
		JSON.stringify({ loaded: loaded, fLoaded: fLoaded, ptr: ptr, sPtr: sPtr });
	`))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	t.Logf("Memory bridge result: %s", val.String())
}

func TestCompileSimpleProgram(t *testing.T) {
	c, err := NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler: %v", err)
	}
	defer c.Close()

	result, err := c.Compile(map[string]string{
		"input.ts": `export function add(a: i32, b: i32): i32 { return a + b; }`,
	}, CompileOptions{
		OptimizeLevel: 0,
		ShrinkLevel:   0,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if len(result.Binary) < 8 {
		t.Fatalf("binary too short: %d bytes", len(result.Binary))
	}

	// Check Wasm magic bytes: \x00asm
	magic := result.Binary[:4]
	if magic[0] != 0x00 || magic[1] != 0x61 || magic[2] != 0x73 || magic[3] != 0x6d {
		t.Errorf("bad wasm magic: %x", magic)
	}

	t.Logf("Compiled %d bytes of Wasm binary", len(result.Binary))
	if result.Text != "" {
		t.Logf("Warnings: %s", result.Text)
	}
}

func TestCompileMatchesNodeJS(t *testing.T) {
	c, err := NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler: %v", err)
	}
	defer c.Close()

	source := `export function add(a: i32, b: i32): i32 { return a + b; }`
	result, err := c.Compile(map[string]string{
		"input.ts": source,
	}, CompileOptions{OptimizeLevel: 0})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if len(result.Binary) < 8 {
		t.Fatalf("binary too short: %d bytes", len(result.Binary))
	}

	magic := result.Binary[:4]
	version := result.Binary[4:8]
	t.Logf("Magic: %v, Version: %v, Size: %d bytes", magic, version, len(result.Binary))

	// Verify Wasm magic number
	if magic[0] != 0x00 || magic[1] != 0x61 || magic[2] != 0x73 || magic[3] != 0x6d {
		t.Errorf("invalid Wasm magic: %v", magic)
	}
	// Verify Wasm version 1
	if version[0] != 0x01 || version[1] != 0x00 || version[2] != 0x00 || version[3] != 0x00 {
		t.Errorf("invalid Wasm version: %v", version)
	}
}

func TestCompileAssignmentChainRegression(t *testing.T) {
	c, err := NewCompiler()
	if err != nil {
		t.Fatalf("NewCompiler: %v", err)
	}
	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	source := `class A {
  x: i64 = 0;
  y: i64 = 0;
}

export function normal_assignment_chain(): void {
  let x = new A();
  let cnt = 0;
  x.x = x.y = cnt++;
  assert(cnt == 1);
}
normal_assignment_chain();

class B {
  _setter_cnt: i32 = 0;
  _getter_cnt: i32 = 0;
  _y: f64 = 0.0;
  set y(z: f64) {
    this._setter_cnt += 1;
    this._y = z;
  }
  get y(): f64 {
    this._getter_cnt += 1;
    return this._y;
  }
}
export function setter_assignment_chain(): void {
  let x = new B();
  x.y = x.y = 1;
  assert(x._setter_cnt == 2);
  assert(x._getter_cnt == 0);
}
setter_assignment_chain();

class C {
  static _setter_cnt: i32 = 0;
  static _y: f64 = 0.0;
  static set y(z: f64) {
    C._setter_cnt += 1;
    C._y = z;
  }
}
export function static_setter_assignment_chain(): void {
  C.y = C.y = 1;
  assert(C._setter_cnt == 2);
}
static_setter_assignment_chain();
`

	result, err := c.Compile(map[string]string{
		"assignment-chain.ts": source,
	}, CompileOptions{
		OptimizeLevel: 0,
		ShrinkLevel:   0,
		Debug:         true,
		Runtime:       "incremental",
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if len(result.Binary) < 8 {
		t.Fatalf("binary too short: %d bytes", len(result.Binary))
	}

	done := make(chan struct{})
	go func() {
		c.Close()
		close(done)
	}()

	select {
	case <-done:
		c = nil
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung after assignment-chain compile")
	}
}

func TestBundleLoadTime(t *testing.T) {
	const iterations = 3
	var total time.Duration

	for i := 0; i < iterations; i++ {
		b := newTestBridge(t)
		start := time.Now()
		if err := LoadBundle(b); err != nil {
			t.Fatalf("LoadBundle: %v", err)
		}
		total += time.Since(start)
		b.Close()
	}

	avg := total / time.Duration(iterations)
	t.Logf("Bundle load time (avg of %d): %s", iterations, avg.Round(time.Millisecond))
	t.Logf("Bundle size: %.1f KB", float64(len(bundleSource))/1024)
}
