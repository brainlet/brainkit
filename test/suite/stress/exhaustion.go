package stress

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Attack: JS code allocates massive arrays
func testExhaustionMemoryBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "mem-stress-bomb.ts", `
		try {
			var arr = [];
			for (var i = 0; i < 100; i++) {
				arr.push("x".repeat(1024 * 1024));
			}
			output({allocated: arr.length + "MB"});
		} catch(e) {
			output({error: e.message});
		}
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive memory bomb")
}

// Attack: deploy code that creates deeply recursive function calls.
func testExhaustionStackOverflow(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "stack-stress-bomb.ts", `
		function recurse(depth) {
			if (depth > 500) return depth;
			return recurse(depth + 1);
		}
		try {
			output({depth: recurse(0)});
		} catch(e) {
			output({error: e.message || "stack overflow"});
		}
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive JS stack overflow")
}

// Attack: deploy code that creates infinite promise chains
func testExhaustionPromiseFlood(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "promise-stress-flood.ts", `
		var count = 0;
		function chain() {
			count++;
			if (count < 10000) {
				return Promise.resolve().then(chain);
			}
			return count;
		}
		try {
			await chain();
			output({promises: count});
		} catch(e) {
			output({error: e.message});
		}
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive promise flood")
}

// Attack: deploy many services simultaneously that all try to use resources
func testExhaustionDeployBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			src := fmt.Sprintf("deploy-stress-bomb-%d.ts", n)
			tk.Deploy(ctx, src, fmt.Sprintf(`
				var t = createTool({id: "stress-bomb-tool-%d", description: "bomb", execute: async () => ({n: %d})});
				kit.register("tool", "stress-bomb-tool-%d", t);
				bus.on("ping", function(msg) { msg.reply({n: %d}); });
			`, n, n, n, n))
		}(i)
	}
	wg.Wait()

	assert.True(t, tk.Alive(ctx), "kernel should survive 50 simultaneous deploys")

	for i := 0; i < 50; i++ {
		tk.Teardown(ctx, fmt.Sprintf("deploy-stress-bomb-%d.ts", i))
	}
}

// Attack: deploy code that does fetch() in a tight loop (network exhaustion)
func testExhaustionFetchBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "fetch-stress-bomb.ts", `
		var count = 0;
		var errors = 0;
		for (var i = 0; i < 100; i++) {
			try {
				await fetch("http://127.0.0.1:1/nonexistent");
				count++;
			} catch(e) { errors++; }
		}
		output({fetched: count, errors: errors});
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive fetch bomb")
}

// Attack: rapid deploy/teardown/redeploy cycle to stress lifecycle management
func testExhaustionLifecycleChurn(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		src := "churn-stress-test.ts"
		tk.Deploy(ctx, src, fmt.Sprintf(`
			var t = createTool({id: "stress-churn-%d", description: "churn", execute: async () => ({})});
			kit.register("tool", "stress-churn-%d", t);
			bus.on("ping-%d", function(msg) { msg.reply({i: %d}); });
		`, i, i, i, i))
		tk.Teardown(ctx, src)
	}

	assert.True(t, tk.Alive(ctx), "kernel should survive 100 deploy/teardown cycles")
	deps := tk.ListDeployments()
	assert.Empty(t, deps, "no deployments should remain after churn")
}

// Attack: output() with enormous payload
func testExhaustionOutputBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "output-stress-bomb.ts", `
		output("x".repeat(10 * 1024 * 1024));
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive 10MB output")
}

// Attack: many concurrent EvalTS from Go side
func testExhaustionConcurrentEvalTS(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx := context.Background()

	var wg sync.WaitGroup
	var errors int64
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := tk.EvalTS(ctx, fmt.Sprintf("__stress_concurrent_%d.ts", n),
				fmt.Sprintf(`return "result-%d";`, n))
			if err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	t.Logf("100 concurrent EvalTS: %d errors", errors)
	assert.True(t, tk.Alive(ctx), "kernel should survive 100 concurrent EvalTS")
}

// Attack: deploy code that creates enormous JSON via bus.publish
func testExhaustionLargePayloadViaJS(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "large-stress-payload.ts", `
		try {
			var big = {data: "x".repeat(5 * 1024 * 1024)};
			var r = bus.publish("incoming.stress-large-test", big);
			output({published: true, replyTo: r.replyTo.length > 0});
		} catch(e) {
			output({error: e.message});
		}
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive 5MB bus.publish from JS")
}

