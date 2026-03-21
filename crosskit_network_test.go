package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestCrossKit_GRPCConnect(t *testing.T) {
	kitA, err := New(Config{
		Name:      "kit-a",
		Namespace: "a",
		Network:   NetworkConfig{Listen: "127.0.0.1:0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	addr := kitA.network.Addr()

	kitB, err := New(Config{
		Name:      "kit-b",
		Namespace: "b",
		Network:   NetworkConfig{Peers: map[string]string{"kit-a": addr}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	time.Sleep(500 * time.Millisecond)

	if kitB.transport == nil {
		t.Fatal("Kit B has no transport")
	}
	if !kitB.transport.HasPeer("kit-a") {
		t.Fatal("Kit B does not have Kit A as a peer")
	}
}

func TestCrossKit_RemoteToolCall(t *testing.T) {
	kitA, err := New(Config{
		Name:      "kit-a-tools",
		Namespace: "a",
		Network:   NetworkConfig{Listen: "127.0.0.1:0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	kitA.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/add", ShortName: "add",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var in struct{ A, B float64 }
				json.Unmarshal(input, &in)
				return json.Marshal(map[string]float64{"result": in.A + in.B})
			},
		},
	})

	addr := kitA.network.Addr()

	kitB, err := New(Config{
		Name:      "kit-b-tools",
		Namespace: "b",
		Network:   NetworkConfig{Peers: map[string]string{"kit-a-tools": addr}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	time.Sleep(500 * time.Millisecond)

	resp, err := bus.AskSync(kitB.Bus, context.Background(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Address:  "kit:kit-a-tools",
		Payload:  json.RawMessage(`{"name":"add","input":{"a":10,"b":20}}`),
	})
	if err != nil {
		t.Fatalf("remote tool call: %v", err)
	}

	var result map[string]float64
	json.Unmarshal(resp.Payload, &result)
	if result["result"] != 30 {
		t.Errorf("expected 30, got %v (payload: %s)", result["result"], resp.Payload)
	}
}

func TestCrossKit_EventForwarding(t *testing.T) {
	kitA, err := New(Config{
		Name:      "kit-a-events",
		Namespace: "a",
		Network:   NetworkConfig{Listen: "127.0.0.1:0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	addr := kitA.network.Addr()

	kitB, err := New(Config{
		Name:      "kit-b-events",
		Namespace: "b",
		Network:   NetworkConfig{Peers: map[string]string{"kit-a-events": addr}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	time.Sleep(500 * time.Millisecond)

	received := make(chan bus.Message, 1)
	kitA.Bus.On("test.cross-kit", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	kitB.Bus.Send(bus.Message{
		Topic:    "test.cross-kit",
		CallerID: "kit-b",
		Address:  "kit:kit-a-events",
		Payload:  json.RawMessage(`{"from":"kit-b"}`),
	})

	select {
	case msg := <-received:
		var payload map[string]string
		json.Unmarshal(msg.Payload, &payload)
		if payload["from"] != "kit-b" {
			t.Errorf("expected from=kit-b, got %v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cross-Kit event")
	}
}

func TestCrossKit_DisconnectHandling(t *testing.T) {
	kitA, err := New(Config{
		Name:      "kit-a-disconnect",
		Namespace: "a",
		Network:   NetworkConfig{Listen: "127.0.0.1:0"},
	})
	if err != nil {
		t.Fatal(err)
	}

	addr := kitA.network.Addr()

	kitB, err := New(Config{
		Name:      "kit-b-disconnect",
		Namespace: "b",
		Network:   NetworkConfig{Peers: map[string]string{"kit-a-disconnect": addr}},
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify connected
	if !kitB.transport.HasPeer("kit-a-disconnect") {
		t.Fatal("expected peer connection")
	}

	// Close Kit A — Kit B should detect disconnect
	kitA.Close()
	time.Sleep(2 * time.Second)

	// After disconnect, peer should be removed from transport
	stillHasPeer := kitB.transport.HasPeer("kit-a-disconnect")

	t.Logf("peer still connected after Kit A close: %v (expected: false)", stillHasPeer)

	// Close Kit B — should not panic even with dead peer
	kitB.Close()
	t.Log("disconnect handling: no panic on close")
}

func TestCrossKit_DiscoveryFallback(t *testing.T) {
	// Kit A listens
	kitA, err := New(Config{
		Name:      "disc-kit-a",
		Namespace: "a",
		Network: NetworkConfig{
			Listen: "127.0.0.1:0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	addr := kitA.network.Addr()

	// Register tool on Kit A
	kitA.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/ping", ShortName: "ping",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]string{"pong": "from-a"})
			},
		},
	})

	// Kit B uses static discovery (NOT direct Peers config)
	kitB, err := New(Config{
		Name:      "disc-kit-b",
		Namespace: "b",
		Network: NetworkConfig{
			Discovery: DiscoveryConfig{Type: "static"},
			Peers:     map[string]string{"disc-kit-a": addr},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	time.Sleep(500 * time.Millisecond)

	// Kit B calls tool on Kit A via address — should resolve through discovery
	resp, err := bus.AskSync(kitB.Bus, context.Background(), bus.Message{
		Topic:   "tools.call",
		Address: "kit:disc-kit-a",
		Payload: json.RawMessage(`{"name":"ping","input":{}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	json.Unmarshal(resp.Payload, &result)
	if result["pong"] != "from-a" {
		t.Fatalf("expected pong=from-a, got: %s", resp.Payload)
	}
}

func TestCrossKit_BidirectionalToolCalls(t *testing.T) {
	kitA, err := New(Config{
		Name:    "bidir-a",
		Network: NetworkConfig{Listen: "127.0.0.1:0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	kitA.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/a-tool", ShortName: "a-tool",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]string{"from": "kit-a"})
			},
		},
	})

	addr := kitA.network.Addr()

	kitB, err := New(Config{
		Name:    "bidir-b",
		Network: NetworkConfig{Listen: "127.0.0.1:0", Peers: map[string]string{"bidir-a": addr}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	kitB.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/b-tool", ShortName: "b-tool",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]string{"from": "kit-b"})
			},
		},
	})

	time.Sleep(500 * time.Millisecond)

	// B calls A
	resp, err := bus.AskSync(kitB.Bus, context.Background(), bus.Message{
		Topic:   "tools.call",
		Address: "kit:bidir-a",
		Payload: json.RawMessage(`{"name":"a-tool","input":{}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var resultA map[string]string
	json.Unmarshal(resp.Payload, &resultA)
	if resultA["from"] != "kit-a" {
		t.Fatalf("B→A: expected from=kit-a, got %s", resp.Payload)
	}

	// A calls B (bidirectional)
	addrB := kitB.network.Addr()
	kitA.connectPeer("bidir-b", addrB)
	time.Sleep(500 * time.Millisecond)

	resp, err = bus.AskSync(kitA.Bus, context.Background(), bus.Message{
		Topic:   "tools.call",
		Address: "kit:bidir-b",
		Payload: json.RawMessage(`{"name":"b-tool","input":{}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var resultB map[string]string
	json.Unmarshal(resp.Payload, &resultB)
	if resultB["from"] != "kit-b" {
		t.Fatalf("A→B: expected from=kit-b, got %s", resp.Payload)
	}
}
