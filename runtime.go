package brainkit

import (
	_ "embed"
	"fmt"
)

//go:embed runtime/brainlet_runtime.js
var brainletRuntimeJS string

// loadRuntime evaluates brainlet-runtime.js in the sandbox.
// Must be called after registerBridges and after the agent-embed bundle is loaded.
func (s *Sandbox) loadRuntime() error {
	val, err := s.agents.Bridge().Eval("brainlet-runtime.js", brainletRuntimeJS)
	if err != nil {
		return fmt.Errorf("brainkit: load runtime: %w", err)
	}
	val.Free()
	return nil
}
