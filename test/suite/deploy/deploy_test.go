package deploy

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestDeploy(t *testing.T) {
	env := suite.Full(t, suite.WithPersistence())
	Run(t, env)
}
