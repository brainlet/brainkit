# Benchmarks

Read TEST_MAP.md before editing.

Same TestEnv pattern as suite tests but for `*testing.B`.

## Pattern

Each domain exports `Run(b *testing.B, env *bench.BenchEnv)`:
- `bench.go` — exported `Run()` with sub-benchmarks
- `<domain>_bench_test.go` — standalone entry

## Adding a benchmark

```go
// In bench/<domain>/bench.go
func Run(b *testing.B, env *bench.BenchEnv) {
    b.Run("my_operation", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            // ... operation to benchmark
        }
    })
}
```

## Running

```bash
go test -bench . ./test/bench/...           # all benchmarks
go test -bench BenchmarkBus ./test/bench/bus/  # single domain
```

## Domains

- `bus/` — roundtrip, tool call, pump throughput
- `deploy/` — deploy 1KB/10KB, restart recovery
- `eval/` — trivial eval, JSON parse
