package fs

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestFS(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
