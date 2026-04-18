// Package harness is the brainkit.Module wrapper around the Harness
// agent-orchestration layer.
//
// Status: WIP. The inner Harness surface (types.go, harness.go,
// config.go, display.go) is subject to churn. The frozen minimum
// every consumer can rely on is the Instance interface in
// instance.go, plus the Event/EventType types it carries. Everything
// else may move without deprecation.
//
// Example:
//
//	harn := harness.NewModule(harness.Config{Harness: harness.HarnessConfig{...}})
//	kit, _ := brainkit.New(brainkit.Config{Modules: []brainkit.Module{harn}})
//	inst := harn.Instance()
//	if inst != nil {
//	    unsubscribe := inst.Subscribe(func(ev harness.Event) { ... })
//	    _ = inst.SendMessage("hello")
//	}
//
// See README.md for the frozen event list and the WIP boundary.
package harness
