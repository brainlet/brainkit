package tracing

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestTracing(t *testing.T) {
	env := suite.Full(t, suite.WithTracing(), suite.WithPersistence())
	Run(t, env)
}
