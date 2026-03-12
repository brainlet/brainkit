// Experiment: QuickJS Fetch Bridge
//
// Goal: Prove that JS running in QuickJS can call a Go-provided fetch()
// and get HTTP responses back. This is the foundation for all three
// library embeddings (AI SDK, Mastra, AssemblyScript).
//
// Tests:
// 1. Basic JS evaluation
// 2. Go -> JS function calls
// 3. JS -> Go function calls (bidirectional)
// 4. Fetch bridge: JS fetch() -> Go net/http -> real endpoint
// 5. Async/Promise with real HTTP
// 6. JSON round-trip between Go and JS

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fastschema/qjs"
)

func main() {
	fmt.Println("=== QuickJS Fetch Bridge Experiment ===")
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Basic JS Evaluation", testBasicEval},
		{"Go -> JS Function Call", testGoCallsJS},
		{"JS -> Go Function Call", testJSCallsGo},
		{"Sync Fetch Bridge (Real HTTP)", testFetchBridgeSync},
		{"Async Fetch Bridge (Promise)", testFetchBridgeAsync},
		{"JSON Round-Trip", testJSONRoundTrip},
	}

	for i, t := range tests {
		fmt.Printf("--- Test %d: %s ---\n", i+1, t.name)
		if err := t.fn(); err != nil {
			log.Fatalf("FAILED: %v\n", err)
		}
		fmt.Println("PASS")
		fmt.Println()
	}

	fmt.Println("=== ALL TESTS PASSED ===")
}

func testBasicEval() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()
	result, err := ctx.Eval("test1.js", qjs.Code(`
		const x = 40;
		const y = 2;
		x + y;
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	val := result.Int32()
	if val != 42 {
		return fmt.Errorf("expected 42, got %d", val)
	}
	fmt.Printf("  42 == %d\n", val)
	return nil
}

func testGoCallsJS() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	_, err = ctx.Eval("define.js", qjs.Code(`
		function greet(name) {
			return "Hello, " + name + "!";
		}
	`))
	if err != nil {
		return fmt.Errorf("define failed: %w", err)
	}

	result, err := ctx.Eval("call.js", qjs.Code(`greet("Go");`))
	if err != nil {
		return fmt.Errorf("call failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	if str != "Hello, Go!" {
		return fmt.Errorf("expected 'Hello, Go!', got '%s'", str)
	}
	fmt.Printf("  \"%s\"\n", str)
	return nil
}

func testJSCallsGo() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	goCallCount := 0
	ctx.SetFunc("goAdd", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return this.Context().NewInt32(0), nil
		}
		a := args[0].Int32()
		b := args[1].Int32()
		goCallCount++
		return this.Context().NewInt32(a + b), nil
	})

	result, err := ctx.Eval("test3.js", qjs.Code(`
		const sum = goAdd(17, 25);
		sum;
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	val := result.Int32()
	if val != 42 {
		return fmt.Errorf("expected 42, got %d", val)
	}
	if goCallCount != 1 {
		return fmt.Errorf("expected Go function called once, got %d", goCallCount)
	}
	fmt.Printf("  goAdd(17, 25) = %d (called %d time)\n", val, goCallCount)
	return nil
}

func testFetchBridgeSync() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Register a sync fetch that makes a real HTTP call from Go
	ctx.SetFunc("__goFetchSync", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("fetch requires a URL argument")
		}
		url := args[0].String()
		fmt.Printf("  Go: fetching %s\n", url)

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

		c := this.Context()
		result := c.NewObject()
		result.SetPropertyStr("status", c.NewInt32(int32(resp.StatusCode)))
		result.SetPropertyStr("body", c.NewString(string(body)))
		result.SetPropertyStr("contentType", c.NewString(resp.Header.Get("Content-Type")))

		return result, nil
	})

	result, err := ctx.Eval("test4.js", qjs.Code(`
		const response = __goFetchSync("https://httpbin.org/json");
		const data = JSON.parse(response.body);
		JSON.stringify({
			status: response.status,
			hasSlideshow: data.slideshow !== undefined,
			contentType: response.contentType
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
	if parsed["status"].(float64) != 200 {
		return fmt.Errorf("expected status 200, got %v", parsed["status"])
	}
	if parsed["hasSlideshow"] != true {
		return fmt.Errorf("expected hasSlideshow=true")
	}
	return nil
}

func testFetchBridgeAsync() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	// Register an async fetch using promises
	ctx.SetAsyncFunc("goFetchAsync", func(this *qjs.This) {
		args := this.Args()
		url := args[0].String()
		fmt.Printf("  Go: async fetching %s\n", url)

		go func() {
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				this.Promise().Reject(this.Context().NewError(fmt.Errorf("fetch failed: %w", err)))
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				this.Promise().Reject(this.Context().NewError(fmt.Errorf("read body failed: %w", err)))
				return
			}

			c := this.Context()
			result := c.NewObject()
			result.SetPropertyStr("status", c.NewInt32(int32(resp.StatusCode)))
			result.SetPropertyStr("body", c.NewString(string(body)))
			this.Promise().Resolve(result)
		}()
	})

	// Use top-level await
	result, err := ctx.Eval("test5.js", qjs.Code(`
		const response = await goFetchAsync("https://httpbin.org/get");
		const data = JSON.parse(response.body);
		JSON.stringify({
			status: response.status,
			url: data.url
		});
	`), qjs.FlagAsync())
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}

	// FlagAsync() resolves the top-level await internally,
	// so result may already be the final value (not a Promise).
	// Try Await() first, fall back to direct value.
	var str string
	if awaited, err := result.Await(); err == nil {
		str = awaited.String()
	} else {
		str = result.String()
	}
	fmt.Printf("  Result: %s\n", str)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}
	if parsed["status"].(float64) != 200 {
		return fmt.Errorf("expected status 200, got %v", parsed["status"])
	}
	return nil
}

func testJSONRoundTrip() error {
	runtime, err := qjs.New()
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer runtime.Close()

	ctx := runtime.Context()

	ctx.SetFunc("goProcessData", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		jsonStr := args[0].String()

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			return nil, fmt.Errorf("JSON parse error: %w", err)
		}

		data["processed"] = true
		data["engine"] = "quickjs-via-wazero"
		if items, ok := data["items"].([]interface{}); ok {
			data["count"] = len(items)
		}

		result, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("JSON marshal error: %w", err)
		}

		return this.Context().NewString(string(result)), nil
	})

	result, err := ctx.Eval("test6.js", qjs.Code(`
		const input = {
			name: "test",
			items: ["alpha", "beta", "gamma"],
			nested: { deep: { value: 42 } }
		};

		const outputJson = goProcessData(JSON.stringify(input));
		const output = JSON.parse(outputJson);

		const checks = {
			namePreserved: output.name === "test",
			processed: output.processed === true,
			engine: output.engine,
			count: output.count,
			nestedPreserved: output.nested.deep.value === 42,
			allPassed: true
		};
		checks.allPassed = checks.namePreserved && checks.processed &&
						   checks.engine === "quickjs-via-wazero" &&
						   checks.count === 3 && checks.nestedPreserved;
		JSON.stringify(checks);
	`))
	if err != nil {
		return fmt.Errorf("eval failed: %w", err)
	}
	defer result.Free()

	str := result.String()
	fmt.Printf("  Result: %s\n", str)

	if !strings.Contains(str, `"allPassed":true`) {
		return fmt.Errorf("round-trip verification failed: %s", str)
	}
	return nil
}
