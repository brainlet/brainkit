package bus

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestBus(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
