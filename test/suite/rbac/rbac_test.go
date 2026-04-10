package rbac

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestRBAC(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
