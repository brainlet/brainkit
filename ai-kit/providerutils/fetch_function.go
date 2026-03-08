// Ported from: packages/provider-utils/src/fetch-function.ts
package providerutils

import "net/http"

// FetchFunction is the standardized HTTP fetch function type.
// In TypeScript this was typeof globalThis.fetch; in Go we use http.Client.Do().
type FetchFunction func(req *http.Request) (*http.Response, error)
