package bus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsTestResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Error    string `json:"error,omitempty"`
	Duration int    `json:"duration"`
}

func startKernelForTestFramework(t *testing.T) *brainkit.Kernel {
	t.Helper()
	env := suite.Minimal(t, suite.WithPersistence())
	return env.Kernel
}

func runJSTestFile(t *testing.T, k *brainkit.Kernel, code string) []jsTestResult {
	t.Helper()
	ctx := context.Background()

	_, err := k.EvalModule(ctx, "__test_file.ts", code)
	require.NoError(t, err)

	resultJSON, err := k.EvalTS(ctx, "__run_tests.ts", `
		var r = await globalThis.__runTests();
		return r;
	`)
	require.NoError(t, err)

	var results []jsTestResult
	require.NoError(t, json.Unmarshal([]byte(resultJSON), &results))
	return results
}

func testFrameworkPassingTests(t *testing.T, _ *suite.TestEnv) {
	k := startKernelForTestFramework(t)

	results := runJSTestFile(t, k, `
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

func testFrameworkFailingTest(t *testing.T, _ *suite.TestEnv) {
	k := startKernelForTestFramework(t)

	results := runJSTestFile(t, k, `
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

func testFrameworkAsyncTests(t *testing.T, _ *suite.TestEnv) {
	k := startKernelForTestFramework(t)

	results := runJSTestFile(t, k, `
		import { test, expect, sleep } from "test";

		test("async with sleep", async () => {
			await sleep(10);
			expect(true).toBeTruthy();
		});
	`)

	require.Len(t, results, 1)
	assert.True(t, results[0].Passed)
}

func testFrameworkDeployAndTest(t *testing.T, _ *suite.TestEnv) {
	k := startKernelForTestFramework(t)

	results := runJSTestFile(t, k, `
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

func testFrameworkHooks(t *testing.T, _ *suite.TestEnv) {
	k := startKernelForTestFramework(t)

	results := runJSTestFile(t, k, `
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

func testFrameworkNotAssertions(t *testing.T, _ *suite.TestEnv) {
	k := startKernelForTestFramework(t)

	results := runJSTestFile(t, k, `
		import { test, expect } from "test";

		test("not assertions", () => {
			expect(42).not.toBe(43);
			expect("hello").not.toContain("xyz");
		});
	`)

	require.Len(t, results, 1)
	assert.True(t, results[0].Passed, "not test error: %s", results[0].Error)
}
