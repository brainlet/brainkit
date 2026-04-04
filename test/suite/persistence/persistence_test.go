package persistence

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestPersistence(t *testing.T) {
	env := suite.Full(t, suite.WithPersistence())
	Run(t, env)
}
