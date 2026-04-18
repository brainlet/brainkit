// Command harness-lite demonstrates the frozen surface of
// modules/harness: NewModule wiring, the Instance interface, and
// the six frozen Event types consumers can rely on across releases.
//
// The harness module is WIP — only the Instance interface and the
// Event / EventType set are stable. Everything else in the package
// (HarnessConfig fields, DisplayState, internal events, subagents,
// observational memory) may move without deprecation. See
// modules/harness/README.md + designs/09-harness-boundary.md.
//
// This example builds a Kit with a minimal harness.Config, resolves
// the Instance, subscribes to events, and issues a SendMessage. If
// the underlying JS Harness backend is not wired in this build, the
// boot step returns an error — the example catches that and prints
// a clear message so readers see the frozen Go-side contract even
// when the JS side is still landing.
//
// Run from the repo root:
//
//	go run ./examples/harness-lite
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/harness"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("harness-lite: %v", err)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "brainkit-harness-lite-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// Minimal HarnessConfig: one default mode pointing at an agent
	// name. `demo-agent` is not registered in this example — that
	// would require a provider + .ts deploy — but the config is
	// valid enough to exercise the Module wiring and the Instance
	// surface shape.
	harnessMod := harness.NewModule(harness.Config{
		Harness: harness.HarnessConfig{
			ID: "harness-lite-demo",
			Modes: []harness.ModeConfig{{
				ID:        "build",
				Name:      "Build",
				Default:   true,
				AgentName: "demo-agent",
			}},
			Permissions: harness.DefaultPermissions(),
		},
	})

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "harness-lite-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Modules:   []brainkit.Module{harnessMod},
	})
	if err != nil {
		if isHarnessJSMissing(err) {
			fmt.Println("Harness JS backend is not wired in this build.")
			fmt.Println("The frozen Go-side contract (Module / Instance / Event) still compiles:")
			fmt.Printf("  module name        : %s\n", harnessMod.Name())
			fmt.Printf("  module status      : %s (WIP)\n", harnessMod.Status())
			fmt.Println("  Instance interface : SendMessage / Abort / Steer / FollowUp / Subscribe / CurrentThread / CurrentMode / Close")
			fmt.Println("  frozen event types : agent_start, agent_end, message_update, tool_start, tool_end, error")
			fmt.Println()
			fmt.Printf("  boot error         : %v\n", err)
			return nil
		}
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	inst := harnessMod.Instance()
	if inst == nil {
		fmt.Println("harness Module initialized without a JS runtime — Instance() is nil.")
		fmt.Println("This example needs a Kit with the JS bridge to exercise SendMessage.")
		return nil
	}

	// Subscribe to the six frozen event types. Any event outside
	// the frozen set flows through the same callback with a raw
	// internal type string — treat those as opaque.
	var seen atomic.Int64
	done := make(chan struct{}, 1)
	unsubscribe := inst.Subscribe(func(ev harness.Event) {
		seen.Add(1)
		label := classify(ev.Type)
		fmt.Printf("  [%s] %s\n", label, truncatePayload(string(ev.Payload), 120))
		if ev.Type == harness.EvAgentEnd || ev.Type == harness.EvError {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})
	defer unsubscribe()

	fmt.Printf("harness Instance wired — mode=%s thread=%s\n",
		inst.CurrentMode(), inst.CurrentThread())
	fmt.Println("sending: hello world")

	if err := inst.SendMessage("hello world"); err != nil {
		fmt.Printf("SendMessage returned: %v\n", err)
		fmt.Println("(expected when the referenced agent is not deployed; the event subscription still observed any events emitted before the error)")
	}

	// Wait briefly for terminal event — if the JS backend replies
	// synchronously, this typically returns immediately.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-ctx.Done():
	}

	fmt.Printf("\nevents observed: %d\n", seen.Load())
	return nil
}

// isHarnessJSMissing matches the error surfaced when the JS-side
// createHarness / init call is absent. Kept as a string match
// because the wrapper error text is the only stable signal — the
// WIP JS shim may or may not be compiled into this build.
func isHarnessJSMissing(err error) bool {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "harness: create JS harness"),
		strings.Contains(msg, "harness: init:"),
		strings.Contains(msg, "__kit.createHarness"),
		strings.Contains(msg, "createHarness is not a function"):
		return true
	}
	return false
}

// classify labels an event by its frozen type. Unknown types pass
// through with their raw internal name.
func classify(t harness.EventType) string {
	switch t {
	case harness.EvAgentStart:
		return "agent_start"
	case harness.EvAgentEnd:
		return "agent_end"
	case harness.EvMessageDelta:
		return "message_update"
	case harness.EvToolStart:
		return "tool_start"
	case harness.EvToolEnd:
		return "tool_end"
	case harness.EvError:
		return "error"
	default:
		return "(non-frozen: " + string(t) + ")"
	}
}

func truncatePayload(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
