package harness

import "time"

// buildJSConfig translates Go config to the JSON shape expected by createHarness().
func (h *Harness) buildJSConfig() map[string]any {
	modes := make([]map[string]any, len(h.config.Modes))
	for i, m := range h.config.Modes {
		modes[i] = map[string]any{
			"id":             m.ID,
			"name":           m.Name,
			"default":        m.Default,
			"defaultModelId": m.DefaultModelID,
			"color":          m.Color,
			"agentName":      m.AgentName,
		}
	}

	cfg := map[string]any{
		"id":    h.config.ID,
		"modes": modes,
	}

	if h.config.ResourceID != "" {
		cfg["resourceId"] = h.config.ResourceID
	}
	if h.config.StateSchema != nil {
		cfg["stateSchema"] = h.config.StateSchema
	}
	if h.config.InitialState != nil {
		cfg["initialState"] = h.config.InitialState
	}
	if len(h.config.Subagents) > 0 {
		subs := make([]map[string]any, len(h.config.Subagents))
		for i, s := range h.config.Subagents {
			subs[i] = map[string]any{
				"id":             s.ID,
				"allowedTools":   s.AllowedTools,
				"defaultModelId": s.DefaultModelID,
				"instructions":   s.Instructions,
			}
		}
		cfg["subagents"] = subs
	}
	if h.config.OMConfig != nil {
		cfg["omConfig"] = map[string]any{
			"defaultObserverModel":  h.config.OMConfig.DefaultObserverModel,
			"defaultReflectorModel": h.config.OMConfig.DefaultReflectorModel,
			"observationThreshold":  h.config.OMConfig.ObservationThreshold,
			"reflectionThreshold":   h.config.OMConfig.ReflectionThreshold,
		}
	}
	if len(h.config.Permissions) > 0 {
		perms := make(map[string]string, len(h.config.Permissions))
		for cat, pol := range h.config.Permissions {
			perms[string(cat)] = string(pol)
		}
		cfg["defaultPermissions"] = perms
	}
	if len(h.config.ToolCategories) > 0 {
		cats := make(map[string]string, len(h.config.ToolCategories))
		for tool, cat := range h.config.ToolCategories {
			cats[tool] = string(cat)
		}
		cfg["toolCategories"] = cats
	}

	return cfg
}

// startHeartbeats starts Go-side heartbeat timers.
func (h *Harness) startHeartbeats() {
	for _, hb := range h.config.HeartbeatHandlers {
		interval := time.Duration(hb.IntervalMs) * time.Millisecond
		if interval <= 0 {
			continue
		}
		ticker := time.NewTicker(interval)

		h.hbMu.Lock()
		h.heartbeats[hb.ID] = ticker
		h.hbMu.Unlock()

		handler := hb.Handler
		if hb.Immediate {
			go func() {
				defer func() { recover() }()
				handler()
			}()
		}

		go func(t *time.Ticker, fn func() error) {
			for range t.C {
				if h.closed {
					return
				}
				func() {
					defer func() { recover() }()
					fn()
				}()
			}
		}(ticker, handler)
	}
}

// stopHeartbeats stops all heartbeat timers and calls shutdown functions.
func (h *Harness) stopHeartbeats() {
	h.hbMu.Lock()
	defer h.hbMu.Unlock()

	for _, ticker := range h.heartbeats {
		ticker.Stop()
	}

	for _, hb := range h.config.HeartbeatHandlers {
		if hb.Shutdown != nil {
			func() {
				defer func() { recover() }()
				hb.Shutdown()
			}()
		}
	}

	h.heartbeats = make(map[string]*time.Ticker)
}
