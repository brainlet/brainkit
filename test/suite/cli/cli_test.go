package cli

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestCLI(t *testing.T) {
	env := suite.Full(t, suite.WithPersistence())
	Run(t, env)
}
