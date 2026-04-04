package health

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestHealth(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
