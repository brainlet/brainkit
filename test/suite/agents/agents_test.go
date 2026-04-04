package agents

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestAgents(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
