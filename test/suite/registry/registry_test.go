package registry

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestRegistry(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
