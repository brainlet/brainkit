package security

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestSecurity(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
