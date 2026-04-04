package deploy_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/bench"
	"github.com/brainlet/brainkit/test/bench/deploy"
)

func BenchmarkDeploy(b *testing.B) {
	env := bench.NewEnv(b)
	deploy.Run(b, env)
}
