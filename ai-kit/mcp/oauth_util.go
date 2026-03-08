// Ported from: packages/mcp/src/util/oauth-util.ts
package mcp

import (
	"net/url"
	"strings"
)

// ResourceURLFromServerURL converts a server URL to a resource URL by removing
// the fragment. RFC 8707 section 2 states that resource URIs "MUST NOT include
// a fragment component". Keeps everything else unchanged (scheme, domain, port,
// path, query).
func ResourceURLFromServerURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	u.Fragment = "" // Remove fragment
	return u, nil
}

// CheckResourceAllowed checks if a requested resource URL matches a configured
// resource URL. A requested resource matches if it has the same scheme, domain,
// port, and its path starts with the configured resource's path.
func CheckResourceAllowed(requestedResource, configuredResource string) (bool, error) {
	requested, err := url.Parse(requestedResource)
	if err != nil {
		return false, err
	}
	configured, err := url.Parse(configuredResource)
	if err != nil {
		return false, err
	}

	// Compare the origin (scheme + host which includes port)
	requestedOrigin := requested.Scheme + "://" + requested.Host
	configuredOrigin := configured.Scheme + "://" + configured.Host
	if requestedOrigin != configuredOrigin {
		return false, nil
	}

	// Handle cases like requested=/foo and configured=/foo/
	if len(requested.Path) < len(configured.Path) {
		return false, nil
	}

	// Ensure both paths end with / for proper comparison
	requestedPath := requested.Path
	if !strings.HasSuffix(requestedPath, "/") {
		requestedPath += "/"
	}
	configuredPath := configured.Path
	if !strings.HasSuffix(configuredPath, "/") {
		configuredPath += "/"
	}

	return strings.HasPrefix(requestedPath, configuredPath), nil
}
