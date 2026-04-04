package cross

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestCross(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
