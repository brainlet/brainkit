package agentembed

import (
	"testing"

	internalagent "github.com/brainlet/brainkit/internal/embed/agent"
)

var (
	_ Tool                                  = internalagent.Tool{}
	_ ToolContext                           = internalagent.ToolContext{}
	_ AgentConfig                           = internalagent.AgentConfig{}
	_ GenerateParams                        = internalagent.GenerateParams{}
	_ StreamParams                          = internalagent.StreamParams{}
	_ SandboxConfig                         = internalagent.SandboxConfig{}
	_ ClientConfig                          = internalagent.ClientConfig{}
	_ func(SandboxConfig) (*Sandbox, error) = NewSandbox
	_ func(ClientConfig) *Client            = NewClient
)

func TestFacadeTypeChecks(t *testing.T) {}
