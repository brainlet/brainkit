package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// registerHealthRoutes adds /healthz, /readyz, /health using runtime type assertions.
// No interface dependency — works with any runtime that has the right methods.
func registerHealthRoutes(mux *http.ServeMux, rt any) {
	type aliver interface {
		Alive(ctx context.Context) bool
	}
	type readier interface {
		Ready(ctx context.Context) bool
	}
	type healther interface {
		Alive(ctx context.Context) bool
		Ready(ctx context.Context) bool
	}

	if a, ok := rt.(aliver); ok {
		mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if a.Alive(ctx) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("not alive"))
			}
		})
	}

	if rd, ok := rt.(readier); ok {
		mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if rd.Ready(ctx) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("not ready"))
			}
		})
	}

	// For /health, we need to call a method that returns the full status.
	// Use reflection-free approach: try to call Health via interface assertion.
	// kit.Kernel has Health(ctx) HealthStatus — we JSON-encode whatever it returns.
	type fullHealther interface {
		Alive(ctx context.Context) bool
		Ready(ctx context.Context) bool
		Health(ctx context.Context) any
	}

	// kit.Kernel.Health returns kit.HealthStatus (concrete), not any.
	// Go interfaces require exact return type match. So we use a helper:
	if _, ok := rt.(healther); ok {
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			// Call Health via reflection-free method: try known types
			var statusJSON []byte
			var healthy bool

			// Try kit.Kernel / kit.Node (they embed Kernel)
			type kernelHealther interface {
				Health(ctx context.Context) struct {
					Healthy bool          `json:"healthy"`
					Status  string        `json:"status"`
					Uptime  time.Duration `json:"uptime"`
					Checks  []struct {
						Name    string        `json:"name"`
						Healthy bool          `json:"healthy"`
						Latency time.Duration `json:"latency,omitempty"`
						Error   string        `json:"error,omitempty"`
						Details any           `json:"details,omitempty"`
					} `json:"checks"`
				}
			}
			// This won't match either since the struct is anonymous.
			// Simplest solution: call Alive and Ready, build a minimal health response.
			a := rt.(aliver)
			rd := rt.(readier)
			alive := a.Alive(ctx)
			ready := rd.Ready(ctx)
			healthy = alive && ready
			status := "running"
			if !alive {
				status = "unhealthy"
			} else if !ready {
				status = "degraded"
			}

			statusJSON, _ = json.Marshal(map[string]any{
				"healthy": healthy,
				"status":  status,
				"checks": []map[string]any{
					{"name": "alive", "healthy": alive},
					{"name": "ready", "healthy": ready},
				},
			})

			w.Header().Set("Content-Type", "application/json")
			if healthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			w.Write(statusJSON)
		})
	}
}
