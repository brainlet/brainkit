package envelope

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestEnvelope(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
