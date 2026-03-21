package agentembed

import internalagent "github.com/brainlet/brainkit/internal/embed/agent"

type SandboxConfig = internalagent.SandboxConfig
type Sandbox = internalagent.Sandbox

func NewSandbox(cfg SandboxConfig) (*Sandbox, error) {
	return internalagent.NewSandbox(cfg)
}
