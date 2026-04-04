package packages

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("packages", func(t *testing.T) {
		t.Run("multi_file_project", func(t *testing.T) { testMultiFileProject(t, env) })
		t.Run("list_and_teardown", func(t *testing.T) { testListAndTeardown(t, env) })
		t.Run("secret_dependency_check", func(t *testing.T) { testSecretDependencyCheck(t, env) })
	})
}
