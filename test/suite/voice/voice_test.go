package voice

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestVoice(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
