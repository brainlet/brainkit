package secrets

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func TestSecrets(t *testing.T) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	Run(t, env)
}
