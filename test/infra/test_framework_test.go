package infra

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startKernelForTesting(t *testing.T) *brainkit.Kernel {
	t.Helper()
	storePath := t.TempDir() + "/test-fw.db"
	store, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{Store: store})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k
}

type testResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Error    string `json:"error,omitempty"`
	Duration int    `json:"duration"`
}

func runTestFile(t *testing.T, k *brainkit.Kernel, code string) []testResult {
	t.Helper()
	ctx := context.Background()

	// Evaluate the test file as a module (registers tests via test() calls)
	_, err := k.EvalModule(ctx, "__test_file.ts", code)
	require.NoError(t, err)

	// Run the registered tests
	resultJSON, err := k.EvalTS(ctx, "__run_tests.ts", `
		var r = await globalThis.__runTests();
		return r;
	`)
	require.NoError(t, err)

	var results []testResult
	require.NoError(t, json.Unmarshal([]byte(resultJSON), &results))
	return results
}

func TestTestFramework_PassingTests(t *testing.T) {
	k := startKernelForTesting(t)

	results := runTestFile(t, k, `
		import { test, expect } from "test";

		test("math works", () => {
			expect(1 + 1).toBe(2);
		});

		test("string contains", () => {
			expect("hello world").toContain("world");
		});

		test("truthiness", () => {
			expect(true).toBeTruthy();
			expect("").toBeFalsy();
		});
	`)

	require.Len(t, results, 3)
	for _, r := range results {
		assert.True(t, r.Passed, "test %q should pass, got error: %s", r.Name, r.Error)
	}
}

func TestTestFramework_FailingTest(t *testing.T) {
	k := startKernelForTesting(t)

	results := runTestFile(t, k, `
		import { test, expect } from "test";

		test("this passes", () => {
			expect(42).toBe(42);
		});

		test("this fails", () => {
			expect(1).toBe(2);
		});
	`)

	require.Len(t, results, 2)
	assert.True(t, results[0].Passed)
	assert.False(t, results[1].Passed)
	assert.Contains(t, results[1].Error, "to be")
}

func TestTestFramework_AsyncTests(t *testing.T) {
	k := startKernelForTesting(t)

	results := runTestFile(t, k, `
		import { test, expect, sleep } from "test";

		test("async with sleep", async () => {
			await sleep(10);
			expect(true).toBeTruthy();
		});
	`)

	require.Len(t, results, 1)
	assert.True(t, results[0].Passed)
}

func TestTestFramework_DeployAndTest(t *testing.T) {
	k := startKernelForTesting(t)

	results := runTestFile(t, k, `
		import { test, expect, deploy } from "test";
		import { bus } from "kit";

		test("deploy and call", async () => {
			await deploy("echo.ts", 'bus.on("ping", (msg) => { msg.reply({ pong: true }); });');

			var result = bus.sendTo("echo.ts", "ping", {});
			expect(result).toBeDefined();
		});
	`)

	require.Len(t, results, 1)
	assert.True(t, results[0].Passed, "deploy test error: %s", results[0].Error)
}

func TestTestFramework_Hooks(t *testing.T) {
	k := startKernelForTesting(t)

	results := runTestFile(t, k, `
		import { test, expect, beforeAll, afterAll } from "test";

		var counter = 0;

		beforeAll(() => { counter = 10; });
		afterAll(() => { counter = 0; });

		test("counter set by beforeAll", () => {
			expect(counter).toBe(10);
		});
	`)

	require.Len(t, results, 1)
	assert.True(t, results[0].Passed, "hook test error: %s", results[0].Error)
}

func TestTestFramework_NotAssertions(t *testing.T) {
	k := startKernelForTesting(t)

	results := runTestFile(t, k, `
		import { test, expect } from "test";

		test("not assertions", () => {
			expect(42).not.toBe(43);
			expect("hello").not.toContain("xyz");
		});
	`)

	require.Len(t, results, 1)
	assert.True(t, results[0].Passed, "not test error: %s", results[0].Error)
}
