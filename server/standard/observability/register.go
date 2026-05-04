// Package observability registers observability YAML module names.
//
// The audit and tracing standard factories intentionally link SQLite-backed
// default stores. Import this profile only in binaries that want those YAML
// module factories.
package observability

import (
	_ "github.com/brainlet/brainkit/modules/audit/standard"
	_ "github.com/brainlet/brainkit/modules/tracing/standard"
)
