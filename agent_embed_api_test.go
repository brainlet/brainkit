package brainkit_test

import (
	"github.com/brainlet/brainkit"
	agentembed "github.com/brainlet/brainkit/agent-embed"
)

var _ func(*brainkit.Kit, agentembed.AgentConfig) (*agentembed.Agent, error) = (*brainkit.Kit).CreateAgent
