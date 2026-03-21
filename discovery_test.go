package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
	transportpkg "github.com/brainlet/brainkit/transport"
)

func TestStaticDiscovery_ResolveAndBrowse(t *testing.T) {
	d := transportpkg.NewStaticDiscovery(map[string]string{
		"server-1": "10.0.1.1:9090",
		"server-2": "10.0.1.2:9090",
	})
	defer d.Close()

	addr, err := d.Resolve("server-1")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "10.0.1.1:9090" {
		t.Errorf("resolve = %q", addr)
	}

	_, err = d.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent peer")
	}

	peers, _ := d.Browse()
	if len(peers) != 2 {
		t.Errorf("browse = %d peers, want 2", len(peers))
	}
}

func TestStaticDiscovery_Register(t *testing.T) {
	d := transportpkg.NewStaticDiscovery(nil)
	defer d.Close()

	d.Register(transportpkg.Peer{Name: "new-peer", Address: "10.0.1.3:9090"})

	addr, err := d.Resolve("new-peer")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "10.0.1.3:9090" {
		t.Errorf("resolve = %q", addr)
	}
}

func TestMulticastDiscovery_AnnounceAndDiscover(t *testing.T) {
	d1, err := transportpkg.NewMulticastDiscovery("_test._tcp")
	if err != nil {
		t.Skipf("multicast not available: %v", err)
	}
	defer d1.Close()

	d2, err := transportpkg.NewMulticastDiscovery("_test._tcp")
	if err != nil {
		t.Skipf("multicast not available: %v", err)
	}
	defer d2.Close()

	d1.Register(transportpkg.Peer{Name: "kit-1", Address: "127.0.0.1:9001"})

	time.Sleep(3 * time.Second)

	addr, err := d2.Resolve("kit-1")
	if err != nil {
		t.Fatalf("d2 did not discover kit-1: %v", err)
	}
	if addr != "127.0.0.1:9001" {
		t.Errorf("addr = %q", addr)
	}
}

func TestGRPCTransport_DiscoveryResolve(t *testing.T) {
	// Kit A listens
	kitA, err := New(Config{
		Name:      "disc-kit-a",
		Namespace: "a",
		Network:   NetworkConfig{Listen: "127.0.0.1:0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	addr := kitA.network.Addr()

	// Register a tool on Kit A
	kitA.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/add", ShortName: "add",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(_ context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var in struct{ A, B float64 }
				json.Unmarshal(input, &in)
				return json.Marshal(map[string]float64{"result": in.A + in.B})
			},
		},
	})

	// Kit B uses static discovery (not Peers map)
	kitB, err := New(Config{
		Name:      "disc-kit-b",
		Namespace: "b",
		Network: NetworkConfig{
			Discovery: DiscoveryConfig{
				Type: "static",
			},
			Peers: map[string]string{"disc-kit-a": addr},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	// Forward should discover and connect to Kit A
	resp, err := bus.AskSync(kitB.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Address:  "kit:disc-kit-a",
		Payload:  json.RawMessage(`{"name":"add","input":{"a":5,"b":3}}`),
	})
	if err != nil {
		t.Fatalf("discovery tool call: %v", err)
	}

	var result map[string]float64
	json.Unmarshal(resp.Payload, &result)
	if result["result"] != 8 {
		t.Errorf("expected 8, got %v (payload: %s)", result["result"], resp.Payload)
	}
}
