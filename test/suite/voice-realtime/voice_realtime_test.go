package voicerealtime

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestVoiceRealtime(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
