package stress

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestStress(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
