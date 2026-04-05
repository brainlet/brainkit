# Bench Test Map

**Purpose:** Performance benchmarks for brainkit core operations.
**Pattern:** Each domain exports `Run(b *testing.B, env *bench.BenchEnv)`. Standalone `_bench_test.go` files create `bench.NewEnv(b)`.
**Env:** `env.go` creates a minimal Kernel with echo tool on memory transport.

## Standalone benchmarks (bench_test.go)

Top-level file with self-contained benchmarks that create their own kernel via `benchKernel(b)`.

| Benchmark | What it measures |
|-----------|-----------------|
| BenchmarkDeploy_1KB | Deploy + teardown cycle for a 1KB .ts handler |
| BenchmarkDeploy_10KB | Deploy + teardown cycle for a 10KB .ts handler (500 padding lines) |
| BenchmarkEvalTS_Trivial | EvalTS round-trip for `return "ok"` |
| BenchmarkEvalTS_JSONParse | EvalTS round-trip for JSON.parse + stringify of 1KB payload |
| BenchmarkBusRoundtrip | Full bus round-trip: SendToService -> subscribe -> receive reply |
| BenchmarkToolCall | Tool call round-trip: Publish ToolCallMsg -> subscribe ToolCallResp |
| BenchmarkPumpThroughput | Message pump throughput: SendToService -> subscribe -> receive |
| BenchmarkRestartRecovery/deployments=10 | Kernel restart recovery time with 10 persisted deployments |
| BenchmarkRestartRecovery/deployments=50 | Kernel restart recovery time with 50 persisted deployments |

## bus/

Domain benchmarks for bus operations. Entry: `BenchmarkBus` in `bus_bench_test.go`.

| Sub-benchmark | What it measures |
|--------------|-----------------|
| roundtrip | SendToService -> subscribe -> receive reply latency |
| tool_call | Publish ToolCallMsg -> subscribe ToolCallResp latency |
| pump_throughput | SendToService -> subscribe -> receive (same as roundtrip, named for throughput focus) |

## deploy/

Domain benchmarks for deploy operations. Entry: `BenchmarkDeploy` in `deploy_bench_test.go`.

| Sub-benchmark | What it measures |
|--------------|-----------------|
| deploy_1KB | Deploy + teardown cycle for 1KB handler |
| deploy_10KB | Deploy + teardown cycle for 10KB handler |
| restart_recovery/deployments=10 | Kernel restart + rehydration with 10 persisted services |
| restart_recovery/deployments=50 | Kernel restart + rehydration with 50 persisted services |

## eval/

Domain benchmarks for JS evaluation. Entry: `BenchmarkEval` in `eval_bench_test.go`.

| Sub-benchmark | What it measures |
|--------------|-----------------|
| trivial | EvalTS overhead for minimal expression |
| json_parse | EvalTS with JSON.parse + stringify of 1KB payload |

## Cross-references

- Domain benchmarks (`bus/`, `deploy/`, `eval/`) use the same `BenchEnv` pattern as suite `TestEnv`
- `bench_test.go` standalone benchmarks duplicate some domain tests (for direct `go test -bench` in root)
- All benchmarks run on memory transport (no containers needed)
- Echo tool registered in both `env.go` (domain) and `bench_test.go` (standalone)
