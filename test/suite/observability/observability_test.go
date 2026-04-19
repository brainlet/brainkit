package observability

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestObservability(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
