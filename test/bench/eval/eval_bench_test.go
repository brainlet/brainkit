package eval_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/bench"
	"github.com/brainlet/brainkit/test/bench/eval"
)

func BenchmarkEval(b *testing.B) {
	env := bench.NewEnv(b)
	eval.Run(b, env)
}
