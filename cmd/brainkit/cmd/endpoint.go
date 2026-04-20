package cmd

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// defaultEndpoint is the final fallback when nothing else resolves.
// Mirrors the gateway default listen addr so a vanilla
// `brainkit start` + `brainkit call` round-trip works with zero
// config on the same machine.
const defaultEndpoint = "http://127.0.0.1:8080"

// resolveEndpoint picks the best URL for the gateway's /api/bus +
// /api/stream endpoints, in priority order:
//
//  1. Explicit `--endpoint / -e` flag  (highest; user intent wins)
//  2. BRAINKIT_ENDPOINT env var         (scripting / CI)
//  3. gateway.listen in ./brainkit.yaml (or --config path)
//  4. http://127.0.0.1:8080             (historic default)
//
// Step 3 reads the same yaml `brainkit start` reads, so running a
// second CLI in the same directory auto-targets the local server
// without a flag — even when `gateway.listen` is non-default.
func resolveEndpoint(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if v, ok := os.LookupEnv("BRAINKIT_ENDPOINT"); ok && v != "" {
		return v
	}
	if ep := endpointFromConfig(cfgFile); ep != "" {
		return ep
	}
	return defaultEndpoint
}

// endpointFromConfig reads gateway.listen out of the yaml at
// `configPath` (or `./brainkit.yaml` when empty) and builds an
// endpoint URL. Returns empty string on any failure — a missing
// yaml is the common case and shouldn't log or error.
//
// `gateway.listen` may be `:8080`, `0.0.0.0:8080`, `127.0.0.1:8080`,
// or `[::]:8080`. For the CLI we always target localhost — the
// listen addr's port is the only load-bearing component.
func endpointFromConfig(configPath string) string {
	path := configPath
	if path == "" {
		path = "brainkit.yaml"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var probe struct {
		Gateway struct {
			Listen string `yaml:"listen"`
		} `yaml:"gateway"`
	}
	if err := yaml.Unmarshal(raw, &probe); err != nil {
		return ""
	}
	listen := strings.TrimSpace(probe.Gateway.Listen)
	if listen == "" {
		return ""
	}
	port := listenPort(listen)
	if port == "" {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port)
}

// listenPort extracts the port from a listen spec. Handles:
//
//	":8080"          → "8080"
//	"127.0.0.1:8080" → "8080"
//	"0.0.0.0:8080"   → "8080"
//	"[::]:8080"      → "8080"
func listenPort(listen string) string {
	// Strip trailing brackets for IPv6 forms like `[::]:8080`.
	if i := strings.LastIndex(listen, "]:"); i != -1 {
		return listen[i+2:]
	}
	if i := strings.LastIndex(listen, ":"); i != -1 {
		return listen[i+1:]
	}
	return ""
}
