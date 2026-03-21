//go:build integration

package kit

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/bus"
)

func buildCronPlugin(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "plugins", "brainkit-plugin-cron")
	binary := filepath.Join(t.TempDir(), "brainkit-plugin-cron")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build cron plugin: %s\n%s", err, out)
	}
	return binary
}

func TestCronPlugin_FullLifecycle(t *testing.T) {
	binary := buildCronPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-cron",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "cron", Binary: binary},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Subscribe to cron.fired events
	firedCh := make(chan bus.Message, 10)
	kit.Bus.On("cron.fired", func(msg bus.Message, _ bus.ReplyFunc) {
		firedCh <- msg
	})

	ctx := context.Background()

	// 1. Create a cron job with 1s interval
	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"create","input":{"name":"test-job","schedule":"1s","action":{"type":"event","topic":"test.cron.tick"}}}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var createResult struct {
		Created string `json:"created"`
	}
	json.Unmarshal(resp.Payload, &createResult)
	if createResult.Created != "test-job" {
		t.Fatalf("create: expected test-job, got %q (payload: %s)", createResult.Created, resp.Payload)
	}

	// 2. List cron jobs
	resp, err = bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"list","input":{}}`),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var listResult struct {
		Jobs []struct {
			Name string `json:"name"`
		} `json:"jobs"`
	}
	json.Unmarshal(resp.Payload, &listResult)
	if len(listResult.Jobs) != 1 {
		t.Fatalf("list: expected 1 job, got %d (payload: %s)", len(listResult.Jobs), resp.Payload)
	}

	// 3. Wait for cron.fired event
	select {
	case msg := <-firedCh:
		var fired struct {
			JobName string `json:"jobName"`
		}
		json.Unmarshal(msg.Payload, &fired)
		if fired.JobName != "test-job" {
			t.Fatalf("cron.fired: expected test-job, got %q", fired.JobName)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cron.fired event")
	}

	// 4. Pause
	_, err = bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"pause","input":{"name":"test-job"}}`),
	})
	if err != nil {
		t.Fatalf("pause: %v", err)
	}

	// 5. Resume
	_, err = bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"resume","input":{"name":"test-job"}}`),
	})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	// 6. Remove
	_, err = bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"remove","input":{"name":"test-job"}}`),
	})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	// 7. List again — should be empty
	resp, err = bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"list","input":{}}`),
	})
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	json.Unmarshal(resp.Payload, &listResult)
	if len(listResult.Jobs) != 0 {
		t.Fatalf("list after remove: expected 0 jobs, got %d", len(listResult.Jobs))
	}
}
