package kit

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/brainlet/brainkit/sdk/messages"
)

// HealthStatus is the full health report from Kernel.Health().
type HealthStatus struct {
	Healthy bool          `json:"healthy"`
	Status  string        `json:"status"` // "running", "starting", "draining", "degraded", "unhealthy"
	Uptime  time.Duration `json:"uptime"`
	Checks  []HealthCheck `json:"checks"`
}

// HealthCheck is a single health check result.
type HealthCheck struct {
	Name    string        `json:"name"`
	Healthy bool          `json:"healthy"`
	Latency time.Duration `json:"latency,omitempty"`
	Error   string        `json:"error,omitempty"`
	Details any           `json:"details,omitempty"`
}

// Alive returns true if the QuickJS runtime can evaluate a trivial expression.
// Used as a Kubernetes liveness probe — fast, checks only the critical path.
func (k *Kernel) Alive(ctx context.Context) bool {
	if k.startedAt.IsZero() {
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := k.EvalTS(checkCtx, "__alive.ts", `return "ok"`)
	return err == nil && result == "ok"
}

// Ready returns true if the Kernel can serve traffic.
// Used as a Kubernetes readiness probe — checks runtime, transport, and drain state.
func (k *Kernel) Ready(ctx context.Context) bool {
	if k.IsDraining() || k.startedAt.IsZero() {
		return false
	}
	health := k.Health(ctx)
	return health.Status == "running" || health.Status == "degraded"
}

// Health returns a comprehensive health status with all checks.
func (k *Kernel) Health(ctx context.Context) HealthStatus {
	var checks []HealthCheck

	checks = append(checks, k.checkRuntime(ctx))
	checks = append(checks, k.checkTransport(ctx))

	// Providers — cached results from periodic probing (no live HTTP)
	// Only report providers that have been probed at least once.
	// Unprobed providers are not unhealthy — they just haven't been checked yet.
	for _, p := range k.providers.ListAIProviders() {
		if p.LastProbed.IsZero() {
			checks = append(checks, HealthCheck{
				Name:    "provider:" + p.Name,
				Healthy: true, // assume healthy until probed
				Details: "not yet probed",
			})
		} else {
			checks = append(checks, HealthCheck{
				Name:    "provider:" + p.Name,
				Healthy: p.Healthy,
				Latency: p.Latency,
				Error:   p.LastError,
			})
		}
	}

	// Embedded storage bridges — HTTP health check
	k.mu.Lock()
	storageNames := make([]string, 0, len(k.storages))
	for name := range k.storages {
		storageNames = append(storageNames, name)
	}
	k.mu.Unlock()
	for _, name := range storageNames {
		checks = append(checks, k.checkStorage(ctx, name))
	}

	// Informational
	checks = append(checks, HealthCheck{
		Name:    "deployments",
		Healthy: true,
		Details: map[string]int{"active": len(k.ListDeployments())},
	})
	checks = append(checks, HealthCheck{
		Name:    "schedules",
		Healthy: true,
		Details: map[string]int{"active": len(k.ListSchedules())},
	})

	// Aggregate
	healthy := true
	for _, c := range checks {
		if !c.Healthy {
			healthy = false
		}
	}

	status := "running"
	if k.startedAt.IsZero() {
		status = "starting"
	} else if k.IsDraining() {
		status = "draining"
	} else if !healthy {
		if criticalCheckFailed(checks) {
			status = "unhealthy"
		} else {
			status = "degraded"
		}
	}

	uptime := time.Duration(0)
	if !k.startedAt.IsZero() {
		uptime = time.Since(k.startedAt)
	}

	return HealthStatus{
		Healthy: healthy,
		Status:  status,
		Uptime:  uptime,
		Checks:  checks,
	}
}

func (k *Kernel) checkRuntime(ctx context.Context) HealthCheck {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	result, err := k.EvalTS(checkCtx, "__health_check.ts", `return "ok"`)
	if err != nil || result != "ok" {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		return HealthCheck{Name: "runtime", Healthy: false, Error: errMsg, Latency: time.Since(start)}
	}
	return HealthCheck{Name: "runtime", Healthy: true, Latency: time.Since(start)}
}

func (k *Kernel) checkTransport(ctx context.Context) HealthCheck {
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	probeTopic := "health.probe." + uuid.NewString()
	received := make(chan bool, 1)

	unsub, err := k.SubscribeRaw(checkCtx, probeTopic, func(msg messages.Message) {
		select {
		case received <- true:
		default:
		}
	})
	if err != nil {
		return HealthCheck{Name: "transport", Healthy: false, Error: err.Error()}
	}
	defer unsub()

	k.PublishRaw(checkCtx, probeTopic, []byte(`{"probe":true}`))

	select {
	case <-received:
		return HealthCheck{Name: "transport", Healthy: true, Latency: time.Since(start)}
	case <-checkCtx.Done():
		return HealthCheck{Name: "transport", Healthy: false, Error: "probe timeout", Latency: time.Since(start)}
	}
}

func (k *Kernel) checkStorage(ctx context.Context, name string) HealthCheck {
	start := time.Now()
	url := k.StorageURL(name)
	if url == "" {
		// InMemory or non-bridge storage — always healthy
		return HealthCheck{Name: "storage:" + name, Healthy: true, Details: "in-memory"}
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/health")
	if err != nil {
		return HealthCheck{Name: "storage:" + name, Healthy: false, Error: err.Error(), Latency: time.Since(start)}
	}
	resp.Body.Close()

	return HealthCheck{Name: "storage:" + name, Healthy: resp.StatusCode == 200, Latency: time.Since(start)}
}

func criticalCheckFailed(checks []HealthCheck) bool {
	for _, c := range checks {
		if !c.Healthy && (c.Name == "runtime" || c.Name == "transport") {
			return true
		}
	}
	return false
}

// HealthJSON returns the full health status as JSON.
// Used by the gateway to avoid concrete type dependency.
func (k *Kernel) HealthJSON(ctx context.Context) json.RawMessage {
	status := k.Health(ctx)
	data, _ := json.Marshal(status)
	return data
}
