//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/registry"
)

func TestFixture_TS_BusSubscribe(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/bus-subscribe.js")

	result, err := kit.EvalModule(context.Background(), "bus-subscribe.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		SubID  bool     `json:"subId"`
		Count  int      `json:"count"`
		Values []int    `json:"values"`
		Topics []string `json:"topics"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.SubID {
		t.Error("no subscription ID returned")
	}
	if out.Count != 2 {
		t.Errorf("expected 2 messages, got %d (values: %v)", out.Count, out.Values)
	}
	t.Logf("bus-subscribe: subId=%v count=%d values=%v topics=%v", out.SubID, out.Count, out.Values, out.Topics)
}

func TestFixture_TS_ToolsRegisterList(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/tools-register-list.js")

	result, err := kit.EvalModule(context.Background(), "tools-register-list.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Registered bool     `json:"registered"`
		ToolCount  int      `json:"toolCount"`
		Found      bool     `json:"found"`
		Names      []string `json:"names"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Found {
		t.Errorf("registered tool not found in list: count=%d names=%v", out.ToolCount, out.Names)
	}
	t.Logf("tools-register-list: registered=%v found=%v count=%d names=%v", out.Registered, out.Found, out.ToolCount, out.Names)
}

func TestFixture_TS_ToolsCall(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/uppercase", ShortName: "uppercase",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Converts text to uppercase",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]string{"result": strings.ToUpper(args.Text)})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/tools-call.js")
	result, err := kit.EvalModule(context.Background(), "tools-call.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct{ Result string }
	json.Unmarshal([]byte(result), &out)

	if out.Result != "HELLO BRAINLET" {
		t.Errorf("result = %q", out.Result)
	}
	t.Logf("fixture tools-call: %q", out.Result)
}

func TestFixture_TS_SandboxContext(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/sandbox-context.js")

	result, err := kit.EvalModule(context.Background(), "sandbox-context.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		ID        string `json:"id"`
		Namespace string `json:"namespace"`
		CallerID  string `json:"callerID"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.ID == "" {
		t.Error("empty id")
	}
	if out.Namespace != "test" {
		t.Errorf("namespace = %q", out.Namespace)
	}
	t.Logf("fixture sandbox-context: %+v", out)
}

func TestFixture_TS_BidirectionalAsync(t *testing.T) {
	tmpDir := t.TempDir()
	key := requireKey(t)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"TEST_TMPDIR": tmpDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/bidirectional-async.js")
	result, err := kit.EvalModule(context.Background(), "bidirectional-async.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	if errMsg, ok := out["error"]; ok && errMsg != nil {
		t.Fatalf("fixture error: %v\nstack: %v", errMsg, out["stack"])
	}
	if out["success"] != true {
		t.Errorf("expected success, got: %v", out)
	}
	t.Logf("bidirectional: generate=%v stream=%v", out["generate"], out["stream"])
}
