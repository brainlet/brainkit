package scheduling

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestScheduling(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
