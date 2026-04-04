package gateway

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestGateway(t *testing.T) {
	env := suite.Full(t)
	Run(t, env)
}