// Attack: deploy code that creates 10,000 timers
func testExhaustionTimerBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "timer-stress-bomb.ts", `
		var count = 0;
		for (var i = 0; i < 10000; i++) {
			setTimeout(function() { count++; }, 1);
		}
		output({timersCreated: 10000});
	`)
	_ = err
	time.Sleep(2 * time.Second)
	assert.True(t, tk.Alive(ctx), "kernel should survive 10K timers")
}

// Attack: 10MB secret value
func testExhaustionSecretValueBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bigValue := strings.Repeat("s", 10*1024*1024)
	pr, err := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "stress-big-secret", Value: bigValue})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		s := string(p)
		if len(s) > 100 {
			s = s[:100]
		}
		t.Logf("10MB secret: %s", s)
	case <-ctx.Done():
		t.Fatal("timeout storing 10MB secret")
	}

	assert.True(t, tk.Alive(ctx), "kernel should survive 10MB secret")
}

// Attack: deploy code that modifies JSON.stringify to return infinite output
func testExhaustionJSONStringifyHijack(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "json-stress-hijack.ts", `
		try {
			var orig = JSON.stringify;
			JSON.stringify = function() { return "x".repeat(100000000); };
			bus.publish("incoming.stress-test", {a: 1});
			output("hijacked");
		} catch(e) {
			output("blocked:" + e.message);
		}
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive JSON.stringify hijack")
}

// Attack: deploy from Go with code that fills the filesystem
func testExhaustionFilesystemFill(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "fs-stress-fill.ts", `
		var written = 0;
		try {
			for (var i = 0; i < 100; i++) {
				fs.writeFileSync("stress-fill-" + i + ".dat", "x".repeat(1024 * 1024));
				written++;
			}
		} catch(e) {}
		output({written: written});
	`)
	_ = err
	assert.True(t, tk.Alive(ctx), "kernel should survive filesystem fill attempt")
}

// Attack: deploy code that does setTimeout(fn, 0) in an infinite loop to starve the pump
func testExhaustionPumpStarvation(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tk := env.Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "pump-stress-starve.ts", `
		var count = 0;
		function starve() {
			count++;
			if (count < 50000) {
				setTimeout(starve, 0);
			}
		}
		starve();
		output({started: true});
	`)
	_ = err

	time.Sleep(3 * time.Second)

	_, err = tk.Deploy(ctx, "after-stress-starve.ts", `output("still works");`)
	if err == nil {
		result, _ := tk.EvalTS(ctx, "__stress_after.ts", `return String(globalThis.__module_result || "");`)
		assert.Equal(t, "still works", result)
	}

	assert.True(t, tk.Alive(ctx), "kernel should survive pump starvation")
}

// Attack: persistence bomb -- save thousands of deployments to the store
func testExhaustionPersistenceBomb(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(tmpDir + "/stress-bomb.db")
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir,
		Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		src := fmt.Sprintf("stress-persist-bomb-%d.ts", i)
		k.Deploy(ctx, src, `output("bomb");`)
	}

	k.Close()

	store2, _ := brainkit.NewSQLiteStore(tmpDir + "/stress-bomb.db")
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "stress-test", CallerID: "stress-test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	assert.True(t, k2.Alive(ctx), "kernel should recover from 100 persisted deployments")

	deps := k2.ListDeployments()
	assert.Equal(t, 100, len(deps), "all 100 deployments should be restored")
}
