package workflows

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestWorkflows(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
