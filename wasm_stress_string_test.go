//go:build stress

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

func TestWASMStress_StringEdgeCases(t *testing.T) {
	for _, runtime := range []string{"stub", "minimal", "incremental"} {
		t.Run("runtime="+runtime, func(t *testing.T) {
			kit := newTestKitNoKey(t)
			ctx := context.Background()

			source := hostTestSource(`
export function run(): i32 {
  host_log("", 0);
  host_log("x", 1);
  host_log("hello from wasm", 1);
  host_log("\u4F60\u597D", 1);
  host_log("\uD83D\uDE00", 1);
  host_log("hi \uD83D\uDE00 \u4F60\u597D end", 1);

  var long: string = "";
  for (let i = 0; i < 1000; i++) {
    long += "A";
  }
  host_log(long, 1);

  host_log('say "hello" they said', 1);
  host_log("path-to-file", 1);
  host_log("1234567890", 1);
  host_log("\uD83C\uDF89", 1);
  host_log("\uD83D\uDE00\uD83D\uDE01\uD83D\uDE02\uD83D\uDE03", 1);

  host_set_state("key-with-dashes", "value with spaces");
  const v1 = host_get_state("key-with-dashes");
  if (v1 != "value with spaces") return 13;

  host_set_state("\u4F60\u597D", "\uD83D\uDE00");
  const v2 = host_get_state("\u4F60\u597D");
  if (v2 != "\uD83D\uDE00") return 14;

  host_set_state("counter", "1");
  host_set_state("counter", "2");
  const v3 = host_get_state("counter");
  if (v3 != "2") return 15;

  const v4 = host_get_state("never-set");
  if (v4 != "") return 16;

  host_send("stress.test.json", '{"key":"value","num":42}');
  host_send("stress.test.unicode", '{"emoji":"\uD83D\uDE00"}');
  host_send("stress.test.empty", '{}');

  for (let i = 0; i < 100; i++) {
    host_set_state("k" + i.toString(), "v" + i.toString());
  }
  for (let i = 0; i < 100; i++) {
    const v = host_get_state("k" + i.toString());
    if (v != "v" + i.toString()) return 20;
  }

  return 0;
}
`)
			_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(`
				await wasm.compile(%s, { name: "stress-strings", runtime: %q });
			`, "`"+source+"`", runtime))
			if err != nil {
				t.Fatalf("compile: %v", err)
			}

			received := make(chan bus.Message, 10)
			kit.Bus.On("stress.test.*", func(msg bus.Message, _ bus.ReplyFunc) {
				received <- msg
			})

			result, err := kit.EvalTS(ctx, "run.ts", `
				var r = await wasm.run("stress-strings");
				return JSON.stringify(r);
			`)
			if err != nil {
				t.Fatalf("run: %v", err)
			}

			var runResult struct {
				ExitCode int `json:"exitCode"`
			}
			json.Unmarshal([]byte(result), &runResult)
			if runResult.ExitCode != 0 {
				t.Fatalf("exit code = %d (subtest %d failed)", runResult.ExitCode, runResult.ExitCode)
			}

			busCount := 0
		drain:
			for {
				select {
				case <-received:
					busCount++
				default:
					break drain
				}
			}
			if busCount < 3 {
				t.Errorf("expected at least 3 bus messages, got %d", busCount)
			}
		})
	}
}

func TestWASMStress_ManyDataSegments(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	var sb strings.Builder
	sb.WriteString(hostTestSource(""))
	sb.WriteString("\nexport function run(): i32 {\n")
	for i := 0; i < 50; i++ {
		sb.WriteString(fmt.Sprintf("  host_log(\"segment-%d-data\", 0);\n", i))
	}
	sb.WriteString("  return 0;\n}\n")
	source := sb.String()

	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(`
		await wasm.compile(%s, { name: "many-segments", runtime: "incremental" });
	`, "`"+source+"`"))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("many-segments");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var rr struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d, want 0", rr.ExitCode)
	}
}

func TestWASMStress_LongStrings(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  var s: string = "";
  for (let i = 0; i < 10000; i++) {
    s += "X";
  }
  if (s.length != 10000) return 1;

  host_set_state("big", s);
  const got = host_get_state("big");
  if (got.length != 10000) return 2;
  if (got != s) return 3;

  host_log(s, 0);
  host_send("stress.long", '{"len":' + s.length.toString() + '}');

  return 0;
}
`)

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "long-strings", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("long-strings");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var rr struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d (subtest %d failed)", rr.ExitCode, rr.ExitCode)
	}
}

func TestWASMStress_LargePayload(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  var json: string = '{"data":"';
  for (let i = 0; i < 5000; i++) {
    json += "A";
  }
  json += '"}';
  host_send("stress.large", json);
  return 0;
}
`)
	received := make(chan bus.Message, 1)
	kit.Bus.On("stress.large", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "large-payload", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("large-payload");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var rr struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d", rr.ExitCode)
	}

	select {
	case msg := <-received:
		var payload struct {
			Data string `json:"data"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if len(payload.Data) != 5000 {
			t.Errorf("payload data length=%d, want 5000", len(payload.Data))
		}
	default:
		t.Error("bus message not received")
	}
}
