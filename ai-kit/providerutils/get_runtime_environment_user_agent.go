// Ported from: packages/provider-utils/src/get-runtime-environment-user-agent.ts
package providerutils

import "runtime"

// GetRuntimeEnvironmentUserAgent returns a user agent string describing the Go runtime.
// In the TypeScript SDK this detects browser/node/bun/deno/edge; in Go we always report Go.
func GetRuntimeEnvironmentUserAgent() string {
	return "runtime/go/" + runtime.Version()
}
