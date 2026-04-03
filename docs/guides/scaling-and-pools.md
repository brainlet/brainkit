# Scaling and Pools

`InstanceManager` manages pools of Kit instances with shared tool registries and configurable scaling strategies.

## Creating a Pool

```go
im := kit.NewInstanceManager()

err := im.SpawnPool("workers", kit.PoolConfig{
    Base: kit.NodeConfig{
        Kernel: kit.KernelConfig{
            Namespace:    "workers",
            FSRoot: "/tmp/workers",
        },
        Messaging: kit.MessagingConfig{
            Transport: "nats",
            NATSURL:   "nats://localhost:4222",
        },
    },
    InitialCount: 3,     // start with 3 instances
    Min:          1,      // never scale below 1
    Max:          10,     // never scale above 10
    Strategy:     kit.NewThresholdStrategy(100, 10), // scale up at 100 pending, down at 10
})
```

Each instance is a full Node with its own Kernel (QuickJS runtime, tool registry). Instances in the same pool share a `ToolRegistry` — tools registered on one are visible on all.

## Manual Scaling

```go
// Scale up — add 2 instances
err := im.Scale("workers", 2)
// Pool now has 5 instances

// Scale down — remove 3 instances
err := im.Scale("workers", -3)
// Pool now has 2 instances

// Scale beyond count — clamped to 0
err := im.Scale("workers", -100)
// Pool now has 0 instances (not an error)
```

Scaling down closes instances in LIFO order (last spawned, first closed). Each close shuts down the full Kernel lifecycle — QuickJS freed, transport closed, goroutines stopped.

## Pool Info

```go
info, err := im.PoolInfo("workers")
// info.Name:    "workers"
// info.Current: 3
// info.Min:     1
// info.Max:     10

names := im.Pools()
// ["workers"]
```

## Kill Pool

```go
err := im.KillPool("workers")
```

Closes all instances and removes the pool. After this, `PoolInfo("workers")` returns `NotFoundError{Resource: "pool", Name: "workers"}`.

## Scaling Strategies

### StaticStrategy

Maintains a fixed instance count. If current differs from target, scales up or down.

```go
strategy := kit.NewStaticStrategy(5) // always 5 instances

// Current=3 → scale-up by 2
// Current=5 → no action
// Current=7 → scale-down by 2
```

### ThresholdStrategy

Scales based on pending message count. Steps up/down by 1 (configurable).

```go
strategy := kit.NewThresholdStrategy(100, 10)
// Scale UP when pending > 100
// Scale DOWN when pending < 10

// Customize steps
strategy := &kit.ThresholdStrategy{
    ScaleUpThreshold:   100,
    ScaleDownThreshold: 10,
    ScaleUpStep:        3,   // add 3 instances at a time
    ScaleDownStep:      1,   // remove 1 at a time
}
```

Respects Min/Max bounds:
- Won't scale above `PoolConfig.Max`
- Won't scale below `PoolConfig.Min`
- Delta is capped: if Max=5 and Current=4, step=3 → only 1 added

### Custom Strategy

Implement the `ScalingStrategy` interface:

```go
type ScalingStrategy interface {
    Evaluate(metrics messaging.MetricsSnapshot, pool PoolInfo) ScalingDecision
}

type ScalingDecision struct {
    Action string // "scale-up", "scale-down", "none"
    Delta  int    // how many to add/remove
    Reason string // human-readable reason (logged)
}
```

```go
type MyStrategy struct{}

func (s *MyStrategy) Evaluate(metrics messaging.MetricsSnapshot, pool kit.PoolInfo) kit.ScalingDecision {
    errorRate := float64(len(metrics.Errors)) / float64(max(len(metrics.Handled), 1))
    if errorRate > 0.1 && pool.Current < pool.Max {
        return kit.ScalingDecision{
            Action: "scale-up",
            Delta:  1,
            Reason: fmt.Sprintf("error rate %.1f%% > 10%%", errorRate*100),
        }
    }
    return kit.ScalingDecision{Action: "none"}
}
```

## EvaluateAndScale

Run the strategy evaluation loop manually:

```go
im.EvaluateAndScale()
```

This iterates all pools with a strategy, evaluates each one, and applies the decision. Call it periodically:

```go
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        im.EvaluateAndScale()
    }
}()
```

## Shared Tool Registry

All instances in a pool share a `ToolRegistry`. Register tools on the shared registry before spawning the pool:

```go
sharedTools := registry.New()

registry.Register(sharedTools, "process-order", registry.TypedTool[OrderInput]{
    Description: "processes an order",
    Execute:     processOrder,
})

err := im.SpawnPool("workers", kit.PoolConfig{
    Base: kit.NodeConfig{
        Kernel: kit.KernelConfig{
            SharedTools: sharedTools,
            // ...
        },
        // ...
    },
    InitialCount: 3,
})
```

The `process-order` tool is available on all 3 instances without re-registration. New instances added via `Scale(+1)` also get the shared registry.

## Error Types

```go
// Pool already exists
err := im.SpawnPool("workers", cfg)
err := im.SpawnPool("workers", cfg) // AlreadyExistsError{Resource: "pool", Name: "workers"}

// Pool not found
err := im.Scale("nonexistent", 1)   // NotFoundError{Resource: "pool", Name: "nonexistent"}
err := im.KillPool("nonexistent")   // NotFoundError{Resource: "pool", Name: "nonexistent"}
_, err := im.PoolInfo("nonexistent") // NotFoundError{Resource: "pool", Name: "nonexistent"}
```

## Memory Considerations

Each Kit instance has its own QuickJS heap (~256MB address space, ~50-80MB actual with Mastra bundle). A pool of 5 instances uses ~400MB actual memory.

For memory-constrained environments, consider using `SharedTools` to avoid duplicate tool registrations, and keep `Max` bounded.
