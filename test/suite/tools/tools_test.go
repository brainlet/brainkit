package tools

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestTools(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
