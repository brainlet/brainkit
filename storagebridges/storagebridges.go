// Package storagebridges links all optional runtime storage bridge backends.
//
// Import backend-specific packages such as storagebridges/sqlite when a binary
// should link only one bridge backend. This aggregate package intentionally
// links every storage bridge backend for convenience.
package storagebridges

import (
	_ "github.com/brainlet/brainkit/storagebridges/sqlite"
)
