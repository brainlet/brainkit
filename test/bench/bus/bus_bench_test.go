package bus_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/bench"
	"github.com/brainlet/brainkit/test/bench/bus"
)

func BenchmarkBus(b *testing.B) {
	env := bench.NewEnv(b)
	bus.Run(b, env)
}
