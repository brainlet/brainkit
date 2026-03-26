package agentembed

import "fmt"

// ErrSandboxClosed is returned when an operation is attempted on a closed sandbox.
var ErrSandboxClosed = fmt.Errorf("agent-embed: sandbox is closed")

// ErrAgentClosed is returned when an operation is attempted on a closed agent.
var ErrAgentClosed = fmt.Errorf("agent-embed: agent is closed")
