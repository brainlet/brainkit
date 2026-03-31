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

	// /health — full health status. Uses HealthJSON interface to avoid
	// concrete type dependency on kit.HealthStatus.
	type healthJSONer interface {
		HealthJSON(ctx context.Context) json.RawMessage
	}

	if hj, ok := rt.(healthJSONer); ok {
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			data := hj.HealthJSON(ctx)
			w.Header().Set("Content-Type", "application/json")
			var parsed struct{ Healthy bool }
			json.Unmarshal(data, &parsed)
			if parsed.Healthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			w.Write(data)
		})
	} else if _, ok := rt.(healther); ok {
		// Fallback: alive + ready only (runtime without HealthJSON)
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			a := rt.(aliver)
			rd := rt.(readier)
			alive := a.Alive(ctx)
			ready := rd.Ready(ctx)
			healthy := alive && ready
			status := "running"
			if !alive {
				status = "unhealthy"
			} else if !ready {
				status = "degraded"
			}
			statusJSON, _ := json.Marshal(map[string]any{
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
